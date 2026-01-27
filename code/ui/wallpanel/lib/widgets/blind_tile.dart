import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../config/constants.dart';
import '../models/device.dart';
import '../providers/device_provider.dart';

/// A tile widget for blind/shutter devices.
/// Shows position as a percentage with a vertical slider.
class BlindTile extends ConsumerStatefulWidget {
  final Device device;

  const BlindTile({super.key, required this.device});

  @override
  ConsumerState<BlindTile> createState() => _BlindTileState();
}

class _BlindTileState extends ConsumerState<BlindTile> {
  double? _draggingValue;
  Timer? _debounceTimer;

  @override
  void dispose() {
    _debounceTimer?.cancel();
    super.dispose();
  }

  int get _position {
    final pos = widget.device.state['position'];
    if (pos is num) return pos.toInt();
    return 0;
  }

  void _onSliderChanged(double value) {
    setState(() => _draggingValue = value);

    _debounceTimer?.cancel();
    _debounceTimer = Timer(
      const Duration(milliseconds: AppConstants.sliderDebounceMs),
      () {
        final level = _draggingValue?.round() ?? value.round();
        setState(() => _draggingValue = null);
        ref.read(roomDevicesProvider.notifier).setPosition(
              widget.device.id,
              level,
            );
      },
    );
  }

  @override
  Widget build(BuildContext context) {
    final device = widget.device;
    final isPending = ref.watch(pendingDevicesProvider).contains(device.id);
    final theme = Theme.of(context);
    final activeColour = theme.colorScheme.primary;

    final displayPosition = _draggingValue?.round() ?? _position;
    final isOpen = displayPosition > 0;

    return Card(
      child: InkWell(
        borderRadius: BorderRadius.circular(16),
        splashFactory: NoSplash.splashFactory,
        highlightColor: Colors.transparent,
        hoverColor: theme.colorScheme.primary.withValues(alpha: 0.05),
        mouseCursor: SystemMouseCursors.basic,
        onTap: null, // Blinds don't have a tap action, just the slider
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              // Device name
              Text(
                device.name,
                style: theme.textTheme.titleMedium,
                textAlign: TextAlign.center,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
              ),
              const SizedBox(height: 8),
              // Position indicator
              Expanded(
                child: Center(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(
                        isOpen ? Icons.blinds : Icons.blinds_closed,
                        size: 32,
                        color: isOpen ? activeColour : Colors.grey.shade500,
                      ),
                      const SizedBox(height: 4),
                      Text(
                        '$displayPosition%',
                        style: TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.w600,
                          color: isOpen ? activeColour : Colors.grey.shade500,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
              // Slider for position
              MouseRegion(
                cursor: !isPending
                    ? SystemMouseCursors.grab
                    : SystemMouseCursors.basic,
                child: SizedBox(
                  height: 32,
                  child: SliderTheme(
                    data: SliderThemeData(
                      trackHeight: 4,
                      thumbShape:
                          const RoundSliderThumbShape(enabledThumbRadius: 8),
                      activeTrackColor: isPending
                          ? activeColour.withValues(alpha: 0.4)
                          : activeColour,
                      inactiveTrackColor: Colors.grey.shade800,
                      thumbColor: !isPending ? activeColour : Colors.grey,
                      overlayShape:
                          const RoundSliderOverlayShape(overlayRadius: 16),
                    ),
                    child: Slider(
                      value: (_draggingValue ?? _position.toDouble()).clamp(0, 100),
                      min: 0,
                      max: 100,
                      onChanged: !isPending ? _onSliderChanged : null,
                    ),
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
