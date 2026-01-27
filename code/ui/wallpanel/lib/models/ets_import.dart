// Models for ETS project file import functionality.
// See: code/core/internal/commissioning/etsimport/types.go

/// Result of parsing an ETS project file.
class ETSParseResult {
  final String importId;
  final String format;
  final String? sourceFile;
  final List<ETSDetectedDevice> devices;
  final List<ETSGroupAddress> unmappedAddresses;
  final List<ETSWarning> warnings;
  final ETSStatistics statistics;

  const ETSParseResult({
    required this.importId,
    required this.format,
    this.sourceFile,
    this.devices = const [],
    this.unmappedAddresses = const [],
    this.warnings = const [],
    required this.statistics,
  });

  factory ETSParseResult.fromJson(Map<String, dynamic> json) {
    return ETSParseResult(
      importId: json['import_id'] as String,
      format: json['format'] as String,
      sourceFile: json['source_file'] as String?,
      devices: _parseDevices(json['devices']),
      unmappedAddresses: _parseAddresses(json['unmapped_addresses']),
      warnings: _parseWarnings(json['warnings']),
      statistics: ETSStatistics.fromJson(
        json['statistics'] as Map<String, dynamic>? ?? {},
      ),
    );
  }

  static List<ETSDetectedDevice> _parseDevices(dynamic value) {
    if (value is! List) return [];
    return value
        .map((d) => ETSDetectedDevice.fromJson(d as Map<String, dynamic>))
        .toList();
  }

  static List<ETSGroupAddress> _parseAddresses(dynamic value) {
    if (value is! List) return [];
    return value
        .map((a) => ETSGroupAddress.fromJson(a as Map<String, dynamic>))
        .toList();
  }

  static List<ETSWarning> _parseWarnings(dynamic value) {
    if (value is! List) return [];
    return value
        .map((w) => ETSWarning.fromJson(w as Map<String, dynamic>))
        .toList();
  }
}

/// A device detected from the ETS project file.
class ETSDetectedDevice {
  final String suggestedId;
  final String suggestedName;
  final String detectedType;
  final double confidence;
  final String suggestedDomain;
  final String? suggestedRoom;
  final String? suggestedArea;
  final String sourceLocation;
  final List<ETSDeviceAddress> addresses;

  // Mutable fields for user editing during preview
  bool import;
  String editedId;
  String editedName;
  String? selectedRoomId;
  String? selectedAreaId;

  ETSDetectedDevice({
    required this.suggestedId,
    required this.suggestedName,
    required this.detectedType,
    required this.confidence,
    required this.suggestedDomain,
    this.suggestedRoom,
    this.suggestedArea,
    required this.sourceLocation,
    this.addresses = const [],
    this.import = true,
    String? editedId,
    String? editedName,
    this.selectedRoomId,
    this.selectedAreaId,
  })  : editedId = editedId ?? suggestedId,
        editedName = editedName ?? suggestedName;

  factory ETSDetectedDevice.fromJson(Map<String, dynamic> json) {
    return ETSDetectedDevice(
      suggestedId: json['suggested_id'] as String,
      suggestedName: json['suggested_name'] as String,
      detectedType: json['detected_type'] as String,
      confidence: (json['confidence'] as num).toDouble(),
      suggestedDomain: json['suggested_domain'] as String,
      suggestedRoom: json['suggested_room'] as String?,
      suggestedArea: json['suggested_area'] as String?,
      sourceLocation: (json['source_location'] as String?) ?? '',
      addresses: _parseAddresses(json['addresses']),
    );
  }

  static List<ETSDeviceAddress> _parseAddresses(dynamic value) {
    if (value is! List) return [];
    return value
        .map((a) => ETSDeviceAddress.fromJson(a as Map<String, dynamic>))
        .toList();
  }

  /// Confidence level as a human-readable string.
  String get confidenceLevel {
    if (confidence >= 0.8) return 'high';
    if (confidence >= 0.5) return 'medium';
    return 'low';
  }

  /// Confidence as a percentage string.
  String get confidencePercent => '${(confidence * 100).round()}%';

  /// Convert to import request format.
  Map<String, dynamic> toImportJson() {
    return {
      'import': import,
      'id': editedId,
      'name': editedName,
      'type': detectedType,
      'domain': suggestedDomain,
      if (selectedRoomId != null) 'room_id': selectedRoomId,
      if (selectedAreaId != null) 'area_id': selectedAreaId,
      'addresses': addresses.map((a) => a.toJson()).toList(),
    };
  }
}

/// A group address mapping for a detected device.
class ETSDeviceAddress {
  final String ga;
  final String name;
  final String dpt;
  final String suggestedFunction;
  final List<String> suggestedFlags;
  final String? description;

  const ETSDeviceAddress({
    required this.ga,
    required this.name,
    required this.dpt,
    required this.suggestedFunction,
    this.suggestedFlags = const [],
    this.description,
  });

