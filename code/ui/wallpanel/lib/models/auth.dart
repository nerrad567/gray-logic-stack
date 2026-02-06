/// Login response matching Go `loginResponse` struct.
/// See: code/core/internal/api/auth.go
class LoginResponse {
  final String accessToken;
  final String refreshToken;
  final String tokenType;
  final int expiresIn;

  const LoginResponse({
    required this.accessToken,
    required this.refreshToken,
    required this.tokenType,
    required this.expiresIn,
  });

  factory LoginResponse.fromJson(Map<String, dynamic> json) {
    return LoginResponse(
      accessToken: json['access_token'] as String,
      refreshToken: json['refresh_token'] as String,
      tokenType: json['token_type'] as String,
      expiresIn: (json['expires_in'] as num).toInt(),
    );
  }
}

/// WebSocket ticket response.
class TicketResponse {
  final String ticket;
  final int expiresIn;

  const TicketResponse({
    required this.ticket,
    required this.expiresIn,
  });

  factory TicketResponse.fromJson(Map<String, dynamic> json) {
    return TicketResponse(
      ticket: json['ticket'] as String,
      expiresIn: (json['expires_in'] as num).toInt(),
    );
  }
}

/// A room the caller can access, returned by GET /auth/me.
class IdentityRoom {
  final String id;
  final String name;
  final String areaId;
  final String areaName;
  final bool canManageScenes;

  const IdentityRoom({
    required this.id,
    required this.name,
    required this.areaId,
    required this.areaName,
    this.canManageScenes = false,
  });

  factory IdentityRoom.fromJson(Map<String, dynamic> json) {
    return IdentityRoom(
      id: json['id'] as String,
      name: json['name'] as String,
      areaId: json['area_id'] as String,
      areaName: json['area_name'] as String,
      canManageScenes: json['can_manage_scenes'] as bool? ?? false,
    );
  }
}

/// User info returned by GET /auth/me for user-type identities.
class IdentityUser {
  final String id;
  final String username;
  final String displayName;
  final String role;

  const IdentityUser({
    required this.id,
    required this.username,
    required this.displayName,
    required this.role,
  });

  factory IdentityUser.fromJson(Map<String, dynamic> json) {
    return IdentityUser(
      id: json['id'] as String,
      username: json['username'] as String,
      displayName: json['display_name'] as String,
      role: json['role'] as String,
    );
  }
}

/// Panel info returned by GET /auth/me for panel-type identities.
class IdentityPanel {
  final String id;
  final String name;

  const IdentityPanel({required this.id, required this.name});

  factory IdentityPanel.fromJson(Map<String, dynamic> json) {
    return IdentityPanel(
      id: json['id'] as String,
      name: json['name'] as String,
    );
  }
}

/// Caller identity returned by GET /auth/me.
/// Contains the caller's type (user/panel), identity details, accessible rooms,
/// and permissions. This is the keystone for identity-driven UI.
class Identity {
  final String type; // "user" or "panel"
  final IdentityUser? user;
  final IdentityPanel? panel;
  final List<IdentityRoom> rooms;
  final List<String> permissions;

  const Identity({
    required this.type,
    this.user,
    this.panel,
    required this.rooms,
    this.permissions = const [],
  });

  factory Identity.fromJson(Map<String, dynamic> json) {
    return Identity(
      type: json['type'] as String,
      user: json['user'] != null
          ? IdentityUser.fromJson(json['user'] as Map<String, dynamic>)
          : null,
      panel: json['panel'] != null
          ? IdentityPanel.fromJson(json['panel'] as Map<String, dynamic>)
          : null,
      rooms: (json['rooms'] as List<dynamic>?)
              ?.map((r) => IdentityRoom.fromJson(r as Map<String, dynamic>))
              .toList() ??
          [],
      permissions: (json['permissions'] as List<dynamic>?)
              ?.map((p) => p as String)
              .toList() ??
          [],
    );
  }

  bool get isPanel => type == 'panel';
  bool get isUser => type == 'user';
  bool get isAdmin => user?.role == 'admin' || user?.role == 'owner';
  bool get isOwner => user?.role == 'owner';

  bool hasPermission(String perm) => permissions.contains(perm);
}

/// Application authentication state.
enum AuthStatus { unauthenticated, authenticating, authenticated, error }

class AuthState {
  final AuthStatus status;
  final String? token;
  final DateTime? tokenExpiresAt;
  final String? errorMessage;
  final Identity? identity;

  const AuthState({
    this.status = AuthStatus.unauthenticated,
    this.token,
    this.tokenExpiresAt,
    this.errorMessage,
    this.identity,
  });

  bool get isAuthenticated => status == AuthStatus.authenticated;

  AuthState copyWith({
    AuthStatus? status,
    String? token,
    DateTime? tokenExpiresAt,
    String? errorMessage,
    Identity? identity,
  }) {
    return AuthState(
      status: status ?? this.status,
      token: token ?? this.token,
      tokenExpiresAt: tokenExpiresAt ?? this.tokenExpiresAt,
      errorMessage: errorMessage ?? this.errorMessage,
      identity: identity ?? this.identity,
    );
  }
}
