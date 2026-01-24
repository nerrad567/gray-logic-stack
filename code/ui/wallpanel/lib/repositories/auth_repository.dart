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

  /// Login with username/password, persist token.
  Future<LoginResponse> login(String username, String password) async {
    final response = await _apiClient.login(username, password);
    await _tokenStorage.setToken(response.accessToken);
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

  /// Clear stored credentials.
  Future<void> logout() async {
    await _tokenStorage.clearToken();
  }

  /// Get the stored token.
  Future<String?> getToken() async {
    return _tokenStorage.getToken();
  }
}
