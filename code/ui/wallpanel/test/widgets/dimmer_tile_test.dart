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
import 'package:wallpanel/widgets/dimmer_tile.dart';

class MockDeviceRepository extends Mock implements DeviceRepository {}

class MockWebSocketService extends Mock implements WebSocketService {}

class MockApiClient extends Mock implements ApiClient {}

class MockTokenStorage extends Mock implements TokenStorage {}

void main() {
  group('DimmerTile', () {
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
              child: DimmerTile(device: device),
            ),
          ),
        ),
      );
    }

    testWidgets('displays device name', (tester) async {
      final device = Device(
        id: 'd1',
        name: 'Lounge Dimmer',
        slug: 'lounge_dimmer',
        type: 'dimmer',
        domain: 'lighting',
        protocol: 'knx',
        capabilities: ['on_off', 'dim'],
        state: {'on': true, 'level': 60},
        healthStatus: 'online',
        createdAt: DateTime(2026),
        updatedAt: DateTime(2026),
      );

      await tester.pumpWidget(buildWidget(device));
      expect(find.text('Lounge Dimmer'), findsOneWidget);
    });

    testWidgets('shows percentage level', (tester) async {
      final device = Device(
        id: 'd1',
        name: 'Dimmer',
        slug: 'dimmer',
        type: 'dimmer',
        domain: 'lighting',
        protocol: 'knx',
        capabilities: ['on_off', 'dim'],
        state: {'on': true, 'level': 75},
        healthStatus: 'online',
        createdAt: DateTime(2026),
        updatedAt: DateTime(2026),
      );

      await tester.pumpWidget(buildWidget(device));
      expect(find.text('75%'), findsOneWidget);
    });

    testWidgets('contains a slider', (tester) async {
      final device = Device(
        id: 'd1',
        name: 'Dimmer',
        slug: 'dimmer',
        type: 'dimmer',
        domain: 'lighting',
        protocol: 'knx',
        capabilities: ['on_off', 'dim'],
        state: {'on': true, 'level': 50},
        healthStatus: 'online',
        createdAt: DateTime(2026),
        updatedAt: DateTime(2026),
      );

      await tester.pumpWidget(buildWidget(device));
      expect(find.byType(Slider), findsOneWidget);
    });

    testWidgets('shows 0% when device is off', (tester) async {
      final device = Device(
        id: 'd1',
        name: 'Dimmer',
        slug: 'dimmer',
        type: 'dimmer',
        domain: 'lighting',
        protocol: 'knx',
        capabilities: ['on_off', 'dim'],
        state: {'on': false, 'level': 0},
        healthStatus: 'online',
        createdAt: DateTime(2026),
        updatedAt: DateTime(2026),
      );

      await tester.pumpWidget(buildWidget(device));
      expect(find.text('0%'), findsOneWidget);
    });

    testWidgets('has circular progress indicator', (tester) async {
      final device = Device(
        id: 'd1',
        name: 'Dimmer',
        slug: 'dimmer',
        type: 'dimmer',
        domain: 'lighting',
        protocol: 'knx',
        capabilities: ['on_off', 'dim'],
        state: {'on': true, 'level': 50},
        healthStatus: 'online',
        createdAt: DateTime(2026),
        updatedAt: DateTime(2026),
      );

      await tester.pumpWidget(buildWidget(device));
      expect(find.byType(CircularProgressIndicator), findsOneWidget);
    });
  });
}
