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

/// Convenience provider that extracts the identity from auth state.
/// Returns null when not authenticated or identity hasn't loaded yet.
final identityProvider = Provider<Identity?>((ref) {
  return ref.watch(authProvider).identity;
});

class AuthNotifier extends Notifier<AuthState> {
  @override
  AuthState build() => const AuthState();

  AuthRepository get _authRepo => ref.read(authRepositoryProvider);
  ApiClient get _apiClient => ref.read(apiClientProvider);
  TokenStorage get _tokenStorage => ref.read(tokenStorageProvider);

  /// Attempt to restore a previous session from stored credentials.
  /// Works for both user (JWT) and panel (X-Panel-Token) auth modes.
  Future<void> restoreSession() async {
    final mode = await _tokenStorage.getAuthMode();
    final coreUrl = await _tokenStorage.getCoreUrl();
    if (coreUrl == null) return;

    if (mode == 'panel') {
      // Panel mode: check for stored panel token
      final panelToken = await _tokenStorage.getPanelToken();
      if (panelToken == null || panelToken.isEmpty) return;

      await _apiClient.configure(coreUrl);

      final healthy = await _apiClient.healthCheck();
      if (!healthy) return;

      // Fetch identity to verify token is still valid
      try {
        final identity = await _apiClient.getMe();
        state = AuthState(
          status: AuthStatus.authenticated,
          identity: identity,
        );
      } catch (_) {
        // Token invalid — stay unauthenticated
        await _tokenStorage.clearPanelToken();
      }
    } else {
      // User mode: check for stored JWT
      final hasToken = await _authRepo.hasToken();
      if (!hasToken) return;

      await _apiClient.configure(coreUrl);

      final healthy = await _apiClient.healthCheck();
      if (!healthy) {
        await _authRepo.logout();
        return;
      }

      final token = await _authRepo.getToken();

      // Fetch identity
      Identity? identity;
      try {
        identity = await _apiClient.getMe();
      } catch (_) {
        // Identity fetch failed — still authenticate with token
      }

      state = AuthState(
        status: AuthStatus.authenticated,
        token: token,
        identity: identity,
      );
    }
  }

  /// Configure the API client with the Core URL and login with user credentials.
  Future<void> login(String coreUrl, String username, String password) async {
    state = state.copyWith(status: AuthStatus.authenticating);

    try {
      await _apiClient.configure(coreUrl);
      await _tokenStorage.setCoreUrl(coreUrl);
      await _tokenStorage.setAuthMode('user');

      final response = await _authRepo.login(username, password);

      // Fetch identity after successful login
      Identity? identity;
      try {
        identity = await _apiClient.getMe();
      } catch (_) {
        // Non-fatal: identity enriches UI but isn't required for auth
      }

      state = AuthState(
        status: AuthStatus.authenticated,
        token: response.accessToken,
        tokenExpiresAt: DateTime.now().add(
          Duration(seconds: response.expiresIn),
        ),
        identity: identity,
      );
    } catch (e) {
      state = AuthState(
        status: AuthStatus.error,
        errorMessage: _formatError(e),
      );
    }
  }

  /// Authenticate as a panel device using a panel token.
  Future<void> panelLogin(String coreUrl, String panelToken) async {
    state = state.copyWith(status: AuthStatus.authenticating);

    try {
      await _apiClient.configure(coreUrl);
      await _tokenStorage.setCoreUrl(coreUrl);
      await _tokenStorage.setAuthMode('panel');
      await _tokenStorage.setPanelToken(panelToken);

      // Verify token is valid via health check + identity fetch
      final healthy = await _apiClient.healthCheck();
      if (!healthy) {
        state = const AuthState(
          status: AuthStatus.error,
          errorMessage: 'Cannot reach Core',
        );
        return;
      }

      final identity = await _apiClient.getMe();

      state = AuthState(
        status: AuthStatus.authenticated,
        identity: identity,
      );
    } catch (e) {
      await _tokenStorage.clearPanelToken();
      state = AuthState(
        status: AuthStatus.error,
        errorMessage: _formatError(e),
      );
    }
  }

  /// Logout: revoke server session, then clear stored credentials.
  Future<void> logout() async {
    final mode = await _tokenStorage.getAuthMode();
    if (mode == 'panel') {
      // Panels don't have sessions to revoke — just clear local state
      await _tokenStorage.clearPanelToken();
    } else {
      await _authRepo.serverLogout();
    }
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
