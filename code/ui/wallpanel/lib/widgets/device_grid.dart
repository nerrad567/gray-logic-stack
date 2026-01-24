import 'package:flutter/material.dart';

import '../models/device.dart';
import 'blind_tile.dart';
import 'dimmer_tile.dart';
import 'sensor_tile.dart';
import 'switch_tile.dart';

/// Responsive grid that routes each device to the correct tile widget
/// based on its type and capabilities.
class DeviceGrid extends StatelessWidget {
  final List<Device> devices;

  const DeviceGrid({super.key, required this.devices});

  @override
  Widget build(BuildContext context) {
    if (devices.isEmpty) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(Icons.devices_other, size: 48, color: Colors.grey.shade600),
            const SizedBox(height: 12),
            Text(
              'No devices in this room',
              style: TextStyle(
                fontSize: 16,
                color: Colors.grey.shade500,
              ),
            ),
          ],
        ),
      );
    }

    return LayoutBuilder(
      builder: (context, constraints) {
        // Calculate grid columns based on available width
        final crossAxisCount = (constraints.maxWidth / 170).floor().clamp(2, 6);

        return GridView.builder(
          padding: const EdgeInsets.all(12),
          gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
            crossAxisCount: crossAxisCount,
            crossAxisSpacing: 12,
            mainAxisSpacing: 12,
            childAspectRatio: 0.85,
          ),
          itemCount: devices.length,
          itemBuilder: (context, index) => _buildTile(devices[index]),
        );
      },
    );
  }

  Widget _buildTile(Device device) {
    // Route to the correct tile based on domain and type first, then capabilities.
    // ValueKey ensures stable positions during state-driven rebuilds.
    if (device.domain == 'sensor') {
      return SensorTile(key: ValueKey(device.id), device: device);
    }
    if (device.domain == 'blinds') {
      return BlindTile(key: ValueKey(device.id), device: device);
    }
    if (device.hasDim) {
      return DimmerTile(key: ValueKey(device.id), device: device);
    }
    if (device.hasOnOff) {
      return SwitchTile(key: ValueKey(device.id), device: device);
    }
    // Fallback: show as a switch tile for any controllable device
    return SwitchTile(key: ValueKey(device.id), device: device);
  }
}
