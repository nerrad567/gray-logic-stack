/// Login response matching Go `loginResponse` struct.
/// See: code/core/internal/api/auth.go
class LoginResponse {
  final String accessToken;
  final String tokenType;
  final int expiresIn;

  const LoginResponse({
    required this.accessToken,
    required this.tokenType,
    required this.expiresIn,
  });

  factory LoginResponse.fromJson(Map<String, dynamic> json) {
    return LoginResponse(
      accessToken: json['access_token'] as String,
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

/// Application authentication state.
enum AuthStatus { unauthenticated, authenticating, authenticated, error }

class AuthState {
  final AuthStatus status;
  final String? token;
  final DateTime? tokenExpiresAt;
  final String? errorMessage;

  const AuthState({
    this.status = AuthStatus.unauthenticated,
    this.token,
    this.tokenExpiresAt,
    this.errorMessage,
  });

  bool get isAuthenticated => status == AuthStatus.authenticated && token != null;

  AuthState copyWith({
    AuthStatus? status,
    String? token,
    DateTime? tokenExpiresAt,
    String? errorMessage,
  }) {
    return AuthState(
      status: status ?? this.status,
      token: token ?? this.token,
      tokenExpiresAt: tokenExpiresAt ?? this.tokenExpiresAt,
      errorMessage: errorMessage ?? this.errorMessage,
    );
  }
}
