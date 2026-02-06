import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/scene.dart';
import '../providers/auth_provider.dart';
import '../providers/location_provider.dart';
import '../providers/scene_provider.dart';
import 'scene_button.dart';
import 'scene_editor_sheet.dart';

/// Horizontal scrollable row of scene activation buttons.
/// Shown at the bottom of the room view.
class SceneBar extends ConsumerWidget {
  const SceneBar({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final scenesAsync = ref.watch(roomScenesProvider);

    return scenesAsync.when(
      data: (scenes) => _buildBar(context, ref, scenes),
      loading: () => const SizedBox(height: 56),
      error: (e, s) => const SizedBox(height: 56),
    );
  }

  Widget _buildBar(BuildContext context, WidgetRef ref, List<Scene> scenes) {
    // Only show enabled scenes
    final enabled = scenes.where((s) => s.enabled).toList();
    final identity = ref.watch(identityProvider);
    final canEdit = identity?.isPanel != true;

    return SizedBox(
      height: 56,
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 12),
        itemCount: enabled.length + (canEdit ? 1 : 0),
        separatorBuilder: (_, i) => const SizedBox(width: 8),
        itemBuilder: (_, index) {
          if (index < enabled.length) {
            return Center(
              child: SceneButton(
                scene: enabled[index],
                onLongPress: canEdit ? () => _openEditor(context, ref, enabled[index]) : null,
              ),
            );
          }
          // Add button at end (only for non-panel identities)
          return Center(child: _AddSceneButton(
            onTap: () => _openEditor(context, ref, null),
          ));
        },
      ),
    );
  }

  void _openEditor(BuildContext context, WidgetRef ref, Scene? scene) async {
    final roomId = ref.read(selectedRoomProvider);
    final result = await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      builder: (context) => SceneEditorSheet(
        scene: scene,
        preselectedRoomId: scene == null ? roomId : null,
      ),
    );
    if (result == true && roomId != null) {
      ref.read(roomScenesProvider.notifier).loadScenes(roomId);
    }
  }
}

class _AddSceneButton extends StatelessWidget {
  final VoidCallback onTap;

  const _AddSceneButton({required this.onTap});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Tooltip(
      message: 'Create scene',
      child: Material(
        color: theme.colorScheme.surfaceContainerHighest.withValues(alpha: 0.5),
        borderRadius: BorderRadius.circular(12),
        child: InkWell(
          borderRadius: BorderRadius.circular(12),
          onTap: onTap,
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(12),
              border: Border.all(
                color: theme.colorScheme.outlineVariant,
                style: BorderStyle.solid,
              ),
            ),
            child: Icon(
              Icons.add,
              size: 18,
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
        ),
      ),
    );
  }
}
