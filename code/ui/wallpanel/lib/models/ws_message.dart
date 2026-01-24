/// WebSocket message types matching Go constants in websocket.go.
/// See: code/core/internal/api/websocket.go
class WSMessageType {
  WSMessageType._();

  static const String subscribe = 'subscribe';
  static const String unsubscribe = 'unsubscribe';
  static const String ping = 'ping';
  static const String pong = 'pong';
  static const String event = 'event';
  static const String response = 'response';
  static const String error = 'error';
}

/// WebSocket channels for subscription.
class WSChannel {
  WSChannel._();

  static const String deviceStateChanged = 'device.state_changed';
  static const String sceneActivated = 'scene.activated';
  static const String deviceHealthChanged = 'device.health_changed';
}

/// A message sent to the WebSocket server.
class WSOutMessage {
  final String type;
  final String? id;
  final dynamic payload;

  const WSOutMessage({
    required this.type,
    this.id,
    this.payload,
  });

  Map<String, dynamic> toJson() {
    return {
      'type': type,
      if (id != null) 'id': id,
      if (payload != null) 'payload': payload,
    };
  }

  /// Create a subscribe message for the given channels.
  factory WSOutMessage.subscribe(List<String> channels, {String? id}) {
    return WSOutMessage(
      type: WSMessageType.subscribe,
      id: id,
      payload: {'channels': channels},
    );
  }

  /// Create an unsubscribe message.
  factory WSOutMessage.unsubscribe(List<String> channels, {String? id}) {
    return WSOutMessage(
      type: WSMessageType.unsubscribe,
      id: id,
      payload: {'channels': channels},
    );
  }

  /// Create a ping message.
  factory WSOutMessage.ping({String? id}) {
    return WSOutMessage(type: WSMessageType.ping, id: id);
  }
}

/// A message received from the WebSocket server.
class WSInMessage {
  final String type;
  final String? id;
  final String? eventType;
  final DateTime? timestamp;
  final dynamic payload;

  const WSInMessage({
    required this.type,
    this.id,
    this.eventType,
    this.timestamp,
    this.payload,
  });

  factory WSInMessage.fromJson(Map<String, dynamic> json) {
    return WSInMessage(
      type: json['type'] as String,
      id: json['id'] as String?,
      eventType: json['event_type'] as String?,
      timestamp: json['timestamp'] != null
          ? DateTime.parse(json['timestamp'] as String)
          : null,
      payload: json['payload'],
    );
  }

  /// Whether this is a device state change event.
  bool get isDeviceStateChanged =>
      type == WSMessageType.event &&
      eventType == WSChannel.deviceStateChanged;

  /// Whether this is an error message.
  bool get isError => type == WSMessageType.error;

  /// Extract the error message string if this is an error.
  String? get errorMessage {
    if (!isError || payload == null) return null;
    if (payload is Map<String, dynamic>) {
      return payload['message'] as String?;
    }
    return null;
  }
}

/// Parsed device state change event payload.
class DeviceStateEvent {
  final String? deviceId;
  final Map<String, dynamic> state;
  final String? healthStatus;

  const DeviceStateEvent({
    this.deviceId,
    this.state = const {},
    this.healthStatus,
  });

  factory DeviceStateEvent.fromPayload(dynamic payload) {
    if (payload is! Map<String, dynamic>) {
      return const DeviceStateEvent();
    }
    return DeviceStateEvent(
      deviceId: payload['device_id'] as String?,
      state: (payload['state'] as Map<String, dynamic>?) ?? {},
      healthStatus: payload['health_status'] as String?,
    );
  }
}