  factory ETSDeviceAddress.fromJson(Map<String, dynamic> json) {
    return ETSDeviceAddress(
      ga: json['ga'] as String,
      name: json['name'] as String,
      dpt: json['dpt'] as String,
      suggestedFunction: json['suggested_function'] as String,
      suggestedFlags: _asStringList(json['suggested_flags']),
      description: json['description'] as String?,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'ga': ga,
      'function': suggestedFunction,
      'dpt': dpt,
      'flags': suggestedFlags,
    };
  }

  static List<String> _asStringList(dynamic value) {
    if (value is List) return value.cast<String>();
    return [];
  }
}

/// A group address that couldn't be mapped to a device.
class ETSGroupAddress {
  final String address;
  final String name;
  final String dpt;
  final String location;
  final String? description;

  const ETSGroupAddress({
    required this.address,
    required this.name,
    required this.dpt,
    required this.location,
    this.description,
  });

  factory ETSGroupAddress.fromJson(Map<String, dynamic> json) {
    return ETSGroupAddress(
      address: json['address'] as String,
      name: json['name'] as String,
      dpt: (json['dpt'] as String?) ?? '',
      location: (json['location'] as String?) ?? '',
      description: json['description'] as String?,
    );
  }
}

/// A warning from the parse operation.
class ETSWarning {
  final String code;
  final String message;
  final String? address;
  final String? deviceId;

  const ETSWarning({
    required this.code,
    required this.message,
    this.address,
    this.deviceId,
  });

  factory ETSWarning.fromJson(Map<String, dynamic> json) {
    return ETSWarning(
      code: json['code'] as String,
      message: json['message'] as String,
      address: json['address'] as String?,
      deviceId: json['device_id'] as String?,
    );
  }
}

/// Statistics from the parse operation.
class ETSStatistics {
  final int totalGroupAddresses;
  final int detectedDevices;
  final int highConfidence;
  final int mediumConfidence;
  final int lowConfidence;
  final int unmappedAddresses;

  const ETSStatistics({
    this.totalGroupAddresses = 0,
    this.detectedDevices = 0,
    this.highConfidence = 0,
    this.mediumConfidence = 0,
    this.lowConfidence = 0,
    this.unmappedAddresses = 0,
  });

  factory ETSStatistics.fromJson(Map<String, dynamic> json) {
    return ETSStatistics(
      totalGroupAddresses: (json['total_group_addresses'] as num?)?.toInt() ?? 0,
      detectedDevices: (json['detected_devices'] as num?)?.toInt() ?? 0,
      highConfidence: (json['high_confidence'] as num?)?.toInt() ?? 0,
      mediumConfidence: (json['medium_confidence'] as num?)?.toInt() ?? 0,
      lowConfidence: (json['low_confidence'] as num?)?.toInt() ?? 0,
      unmappedAddresses: (json['unmapped_addresses'] as num?)?.toInt() ?? 0,
    );
  }
}

/// Request to import devices from a parsed ETS file.
class ETSImportRequest {
  final String importId;
  final List<ETSDetectedDevice> devices;
  final ETSImportOptions options;

  const ETSImportRequest({
    required this.importId,
    required this.devices,
    this.options = const ETSImportOptions(),
  });

  Map<String, dynamic> toJson() {
    return {
      'import_id': importId,
      'devices': devices.map((d) => d.toImportJson()).toList(),
      'options': options.toJson(),
    };
  }
}

/// Options for the import operation.
class ETSImportOptions {
  final bool skipExisting;
  final bool updateExisting;
  final bool dryRun;

  const ETSImportOptions({
    this.skipExisting = false,
    this.updateExisting = false,
    this.dryRun = false,
  });

  Map<String, dynamic> toJson() {
    return {
      'skip_existing': skipExisting,
      'update_existing': updateExisting,
      'dry_run': dryRun,
    };
  }
}

/// Response from the import operation.
class ETSImportResponse {
  final String importId;
  final int created;
  final int updated;
  final int skipped;
  final List<ETSImportError> errors;

  const ETSImportResponse({
    required this.importId,
    this.created = 0,
    this.updated = 0,
    this.skipped = 0,
    this.errors = const [],
  });

  factory ETSImportResponse.fromJson(Map<String, dynamic> json) {
    return ETSImportResponse(
      importId: json['import_id'] as String,
      created: (json['created'] as num?)?.toInt() ?? 0,
      updated: (json['updated'] as num?)?.toInt() ?? 0,
      skipped: (json['skipped'] as num?)?.toInt() ?? 0,
      errors: _parseErrors(json['errors']),
    );
  }

  static List<ETSImportError> _parseErrors(dynamic value) {
    if (value is! List) return [];
    return value
        .map((e) => ETSImportError.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  /// Whether the import was fully successful (no errors).
  bool get isSuccess => errors.isEmpty;

  /// Total devices processed.
  int get total => created + updated + skipped;
}

/// An error for a specific device during import.
class ETSImportError {
  final String deviceId;
  final String message;

  const ETSImportError({
    required this.deviceId,
    required this.message,
  });

  factory ETSImportError.fromJson(Map<String, dynamic> json) {
    return ETSImportError(
      deviceId: json['device_id'] as String,
      message: json['message'] as String,
    );
  }
}
