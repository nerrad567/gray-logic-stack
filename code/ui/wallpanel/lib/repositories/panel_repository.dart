import '../models/panel.dart';
import '../services/api_client.dart';

/// Handles panel CRUD and room assignment via the Core API.
class PanelRepository {
  final ApiClient _apiClient;

  PanelRepository({required ApiClient apiClient}) : _apiClient = apiClient;

  /// Get all panels.
  Future<List<Panel>> getPanels() async {
    final response = await _apiClient.getPanels();
    return response.panels;
  }

  /// Create a new panel. Returns the full response including the one-time token.
  Future<PanelCreateResponse> createPanel(Map<String, dynamic> data) async {
    return _apiClient.createPanel(data);
  }

  /// Get a single panel by ID with its room assignments.
  Future<PanelDetailResponse> getPanel(String id) async {
    return _apiClient.getPanel(id);
  }

  /// Update a panel (PATCH semantics).
  Future<Panel> updatePanel(String id, Map<String, dynamic> data) async {
    return _apiClient.updatePanel(id, data);
  }

  /// Delete a panel.
  Future<void> deletePanel(String id) async {
    await _apiClient.deletePanel(id);
  }

  /// Get room IDs assigned to a panel.
  Future<List<String>> getPanelRooms(String id) async {
    final response = await _apiClient.getPanelRooms(id);
    return response.roomIds;
  }

  /// Replace all room assignments for a panel.
  Future<PanelRoomsResponse> setPanelRooms(String id, List<String> roomIds) async {
    return _apiClient.setPanelRooms(id, roomIds);
  }
}
