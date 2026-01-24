import '../models/area.dart';
import '../models/room.dart';
import '../services/api_client.dart';

/// Fetches area and room data from the Core API.
class LocationRepository {
  final ApiClient _apiClient;

  LocationRepository({required ApiClient apiClient}) : _apiClient = apiClient;

  /// Get all areas.
  Future<List<Area>> getAreas() async {
    final response = await _apiClient.getAreas();
    return response.areas;
  }

  /// Get all rooms.
  Future<List<Room>> getRooms() async {
    final response = await _apiClient.getRooms();
    return response.rooms;
  }

  /// Get rooms filtered by area.
  Future<List<Room>> getRoomsByArea(String areaId) async {
    final response = await _apiClient.getRooms(areaId: areaId);
    return response.rooms;
  }
}
