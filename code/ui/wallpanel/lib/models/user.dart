/// User model matching Go `auth.User` struct.
/// See: code/core/internal/auth/types.go
class User {
  final String id;
  final String username;
  final String displayName;
  final String? email;
  final String role;
  final bool isActive;
  final String? createdBy;
  final DateTime createdAt;
  final DateTime updatedAt;

  const User({
    required this.id,
    required this.username,
    required this.displayName,
    this.email,
    required this.role,
    required this.isActive,
    this.createdBy,
    required this.createdAt,
    required this.updatedAt,
  });

  factory User.fromJson(Map<String, dynamic> json) {
    return User(
      id: json['id'] as String,
      username: json['username'] as String,
      displayName: json['display_name'] as String,
      email: json['email'] as String?,
      role: json['role'] as String,
      isActive: (json['is_active'] as bool?) ?? true,
      createdBy: json['created_by'] as String?,
      createdAt: DateTime.parse(json['created_at'] as String),
      updatedAt: DateTime.parse(json['updated_at'] as String),
    );
  }
}

/// Response wrapper for GET /users.
class UserListResponse {
  final List<User> users;
  final int count;

  const UserListResponse({required this.users, required this.count});

  factory UserListResponse.fromJson(Map<String, dynamic> json) {
    final list = (json['users'] as List<dynamic>?) ?? [];
    return UserListResponse(
      users: list.map((u) => User.fromJson(u as Map<String, dynamic>)).toList(),
      count: (json['count'] as num?)?.toInt() ?? list.length,
    );
  }
}

/// User session (refresh token metadata) from GET /users/{id}/sessions.
/// Matches Go `auth.RefreshToken` (with token_hash excluded via json:"-").
class UserSession {
  final String id;
  final String userId;
  final String familyId;
  final String? deviceInfo;
  final DateTime expiresAt;
  final bool revoked;
  final DateTime createdAt;

  const UserSession({
    required this.id,
    required this.userId,
    required this.familyId,
    this.deviceInfo,
    required this.expiresAt,
    required this.revoked,
    required this.createdAt,
  });

  factory UserSession.fromJson(Map<String, dynamic> json) {
    return UserSession(
      id: json['id'] as String,
      userId: json['user_id'] as String,
      familyId: json['family_id'] as String,
      deviceInfo: json['device_info'] as String?,
      expiresAt: DateTime.parse(json['expires_at'] as String),
      revoked: (json['revoked'] as bool?) ?? false,
      createdAt: DateTime.parse(json['created_at'] as String),
    );
  }
}

/// Response wrapper for GET /users/{id}/sessions.
class UserSessionListResponse {
  final List<UserSession> sessions;
  final int count;

  const UserSessionListResponse({required this.sessions, required this.count});

  factory UserSessionListResponse.fromJson(Map<String, dynamic> json) {
    final list = (json['sessions'] as List<dynamic>?) ?? [];
    return UserSessionListResponse(
      sessions: list
          .map((s) => UserSession.fromJson(s as Map<String, dynamic>))
          .toList(),
      count: (json['count'] as num?)?.toInt() ?? list.length,
    );
  }
}

/// Room access grant from GET /users/{id}/rooms.
/// Matches Go `auth.RoomAccess`.
class RoomAccessGrant {
  final String roomId;
  final bool canManageScenes;

  const RoomAccessGrant({
    required this.roomId,
    this.canManageScenes = false,
  });

  factory RoomAccessGrant.fromJson(Map<String, dynamic> json) {
    return RoomAccessGrant(
      roomId: json['room_id'] as String,
      canManageScenes: (json['can_manage_scenes'] as bool?) ?? false,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'room_id': roomId,
      'can_manage_scenes': canManageScenes,
    };
  }
}

/// Response wrapper for GET/PUT /users/{id}/rooms.
class RoomAccessResponse {
  final List<RoomAccessGrant> rooms;
  final int count;

  const RoomAccessResponse({required this.rooms, required this.count});

  factory RoomAccessResponse.fromJson(Map<String, dynamic> json) {
    final list = (json['rooms'] as List<dynamic>?) ?? [];
    return RoomAccessResponse(
      rooms: list
          .map((r) => RoomAccessGrant.fromJson(r as Map<String, dynamic>))
          .toList(),
      count: (json['count'] as num?)?.toInt() ?? list.length,
    );
  }
}
