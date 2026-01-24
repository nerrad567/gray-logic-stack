import 'package:wallpanel/models/device.dart';
import 'package:wallpanel/models/scene.dart';

/// Shared test fixtures for creating mock devices and scenes.
/// Used across provider and widget tests.
class TestFixtures {
  TestFixtures._();

  static Device makeSwitch({
    String id = 'dev-switch-1',
    String name = 'Test Switch',
    bool on = false,
    String healthStatus = 'online',
    String roomId = 'living_room',
  }) {
    return Device(
      id: id,
      name: name,
      slug: name.toLowerCase().replaceAll(' ', '_'),
      roomId: roomId,
      type: 'switch',
      domain: 'lighting',
      protocol: 'knx',
      capabilities: ['on_off'],
      state: {'on': on},
      healthStatus: healthStatus,
      createdAt: DateTime(2026),
      updatedAt: DateTime(2026),
    );
  }

  static Device makeDimmer({
    String id = 'dev-dimmer-1',
    String name = 'Test Dimmer',
    bool on = false,
    int level = 0,
    String healthStatus = 'online',
    String roomId = 'living_room',
  }) {
    return Device(
      id: id,
      name: name,
      slug: name.toLowerCase().replaceAll(' ', '_'),
      roomId: roomId,
      type: 'dimmer',
      domain: 'lighting',
      protocol: 'knx',
      capabilities: ['on_off', 'dim'],
      state: {'on': on, 'level': level},
      healthStatus: healthStatus,
      createdAt: DateTime(2026),
      updatedAt: DateTime(2026),
    );
  }

  static Scene makeScene({
    String id = 'scene-1',
    String name = 'Test Scene',
    String? icon,
    String? colour,
    bool enabled = true,
    int sortOrder = 0,
    String roomId = 'living_room',
    List<SceneAction> actions = const [],
  }) {
    return Scene(
      id: id,
      name: name,
      slug: name.toLowerCase().replaceAll(' ', '_'),
      roomId: roomId,
      icon: icon,
      colour: colour,
      enabled: enabled,
      sortOrder: sortOrder,
      actions: actions,
      createdAt: DateTime(2026),
      updatedAt: DateTime(2026),
    );
  }

  /// Sample API response JSON for a device list.
  static Map<String, dynamic> deviceListJson({int count = 2}) {
    return {
      'devices': List.generate(count, (i) => {
        'id': 'dev-$i',
        'name': 'Device $i',
        'slug': 'device_$i',
        'room_id': 'living_room',
        'type': i.isEven ? 'switch' : 'dimmer',
        'domain': 'lighting',
        'protocol': 'knx',
        'capabilities': i.isEven ? ['on_off'] : ['on_off', 'dim'],
        'state': i.isEven ? {'on': false} : {'on': true, 'level': 50},
        'health_status': 'online',
        'created_at': '2026-01-01T00:00:00Z',
        'updated_at': '2026-01-01T00:00:00Z',
      }),
      'count': count,
    };
  }

  /// Sample API response JSON for a scene list.
  static Map<String, dynamic> sceneListJson({int count = 2}) {
    return {
      'scenes': List.generate(count, (i) => {
        'id': 'scene-$i',
        'name': 'Scene $i',
        'slug': 'scene_$i',
        'room_id': 'living_room',
        'enabled': true,
        'sort_order': i,
        'actions': [
          {
            'device_id': 'dev-0',
            'command': 'turn_on',
          }
        ],
        'created_at': '2026-01-01T00:00:00Z',
        'updated_at': '2026-01-01T00:00:00Z',
      }),
      'count': count,
    };
  }

  /// Sample WebSocket device state change payload.
  static Map<String, dynamic> wsDeviceStateChanged({
    String deviceId = 'dev-001',
    Map<String, dynamic> state = const {'on': true},
  }) {
    return {
      'type': 'event',
      'channel': 'device.state_changed',
      'payload': {
        'device_id': deviceId,
        'state': state,
        'timestamp': DateTime.now().toUtc().toIso8601String(),
      },
    };
  }
}
