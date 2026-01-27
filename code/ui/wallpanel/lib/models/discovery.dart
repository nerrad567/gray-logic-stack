/// Models for KNX bus discovery data (passive scanning).

class DiscoveryData {
  final List<DiscoveredGA> groupAddresses;
  final List<DiscoveredDevice> devices;
  final DiscoverySummary summary;

  DiscoveryData({
    required this.groupAddresses,
    required this.devices,
    required this.summary,
  });

  factory DiscoveryData.fromJson(Map<String, dynamic> json) {
    return DiscoveryData(
      groupAddresses: (json['group_addresses'] as List<dynamic>?)
              ?.map((e) => DiscoveredGA.fromJson(e as Map<String, dynamic>))
              .toList() ??
          [],
      devices: (json['devices'] as List<dynamic>?)
              ?.map((e) => DiscoveredDevice.fromJson(e as Map<String, dynamic>))
              .toList() ??
          [],
      summary: DiscoverySummary.fromJson(
          json['summary'] as Map<String, dynamic>? ?? {}),
    );
  }
}

class DiscoveredGA {
  final String groupAddress;
  final String lastSeen;
  final String lastSeenAgo;
  final int messageCount;
  final bool hasReadResponse;

  DiscoveredGA({
    required this.groupAddress,
    required this.lastSeen,
    required this.lastSeenAgo,
    required this.messageCount,
    required this.hasReadResponse,
  });

  factory DiscoveredGA.fromJson(Map<String, dynamic> json) {
    return DiscoveredGA(
      groupAddress: json['group_address'] as String? ?? '',
      lastSeen: json['last_seen'] as String? ?? '',
      lastSeenAgo: json['last_seen_ago'] as String? ?? '',
      messageCount: json['message_count'] as int? ?? 0,
      hasReadResponse: json['has_read_response'] as bool? ?? false,
    );
  }
}

class DiscoveredDevice {
  final String individualAddress;
  final String lastSeen;
  final String lastSeenAgo;
  final int messageCount;

  DiscoveredDevice({
    required this.individualAddress,
    required this.lastSeen,
    required this.lastSeenAgo,
    required this.messageCount,
  });

  factory DiscoveredDevice.fromJson(Map<String, dynamic> json) {
    return DiscoveredDevice(
      individualAddress: json['individual_address'] as String? ?? '',
      lastSeen: json['last_seen'] as String? ?? '',
      lastSeenAgo: json['last_seen_ago'] as String? ?? '',
      messageCount: json['message_count'] as int? ?? 0,
    );
  }
}

class DiscoverySummary {
  final int totalGroupAddresses;
  final int totalDevices;
  final int respondingAddresses;
  final int activeLast5Min;
  final int activeLast1Hour;

  DiscoverySummary({
    required this.totalGroupAddresses,
    required this.totalDevices,
    required this.respondingAddresses,
    required this.activeLast5Min,
    required this.activeLast1Hour,
  });

  factory DiscoverySummary.fromJson(Map<String, dynamic> json) {
    return DiscoverySummary(
      totalGroupAddresses: json['total_group_addresses'] as int? ?? 0,
      totalDevices: json['total_devices'] as int? ?? 0,
      respondingAddresses: json['responding_addresses'] as int? ?? 0,
      activeLast5Min: json['active_last_5min'] as int? ?? 0,
      activeLast1Hour: json['active_last_1hour'] as int? ?? 0,
    );
  }
}
