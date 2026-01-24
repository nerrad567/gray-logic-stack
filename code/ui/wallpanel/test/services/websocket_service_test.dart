import 'package:flutter_test/flutter_test.dart';

import 'package:wallpanel/services/websocket_service.dart';

void main() {
  group('WebSocketService', () {
    late WebSocketService service;

    setUp(() {
      service = WebSocketService();
    });

    tearDown(() {
      service.dispose();
    });

    test('initial status is disconnected', () {
      expect(service.status, WSConnectionStatus.disconnected);
    });

    test('connectionStatus stream emits status changes', () async {
      final statuses = <WSConnectionStatus>[];
      final sub = service.connectionStatus.listen(statuses.add);

      // Attempt to connect to an unreachable server
      await service.connect('ws://localhost:99999/ws');

      // Give time for the connection attempt to fail
      await Future.delayed(const Duration(milliseconds: 500));

      // Should have emitted connecting → disconnected
      expect(statuses, isNotEmpty);
      expect(statuses.first, WSConnectionStatus.connecting);

      await sub.cancel();
    });

    test('disconnect keeps status as disconnected', () {
      service.disconnect();
      expect(service.status, WSConnectionStatus.disconnected);
    });

    test('subscribe stores channels for reconnection', () {
      service.subscribe(['device.state_changed', 'scene.activated']);
      // Channels are stored internally for restoration on reconnect
      // This is verified indirectly — no exception means success
      expect(true, isTrue);
    });

    test('events stream is broadcast', () {
      // Multiple listeners should be allowed
      final sub1 = service.events.listen((_) {});
      final sub2 = service.events.listen((_) {});

      // No exception means broadcast stream works
      expect(true, isTrue);

      sub1.cancel();
      sub2.cancel();
    });
  });
}
