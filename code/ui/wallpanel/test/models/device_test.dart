import 'package:flutter_test/flutter_test.dart';
import 'package:wallpanel/models/device.dart';

void main() {
  group('Device', () {
    final sampleJson = <String, dynamic>{
      'id': 'dev-001',
      'name': 'Living Room Light',
      'slug': 'living_room_light',
      'room_id': 'living_room',
      'area_id': 'ground_floor',
      'type': 'dimmer',
      'domain': 'lighting',
      'protocol': 'knx',
      'address': {'ga_switch': '1/1/1', 'ga_dim': '1/1/2'},
      'gateway_id': 'knx-gw-1',
      'capabilities': ['on_off', 'dim'],
      'config': {'min_level': 10},
      'state': {'on': true, 'level': 75},
      'state_updated_at': '2026-01-20T10:30:00Z',
      'health_status': 'online',
      'health_last_seen': '2026-01-20T10:30:00Z',
      'phm_enabled': true,
      'phm_baseline': {'avg_on_hours': 8.5},
      'manufacturer': 'ABB',
      'model': 'SA/S 4.16.6.2',
      'firmware_version': '2.1.0',
      'created_at': '2026-01-01T00:00:00Z',
      'updated_at': '2026-01-20T10:30:00Z',
    };

    test('fromJson parses all fields correctly', () {
      final device = Device.fromJson(sampleJson);

      expect(device.id, 'dev-001');
      expect(device.name, 'Living Room Light');
      expect(device.slug, 'living_room_light');
      expect(device.roomId, 'living_room');
      expect(device.areaId, 'ground_floor');
      expect(device.type, 'dimmer');
      expect(device.domain, 'lighting');
      expect(device.protocol, 'knx');
      expect(device.address['ga_switch'], '1/1/1');
      expect(device.gatewayId, 'knx-gw-1');
      expect(device.capabilities, ['on_off', 'dim']);
      expect(device.config['min_level'], 10);
      expect(device.state['on'], true);
      expect(device.state['level'], 75);
      expect(device.stateUpdatedAt, isNotNull);
      expect(device.healthStatus, 'online');
      expect(device.phmEnabled, true);
      expect(device.phmBaseline!['avg_on_hours'], 8.5);
      expect(device.manufacturer, 'ABB');
      expect(device.model, 'SA/S 4.16.6.2');
      expect(device.firmwareVersion, '2.1.0');
    });

    test('fromJson handles missing optional fields', () {
      final minimal = <String, dynamic>{
        'id': 'dev-002',
        'name': 'Switch',
        'slug': 'switch',
        'type': 'switch',
        'domain': 'lighting',
        'protocol': 'knx',
        'created_at': '2026-01-01T00:00:00Z',
        'updated_at': '2026-01-01T00:00:00Z',
      };

      final device = Device.fromJson(minimal);

      expect(device.roomId, isNull);
      expect(device.areaId, isNull);
      expect(device.address, isEmpty);
      expect(device.gatewayId, isNull);
      expect(device.capabilities, isEmpty);
      expect(device.config, isEmpty);
      expect(device.state, isEmpty);
      expect(device.stateUpdatedAt, isNull);
      expect(device.healthStatus, 'unknown');
      expect(device.phmEnabled, false);
      expect(device.manufacturer, isNull);
    });

    test('toJson produces correct keys', () {
      final device = Device.fromJson(sampleJson);
      final json = device.toJson();

      expect(json['id'], 'dev-001');
      expect(json['room_id'], 'living_room');
      expect(json['health_status'], 'online');
      expect(json['phm_enabled'], true);
      expect(json['created_at'], contains('2026'));
    });

    test('toJson omits null optional fields', () {
      final device = Device.fromJson({
        'id': 'dev-003',
        'name': 'Minimal',
        'slug': 'minimal',
        'type': 'switch',
        'domain': 'lighting',
        'protocol': 'knx',
        'created_at': '2026-01-01T00:00:00Z',
        'updated_at': '2026-01-01T00:00:00Z',
      });
      final json = device.toJson();

      expect(json.containsKey('room_id'), false);
      expect(json.containsKey('area_id'), false);
      expect(json.containsKey('gateway_id'), false);
      expect(json.containsKey('manufacturer'), false);
    });

    test('round-trip fromJson/toJson preserves data', () {
      final device = Device.fromJson(sampleJson);
      final json = device.toJson();
      final restored = Device.fromJson(json);

      expect(restored.id, device.id);
      expect(restored.name, device.name);
      expect(restored.state['on'], device.state['on']);
      expect(restored.state['level'], device.state['level']);
      expect(restored.capabilities, device.capabilities);
    });

    test('convenience getters work correctly', () {
      final device = Device.fromJson(sampleJson);

      expect(device.isOn, true);
      expect(device.level, 75);
      expect(device.hasOnOff, true);
      expect(device.hasDim, true);
      expect(device.isOnline, true);
    });

    test('isOn returns false when state has no on key', () {
      final device = Device.fromJson({
        'id': 'dev-004',
        'name': 'Off Light',
        'slug': 'off_light',
        'type': 'switch',
        'domain': 'lighting',
        'protocol': 'knx',
        'state': {'on': false},
        'created_at': '2026-01-01T00:00:00Z',
        'updated_at': '2026-01-01T00:00:00Z',
      });

      expect(device.isOn, false);
    });

    test('copyWith updates specified fields', () {
      final device = Device.fromJson(sampleJson);
      final updated = device.copyWith(
        state: {'on': false, 'level': 0},
        healthStatus: 'offline',
      );

      expect(updated.isOn, false);
      expect(updated.level, 0);
      expect(updated.healthStatus, 'offline');
      // Unchanged fields preserved
      expect(updated.id, device.id);
      expect(updated.name, device.name);
    });
  });

  group('DeviceListResponse', () {
    test('fromJson parses device list', () {
      final json = <String, dynamic>{
        'devices': [
          {
            'id': 'dev-001',
            'name': 'Light 1',
            'slug': 'light_1',
            'type': 'switch',
            'domain': 'lighting',
            'protocol': 'knx',
            'created_at': '2026-01-01T00:00:00Z',
            'updated_at': '2026-01-01T00:00:00Z',
          },
          {
            'id': 'dev-002',
            'name': 'Light 2',
            'slug': 'light_2',
            'type': 'dimmer',
            'domain': 'lighting',
            'protocol': 'knx',
            'created_at': '2026-01-01T00:00:00Z',
            'updated_at': '2026-01-01T00:00:00Z',
          },
        ],
        'count': 2,
      };

      final response = DeviceListResponse.fromJson(json);
      expect(response.devices.length, 2);
      expect(response.count, 2);
      expect(response.devices[0].id, 'dev-001');
      expect(response.devices[1].type, 'dimmer');
    });

    test('fromJson handles empty list', () {
      final response = DeviceListResponse.fromJson({'devices': [], 'count': 0});
      expect(response.devices, isEmpty);
      expect(response.count, 0);
    });
  });

  group('CommandResponse', () {
    test('fromJson parses correctly', () {
      final response = CommandResponse.fromJson({
        'command_id': 'cmd-123',
        'status': 'accepted',
        'message': 'Command queued',
      });

      expect(response.commandId, 'cmd-123');
      expect(response.status, 'accepted');
      expect(response.message, 'Command queued');
    });
  });
}
