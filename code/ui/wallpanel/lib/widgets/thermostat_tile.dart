import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/device.dart';
import '../providers/device_provider.dart';

/// A tile widget for thermostat devices.
/// Shows current temperature, setpoint, and allows adjusting the setpoint.
class ThermostatTile extends ConsumerWidget {
  final Device device;

  const ThermostatTile({super.key, required this.device});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final isOnline = device.isOnline;
    final isPending = ref.watch(pendingDevicesProvider).contains(device.id);

    // Get temperature values from state
    final currentTemp = _getTemperature(device.state, 'temperature');
    final setpoint = _getTemperature(device.state, 'setpoint');
    final isHeating = device.state['heating'] == true ||
        device.state['mode'] == 'heating' ||
        (currentTemp != null && setpoint != null && currentTemp < setpoint);

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

            // Current temperature (large)
            Row(
              mainAxisAlignment: MainAxisAlignment.center,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Icon(
                  Icons.thermostat,
                  size: 24,
                  color: isHeating
                      ? Colors.orange
                      : theme.colorScheme.primary,
                ),
                const SizedBox(width: 4),
                Text(
                  currentTemp != null
                      ? currentTemp.toStringAsFixed(1)
                      : '--',
                  style: TextStyle(
                    fontSize: 32,
                    fontWeight: FontWeight.bold,
                    color: theme.colorScheme.onSurface,
                  ),
                ),
                Text(
                  '°C',
                  style: TextStyle(
                    fontSize: 16,
                    color: Colors.grey.shade400,
                  ),
                ),
              ],
            ),

            // Valve output indicator (heating_output from thermostat PID)
            if (_getHeatingOutput(device.state) != null) ...[
              const SizedBox(height: 4),
              _ValveIndicator(percent: _getHeatingOutput(device.state)!),
            ],

            const Spacer(),

            // Setpoint controls
            if (setpoint != null || device.capabilities.contains('temperature_set'))
              _SetpointControls(
                device: device,
                setpoint: setpoint,
                isOnline: isOnline,
                isPending: isPending,
                theme: theme,
                ref: ref,
              )
            else
              // Just show "Current" label for read-only sensors
              Text(
                'Current',
                style: TextStyle(
                  fontSize: 12,
                  color: Colors.grey.shade400,
                ),
              ),
          ],
        ),
      ),
    );
  }

  double? _getTemperature(Map<String, dynamic> state, String key) {
    final value = state[key];
    if (value is num) return value.toDouble();
    return null;
  }

  double? _getHeatingOutput(Map<String, dynamic> state) {
    final value = state['heating_output'];
    if (value is num) return value.toDouble();
    return null;
  }
}

class _SetpointControls extends StatefulWidget {
  final Device device;
  final double? setpoint;
  final bool isOnline;
  final bool isPending;
  final ThemeData theme;
  final WidgetRef ref;

  const _SetpointControls({
    required this.device,
    required this.setpoint,
    required this.isOnline,
    required this.isPending,
    required this.theme,
    required this.ref,
  });

  @override
  State<_SetpointControls> createState() => _SetpointControlsState();
}

class _SetpointControlsState extends State<_SetpointControls> {
  /// Local optimistic setpoint — null means use the server value.
  double? _localSetpoint;

  /// Timer for hold-to-repeat acceleration.
  Timer? _repeatTimer;

  /// Safety timer — kills the repeat timer if onLongPressEnd never fires
  /// (e.g. browser context menu steals the touch event).
  Timer? _safetyTimer;

  /// Debounce timer — sends final value to backend after user stops adjusting.
  Timer? _debounceTimer;

  /// How many repeat ticks have fired in the current hold gesture.
  int _repeatCount = 0;

  /// The effective setpoint to display (local override or server value).
  double? get _displaySetpoint => _localSetpoint ?? widget.setpoint;

  @override
  void didUpdateWidget(_SetpointControls old) {
    super.didUpdateWidget(old);
    // When the server confirms a value and we're not mid-gesture, clear local.
    if (_repeatTimer == null && _debounceTimer == null) {
      _localSetpoint = null;
    }
  }

  @override
  void dispose() {
    _repeatTimer?.cancel();
    _safetyTimer?.cancel();
    _debounceTimer?.cancel();
    super.dispose();
  }

  /// Kill any running repeat timer (idempotent).
  void _cancelRepeat() {
    _repeatTimer?.cancel();
    _repeatTimer = null;
    _safetyTimer?.cancel();
    _safetyTimer = null;
    _repeatCount = 0;
  }

  /// Step once and start the hold-to-repeat timer.
  void _onHoldStart(double direction) {
    // Always kill previous timer first — prevents orphan timers when
    // rapidly alternating between + and -.
    _cancelRepeat();
    _step(direction);
    _repeatTimer = Timer.periodic(const Duration(milliseconds: 300), (_) {
      if (!mounted) {
        _cancelRepeat();
        return;
      }
      _repeatCount++;
      // Accelerate: after 4 ticks double-step, after 8 triple-step.
      final steps = _repeatCount < 4 ? 1 : (_repeatCount < 8 ? 2 : 3);
      for (var i = 0; i < steps; i++) {
        _step(direction);
      }
    });
    // Safety kill: if onLongPressEnd never fires (e.g. focus loss, browser
    // event theft), auto-stop after 8 seconds and send what we have.
    _safetyTimer = Timer(const Duration(seconds: 8), () {
      _onHoldEnd();
    });
  }

  /// Stop repeating and debounce-send to backend.
  void _onHoldEnd() {
    _cancelRepeat();
    _debounceSend();
  }

