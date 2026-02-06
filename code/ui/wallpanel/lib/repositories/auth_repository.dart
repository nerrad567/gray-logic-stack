import '../models/auth.dart';
import '../services/api_client.dart';
import '../services/token_storage.dart';

/// Handles authentication flow: login, token persistence, WS tickets.
class AuthRepository {
  final ApiClient _apiClient;
  final TokenStorage _tokenStorage;

  AuthRepository({
    required ApiClient apiClient,
    required TokenStorage tokenStorage,
  })  : _apiClient = apiClient,
        _tokenStorage = tokenStorage;

  /// Login with username/password, persist both tokens.
  Future<LoginResponse> login(String username, String password) async {
    final response = await _apiClient.login(username, password);
    await _tokenStorage.setToken(response.accessToken);
    await _tokenStorage.setRefreshToken(response.refreshToken);
    return response;
  }

  /// Get a single-use WebSocket ticket (requires valid auth token).
  Future<String> getWsTicket() async {
    final response = await _apiClient.getWsTicket();
    return response.ticket;
  }

  /// Check if we have a stored token.
  Future<bool> hasToken() async {
    final token = await _tokenStorage.getToken();
    return token != null && token.isNotEmpty;
  }

  /// Revoke server session, then clear stored credentials.
  Future<void> serverLogout() async {
    try {
      final refreshToken = await _tokenStorage.getRefreshToken();
      await _apiClient.logout(refreshToken);
    } catch (_) {
      // Best-effort: server may be unreachable. Clear local state regardless.
    }
    await _tokenStorage.clearToken();
    await _tokenStorage.clearRefreshToken();
  }

  /// Clear stored credentials without server call (legacy/fallback).
  Future<void> logout() async {
    await _tokenStorage.clearToken();
    await _tokenStorage.clearRefreshToken();
  }

  /// Change current user's password. Server revokes all sessions on success.
  Future<void> changePassword(String currentPassword, String newPassword) async {
    await _apiClient.changePassword(currentPassword, newPassword);
    // Server revokes all sessions — clear local tokens
    await _tokenStorage.clearToken();
    await _tokenStorage.clearRefreshToken();
  }

  /// Get the stored token.
  Future<String?> getToken() async {
    return _tokenStorage.getToken();
  }
}
