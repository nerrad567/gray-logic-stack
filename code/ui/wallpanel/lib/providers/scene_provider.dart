import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_riverpod/legacy.dart';

import '../models/scene.dart';
import '../models/ws_message.dart';
import '../repositories/scene_repository.dart';
import 'auth_provider.dart';
import 'connection_provider.dart';

/// Provides the SceneRepository.
final sceneRepositoryProvider = Provider<SceneRepository>((ref) {
  return SceneRepository(apiClient: ref.watch(apiClientProvider));
});

/// Provides the list of scenes for the configured room.
final roomScenesProvider =
    NotifierProvider<RoomScenesNotifier, AsyncValue<List<Scene>>>(
  RoomScenesNotifier.new,
);

/// Persistent: which scene is active per room (from API + WebSocket).
/// Maps roomID -> sceneID. Updated on scene load and cross-device via WebSocket.
final activeScenePerRoomProvider = StateProvider<Map<String, String>>((ref) => {});

/// Transient: which scene is currently being activated (spinner, ~800ms).
final activatingSceneIdProvider = StateProvider<String?>((ref) => null);

/// Provides all scenes across all rooms (admin view).
final allScenesProvider =
    NotifierProvider<AllScenesNotifier, AsyncValue<List<Scene>>>(
  AllScenesNotifier.new,
);

class RoomScenesNotifier extends Notifier<AsyncValue<List<Scene>>> {
  StreamSubscription<WSInMessage>? _wsSubscription;

  @override
  AsyncValue<List<Scene>> build() {
    _listenToWebSocket();
    ref.onDispose(() => _wsSubscription?.cancel());
    return const AsyncValue.loading();
  }

  SceneRepository get _sceneRepo => ref.read(sceneRepositoryProvider);

  /// Load scenes for a room from the API.
  /// Parses active_scenes from the response and updates the persistent provider.
  Future<void> loadScenes(String roomId) async {
    state = const AsyncValue.loading();
    try {
      final response = await _sceneRepo.getScenesResponse(roomId: roomId);
      final scenes = response.scenes;
      // Sort by sort_order for consistent UI display
      scenes.sort((a, b) => a.sortOrder.compareTo(b.sortOrder));
      state = AsyncValue.data(scenes);

      // Update active scenes from API response
      if (response.activeScenes.isNotEmpty) {
        final current = Map<String, String>.from(
            ref.read(activeScenePerRoomProvider));
        current.addAll(response.activeScenes);
        ref.read(activeScenePerRoomProvider.notifier).state = current;
      }
    } catch (e, st) {
      state = AsyncValue.error(e, st);
    }
  }

  /// Activate a scene with visual feedback.
  Future<void> activateScene(String sceneId, {String? roomId}) async {
    // Set transient activating indicator (spinner)
    ref.read(activatingSceneIdProvider.notifier).state = sceneId;

    try {
      await _sceneRepo.activate(sceneId);

      // Optimistically update persistent active scene for the room
      if (roomId != null) {
        final current = Map<String, String>.from(
            ref.read(activeScenePerRoomProvider));
        current[roomId] = sceneId;
        ref.read(activeScenePerRoomProvider.notifier).state = current;
      }
    } finally {
      // Clear activating indicator after brief delay (animation time)
      Future.delayed(const Duration(milliseconds: 800), () {
        if (ref.mounted) {
          ref.read(activatingSceneIdProvider.notifier).state = null;
        }
      });
    }
  }

  /// Listen to WebSocket for cross-device scene activation events.
  void _listenToWebSocket() {
    final wsService = ref.read(webSocketServiceProvider);
    _wsSubscription = wsService.events.listen((msg) {
      if (msg.type == WSMessageType.event &&
          msg.eventType == WSChannel.sceneActivated) {
        _handleSceneActivated(msg.payload);
      }
    });
  }

  /// Update active scene tracking from a WebSocket event.
  void _handleSceneActivated(dynamic payload) {
    if (payload is! Map<String, dynamic>) return;
    final roomId = payload['room_id'] as String?;
    final sceneId = payload['scene_id'] as String?;
    if (roomId == null || sceneId == null) return;

    final current = Map<String, String>.from(
        ref.read(activeScenePerRoomProvider));
    current[roomId] = sceneId;
    ref.read(activeScenePerRoomProvider.notifier).state = current;
  }
}

class AllScenesNotifier extends Notifier<AsyncValue<List<Scene>>> {
  @override
  AsyncValue<List<Scene>> build() => const AsyncValue.loading();

  SceneRepository get _sceneRepo => ref.read(sceneRepositoryProvider);

  /// Load all scenes (admin).
  Future<void> load() async {
    state = const AsyncValue.loading();
    try {
      final scenes = await _sceneRepo.getAllScenes();
      scenes.sort((a, b) => a.sortOrder.compareTo(b.sortOrder));
      state = AsyncValue.data(scenes);
    } catch (e, st) {
      state = AsyncValue.error(e, st);
    }
  }

  /// Activate a scene with visual feedback (used by panel mode).
  Future<void> activateScene(String sceneId) async {
    ref.read(activatingSceneIdProvider.notifier).state = sceneId;
    try {
      await _sceneRepo.activate(sceneId);
    } finally {
      Future.delayed(const Duration(milliseconds: 800), () {
        if (ref.mounted) {
          ref.read(activatingSceneIdProvider.notifier).state = null;
        }
      });
    }
  }

  /// Create a new scene and refresh list.
  Future<Scene> createScene(Map<String, dynamic> data) async {
    final scene = await _sceneRepo.createScene(data);
    await load();
    return scene;
  }

  /// Update a scene and refresh list.
  Future<Scene> updateScene(String id, Map<String, dynamic> data) async {
    final scene = await _sceneRepo.updateScene(id, data);
    await load();
    return scene;
  }

  /// Delete a scene and refresh list.
  Future<void> deleteScene(String id) async {
    await _sceneRepo.deleteScene(id);
    await load();
  }
}
