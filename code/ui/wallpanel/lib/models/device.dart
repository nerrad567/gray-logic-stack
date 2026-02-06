/// Device model matching the Go `device.Device` struct.
/// See: code/core/internal/device/types.go
class Device {
  final String id;
  final String name;
  final String slug;
  final String? roomId;
  final String? areaId;
  final String type;
  final String domain;
  final String protocol;
  final Map<String, dynamic> address;
  final String? gatewayId;
  final List<String> capabilities;
  final Map<String, dynamic> config;
  final Map<String, dynamic> state;
  final DateTime? stateUpdatedAt;
  final String healthStatus;
  final DateTime? healthLastSeen;
  final bool phmEnabled;
  final Map<String, dynamic>? phmBaseline;
  final String? manufacturer;
  final String? model;
  final String? firmwareVersion;
  final DateTime createdAt;
  final DateTime updatedAt;

  const Device({
    required this.id,
    required this.name,
    required this.slug,
    this.roomId,
    this.areaId,
    required this.type,
    required this.domain,
    required this.protocol,
    this.address = const {},
    this.gatewayId,
    this.capabilities = const [],
    this.config = const {},
    this.state = const {},
    this.stateUpdatedAt,
    this.healthStatus = 'unknown',
    this.healthLastSeen,
    this.phmEnabled = false,
    this.phmBaseline,
    this.manufacturer,
    this.model,
    this.firmwareVersion,
    required this.createdAt,
    required this.updatedAt,
  });

  /// Creates a Device from a JSON map (API response).
  factory Device.fromJson(Map<String, dynamic> json) {
    return Device(
      id: json['id'] as String,
      name: json['name'] as String,
      slug: json['slug'] as String,
      roomId: json['room_id'] as String?,
      areaId: json['area_id'] as String?,
      type: json['type'] as String,
      domain: json['domain'] as String,
      protocol: json['protocol'] as String,
      address: _asMap(json['address']),
      gatewayId: json['gateway_id'] as String?,
      capabilities: _asStringList(json['capabilities']),
      config: _asMap(json['config']),
      state: _asMap(json['state']),
      stateUpdatedAt: _parseDateTime(json['state_updated_at']),
      healthStatus: (json['health_status'] as String?) ?? 'unknown',
      healthLastSeen: _parseDateTime(json['health_last_seen']),
      phmEnabled: (json['phm_enabled'] as bool?) ?? false,
      phmBaseline: json['phm_baseline'] as Map<String, dynamic>?,
      manufacturer: json['manufacturer'] as String?,
      model: json['model'] as String?,
      firmwareVersion: json['firmware_version'] as String?,
      createdAt: DateTime.parse(json['created_at'] as String),
      updatedAt: DateTime.parse(json['updated_at'] as String),
    );
  }

