import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/scene.dart';
import '../providers/scene_provider.dart';

/// A button for activating a scene with three visual states:
/// - **Active** (persistent): Filled background, bold border, check icon
/// - **Activating** (transient ~800ms): Spinner overlay
/// - **Inactive** (default): Muted style
///
/// Long-press opens the scene editor.
class SceneButton extends ConsumerWidget {
  final Scene scene;
  final String? roomId;
  final VoidCallback? onLongPress;

  const SceneButton({
    super.key,
    required this.scene,
    this.roomId,
    this.onLongPress,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final activeScenes = ref.watch(activeScenePerRoomProvider);
    final isActive = roomId != null && activeScenes[roomId] == scene.id;
    final isActivating = ref.watch(activatingSceneIdProvider) == scene.id;

    final theme = Theme.of(context);
    final accentColour = _parseColour(scene.colour) ?? theme.colorScheme.secondary;

    // Visual state hierarchy: activating > active > inactive
    final Color bgColor;
    final Color borderColor;
    final double borderWidth;
    final FontWeight fontWeight;

    if (isActivating) {
      bgColor = accentColour.withValues(alpha: 0.3);
      borderColor = accentColour;
      borderWidth = 2;
      fontWeight = FontWeight.w600;
    } else if (isActive) {
      bgColor = accentColour.withValues(alpha: 0.25);
      borderColor = accentColour;
      borderWidth = 2;
      fontWeight = FontWeight.w600;
    } else {
      bgColor = accentColour.withValues(alpha: 0.1);
      borderColor = accentColour.withValues(alpha: 0.3);
      borderWidth = 1;
      fontWeight = FontWeight.w500;
    }

    return AnimatedContainer(
      duration: const Duration(milliseconds: 200),
      child: Material(
        color: bgColor,
        borderRadius: BorderRadius.circular(12),
        child: InkWell(
          borderRadius: BorderRadius.circular(12),
          onTap: isActivating
              ? null
              : () => ref.read(roomScenesProvider.notifier)
                  .activateScene(scene.id, roomId: roomId),
          onLongPress: onLongPress,
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: borderColor, width: borderWidth),
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
                    fontWeight: fontWeight,
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
                ] else if (isActive) ...[
                  const SizedBox(width: 8),
                  Icon(Icons.check_circle, size: 14, color: accentColour),
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
