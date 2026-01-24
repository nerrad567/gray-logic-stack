import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../config/constants.dart';
import '../models/device.dart';
import '../providers/device_provider.dart';

/// A tile widget for dimmable devices (light dimmers).
/// Shows device name, brightness indicator (tap to toggle), and a slider.
/// During pending state, the tile pulses subtly and the slider is disabled.
class DimmerTile extends ConsumerStatefulWidget {
  final Device device;

  const DimmerTile({super.key, required this.device});

  @override
  ConsumerState<DimmerTile> createState() => _DimmerTileState();
}

class _DimmerTileState extends ConsumerState<DimmerTile> {
  double? _draggingValue;
  double? _sentValue;
  Timer? _debounceTimer;

  @override
  void dispose() {
    _debounceTimer?.cancel();
    super.dispose();
  }

  void _onSliderChanged(double value) {
    setState(() => _draggingValue = value);

    _debounceTimer?.cancel();
    _debounceTimer = Timer(
      const Duration(milliseconds: AppConstants.sliderDebounceMs),
      () {
        final level = _draggingValue?.round() ?? value.round();
        // Keep the sent value so the slider doesn't snap back while pending
        setState(() {
          _sentValue = _draggingValue;
          _draggingValue = null;
        });
        ref.read(roomDevicesProvider.notifier).setLevel(
              widget.device.id,
              level,
            );
      },
    );
  }

  void _onTapToggle() {
    ref.read(roomDevicesProvider.notifier).toggleDevice(widget.device.id);
  }

  @override
  Widget build(BuildContext context) {
    final device = widget.device;
    final isPending = ref.watch(pendingDevicesProvider).contains(device.id);
    final isOn = device.isOn;
    final isOnline = device.isOnline;
    final theme = Theme.of(context);
    final activeColour = theme.colorScheme.primary;

    // Clear _sentValue only when device state has caught up to what we sent
    if (_sentValue != null && device.level == _sentValue!.round()) {
      _sentValue = null;
    }

    // Display value priority: actively dragging > sent but pending > device state
    final displayLevel = _draggingValue?.round()
        ?? _sentValue?.round()
        ?? device.level;

    return Card(
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
            // Brightness indicator + tap to toggle
            Expanded(
              child: InkWell(
                borderRadius: BorderRadius.circular(32),
                splashFactory: NoSplash.splashFactory,
                highlightColor: Colors.transparent,
                hoverColor: activeColour.withValues(alpha: 0.05),
                onTap: isOnline && !isPending ? _onTapToggle : null,
                mouseCursor: isOnline && !isPending
                    ? SystemMouseCursors.click
                    : SystemMouseCursors.basic,
                child: Center(
                  child: isPending
                      ? _PendingBrightnessIndicator(
                          level: displayLevel,
                          isOn: isOn,
                          activeColour: activeColour,
                        )
                      : _BrightnessIndicator(
                          level: displayLevel,
                          isOn: isOn,
                          activeColour: activeColour,
                        ),
                ),
              ),
            ),
            const SizedBox(height: 4),
            // Slider — disabled during pending state
            MouseRegion(
              cursor: isOnline && !isPending
                  ? SystemMouseCursors.grab
                  : SystemMouseCursors.basic,
              child: SizedBox(
                height: 32,
                child: SliderTheme(
                  data: SliderThemeData(
                    trackHeight: 4,
                    thumbShape: const RoundSliderThumbShape(enabledThumbRadius: 8),
                    activeTrackColor: isPending
                        ? activeColour.withValues(alpha: 0.4)
                        : activeColour,
                    inactiveTrackColor: Colors.grey.shade800,
                    thumbColor: isOnline && !isPending ? activeColour : Colors.grey,
                    overlayShape: const RoundSliderOverlayShape(overlayRadius: 16),
                  ),
                  child: Slider(
                    value: (_draggingValue ?? _sentValue ?? device.level.toDouble()).clamp(0, 100),
                    min: 0,
                    max: 100,
                    onChanged: isOnline && !isPending ? _onSliderChanged : null,
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// Normal brightness circle showing level percentage.
class _BrightnessIndicator extends StatelessWidget {
  final int level;
  final bool isOn;
  final Color activeColour;

  const _BrightnessIndicator({
    required this.level,
    required this.isOn,
    required this.activeColour,
  });

  @override
  Widget build(BuildContext context) {
    return Stack(
      alignment: Alignment.center,
      children: [
        SizedBox(
          width: 56,
          height: 56,
          child: CircularProgressIndicator(
            value: level / 100.0,
            strokeWidth: 4,
            backgroundColor: Colors.grey.shade800,
            valueColor: AlwaysStoppedAnimation<Color>(
              isOn ? activeColour : Colors.grey.shade600,
            ),
          ),
        ),
        Text(
          '$level%',
          style: TextStyle(
            fontSize: 14,
            fontWeight: FontWeight.w600,
            color: isOn ? activeColour : Colors.grey.shade500,
          ),
        ),
      ],
    );
  }
}

/// The same brightness circle but with a gentle opacity pulse to indicate pending.
class _PendingBrightnessIndicator extends StatefulWidget {
  final int level;
  final bool isOn;
  final Color activeColour;

  const _PendingBrightnessIndicator({
    required this.level,
    required this.isOn,
    required this.activeColour,
  });

  @override
  State<_PendingBrightnessIndicator> createState() =>
      _PendingBrightnessIndicatorState();
}

class _PendingBrightnessIndicatorState extends State<_PendingBrightnessIndicator>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;
  late final Animation<double> _opacity;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 800),
    )..repeat(reverse: true);
    _opacity = Tween<double>(begin: 0.2, end: 1.0).animate(
      CurvedAnimation(parent: _controller, curve: Curves.easeInOut),
    );
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return FadeTransition(
      opacity: _opacity,
      child: Stack(
        alignment: Alignment.center,
        children: [
          SizedBox(
            width: 56,
            height: 56,
            child: CircularProgressIndicator(
              value: widget.level / 100.0,
              strokeWidth: 4,
              backgroundColor: Colors.grey.shade800,
              valueColor: AlwaysStoppedAnimation<Color>(
                widget.activeColour.withValues(alpha: 0.7),
              ),
            ),
          ),
          Text(
            '${widget.level}%',
            style: TextStyle(
              fontSize: 14,
              fontWeight: FontWeight.w600,
              color: widget.activeColour.withValues(alpha: 0.7),
            ),
          ),
        ],
      ),
    );
  }
}
