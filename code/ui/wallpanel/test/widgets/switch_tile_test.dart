import 'package:flutter/material.dart';
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
import 'package:wallpanel/widgets/switch_tile.dart';

class MockDeviceRepository extends Mock implements DeviceRepository {}

class MockWebSocketService extends Mock implements WebSocketService {}

class MockApiClient extends Mock implements ApiClient {}

class MockTokenStorage extends Mock implements TokenStorage {}

void main() {
  group('SwitchTile', () {
    late MockDeviceRepository mockRepo;
    late MockWebSocketService mockWs;

    setUp(() {
      mockRepo = MockDeviceRepository();
      mockWs = MockWebSocketService();
      when(() => mockWs.events).thenAnswer((_) => const Stream.empty());
      when(() => mockWs.connectionStatus).thenAnswer((_) => const Stream.empty());
    });

    Widget buildWidget(Device device) {
      return ProviderScope(
        overrides: [
          deviceRepositoryProvider.overrideWithValue(mockRepo),
          webSocketServiceProvider.overrideWithValue(mockWs),
          tokenStorageProvider.overrideWithValue(MockTokenStorage()),
          apiClientProvider.overrideWithValue(MockApiClient()),
        ],
        child: MaterialApp(
          home: Scaffold(
            body: SizedBox(
              width: 170,
              height: 200,
              child: SwitchTile(device: device),
            ),
          ),
        ),
      );
    }

    testWidgets('displays device name', (tester) async {
      final device = Device(
        id: 'd1',
        name: 'Kitchen Light',
        slug: 'kitchen_light',
        type: 'switch',
        domain: 'lighting',
        protocol: 'knx',
        capabilities: ['on_off'],
        state: {'on': false},
        healthStatus: 'online',
        createdAt: DateTime(2026),
        updatedAt: DateTime(2026),
      );

      await tester.pumpWidget(buildWidget(device));
      expect(find.text('Kitchen Light'), findsOneWidget);
    });

    testWidgets('shows ON status when device is on', (tester) async {
      final device = Device(
        id: 'd1',
        name: 'Light',
        slug: 'light',
        type: 'switch',
        domain: 'lighting',
        protocol: 'knx',
        capabilities: ['on_off'],
        state: {'on': true},
        healthStatus: 'online',
        createdAt: DateTime(2026),
        updatedAt: DateTime(2026),
      );

      await tester.pumpWidget(buildWidget(device));
      expect(find.text('ON'), findsOneWidget);
    });

    testWidgets('shows OFF status when device is off', (tester) async {
      final device = Device(
        id: 'd1',
        name: 'Light',
        slug: 'light',
        type: 'switch',
        domain: 'lighting',
        protocol: 'knx',
        capabilities: ['on_off'],
        state: {'on': false},
        healthStatus: 'online',
        createdAt: DateTime(2026),
        updatedAt: DateTime(2026),
      );

      await tester.pumpWidget(buildWidget(device));
      expect(find.text('OFF'), findsOneWidget);
    });

    testWidgets('shows Offline when device is unreachable', (tester) async {
      final device = Device(
        id: 'd1',
        name: 'Light',
        slug: 'light',
        type: 'switch',
        domain: 'lighting',
        protocol: 'knx',
        capabilities: ['on_off'],
        state: {'on': false},
        healthStatus: 'offline',
        createdAt: DateTime(2026),
        updatedAt: DateTime(2026),
      );

      await tester.pumpWidget(buildWidget(device));
      expect(find.text('Offline'), findsOneWidget);
    });

    testWidgets('has power icon', (tester) async {
      final device = Device(
        id: 'd1',
        name: 'Light',
        slug: 'light',
        type: 'switch',
        domain: 'lighting',
        protocol: 'knx',
        capabilities: ['on_off'],
        state: {'on': false},
        healthStatus: 'online',
        createdAt: DateTime(2026),
        updatedAt: DateTime(2026),
      );

      await tester.pumpWidget(buildWidget(device));
      expect(find.byIcon(Icons.power_settings_new), findsOneWidget);
    });
  });
}
