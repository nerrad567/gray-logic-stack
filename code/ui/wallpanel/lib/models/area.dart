/// Represents a logical grouping within a site (floor, building, wing).
class Area {
  final String id;
  final String siteId;
  final String name;
  final String slug;
  final String type;
  final int sortOrder;
  final DateTime createdAt;
  final DateTime updatedAt;

  const Area({
    required this.id,
    required this.siteId,
    required this.name,
    required this.slug,
    required this.type,
    this.sortOrder = 0,
    required this.createdAt,
    required this.updatedAt,
  });

  factory Area.fromJson(Map<String, dynamic> json) {
    return Area(
      id: json['id'] as String,
      siteId: json['site_id'] as String,
      name: json['name'] as String,
      slug: json['slug'] as String,
      type: json['type'] as String,
      sortOrder: (json['sort_order'] as num?)?.toInt() ?? 0,
      createdAt: DateTime.parse(json['created_at'] as String),
      updatedAt: DateTime.parse(json['updated_at'] as String),
    );
  }

  Map<String, dynamic> toJson() => {
        'id': id,
        'site_id': siteId,
        'name': name,
        'slug': slug,
        'type': type,
        'sort_order': sortOrder,
        'created_at': createdAt.toUtc().toIso8601String(),
        'updated_at': updatedAt.toUtc().toIso8601String(),
      };
}

/// API response wrapper for area list endpoint.
class AreaListResponse {
  final List<Area> areas;
  final int count;

  const AreaListResponse({required this.areas, required this.count});

  factory AreaListResponse.fromJson(Map<String, dynamic> json) {
    final list = (json['areas'] as List<dynamic>?) ?? [];
    return AreaListResponse(
      areas: list.map((a) => Area.fromJson(a as Map<String, dynamic>)).toList(),
      count: (json['count'] as num?)?.toInt() ?? list.length,
    );
  }
}
