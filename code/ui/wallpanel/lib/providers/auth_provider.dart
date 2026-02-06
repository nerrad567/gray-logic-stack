import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/auth.dart';
import '../repositories/auth_repository.dart';
import '../services/api_client.dart';
import '../services/token_storage.dart';

/// Provides the singleton TokenStorage instance.
final tokenStorageProvider = Provider<TokenStorage>((ref) {
  return TokenStorage();
});

/// Provides the singleton ApiClient instance.
final apiClientProvider = Provider<ApiClient>((ref) {
  final tokenStorage = ref.watch(tokenStorageProvider);
  return ApiClient(tokenStorage: tokenStorage);
});

/// Provides the AuthRepository.
final authRepositoryProvider = Provider<AuthRepository>((ref) {
  return AuthRepository(
    apiClient: ref.watch(apiClientProvider),
    tokenStorage: ref.watch(tokenStorageProvider),
  );
});

/// Manages authentication state.
final authProvider = NotifierProvider<AuthNotifier, AuthState>(
  AuthNotifier.new,
);

class AuthNotifier extends Notifier<AuthState> {
  @override
  AuthState build() => const AuthState();

  AuthRepository get _authRepo => ref.read(authRepositoryProvider);
  ApiClient get _apiClient => ref.read(apiClientProvider);
  TokenStorage get _tokenStorage => ref.read(tokenStorageProvider);

  /// Attempt to restore a previous session from stored token.
  Future<void> restoreSession() async {
    final hasToken = await _authRepo.hasToken();
    if (!hasToken) return;

    final coreUrl = await _tokenStorage.getCoreUrl();
    if (coreUrl == null) return;

    await _apiClient.configure(coreUrl);

    // Verify the token is still valid via health check
    final healthy = await _apiClient.healthCheck();
    if (healthy) {
      final token = await _authRepo.getToken();
      state = AuthState(
        status: AuthStatus.authenticated,
        token: token,
      );
    } else {
      await _authRepo.logout();
    }
  }

  /// Configure the API client with the Core URL and login.
  Future<void> login(String coreUrl, String username, String password) async {
    state = state.copyWith(status: AuthStatus.authenticating);

    try {
      await _apiClient.configure(coreUrl);
      await _tokenStorage.setCoreUrl(coreUrl);

      final response = await _authRepo.login(username, password);

      state = AuthState(
        status: AuthStatus.authenticated,
        token: response.accessToken,
        tokenExpiresAt: DateTime.now().add(
          Duration(seconds: response.expiresIn),
        ),
      );
    } catch (e) {
      state = AuthState(
        status: AuthStatus.error,
        errorMessage: _formatError(e),
      );
    }
  }

  /// Logout: revoke server session, then clear stored credentials.
  Future<void> logout() async {
    await _authRepo.serverLogout();
    state = const AuthState();
  }

  /// Force re-login: clear local state without server call (used after password change).
  void forceRelogin() {
    state = const AuthState();
  }

  String _formatError(dynamic error) {
    if (error.toString().contains('401')) return 'Invalid credentials';
    if (error.toString().contains('SocketException')) return 'Cannot reach Core';
    if (error.toString().contains('TimeoutException')) return 'Connection timed out';
    return 'Login failed';
  }
}
