import '../models/device.dart';
import '../services/api_client.dart';

/// Fetches devices and sends commands to the Core API.
class DeviceRepository {
  final ApiClient _apiClient;

  DeviceRepository({required ApiClient apiClient}) : _apiClient = apiClient;

  /// Get all devices in a room.
  Future<List<Device>> getDevicesByRoom(String roomId) async {
    final response = await _apiClient.getDevices(roomId: roomId);
    return response.devices;
  }

  /// Get a single device by ID.
  Future<Device> getDevice(String id) async {
    return _apiClient.getDevice(id);
  }

  /// Send an on command to a device.
  Future<CommandResponse> turnOn(String deviceId) async {
    return _apiClient.setDeviceState(deviceId, command: 'on');
  }

  /// Send an off command to a device.
  Future<CommandResponse> turnOff(String deviceId) async {
    return _apiClient.setDeviceState(deviceId, command: 'off');
  }

  /// Toggle a device's on/off state.
  Future<CommandResponse> toggle(String deviceId, {required bool currentlyOn}) async {
    return currentlyOn ? turnOff(deviceId) : turnOn(deviceId);
  }

  /// Set brightness level (0-100) for a dimmable device.
  Future<CommandResponse> setLevel(String deviceId, int level) async {
    return _apiClient.setDeviceState(
      deviceId,
      command: 'dim',
      parameters: {'level': level},
    );
  }

  /// Set blind position (0-100) for a blind/shutter device.
  Future<CommandResponse> setPosition(String deviceId, int position) async {
    return _apiClient.setDeviceState(
      deviceId,
      command: 'set_position',
      parameters: {'position': position},
    );
  }

  /// Send a generic command with parameters.
  Future<CommandResponse> sendCommand(
    String deviceId, {
    required String command,
    Map<String, dynamic>? parameters,
  }) async {
    return _apiClient.setDeviceState(
      deviceId,
      command: command,
      parameters: parameters,
    );
  }
}
