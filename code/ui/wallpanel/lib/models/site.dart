/// Represents a physical property managed by Gray Logic.
/// There is typically one site per deployment.
class Site {
  final String id;
  final String name;
  final String slug;
  final String? address;
  final double? latitude;
  final double? longitude;
  final String timezone;
  final double? elevationM;
  final List<String> modesAvailable;
  final String modeCurrent;
  final Map<String, dynamic> settings;
  final DateTime createdAt;
  final DateTime updatedAt;

  const Site({
    required this.id,
    required this.name,
    required this.slug,
    this.address,
    this.latitude,
    this.longitude,
    required this.timezone,
    this.elevationM,
    this.modesAvailable = const ['home', 'away', 'night', 'holiday'],
    this.modeCurrent = 'home',
    this.settings = const {},
    required this.createdAt,
    required this.updatedAt,
  });

  factory Site.fromJson(Map<String, dynamic> json) {
    return Site(
      id: json['id'] as String,
      name: json['name'] as String,
      slug: json['slug'] as String,
      address: json['address'] as String?,
      latitude: (json['latitude'] as num?)?.toDouble(),
      longitude: (json['longitude'] as num?)?.toDouble(),
      timezone: json['timezone'] as String? ?? 'UTC',
      elevationM: (json['elevation_m'] as num?)?.toDouble(),
      modesAvailable: (json['modes_available'] as List<dynamic>?)
              ?.map((e) => e as String)
              .toList() ??
          const ['home', 'away', 'night', 'holiday'],
      modeCurrent: json['mode_current'] as String? ?? 'home',
      settings: (json['settings'] as Map<String, dynamic>?) ?? {},
      createdAt: DateTime.parse(json['created_at'] as String),
      updatedAt: DateTime.parse(json['updated_at'] as String),
    );
  }

  Map<String, dynamic> toJson() => {
        'id': id,
        'name': name,
        'slug': slug,
        if (address != null) 'address': address,
        if (latitude != null) 'latitude': latitude,
        if (longitude != null) 'longitude': longitude,
        'timezone': timezone,
        if (elevationM != null) 'elevation_m': elevationM,
        'modes_available': modesAvailable,
        'mode_current': modeCurrent,
        'settings': settings,
        'created_at': createdAt.toUtc().toIso8601String(),
        'updated_at': updatedAt.toUtc().toIso8601String(),
      };
}
