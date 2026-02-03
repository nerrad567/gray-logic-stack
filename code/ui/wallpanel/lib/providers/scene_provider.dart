import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_riverpod/legacy.dart';

import '../models/scene.dart';
import '../repositories/scene_repository.dart';
import 'auth_provider.dart';

/// Provides the SceneRepository.
final sceneRepositoryProvider = Provider<SceneRepository>((ref) {
  return SceneRepository(apiClient: ref.watch(apiClientProvider));
});

/// Provides the list of scenes for the configured room.
final roomScenesProvider =
    NotifierProvider<RoomScenesNotifier, AsyncValue<List<Scene>>>(
  RoomScenesNotifier.new,
);

/// Tracks which scene is currently being activated (for UI feedback).
final activeSceneIdProvider = StateProvider<String?>((ref) => null);

class RoomScenesNotifier extends Notifier<AsyncValue<List<Scene>>> {
  @override
  AsyncValue<List<Scene>> build() => const AsyncValue.loading();

  SceneRepository get _sceneRepo => ref.read(sceneRepositoryProvider);

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
    ref.read(activeSceneIdProvider.notifier).state = sceneId;

    try {
      await _sceneRepo.activate(sceneId);
    } finally {
      // Clear active indicator after a brief delay (animation time)
      Future.delayed(const Duration(milliseconds: 800), () {
        if (ref.mounted) {
          ref.read(activeSceneIdProvider.notifier).state = null;
        }
      });
    }
  }
}
