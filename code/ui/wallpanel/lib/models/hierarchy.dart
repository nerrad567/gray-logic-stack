/// Models for the GET /hierarchy single-call endpoint.
/// Returns the full site → areas → rooms tree with device/scene counts.

class HierarchyRoom {
  final String id;
  final String name;
  final String type;
  final int sortOrder;
  final int deviceCount;
  final int sceneCount;
  final Map<String, String> zones;

  const HierarchyRoom({
    required this.id,
    required this.name,
    required this.type,
    this.sortOrder = 0,
    this.deviceCount = 0,
    this.sceneCount = 0,
    this.zones = const {},
  });

  factory HierarchyRoom.fromJson(Map<String, dynamic> json) {
    return HierarchyRoom(
      id: json['id'] as String,
      name: json['name'] as String,
      type: json['type'] as String,
      sortOrder: (json['sort_order'] as num?)?.toInt() ?? 0,
      deviceCount: (json['device_count'] as num?)?.toInt() ?? 0,
      sceneCount: (json['scene_count'] as num?)?.toInt() ?? 0,
      zones: (json['zones'] as Map<String, dynamic>?)
              ?.map((k, v) => MapEntry(k, v as String)) ??
          {},
    );
  }
}

class HierarchyArea {
  final String id;
  final String name;
  final String type;
  final int sortOrder;
  final List<HierarchyRoom> rooms;
  final int roomCount;

  const HierarchyArea({
    required this.id,
    required this.name,
    required this.type,
    this.sortOrder = 0,
    this.rooms = const [],
    this.roomCount = 0,
  });

  factory HierarchyArea.fromJson(Map<String, dynamic> json) {
    final roomList = (json['rooms'] as List<dynamic>?) ?? [];
    return HierarchyArea(
      id: json['id'] as String,
      name: json['name'] as String,
      type: json['type'] as String,
      sortOrder: (json['sort_order'] as num?)?.toInt() ?? 0,
      rooms: roomList
          .map((r) => HierarchyRoom.fromJson(r as Map<String, dynamic>))
          .toList(),
      roomCount: (json['room_count'] as num?)?.toInt() ?? roomList.length,
    );
  }
}

class HierarchySite {
  final String id;
  final String name;
  final String modeCurrent;
  final List<HierarchyArea> areas;
  final int areaCount;

  const HierarchySite({
    required this.id,
    required this.name,
    this.modeCurrent = 'home',
    this.areas = const [],
    this.areaCount = 0,
  });

  factory HierarchySite.fromJson(Map<String, dynamic> json) {
    final areaList = (json['areas'] as List<dynamic>?) ?? [];
    return HierarchySite(
      id: json['id'] as String,
      name: json['name'] as String,
      modeCurrent: json['mode_current'] as String? ?? 'home',
      areas: areaList
          .map((a) => HierarchyArea.fromJson(a as Map<String, dynamic>))
          .toList(),
      areaCount: (json['area_count'] as num?)?.toInt() ?? areaList.length,
    );
  }
}

/// Response wrapper for GET /hierarchy.
class HierarchyResponse {
  final HierarchySite site;

  const HierarchyResponse({required this.site});

  factory HierarchyResponse.fromJson(Map<String, dynamic> json) {
    return HierarchyResponse(
      site: HierarchySite.fromJson(json['site'] as Map<String, dynamic>),
    );
  }
}
