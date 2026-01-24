import '../models/scene.dart';
import '../services/api_client.dart';

/// Fetches scenes and triggers activations via the Core API.
class SceneRepository {
  final ApiClient _apiClient;

  SceneRepository({required ApiClient apiClient}) : _apiClient = apiClient;

  /// Get all scenes for a room.
  Future<List<Scene>> getScenesByRoom(String roomId) async {
    final response = await _apiClient.getScenes(roomId: roomId);
    return response.scenes;
  }

  /// Activate a scene by ID.
  Future<ActivateResponse> activate(String sceneId) async {
    return _apiClient.activateScene(sceneId);
  }
}
