import 'dart:async';
import 'dart:math';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../config/constants.dart';
import '../models/ws_message.dart';
import '../services/websocket_service.dart';
import 'auth_provider.dart';

/// Provides the singleton WebSocketService instance.
final webSocketServiceProvider = Provider<WebSocketService>((ref) {
  final service = WebSocketService();
  ref.onDispose(() => service.dispose());
  return service;
});

/// Exposes the current WebSocket connection status as a stream.
/// Seeds with the current status so late subscribers don't get stuck in loading.
final connectionStatusProvider = StreamProvider<WSConnectionStatus>((ref) {
  final wsService = ref.watch(webSocketServiceProvider);
  // Prepend the current status so the provider has data immediately,
  // even if the broadcast stream already emitted before this listener attached.
  return wsService.connectionStatus.transform(
    StreamTransformer<WSConnectionStatus, WSConnectionStatus>.fromBind(
      (stream) => Stream.value(wsService.status).asyncExpand(
        (initial) async* {
          yield initial;
          await for (final status in stream) {
            yield status;
          }
        },
      ),
    ),
  );
});

/// Exposes incoming WebSocket events as a stream.
final wsEventsProvider = StreamProvider<WSInMessage>((ref) {
  final wsService = ref.watch(webSocketServiceProvider);
  return wsService.events;
});

/// Manages the WebSocket connection lifecycle.
/// Connects when authenticated, reconnects on disconnect.
final connectionManagerProvider = Provider<ConnectionManager>((ref) {
  final manager = ConnectionManager(ref);
  ref.onDispose(() => manager.dispose());
  return manager;
});

class ConnectionManager {
  final Ref _ref;
  Timer? _reconnectTimer;
  StreamSubscription<WSConnectionStatus>? _statusSubscription;
  int _reconnectAttempts = 0;
  bool _disposed = false;

  ConnectionManager(this._ref);

  WebSocketService get _wsService => _ref.read(webSocketServiceProvider);

  /// Establish WebSocket connection (call after successful login).
  Future<void> connect() async {
    if (_disposed) return;

    // Cancel any pending reconnect.
    _reconnectTimer?.cancel();

    // Cancel the previous status listener to prevent stacked subscriptions.
    await _statusSubscription?.cancel();
    _statusSubscription = null;

    final authRepo = _ref.read(authRepositoryProvider);
    final apiClient = _ref.read(apiClientProvider);

    try {
      final ticket = await authRepo.getWsTicket();
      final wsUrl = apiClient.getWsUrl(ticket);
      await _wsService.connect(wsUrl);

      // Reset backoff on successful connection.
      _reconnectAttempts = 0;

      // Subscribe to device state changes.
      _wsService.subscribe([
        WSChannel.deviceStateChanged,
        WSChannel.sceneActivated,
      ]);

      // Listen for disconnects — single subscription, replaced on each connect().
      _statusSubscription = _wsService.connectionStatus.listen((status) {
        if (status == WSConnectionStatus.disconnected) {
          _scheduleReconnect();
        }
      });
    } catch (e) {
      _scheduleReconnect();
    }
  }

  /// Disconnect intentionally (e.g., on logout).
  void disconnect() {
    _reconnectTimer?.cancel();
    _statusSubscription?.cancel();
    _statusSubscription = null;
    _reconnectAttempts = 0;
    _wsService.disconnect();
  }

  /// Clean up all resources.
  void dispose() {
    _disposed = true;
    disconnect();
  }

  void _scheduleReconnect() {
    if (_disposed) return;
    _reconnectTimer?.cancel();

    final delayMs = min(
      AppConstants.wsReconnectBaseMs * pow(2, _reconnectAttempts).toInt(),
      AppConstants.wsReconnectMaxMs,
    );
    _reconnectAttempts++;

    _reconnectTimer = Timer(Duration(milliseconds: delayMs), () async {
      if (!_disposed) {
        await connect();
      }
    });
  }
}
