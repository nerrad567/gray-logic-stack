/// System metrics data model for the Admin dashboard.
///
/// Maps to the /api/v1/metrics endpoint response.
class SystemMetrics {
  final String timestamp;
  final String version;
  final int uptimeSeconds;
  final RuntimeMetrics runtime;
  final WebSocketMetrics websocket;
  final MQTTMetrics mqtt;
  final KNXBridgeMetrics? knxBridge;
  final DeviceMetrics devices;
  final DatabaseMetrics database;

  SystemMetrics({
    required this.timestamp,
    required this.version,
    required this.uptimeSeconds,
    required this.runtime,
    required this.websocket,
    required this.mqtt,
    this.knxBridge,
    required this.devices,
    required this.database,
  });

  factory SystemMetrics.fromJson(Map<String, dynamic> json) {
    return SystemMetrics(
      timestamp: json['timestamp'] as String,
      version: json['version'] as String,
      uptimeSeconds: json['uptime_seconds'] as int,
      runtime: RuntimeMetrics.fromJson(json['runtime'] as Map<String, dynamic>),
      websocket:
          WebSocketMetrics.fromJson(json['websocket'] as Map<String, dynamic>),
      mqtt: MQTTMetrics.fromJson(json['mqtt'] as Map<String, dynamic>),
      knxBridge: json['knx_bridge'] != null
          ? KNXBridgeMetrics.fromJson(
              json['knx_bridge'] as Map<String, dynamic>)
          : null,
      devices: DeviceMetrics.fromJson(json['devices'] as Map<String, dynamic>),
      database:
          DatabaseMetrics.fromJson(json['database'] as Map<String, dynamic>),
    );
  }

  /// Returns uptime as a human-readable string (e.g., "2h 15m").
  String get uptimeFormatted {
    final hours = uptimeSeconds ~/ 3600;
    final minutes = (uptimeSeconds % 3600) ~/ 60;
    final seconds = uptimeSeconds % 60;

    if (hours > 0) {
      return '${hours}h ${minutes}m';
    } else if (minutes > 0) {
      return '${minutes}m ${seconds}s';
    } else {
      return '${seconds}s';
    }
  }
}

class RuntimeMetrics {
  final int goroutines;
  final double memoryAllocMB;
  final double memoryTotalMB;
  final int numGC;

  RuntimeMetrics({
    required this.goroutines,
    required this.memoryAllocMB,
    required this.memoryTotalMB,
    required this.numGC,
  });

  factory RuntimeMetrics.fromJson(Map<String, dynamic> json) {
    return RuntimeMetrics(
      goroutines: json['goroutines'] as int,
      memoryAllocMB: (json['memory_alloc_mb'] as num).toDouble(),
      memoryTotalMB: (json['memory_total_mb'] as num).toDouble(),
      numGC: json['num_gc'] as int,
    );
  }
}

class WebSocketMetrics {
  final int connectedClients;

  WebSocketMetrics({required this.connectedClients});

  factory WebSocketMetrics.fromJson(Map<String, dynamic> json) {
    return WebSocketMetrics(
      connectedClients: json['connected_clients'] as int,
    );
  }
}

class MQTTMetrics {
  final bool connected;

  MQTTMetrics({required this.connected});

  factory MQTTMetrics.fromJson(Map<String, dynamic> json) {
    return MQTTMetrics(
      connected: json['connected'] as bool,
    );
  }
}

class KNXBridgeMetrics {
  final bool connected;
  final String status;
  final int telegramsTx;
  final int telegramsRx;
  final int devicesManaged;

  KNXBridgeMetrics({
    required this.connected,
    required this.status,
    required this.telegramsTx,
    required this.telegramsRx,
    required this.devicesManaged,
  });

  factory KNXBridgeMetrics.fromJson(Map<String, dynamic> json) {
    return KNXBridgeMetrics(
      connected: json['connected'] as bool,
      status: json['status'] as String,
      telegramsTx: json['telegrams_tx'] as int,
      telegramsRx: json['telegrams_rx'] as int,
      devicesManaged: json['devices_managed'] as int,
    );
  }
}

class DeviceMetrics {
  final int total;
  final Map<String, int> byHealth;
  final Map<String, int> byDomain;

  DeviceMetrics({
    required this.total,
    required this.byHealth,
    required this.byDomain,
  });

  factory DeviceMetrics.fromJson(Map<String, dynamic> json) {
    return DeviceMetrics(
      total: json['total'] as int,
      byHealth: (json['by_health'] as Map<String, dynamic>)
          .map((k, v) => MapEntry(k, v as int)),
      byDomain: (json['by_domain'] as Map<String, dynamic>)
          .map((k, v) => MapEntry(k, v as int)),
    );
  }
}

class DatabaseMetrics {
  final int openConnections;
  final int inUse;
  final int idle;
  final int waitCount;

  DatabaseMetrics({
    required this.openConnections,
    required this.inUse,
    required this.idle,
    required this.waitCount,
  });

  factory DatabaseMetrics.fromJson(Map<String, dynamic> json) {
    return DatabaseMetrics(
      openConnections: json['open_connections'] as int,
      inUse: json['in_use'] as int,
      idle: json['idle'] as int,
      waitCount: json['wait_count'] as int,
    );
  }
}
