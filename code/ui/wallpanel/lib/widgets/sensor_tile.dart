import 'package:flutter/material.dart';

import '../models/device.dart';

/// A read-only tile for sensor devices (temperature, presence, humidity).
/// Shows the sensor reading without any interactive controls.
class SensorTile extends StatelessWidget {
  final Device device;

  const SensorTile({super.key, required this.device});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final reading = _getReading();
    final icon = _getIcon();
    final hasValue = reading.value != null;
    final isPresenceSensor = _isPresenceSensor();
    final isOccupied = isPresenceSensor && reading.value == 1.0;
    final isInactive = isPresenceSensor && !isOccupied;

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
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
            const Spacer(),
            // Sensor icon — greys out when presence is empty
            Icon(
              icon,
              size: 32,
              color: isOccupied
                  ? Colors.green
                  : isInactive
                      ? Colors.grey.shade700
                      : hasValue
                          ? theme.colorScheme.primary
                          : Colors.grey.shade600,
            ),
            const SizedBox(height: 8),
            // Reading value
            Text(
              hasValue ? reading.display : '--',
              style: TextStyle(
                fontSize: hasValue ? 24 : 18,
                fontWeight: FontWeight.bold,
                color: hasValue
                    ? theme.colorScheme.onSurface
                    : Colors.grey.shade500,
              ),
            ),
            const Spacer(),
            // Label
            Text(
              reading.label,
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

  bool _isPresenceSensor() {
    return device.type == 'presence_sensor' ||
        device.state.containsKey('presence');
  }

  _SensorReading _getReading() {
    final state = device.state;

    // Temperature sensor
    if (state.containsKey('temperature')) {
      final temp = state['temperature'];
      if (temp is num) {
        return _SensorReading(
          value: temp.toDouble(),
          display: '${temp.toStringAsFixed(1)}°C',
          label: 'Temperature',
        );
      }
    }

    // Presence sensor
    if (state.containsKey('presence')) {
      final presence = state['presence'];
      return _SensorReading(
        value: presence == true ? 1.0 : 0.0,
        display: presence == true ? 'Occupied' : 'Empty',
        label: 'Presence',
      );
    }

    // Humidity sensor
    if (state.containsKey('humidity')) {
      final humidity = state['humidity'];
      if (humidity is num) {
        return _SensorReading(
          value: humidity.toDouble(),
          display: '${humidity.toStringAsFixed(0)}%',
          label: 'Humidity',
        );
      }
    }

    // Fallback based on device type
    if (device.type == 'temperature_sensor') {
      return _SensorReading(value: null, display: '--', label: 'Temperature');
    }
    if (device.type == 'presence_sensor') {
      return _SensorReading(value: null, display: '--', label: 'Presence');
    }

    return _SensorReading(value: null, display: '--', label: 'Sensor');
  }

  IconData _getIcon() {
    if (device.type == 'temperature_sensor' ||
        device.state.containsKey('temperature')) {
      return Icons.thermostat;
    }
    if (device.type == 'presence_sensor' ||
        device.state.containsKey('presence')) {
      return Icons.sensors;
    }
    if (device.type == 'humidity_sensor' ||
        device.state.containsKey('humidity')) {
      return Icons.water_drop;
    }
    return Icons.speed;
  }
}

class _SensorReading {
  final double? value;
  final String display;
  final String label;

  const _SensorReading({
    required this.value,
    required this.display,
    required this.label,
  });
}
