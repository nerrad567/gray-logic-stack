import 'package:flutter_test/flutter_test.dart';
import 'package:wallpanel/models/scene.dart';

void main() {
  group('Scene', () {
    final sampleJson = <String, dynamic>{
      'id': 'scene-001',
      'name': 'Movie Night',
      'slug': 'movie_night',
      'description': 'Dim lights for cinema mode',
      'room_id': 'living_room',
      'area_id': 'ground_floor',
      'enabled': true,
      'priority': 80,
      'icon': 'movie',
      'colour': '#4ECDC4',
      'category': 'entertainment',
      'actions': [
        {
          'device_id': 'dev-001',
          'command': 'set_level',
          'parameters': {'level': 20},
          'delay_ms': 0,
          'fade_ms': 2000,
          'parallel': false,
          'continue_on_error': false,
          'sort_order': 0,
        },
        {
          'device_id': 'dev-002',
          'command': 'turn_off',
          'delay_ms': 500,
          'fade_ms': 0,
          'parallel': true,
          'continue_on_error': true,
          'sort_order': 1,
        },
      ],
      'sort_order': 5,
      'created_at': '2026-01-01T00:00:00Z',
      'updated_at': '2026-01-15T12:00:00Z',
    };

    test('fromJson parses all fields correctly', () {
      final scene = Scene.fromJson(sampleJson);

      expect(scene.id, 'scene-001');
      expect(scene.name, 'Movie Night');
      expect(scene.slug, 'movie_night');
      expect(scene.description, 'Dim lights for cinema mode');
      expect(scene.roomId, 'living_room');
      expect(scene.areaId, 'ground_floor');
      expect(scene.enabled, true);
      expect(scene.priority, 80);
      expect(scene.icon, 'movie');
      expect(scene.colour, '#4ECDC4');
      expect(scene.category, 'entertainment');
      expect(scene.actions.length, 2);
      expect(scene.sortOrder, 5);
    });

    test('fromJson handles missing optional fields', () {
      final minimal = <String, dynamic>{
        'id': 'scene-002',
        'name': 'All Off',
        'slug': 'all_off',
        'created_at': '2026-01-01T00:00:00Z',
        'updated_at': '2026-01-01T00:00:00Z',
      };

      final scene = Scene.fromJson(minimal);

      expect(scene.description, isNull);
      expect(scene.roomId, isNull);
      expect(scene.enabled, true);
      expect(scene.priority, 50);
      expect(scene.icon, isNull);
      expect(scene.colour, isNull);
      expect(scene.category, isNull);
      expect(scene.actions, isEmpty);
      expect(scene.sortOrder, 0);
    });

    test('toJson produces correct keys', () {
      final scene = Scene.fromJson(sampleJson);
      final json = scene.toJson();

      expect(json['id'], 'scene-001');
      expect(json['room_id'], 'living_room');
      expect(json['sort_order'], 5);
      expect(json['actions'], isList);
      expect((json['actions'] as List).length, 2);
    });

    test('toJson omits null optional fields', () {
      final scene = Scene.fromJson({
        'id': 'scene-003',
        'name': 'Minimal',
        'slug': 'minimal',
        'created_at': '2026-01-01T00:00:00Z',
        'updated_at': '2026-01-01T00:00:00Z',
      });
      final json = scene.toJson();

      expect(json.containsKey('description'), false);
      expect(json.containsKey('room_id'), false);
      expect(json.containsKey('icon'), false);
      expect(json.containsKey('colour'), false);
    });

    test('round-trip fromJson/toJson preserves data', () {
      final scene = Scene.fromJson(sampleJson);
      final json = scene.toJson();
      final restored = Scene.fromJson(json);

      expect(restored.id, scene.id);
      expect(restored.name, scene.name);
      expect(restored.actions.length, scene.actions.length);
      expect(restored.priority, scene.priority);
      expect(restored.sortOrder, scene.sortOrder);
    });
  });

  group('SceneAction', () {
    test('fromJson parses all fields', () {
      final action = SceneAction.fromJson({
        'device_id': 'dev-001',
        'command': 'set_level',
        'parameters': {'level': 50},
        'delay_ms': 100,
        'fade_ms': 2000,
        'parallel': true,
        'continue_on_error': true,
        'sort_order': 3,
      });

      expect(action.deviceId, 'dev-001');
      expect(action.command, 'set_level');
      expect(action.parameters!['level'], 50);
      expect(action.delayMs, 100);
      expect(action.fadeMs, 2000);
      expect(action.parallel, true);
      expect(action.continueOnError, true);
      expect(action.sortOrder, 3);
    });

    test('fromJson uses defaults for missing fields', () {
      final action = SceneAction.fromJson({
        'device_id': 'dev-002',
        'command': 'turn_on',
      });

      expect(action.parameters, isNull);
      expect(action.delayMs, 0);
      expect(action.fadeMs, 0);
      expect(action.parallel, false);
      expect(action.continueOnError, false);
      expect(action.sortOrder, 0);
    });

    test('toJson produces correct output', () {
      final action = SceneAction(
        deviceId: 'dev-001',
        command: 'set_level',
        parameters: {'level': 75},
        fadeMs: 1000,
      );
      final json = action.toJson();

      expect(json['device_id'], 'dev-001');
      expect(json['command'], 'set_level');
      expect(json['parameters'], {'level': 75});
      expect(json['fade_ms'], 1000);
      expect(json['delay_ms'], 0);
    });

    test('toJson omits null parameters', () {
      final action = SceneAction(deviceId: 'dev-001', command: 'turn_off');
      final json = action.toJson();

      expect(json.containsKey('parameters'), false);
    });
  });

  group('SceneListResponse', () {
    test('fromJson parses scene list', () {
      final response = SceneListResponse.fromJson({
        'scenes': [
          {
            'id': 'scene-001',
            'name': 'Scene 1',
            'slug': 'scene_1',
            'created_at': '2026-01-01T00:00:00Z',
            'updated_at': '2026-01-01T00:00:00Z',
          },
        ],
        'count': 1,
      });

      expect(response.scenes.length, 1);
      expect(response.count, 1);
      expect(response.scenes[0].name, 'Scene 1');
    });

    test('fromJson handles empty list', () {
      final response = SceneListResponse.fromJson({'scenes': [], 'count': 0});
      expect(response.scenes, isEmpty);
    });
  });

  group('ActivateResponse', () {
    test('fromJson parses correctly', () {
      final response = ActivateResponse.fromJson({
        'execution_id': 'exec-456',
        'status': 'running',
        'message': 'Scene activated',
      });

      expect(response.executionId, 'exec-456');
      expect(response.status, 'running');
      expect(response.message, 'Scene activated');
    });
  });
}
