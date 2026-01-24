import 'dart:async';
import 'dart:convert';

import 'package:web_socket_channel/web_socket_channel.dart';

import '../config/constants.dart';
import '../models/ws_message.dart';

/// WebSocket connection status.
enum WSConnectionStatus { disconnected, connecting, connected }

/// Manages the WebSocket connection to Gray Logic Core.
/// Handles subscription restoration on reconnect.
/// Reconnection logic is owned by [ConnectionManager] — this class
/// only reports status changes.
class WebSocketService {
  WebSocketChannel? _channel;
  StreamSubscription? _channelSubscription;
  Timer? _pingTimer;
  final List<String> _subscribedChannels = [];
  bool _intentionalClose = false;

  final _eventController = StreamController<WSInMessage>.broadcast();
  final _statusController = StreamController<WSConnectionStatus>.broadcast();

  WSConnectionStatus _status = WSConnectionStatus.disconnected;

  /// Stream of incoming WebSocket events.
  Stream<WSInMessage> get events => _eventController.stream;

  /// Stream of connection status changes.
  Stream<WSConnectionStatus> get connectionStatus => _statusController.stream;

  /// Current connection status.
  WSConnectionStatus get status => _status;

  /// Connect to the WebSocket server using a ticket URL.
  /// Returns normally on success; throws on failure.
  Future<void> connect(String wsUrl) async {
    _intentionalClose = false;

    // Tear down any previous connection cleanly before opening a new one.
    await _closeExistingConnection();

    _setStatus(WSConnectionStatus.connecting);

    try {
      _channel = WebSocketChannel.connect(Uri.parse(wsUrl));
      await _channel!.ready;

      _setStatus(WSConnectionStatus.connected);
      _startPingTimer();

      // Listen to the new channel's stream. Store the subscription so we can
      // cancel it explicitly on the next connect() or disconnect().
      _channelSubscription = _channel!.stream.listen(
        _onMessage,
        onDone: _onDisconnect,
        onError: _onError,
      );

      // Restore subscriptions if reconnecting.
      if (_subscribedChannels.isNotEmpty) {
        subscribe(_subscribedChannels);
      }
    } catch (e) {
      _setStatus(WSConnectionStatus.disconnected);
      rethrow;
    }
  }

  /// Subscribe to event channels.
  void subscribe(List<String> channels) {
    for (final ch in channels) {
      if (!_subscribedChannels.contains(ch)) {
        _subscribedChannels.add(ch);
      }
    }

    if (_status != WSConnectionStatus.connected) return;
    _send(WSOutMessage.subscribe(channels));
  }

  /// Unsubscribe from event channels.
  void unsubscribe(List<String> channels) {
    _subscribedChannels.removeWhere((ch) => channels.contains(ch));

    if (_status != WSConnectionStatus.connected) return;
    _send(WSOutMessage.unsubscribe(channels));
  }

  /// Disconnect and stop reconnecting.
  void disconnect() {
    _intentionalClose = true;
    _cleanup();
    _setStatus(WSConnectionStatus.disconnected);
  }

  /// Dispose all resources.
  void dispose() {
    disconnect();
    _eventController.close();
    _statusController.close();
  }

  // --- Private ---

  void _onMessage(dynamic data) {
    if (data is! String) return;

    try {
      final json = jsonDecode(data) as Map<String, dynamic>;
      final msg = WSInMessage.fromJson(json);
      _eventController.add(msg);
    } catch (e) {
      // Malformed message — ignore.
    }
  }

  void _onDisconnect() {
    _cleanup();
    if (!_intentionalClose) {
      _setStatus(WSConnectionStatus.disconnected);
    }
  }

  void _onError(dynamic error) {
    _cleanup();
    if (!_intentionalClose) {
      _setStatus(WSConnectionStatus.disconnected);
    }
  }

  void _startPingTimer() {
    _pingTimer?.cancel();
    _pingTimer = Timer.periodic(
      const Duration(milliseconds: AppConstants.wsPingIntervalMs),
      (_) => _send(WSOutMessage.ping()),
    );
  }

  void _send(WSOutMessage message) {
    if (_channel == null) return;
    try {
      _channel!.sink.add(jsonEncode(message.toJson()));
    } catch (_) {
      // Channel closed — will be handled by onDone/onError.
    }
  }

  /// Close the previous channel and cancel its stream subscription.
  /// This prevents stale onDone/onError callbacks from affecting a new connection.
  Future<void> _closeExistingConnection() async {
    _pingTimer?.cancel();
    await _channelSubscription?.cancel();
    _channelSubscription = null;
    try {
      _channel?.sink.close();
    } catch (_) {}
    _channel = null;
  }

  void _cleanup() {
    _pingTimer?.cancel();
    _channelSubscription?.cancel();
    _channelSubscription = null;
    try {
      _channel?.sink.close();
    } catch (_) {}
    _channel = null;
  }

  void _setStatus(WSConnectionStatus newStatus) {
    if (_status == newStatus) return;
    _status = newStatus;
    _statusController.add(newStatus);
  }
}