  /// Apply a single 0.5° step locally (optimistic).
  void _step(double direction) {
    if (!mounted) return;
    setState(() {
      final current = _displaySetpoint ?? 20.0;
      _localSetpoint = (current + direction * 0.5).clamp(5.0, 35.0);
    });
    // Reset debounce on every step so we only send after user stops.
    _debounceTimer?.cancel();
    _debounceTimer = null;
  }

  /// Send the final value to the backend after a short pause.
  void _debounceSend() {
    _debounceTimer?.cancel();
    _debounceTimer = Timer(const Duration(milliseconds: 400), () {
      _debounceTimer = null;
      if (!mounted) return;
      final value = _localSetpoint;
      if (value != null) {
        widget.ref
            .read(roomDevicesProvider.notifier)
            .setSetpoint(widget.device.id, value);
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final canAdjust =
        widget.isOnline && !widget.isPending && _displaySetpoint != null;
    final isLocal = _localSetpoint != null;

    return Column(
      children: [
        Text(
          'Setpoint',
          style: TextStyle(
            fontSize: 11,
            color: Colors.grey.shade500,
          ),
        ),
        const SizedBox(height: 4),
        Row(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            // Decrease button
            Expanded(
              child: _AdjustButton(
                icon: Icons.remove,
                onPressed: canAdjust ? () => _step(-1) : null,
                onHoldStart: canAdjust ? () => _onHoldStart(-1) : null,
                onHoldEnd: canAdjust ? _onHoldEnd : null,
                theme: widget.theme,
              ),
            ),

            // Setpoint display — highlight while local (unsent)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 4),
              child: Text(
                _displaySetpoint != null
                    ? '${_displaySetpoint!.toStringAsFixed(1)}°'
                    : '--',
                style: TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.w600,
                  color: isLocal
                      ? widget.theme.colorScheme.primary
                      : widget.isPending
                          ? widget.theme.colorScheme.primary
                          : widget.theme.colorScheme.onSurface,
                ),
              ),
            ),

            // Increase button
            Expanded(
              child: _AdjustButton(
                icon: Icons.add,
                onPressed: canAdjust ? () => _step(1) : null,
                onHoldStart: canAdjust ? () => _onHoldStart(1) : null,
                onHoldEnd: canAdjust ? _onHoldEnd : null,
                theme: widget.theme,
              ),
            ),
          ],
        ),
      ],
    );
  }
}

class _AdjustButton extends StatefulWidget {
  final IconData icon;
  final VoidCallback? onPressed;
  final VoidCallback? onHoldStart;
  final VoidCallback? onHoldEnd;
  final ThemeData theme;

  const _AdjustButton({
    required this.icon,
    required this.onPressed,
    this.onHoldStart,
    this.onHoldEnd,
    required this.theme,
  });

  @override
  State<_AdjustButton> createState() => _AdjustButtonState();
}

class _AdjustButtonState extends State<_AdjustButton> {
  bool _pressed = false;

  void _setPressed(bool v) {
    if (v != _pressed) setState(() => _pressed = v);
  }

  @override
  Widget build(BuildContext context) {
    final enabled = widget.onPressed != null;
    final accent = widget.theme.colorScheme.primary;

    return GestureDetector(
      behavior: HitTestBehavior.opaque,
      onTapDown: (_) => _setPressed(true),
      onTapUp: (_) => _setPressed(false),
      onTapCancel: () => _setPressed(false),
      onTap: () {
        widget.onHoldEnd?.call();
        widget.onPressed?.call();
        widget.onHoldEnd?.call();
      },
      onLongPressStart: (_) {
        _setPressed(true);
        widget.onHoldStart?.call();
      },
      onLongPressEnd: (_) {
        _setPressed(false);
        widget.onHoldEnd?.call();
      },
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 80),
        height: 48,
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(10),
          color: _pressed && enabled
              ? accent.withValues(alpha: 0.25)
              : enabled
                  ? accent.withValues(alpha: 0.08)
                  : Colors.grey.shade900,
          border: Border.all(
            color: enabled
                ? accent.withValues(alpha: _pressed ? 0.8 : 0.35)
                : Colors.grey.shade700,
            width: _pressed ? 1.5 : 1.0,
          ),
        ),
        child: Icon(
          widget.icon,
          size: 22,
          color: enabled ? accent : Colors.grey.shade600,
        ),
      ),
    );
  }
}

/// Compact valve output indicator showing a flame icon with warm colour
/// gradient based on the heating output percentage (0-100%).
class _ValveIndicator extends StatelessWidget {
  final double percent;

  const _ValveIndicator({required this.percent});

  @override
  Widget build(BuildContext context) {
    final t = (percent / 100).clamp(0.0, 1.0);
    final color = _heatColor(t);
    final isActive = percent > 0;

    return Row(
      mainAxisAlignment: MainAxisAlignment.center,
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(
          Icons.local_fire_department,
          size: 14,
          color: isActive ? color : Colors.grey.shade700,
        ),
        const SizedBox(width: 3),
        Text(
          '${percent.toInt()}%',
          style: TextStyle(
            fontSize: 11,
            fontWeight: FontWeight.w600,
            color: isActive ? color : Colors.grey.shade600,
          ),
        ),
      ],
    );
  }

  /// Warm colour gradient: cold steel → amber → deep orange → red.
  static Color _heatColor(double t) {
    const stops = [
      Color(0xFF5C7A99), //  0% — cool steel blue
      Color(0xFFFFA726), // 33% — amber
      Color(0xFFFF5722), // 66% — deep orange
      Color(0xFFD32F2F), // 100% — red
    ];
    final scaled = t * (stops.length - 1);
    final i = scaled.floor().clamp(0, stops.length - 2);
    final frac = scaled - i;
    return Color.lerp(stops[i], stops[i + 1], frac)!;
  }
}
