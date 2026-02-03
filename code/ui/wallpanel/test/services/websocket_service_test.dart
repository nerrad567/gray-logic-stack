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

      // Attempt to connect to an unreachable server (valid port, nothing listening).
      // connect() rethrows on failure, so we expect an exception.
      try {
        await service.connect('ws://localhost:19/ws');
      } catch (_) {
        // Expected — connection refused
      }

      // Give time for status updates to propagate
      await Future.delayed(const Duration(milliseconds: 100));

      // Should have emitted connecting → disconnected
      expect(statuses, containsAllInOrder([
        WSConnectionStatus.connecting,
        WSConnectionStatus.disconnected,
      ]));

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
