/// Represents a physical space within an area.
class Room {
  final String id;
  final String areaId;
  final String name;
  final String slug;
  final String type;
  final int sortOrder;
  final String? climateZoneId;
  final String? audioZoneId;
  final Map<String, dynamic> settings;
  final DateTime createdAt;
  final DateTime updatedAt;

  const Room({
    required this.id,
    required this.areaId,
    required this.name,
    required this.slug,
    required this.type,
    this.sortOrder = 0,
    this.climateZoneId,
    this.audioZoneId,
    this.settings = const {},
    required this.createdAt,
    required this.updatedAt,
  });

  factory Room.fromJson(Map<String, dynamic> json) {
    return Room(
      id: json['id'] as String,
      areaId: json['area_id'] as String,
      name: json['name'] as String,
      slug: json['slug'] as String,
      type: json['type'] as String,
      sortOrder: (json['sort_order'] as num?)?.toInt() ?? 0,
      climateZoneId: json['climate_zone_id'] as String?,
      audioZoneId: json['audio_zone_id'] as String?,
      settings: (json['settings'] as Map<String, dynamic>?) ?? {},
      createdAt: DateTime.parse(json['created_at'] as String),
      updatedAt: DateTime.parse(json['updated_at'] as String),
    );
  }

  Map<String, dynamic> toJson() => {
        'id': id,
        'area_id': areaId,
        'name': name,
        'slug': slug,
        'type': type,
        'sort_order': sortOrder,
        if (climateZoneId != null) 'climate_zone_id': climateZoneId,
        if (audioZoneId != null) 'audio_zone_id': audioZoneId,
        'settings': settings,
        'created_at': createdAt.toUtc().toIso8601String(),
        'updated_at': updatedAt.toUtc().toIso8601String(),
      };
}

/// API response wrapper for room list endpoint.
class RoomListResponse {
  final List<Room> rooms;
  final int count;

  const RoomListResponse({required this.rooms, required this.count});

  factory RoomListResponse.fromJson(Map<String, dynamic> json) {
    final list = (json['rooms'] as List<dynamic>?) ?? [];
    return RoomListResponse(
      rooms: list.map((r) => Room.fromJson(r as Map<String, dynamic>)).toList(),
      count: (json['count'] as num?)?.toInt() ?? list.length,
    );
  }
}
