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

class _SetpointControls extends StatelessWidget {
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
  Widget build(BuildContext context) {
    final canAdjust = isOnline && !isPending && setpoint != null;

    return Column(
      children: [
        // Setpoint label
        Text(
          'Setpoint',
          style: TextStyle(
            fontSize: 11,
            color: Colors.grey.shade500,
          ),
        ),
        const SizedBox(height: 4),

        // Setpoint value with +/- buttons
        Row(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            // Decrease button
            _AdjustButton(
              icon: Icons.remove,
              onPressed: canAdjust
                  ? () => _adjustSetpoint(-0.5)
                  : null,
              theme: theme,
            ),

            // Setpoint display
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 8),
              child: Text(
                setpoint != null
                    ? '${setpoint!.toStringAsFixed(1)}°'
                    : '--',
                style: TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.w600,
                  color: isPending
                      ? theme.colorScheme.primary
                      : theme.colorScheme.onSurface,
                ),
              ),
            ),

            // Increase button
            _AdjustButton(
              icon: Icons.add,
              onPressed: canAdjust
                  ? () => _adjustSetpoint(0.5)
                  : null,
              theme: theme,
            ),
          ],
        ),
      ],
    );
  }

  void _adjustSetpoint(double delta) {
    final newSetpoint = (setpoint ?? 20.0) + delta;
    // Clamp to reasonable range (5-35°C)
    final clamped = newSetpoint.clamp(5.0, 35.0);

    ref.read(roomDevicesProvider.notifier).setSetpoint(device.id, clamped);
  }
}

class _AdjustButton extends StatelessWidget {
  final IconData icon;
  final VoidCallback? onPressed;
  final ThemeData theme;

  const _AdjustButton({
    required this.icon,
    required this.onPressed,
    required this.theme,
  });

  @override
  Widget build(BuildContext context) {
    final enabled = onPressed != null;

    return InkWell(
      onTap: onPressed,
      borderRadius: BorderRadius.circular(8),
      child: Container(
        width: 32,
        height: 32,
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(8),
          border: Border.all(
            color: enabled
                ? theme.colorScheme.primary.withValues(alpha: 0.5)
                : Colors.grey.shade700,
          ),
        ),
        child: Icon(
          icon,
          size: 18,
          color: enabled
              ? theme.colorScheme.primary
              : Colors.grey.shade600,
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
