import '../models/scene.dart';
import '../services/api_client.dart';

/// Fetches scenes and triggers activations via the Core API.
class SceneRepository {
  final ApiClient _apiClient;

  SceneRepository({required ApiClient apiClient}) : _apiClient = apiClient;

  /// Get all scenes (admin: no room filter).
  Future<List<Scene>> getAllScenes() async {
    final response = await _apiClient.getScenes();
    return response.scenes;
  }

  /// Get all scenes for a room.
  Future<List<Scene>> getScenesByRoom(String roomId) async {
    final response = await _apiClient.getScenes(roomId: roomId);
    return response.scenes;
  }

  /// Get a single scene by ID.
  Future<Scene> getScene(String sceneId) async {
    return _apiClient.getScene(sceneId);
  }

  /// Create a new scene.
  Future<Scene> createScene(Map<String, dynamic> data) async {
    return _apiClient.createScene(data);
  }

  /// Update a scene (PATCH merge semantics).
  Future<Scene> updateScene(String sceneId, Map<String, dynamic> data) async {
    return _apiClient.updateScene(sceneId, data);
  }

  /// Delete a scene.
  Future<void> deleteScene(String sceneId) async {
    await _apiClient.deleteScene(sceneId);
  }

  /// Activate a scene by ID.
  Future<ActivateResponse> activate(String sceneId) async {
    return _apiClient.activateScene(sceneId);
  }

  /// Get execution history for a scene.
  Future<SceneExecutionResponse> getExecutions(String sceneId) async {
    return _apiClient.getSceneExecutions(sceneId);
  }
}
