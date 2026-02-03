import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import 'package:wallpanel/models/device.dart';
import 'package:wallpanel/providers/device_provider.dart';
import 'package:wallpanel/repositories/device_repository.dart';
import 'package:wallpanel/services/websocket_service.dart';
import 'package:wallpanel/providers/connection_provider.dart';
import 'package:wallpanel/providers/auth_provider.dart';
import 'package:wallpanel/services/api_client.dart';
import 'package:wallpanel/services/token_storage.dart';

class MockDeviceRepository extends Mock implements DeviceRepository {}

class MockWebSocketService extends Mock implements WebSocketService {}

class MockApiClient extends Mock implements ApiClient {}

class MockTokenStorage extends Mock implements TokenStorage {}

Device _makeDevice(String id, {bool on = false, int level = 0}) {
  return Device(
    id: id,
    name: 'Device $id',
    slug: 'device_$id',
    type: 'dimmer',
    domain: 'lighting',
    protocol: 'knx',
    capabilities: ['on_off', 'dim'],
    state: {'on': on, 'level': level},
    healthStatus: 'online',
    createdAt: DateTime(2026),
    updatedAt: DateTime(2026),
  );
}

void main() {
  group('RoomDevicesNotifier', () {
    late MockDeviceRepository mockRepo;
    late MockWebSocketService mockWs;
    late ProviderContainer container;

    setUp(() {
      mockRepo = MockDeviceRepository();
      mockWs = MockWebSocketService();

      when(() => mockWs.events).thenAnswer((_) => const Stream.empty());
      when(() => mockWs.connectionStatus).thenAnswer((_) => const Stream.empty());

      container = ProviderContainer(overrides: [
        deviceRepositoryProvider.overrideWithValue(mockRepo),
        webSocketServiceProvider.overrideWithValue(mockWs),
        tokenStorageProvider.overrideWithValue(MockTokenStorage()),
        apiClientProvider.overrideWithValue(MockApiClient()),
      ]);
    });

    tearDown(() {
      container.dispose();
    });

    test('loadDevices fetches from repository', () async {
      final devices = [_makeDevice('d1'), _makeDevice('d2', on: true)];
      when(() => mockRepo.getDevicesByRoom('living_room'))
          .thenAnswer((_) async => devices);

      final notifier = container.read(roomDevicesProvider.notifier);
      await notifier.loadDevices('living_room');

      final state = container.read(roomDevicesProvider);
      expect(state.value!.length, 2);
      expect(state.value![1].isOn, true);
    });

    test('loadDevices sets error state on failure', () async {
      when(() => mockRepo.getDevicesByRoom('bad_room'))
          .thenThrow(Exception('Not found'));

      final notifier = container.read(roomDevicesProvider.notifier);
      await notifier.loadDevices('bad_room');

      final state = container.read(roomDevicesProvider);
      expect(state.hasError, true);
    });

    test('toggleDevice marks device as pending and sends command', () async {
      final devices = [_makeDevice('d1', on: false)];
      when(() => mockRepo.getDevicesByRoom('room'))
          .thenAnswer((_) async => devices);
      when(() => mockRepo.toggle('d1', currentlyOn: false))
          .thenAnswer((_) async => const CommandResponse(
                commandId: 'c1', status: 'accepted', message: 'ok'));

      final notifier = container.read(roomDevicesProvider.notifier);
      await notifier.loadDevices('room');

      // Toggle — sends command, marks pending (does NOT change state)
      unawaited(notifier.toggleDevice('d1'));

      // Allow microtask to complete
      await Future.delayed(Duration.zero);

      // Device state unchanged — waiting for WebSocket confirmation
      final state = container.read(roomDevicesProvider);
      expect(state.value![0].isOn, false);

      // But device should be marked as pending
      final pending = container.read(pendingDevicesProvider);
      expect(pending.contains('d1'), true);
    });

    test('toggleDevice rolls back on API error', () async {
      final devices = [_makeDevice('d1', on: true)];
      when(() => mockRepo.getDevicesByRoom('room'))
          .thenAnswer((_) async => devices);
      when(() => mockRepo.toggle('d1', currentlyOn: true))
          .thenThrow(Exception('Network error'));

      final notifier = container.read(roomDevicesProvider.notifier);
      await notifier.loadDevices('room');
      await notifier.toggleDevice('d1');

      final state = container.read(roomDevicesProvider);
      // Should have rolled back to ON (original state)
      expect(state.value![0].isOn, true);
    });

    test('setLevel marks device as pending and sends command', () async {
      final devices = [_makeDevice('d1', on: true, level: 50)];
      when(() => mockRepo.getDevicesByRoom('room'))
          .thenAnswer((_) async => devices);
      when(() => mockRepo.setLevel('d1', 80))
          .thenAnswer((_) async => const CommandResponse(
                commandId: 'c2', status: 'accepted', message: 'ok'));

      final notifier = container.read(roomDevicesProvider.notifier);
      await notifier.loadDevices('room');
      unawaited(notifier.setLevel('d1', 80));
      await Future.delayed(Duration.zero);

      // Level unchanged — waiting for WebSocket confirmation
      final state = container.read(roomDevicesProvider);
      expect(state.value![0].level, 50);

      // But device should be marked as pending
      final pending = container.read(pendingDevicesProvider);
      expect(pending.contains('d1'), true);
    });

    test('setLevel rolls back on API error', () async {
      final devices = [_makeDevice('d1', on: true, level: 50)];
      when(() => mockRepo.getDevicesByRoom('room'))
          .thenAnswer((_) async => devices);
      when(() => mockRepo.setLevel('d1', 80))
          .thenThrow(Exception('Timeout'));

      final notifier = container.read(roomDevicesProvider.notifier);
      await notifier.loadDevices('room');
      await notifier.setLevel('d1', 80);

      final state = container.read(roomDevicesProvider);
      // Should have rolled back to 50
      expect(state.value![0].level, 50);
    });
  });
}
