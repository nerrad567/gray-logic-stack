import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/scene.dart';
import '../providers/scene_provider.dart';
import 'scene_button.dart';

/// Horizontal scrollable row of scene activation buttons.
/// Shown at the bottom of the room view.
class SceneBar extends ConsumerWidget {
  const SceneBar({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final scenesAsync = ref.watch(roomScenesProvider);

    return scenesAsync.when(
      data: (scenes) => _buildBar(scenes),
      loading: () => const SizedBox(height: 56),
      error: (e, s) => const SizedBox(height: 56),
    );
  }

  Widget _buildBar(List<Scene> scenes) {
    if (scenes.isEmpty) return const SizedBox.shrink();

    // Only show enabled scenes
    final enabled = scenes.where((s) => s.enabled).toList();
    if (enabled.isEmpty) return const SizedBox.shrink();

    return SizedBox(
      height: 56,
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 12),
        itemCount: enabled.length,
        separatorBuilder: (_, i) => const SizedBox(width: 8),
        itemBuilder: (_, index) => Center(
          child: SceneButton(scene: enabled[index]),
        ),
      ),
    );
  }
}
