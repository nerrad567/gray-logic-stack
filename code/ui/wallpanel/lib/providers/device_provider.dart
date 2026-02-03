import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../config/constants.dart';
import '../models/device.dart';
import '../models/ws_message.dart';
import '../repositories/device_repository.dart';
import 'auth_provider.dart';
import 'connection_provider.dart';

/// Provides the DeviceRepository.
final deviceRepositoryProvider = Provider<DeviceRepository>((ref) {
  return DeviceRepository(apiClient: ref.watch(apiClientProvider));
});

/// Tracks which devices are currently in a "pending" state (waiting for
/// bridge/WebSocket confirmation). Widgets use this to show a loading indicator.
final pendingDevicesProvider =
    NotifierProvider<PendingDevicesNotifier, Set<String>>(
  PendingDevicesNotifier.new,
);

class PendingDevicesNotifier extends Notifier<Set<String>> {
  @override
  Set<String> build() => {};

  void markPending(String deviceId) {
    state = {...state, deviceId};
  }

  void clearPending(String deviceId) {
    state = state.where((id) => id != deviceId).toSet();
  }

  bool isPending(String deviceId) => state.contains(deviceId);
}

/// Provides the list of devices for the configured room.
/// Listens to WebSocket events for real-time state updates.
final roomDevicesProvider =
    NotifierProvider<RoomDevicesNotifier, AsyncValue<List<Device>>>(
  RoomDevicesNotifier.new,
);

class RoomDevicesNotifier extends Notifier<AsyncValue<List<Device>>> {
  StreamSubscription<WSInMessage>? _wsSubscription;
  final Map<String, Timer> _timeoutTimers = {};

  @override
  AsyncValue<List<Device>> build() {
    _listenToWebSocket();

    ref.onDispose(() {
      _wsSubscription?.cancel();
      for (final timer in _timeoutTimers.values) {
        timer.cancel();
      }
      _timeoutTimers.clear();
    });

    return const AsyncValue.loading();
  }

  DeviceRepository get _deviceRepo => ref.read(deviceRepositoryProvider);
  PendingDevicesNotifier get _pending => ref.read(pendingDevicesProvider.notifier);

  /// Load devices for a room from the API.
  /// Use '__all__' to load all devices regardless of room assignment.
  Future<void> loadDevices(String roomId) async {
    state = const AsyncValue.loading();
    try {
      final List<Device> devices;
      if (roomId == '__all__') {
        // Load all devices (no room filter)
        devices = await _deviceRepo.getAllDevices();
      } else {
        devices = await _deviceRepo.getDevicesByRoom(roomId);
      }
      state = AsyncValue.data(devices);
    } catch (e, st) {
      state = AsyncValue.error(e, st);
    }
  }

  /// Toggle a device's on/off state with pending confirmation pattern.
  /// Instead of flipping state immediately, marks device as "pending" and
  /// waits for WebSocket confirmation from the backend/bridge.
  Future<void> toggleDevice(String deviceId) async {
    final devices = state.value;
    if (devices == null) return;

    final index = devices.indexWhere((d) => d.id == deviceId);
    if (index == -1) return;

    final device = devices[index];
    final wasOn = device.isOn;

    // 1. Mark device as pending (widgets show spinner/pulse)
    _pending.markPending(deviceId);

    // 2. Start timeout timer — if no WS confirmation within 3s, clear pending
    _startTimeoutTimer(deviceId);

    // 3. Send command to API
    try {
      await _deviceRepo.toggle(deviceId, currentlyOn: wasOn);
    } catch (_) {
      // API call failed — immediately clear pending state
      _cancelTimeoutTimer(deviceId);
      _pending.clearPending(deviceId);
    }
  }

  /// Set blind position with pending confirmation pattern.
  Future<void> setPosition(String deviceId, int position) async {
    final devices = state.value;
    if (devices == null) return;

    final index = devices.indexWhere((d) => d.id == deviceId);
    if (index == -1) return;

    _pending.markPending(deviceId);
    _startTimeoutTimer(deviceId);

    try {
      await _deviceRepo.setPosition(deviceId, position);
    } catch (_) {
      _cancelTimeoutTimer(deviceId);
      _pending.clearPending(deviceId);
    }
  }

  /// Set brightness level with pending confirmation pattern.
  Future<void> setLevel(String deviceId, int level) async {
    final devices = state.value;
    if (devices == null) return;

    final index = devices.indexWhere((d) => d.id == deviceId);
    if (index == -1) return;

    // 1. Mark device as pending
    _pending.markPending(deviceId);

    // 2. Start timeout timer
    _startTimeoutTimer(deviceId);

    // 3. Send command
    try {
      await _deviceRepo.setLevel(deviceId, level);
    } catch (_) {
      _cancelTimeoutTimer(deviceId);
      _pending.clearPending(deviceId);
    }
  }

  /// Set thermostat setpoint with pending confirmation pattern.
  Future<void> setSetpoint(String deviceId, double setpoint) async {
    final devices = state.value;
    if (devices == null) return;

    final index = devices.indexWhere((d) => d.id == deviceId);
    if (index == -1) return;

    _pending.markPending(deviceId);
    _startTimeoutTimer(deviceId);

    try {
      await _deviceRepo.setSetpoint(deviceId, setpoint);
    } catch (_) {
      _cancelTimeoutTimer(deviceId);
      _pending.clearPending(deviceId);
    }
  }

  /// Listen to WebSocket events for real-time device state updates.
  void _listenToWebSocket() {
    final wsService = ref.read(webSocketServiceProvider);
    _wsSubscription = wsService.events.listen((msg) {
      if (msg.isDeviceStateChanged) {
        final event = DeviceStateEvent.fromPayload(msg.payload);
        if (event.deviceId != null) {
          // WebSocket confirmed the state — clear pending and apply
          _cancelTimeoutTimer(event.deviceId!);
          _pending.clearPending(event.deviceId!);
          _updateDeviceState(event.deviceId!, event.state);
        }
      }
    });
  }

  /// Update a device's state in the local list.
  void _updateDeviceState(String deviceId, Map<String, dynamic> newState) {
    final devices = state.value;
    if (devices == null) return;

    final updated = devices.map((d) {
      if (d.id == deviceId) {
        final mergedState = {...d.state, ...newState};
        return d.copyWith(
          state: mergedState,
          stateUpdatedAt: DateTime.now().toUtc(),
        );
      }
      return d;
    }).toList();

    state = AsyncValue.data(updated);
  }

  /// Start a timer that clears pending state if no WebSocket confirmation arrives.
  void _startTimeoutTimer(String deviceId) {
    _cancelTimeoutTimer(deviceId);
    _timeoutTimers[deviceId] = Timer(
      const Duration(milliseconds: AppConstants.optimisticRollbackMs),
      () {
        _pending.clearPending(deviceId);
        _timeoutTimers.remove(deviceId);
      },
    );
  }

  void _cancelTimeoutTimer(String deviceId) {
    _timeoutTimers[deviceId]?.cancel();
    _timeoutTimers.remove(deviceId);
  }
}
