import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/scene.dart';
import '../repositories/scene_repository.dart';
import 'auth_provider.dart';

/// Provides the SceneRepository.
final sceneRepositoryProvider = Provider<SceneRepository>((ref) {
  return SceneRepository(apiClient: ref.watch(apiClientProvider));
});

/// Provides the list of scenes for the configured room.
final roomScenesProvider =
    StateNotifierProvider<RoomScenesNotifier, AsyncValue<List<Scene>>>((ref) {
  return RoomScenesNotifier(ref);
});

/// Tracks which scene is currently being activated (for UI feedback).
final activeSceneIdProvider = StateProvider<String?>((ref) => null);

class RoomScenesNotifier extends StateNotifier<AsyncValue<List<Scene>>> {
  final Ref _ref;

  RoomScenesNotifier(this._ref) : super(const AsyncValue.loading());

  SceneRepository get _sceneRepo => _ref.read(sceneRepositoryProvider);

  /// Load scenes for a room from the API.
  Future<void> loadScenes(String roomId) async {
    state = const AsyncValue.loading();
    try {
      final scenes = await _sceneRepo.getScenesByRoom(roomId);
      // Sort by sort_order for consistent UI display
      scenes.sort((a, b) => a.sortOrder.compareTo(b.sortOrder));
      state = AsyncValue.data(scenes);
    } catch (e, st) {
      state = AsyncValue.error(e, st);
    }
  }

  /// Activate a scene with visual feedback.
  Future<void> activateScene(String sceneId) async {
    // Set active indicator
    _ref.read(activeSceneIdProvider.notifier).state = sceneId;

    try {
      await _sceneRepo.activate(sceneId);
    } finally {
      // Clear active indicator after a brief delay (animation time)
      Future.delayed(const Duration(milliseconds: 800), () {
        if (mounted) {
          _ref.read(activeSceneIdProvider.notifier).state = null;
        }
      });
    }
  }
}
