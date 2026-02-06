/// Panel model matching Go `auth.Panel` struct.
/// See: code/core/internal/auth/types.go
class Panel {
  final String id;
  final String name;
  final bool isActive;
  final DateTime? lastSeenAt;
  final String? createdBy;
  final DateTime createdAt;

  const Panel({
    required this.id,
    required this.name,
    required this.isActive,
    this.lastSeenAt,
    this.createdBy,
    required this.createdAt,
  });

  factory Panel.fromJson(Map<String, dynamic> json) {
    return Panel(
      id: json['id'] as String,
      name: json['name'] as String,
      isActive: (json['is_active'] as bool?) ?? true,
      lastSeenAt: json['last_seen_at'] != null
          ? DateTime.parse(json['last_seen_at'] as String)
          : null,
      createdBy: json['created_by'] as String?,
      createdAt: DateTime.parse(json['created_at'] as String),
    );
  }
}

/// Response wrapper for GET /panels.
class PanelListResponse {
  final List<Panel> panels;
  final int count;

  const PanelListResponse({required this.panels, required this.count});

  factory PanelListResponse.fromJson(Map<String, dynamic> json) {
    final list = (json['panels'] as List<dynamic>?) ?? [];
    return PanelListResponse(
      panels: list.map((p) => Panel.fromJson(p as Map<String, dynamic>)).toList(),
      count: (json['count'] as num?)?.toInt() ?? list.length,
    );
  }
}

/// Response wrapper for POST /panels (includes one-time token).
class PanelCreateResponse {
  final Panel panel;
  final String panelToken;

  const PanelCreateResponse({required this.panel, required this.panelToken});

  factory PanelCreateResponse.fromJson(Map<String, dynamic> json) {
    return PanelCreateResponse(
      panel: Panel.fromJson(json['panel'] as Map<String, dynamic>),
      panelToken: json['panel_token'] as String,
    );
  }
}

/// Response wrapper for GET /panels/{id} (includes room assignments).
class PanelDetailResponse {
  final Panel panel;
  final List<String> rooms;

  const PanelDetailResponse({required this.panel, required this.rooms});

  factory PanelDetailResponse.fromJson(Map<String, dynamic> json) {
    final roomList = (json['rooms'] as List<dynamic>?) ?? [];
    return PanelDetailResponse(
      panel: Panel.fromJson(json['panel'] as Map<String, dynamic>),
      rooms: roomList.map((r) => r as String).toList(),
    );
  }
}

/// Response wrapper for GET/PUT /panels/{id}/rooms.
class PanelRoomsResponse {
  final List<String> roomIds;
  final int count;

  const PanelRoomsResponse({required this.roomIds, required this.count});

  factory PanelRoomsResponse.fromJson(Map<String, dynamic> json) {
    final list = (json['room_ids'] as List<dynamic>?) ?? [];
    return PanelRoomsResponse(
      roomIds: list.map((r) => r as String).toList(),
      count: (json['count'] as num?)?.toInt() ?? list.length,
    );
  }
}
