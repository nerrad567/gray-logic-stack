import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/scene.dart';
import '../providers/scene_provider.dart';

/// A button for activating a scene. Shows a brief pulse animation on activation.
class SceneButton extends ConsumerWidget {
  final Scene scene;

  const SceneButton({super.key, required this.scene});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final activeSceneId = ref.watch(activeSceneIdProvider);
    final isActivating = activeSceneId == scene.id;
    final theme = Theme.of(context);
    final accentColour = _parseColour(scene.colour) ?? theme.colorScheme.secondary;

    return AnimatedContainer(
      duration: const Duration(milliseconds: 200),
      child: Material(
        color: isActivating
            ? accentColour.withValues(alpha: 0.3)
            : accentColour.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(12),
        child: InkWell(
          borderRadius: BorderRadius.circular(12),
          onTap: isActivating
              ? null
              : () => ref.read(roomScenesProvider.notifier).activateScene(scene.id),
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(12),
              border: Border.all(
                color: isActivating
                    ? accentColour
                    : accentColour.withValues(alpha: 0.3),
                width: isActivating ? 2 : 1,
              ),
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                if (scene.icon != null) ...[
                  Icon(
                    _getIconData(scene.icon!),
                    size: 18,
                    color: accentColour,
                  ),
                  const SizedBox(width: 8),
                ],
                Text(
                  scene.name,
                  style: TextStyle(
                    fontSize: 14,
                    fontWeight: FontWeight.w500,
                    color: accentColour,
                  ),
                ),
                if (isActivating) ...[
                  const SizedBox(width: 8),
                  SizedBox(
                    width: 12,
                    height: 12,
                    child: CircularProgressIndicator(
                      strokeWidth: 2,
                      valueColor: AlwaysStoppedAnimation<Color>(accentColour),
                    ),
                  ),
                ],
              ],
            ),
          ),
        ),
      ),
    );
  }

  Color? _parseColour(String? hex) {
    if (hex == null || hex.isEmpty) return null;
    final cleaned = hex.replaceFirst('#', '');
    if (cleaned.length != 6) return null;
    final value = int.tryParse(cleaned, radix: 16);
    if (value == null) return null;
    return Color(0xFF000000 | value);
  }

  IconData _getIconData(String iconName) {
    // Map common scene icon names to Material icons
    const iconMap = {
      'movie': Icons.movie,
      'movie_night': Icons.movie,
      'cinema': Icons.movie,
      'reading': Icons.menu_book,
      'book': Icons.menu_book,
      'bright': Icons.wb_sunny,
      'sun': Icons.wb_sunny,
      'relax': Icons.spa,
      'night': Icons.nightlight_round,
      'off': Icons.power_settings_new,
      'all_off': Icons.power_settings_new,
      'morning': Icons.wb_twilight,
      'evening': Icons.nights_stay,
      'party': Icons.celebration,
      'dinner': Icons.restaurant,
      'welcome': Icons.waving_hand,
    };
    return iconMap[iconName.toLowerCase()] ?? Icons.play_circle_outline;
  }
}
