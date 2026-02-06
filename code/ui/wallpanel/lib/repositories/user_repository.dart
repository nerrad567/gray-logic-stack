import '../models/user.dart';
import '../services/api_client.dart';

/// Handles user CRUD, session management, and room access via the Core API.
class UserRepository {
  final ApiClient _apiClient;

  UserRepository({required ApiClient apiClient}) : _apiClient = apiClient;

  /// Get all users.
  Future<List<User>> getUsers() async {
    final response = await _apiClient.getUsers();
    return response.users;
  }

  /// Get a single user by ID.
  Future<User> getUser(String id) async {
    return _apiClient.getUser(id);
  }

  /// Create a new user.
  Future<User> createUser(Map<String, dynamic> data) async {
    return _apiClient.createUser(data);
  }

  /// Update a user (PATCH semantics).
  Future<User> updateUser(String id, Map<String, dynamic> data) async {
    return _apiClient.updateUser(id, data);
  }

  /// Delete a user.
  Future<void> deleteUser(String id) async {
    await _apiClient.deleteUser(id);
  }

  /// Get active sessions for a user.
  Future<List<UserSession>> getUserSessions(String id) async {
    final response = await _apiClient.getUserSessions(id);
    return response.sessions;
  }

  /// Revoke all sessions for a user.
  Future<void> revokeUserSessions(String id) async {
    await _apiClient.revokeUserSessions(id);
  }

  /// Get room access grants for a user.
  Future<List<RoomAccessGrant>> getUserRooms(String id) async {
    final response = await _apiClient.getUserRooms(id);
    return response.rooms;
  }

  /// Replace all room access grants for a user.
  Future<List<RoomAccessGrant>> setUserRooms(
      String id, List<RoomAccessGrant> rooms) async {
    final response = await _apiClient.setUserRooms(id, rooms);
    return response.rooms;
  }
}