  /// Serialises the Device to a JSON map.
  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'name': name,
      'slug': slug,
      if (roomId != null) 'room_id': roomId,
      if (areaId != null) 'area_id': areaId,
      'type': type,
      'domain': domain,
      'protocol': protocol,
      'address': address,
      if (gatewayId != null) 'gateway_id': gatewayId,
      'capabilities': capabilities,
      'config': config,
      'state': state,
      if (stateUpdatedAt != null)
        'state_updated_at': stateUpdatedAt!.toUtc().toIso8601String(),
      'health_status': healthStatus,
      if (healthLastSeen != null)
        'health_last_seen': healthLastSeen!.toUtc().toIso8601String(),
      'phm_enabled': phmEnabled,
      if (phmBaseline != null) 'phm_baseline': phmBaseline,
      if (manufacturer != null) 'manufacturer': manufacturer,
      if (model != null) 'model': model,
      if (firmwareVersion != null) 'firmware_version': firmwareVersion,
      'created_at': createdAt.toUtc().toIso8601String(),
      'updated_at': updatedAt.toUtc().toIso8601String(),
    };
  }

  /// Returns a copy with updated fields.
  Device copyWith({
    String? id,
    String? name,
    String? slug,
    String? roomId,
    String? areaId,
    String? type,
    String? domain,
    String? protocol,
    Map<String, dynamic>? address,
    String? gatewayId,
    List<String>? capabilities,
    Map<String, dynamic>? config,
    Map<String, dynamic>? state,
    DateTime? stateUpdatedAt,
    String? healthStatus,
    DateTime? healthLastSeen,
    bool? phmEnabled,
    Map<String, dynamic>? phmBaseline,
    String? manufacturer,
    String? model,
    String? firmwareVersion,
    DateTime? createdAt,
    DateTime? updatedAt,
  }) {
    return Device(
      id: id ?? this.id,
      name: name ?? this.name,
      slug: slug ?? this.slug,
      roomId: roomId ?? this.roomId,
      areaId: areaId ?? this.areaId,
      type: type ?? this.type,
      domain: domain ?? this.domain,
      protocol: protocol ?? this.protocol,
      address: address ?? this.address,
      gatewayId: gatewayId ?? this.gatewayId,
      capabilities: capabilities ?? this.capabilities,
      config: config ?? this.config,
      state: state ?? this.state,
      stateUpdatedAt: stateUpdatedAt ?? this.stateUpdatedAt,
      healthStatus: healthStatus ?? this.healthStatus,
      healthLastSeen: healthLastSeen ?? this.healthLastSeen,
      phmEnabled: phmEnabled ?? this.phmEnabled,
      phmBaseline: phmBaseline ?? this.phmBaseline,
      manufacturer: manufacturer ?? this.manufacturer,
      model: model ?? this.model,
      firmwareVersion: firmwareVersion ?? this.firmwareVersion,
      createdAt: createdAt ?? this.createdAt,
      updatedAt: updatedAt ?? this.updatedAt,
    );
  }

  // --- Convenience getters for M1.5 tile widgets ---

  /// Whether the device is currently on (light switches, dimmers).
  bool get isOn => state['on'] == true;

  /// Current brightness level (0-100) for dimmable devices.
  int get level => (state['level'] as num?)?.toInt() ?? 0;

  /// Whether this device supports on/off control.
  bool get hasOnOff => capabilities.contains('on_off');

  /// Whether this device supports dimming.
  bool get hasDim => capabilities.contains('dim');

  /// Whether this device supports position control (blinds).
  bool get hasPosition => capabilities.contains('position');

  /// Whether this device supports tilt/slat angle control.
  bool get hasTilt => capabilities.contains('tilt');

  /// Whether this device supports temperature setpoint control.
  bool get hasTemperatureSet => capabilities.contains('temperature_set');

  /// Whether this device supports colour temperature control.
  bool get hasColorTemp => capabilities.contains('color_temp');

  /// Whether this device supports RGB colour control.
  bool get hasColorRGB => capabilities.contains('color_rgb');

  /// Whether this device supports fan speed control.
  bool get hasSpeed => capabilities.contains('speed');

  /// Control capabilities — anything that can be commanded in a scene.
  static const _controlCapabilities = {
    'on_off', 'dim', 'position', 'tilt', 'temperature_set',
    'color_temp', 'color_rgb', 'speed', 'lock_unlock', 'arm_disarm',
  };

  /// True if device has any control capability (sensors are not commandable).
  bool get isCommandable =>
      capabilities.any((c) => _controlCapabilities.contains(c));

  /// Device types that are controlled by HVAC loops, not user scenes.
  static const _hvacInfrastructureTypes = {
    'heating_actuator', 'valve_actuator',
  };

  /// True if device is a valid target for scene actions.
  /// Excludes sensors (not commandable) and HVAC infrastructure (loop-controlled).
  bool get isSceneTarget =>
      isCommandable && !_hvacInfrastructureTypes.contains(type);

  /// Whether the device is considered reachable for UI interaction.
  /// Devices with 'unknown' health are treated as available (no health data yet).
  /// Only explicitly 'offline' devices are disabled.
  bool get isOnline => healthStatus != 'offline';

  // --- Helpers ---

  static Map<String, dynamic> _asMap(dynamic value) {
    if (value is Map<String, dynamic>) return value;
    if (value is Map) return Map<String, dynamic>.from(value);
    return {};
  }

  static List<String> _asStringList(dynamic value) {
    if (value is List) return value.cast<String>();
    return [];
  }

  static DateTime? _parseDateTime(dynamic value) {
    if (value == null) return null;
    return DateTime.parse(value as String);
  }
}

/// Response wrapper for GET /devices.
class DeviceListResponse {
  final List<Device> devices;
  final int count;

  const DeviceListResponse({required this.devices, required this.count});

  factory DeviceListResponse.fromJson(Map<String, dynamic> json) {
    final list = (json['devices'] as List<dynamic>?) ?? [];
    return DeviceListResponse(
      devices: list
          .map((d) => Device.fromJson(d as Map<String, dynamic>))
          .toList(),
      count: (json['count'] as num?)?.toInt() ?? list.length,
    );
  }
}

/// Response for PUT /devices/{id}/state (202 Accepted).
class CommandResponse {
  final String commandId;
  final String status;
  final String message;

  const CommandResponse({
    required this.commandId,
    required this.status,
    required this.message,
  });

  factory CommandResponse.fromJson(Map<String, dynamic> json) {
    return CommandResponse(
      commandId: json['command_id'] as String,
      status: json['status'] as String,
      message: json['message'] as String,
    );
  }
}
