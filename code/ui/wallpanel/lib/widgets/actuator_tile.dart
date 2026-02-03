import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/device.dart';

/// A read-only tile for multi-channel actuators in the Distribution Board.
///
/// Displays the actuator name and a compact grid of channel indicators,
/// each showing the channel letter, ON/OFF or percentage status, and
/// a truncated load name. Spare channels (no load assigned) show "--".
///
/// Channel metadata is read from `device.config['channels']`, which is
/// a map keyed by channel letter (A-F) containing load_name, load_type, etc.
class ActuatorTile extends ConsumerWidget {
  final Device device;

  const ActuatorTile({super.key, required this.device});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final channels = _parseChannels(device.config);
    final isOnline = device.isOnline;

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(10),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Header: device name + online indicator
            Row(
              children: [
                Expanded(
                  child: Text(
                    device.name,
                    style: theme.textTheme.titleSmall?.copyWith(
                      fontWeight: FontWeight.w600,
                      fontSize: 12,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                Container(
                  width: 8,
                  height: 8,
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    color: isOnline ? Colors.green : Colors.grey.shade600,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 6),
            // Channel indicators in a 3x2 or 2x3 grid
            Expanded(
              child: _ChannelGrid(
                channels: channels,
                deviceState: device.state,
                isValveType: device.type.contains('heating'),
              ),
            ),
          ],
        ),
      ),
    );
  }

  /// Parse channel metadata from the device config map.
  static List<_ChannelInfo> _parseChannels(Map<String, dynamic> config) {
    final raw = config['channels'];
    if (raw is! Map) return [];

    final letters = raw.keys.toList()..sort();
    return letters.map((letter) {
      final ch = raw[letter];
      if (ch is! Map) {
        return _ChannelInfo(letter: letter.toString(), loadName: null, loadType: null);
      }
      return _ChannelInfo(
        letter: letter.toString(),
        loadName: ch['load_name'] as String?,
        loadType: ch['load_type'] as String?,
      );
    }).toList();
  }
}

/// Parsed channel metadata for display.
class _ChannelInfo {
  final String letter;
  final String? loadName;
  final String? loadType;

  const _ChannelInfo({
    required this.letter,
    this.loadName,
    this.loadType,
  });

  bool get isSpare => loadName == null || loadName!.isEmpty;
}

/// A compact grid of channel status indicators.
class _ChannelGrid extends StatelessWidget {
  final List<_ChannelInfo> channels;
  final Map<String, dynamic> deviceState;
  final bool isValveType;

  const _ChannelGrid({
    required this.channels,
    required this.deviceState,
    required this.isValveType,
  });

  @override
  Widget build(BuildContext context) {
    if (channels.isEmpty) {
      return Center(
        child: Text(
          'No channels',
          style: TextStyle(fontSize: 11, color: Colors.grey.shade500),
        ),
      );
    }

    return GridView.count(
      crossAxisCount: 3,
      crossAxisSpacing: 4,
      mainAxisSpacing: 4,
      childAspectRatio: 1.3,
      physics: const NeverScrollableScrollPhysics(),
      children: channels.map((ch) {
        return _ChannelIndicator(
          channel: ch,
          deviceState: deviceState,
          isValveType: isValveType,
        );
      }).toList(),
    );
  }
}

/// A single channel indicator box showing letter, status, and load name.
class _ChannelIndicator extends StatelessWidget {
  final _ChannelInfo channel;
  final Map<String, dynamic> deviceState;
  final bool isValveType;

  const _ChannelIndicator({
    required this.channel,
    required this.deviceState,
    required this.isValveType,
  });

  @override
  Widget build(BuildContext context) {
    final isSpare = channel.isSpare;
    final statusText = _getStatusText();
    final isActive = _isActive();

    final bgColor = isSpare
        ? Colors.grey.shade800
        : isActive
            ? Colors.green.withValues(alpha: 0.2)
            : Colors.grey.shade900;

    final borderColor = isSpare
        ? Colors.grey.shade700
        : isActive
            ? Colors.green.shade700
            : Colors.grey.shade700;

    return Container(
      decoration: BoxDecoration(
        color: bgColor,
        borderRadius: BorderRadius.circular(6),
        border: Border.all(color: borderColor, width: 1),
      ),
      padding: const EdgeInsets.symmetric(horizontal: 3, vertical: 2),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          // Channel letter + status
          Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Text(
                channel.letter,
                style: TextStyle(
                  fontSize: 10,
                  fontWeight: FontWeight.w700,
                  color: isSpare ? Colors.grey.shade600 : Colors.white,
                ),
              ),
              const SizedBox(width: 3),
              Text(
                statusText,
                style: TextStyle(
                  fontSize: 9,
                  fontWeight: FontWeight.w600,
                  color: isSpare
                      ? Colors.grey.shade600
                      : isActive
                          ? Colors.green.shade400
                          : Colors.grey.shade400,
                ),
              ),
            ],
          ),
          // Load name (truncated)
          if (!isSpare) ...[
            const SizedBox(height: 1),
            Text(
              _truncateLoadName(channel.loadName!),
              style: TextStyle(
                fontSize: 8,
                color: Colors.grey.shade400,
              ),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              textAlign: TextAlign.center,
            ),
          ],
          if (isSpare)
            Text(
              'Spare',
              style: TextStyle(
                fontSize: 8,
                color: Colors.grey.shade600,
                fontStyle: FontStyle.italic,
              ),
            ),
        ],
      ),
    );
  }

  /// Get the status text for this channel from the device state.
  /// State keys are prefixed: ch_a_switch_status, ch_a_valve_status, etc.
  String _getStatusText() {
    if (channel.isSpare) return '--';

    final prefix = 'ch_${channel.letter.toLowerCase()}';

    if (isValveType) {
      // Look for valve percentage
      final valveKey = '${prefix}_valve_status';
      final val = deviceState[valveKey];
      if (val is num) return '${val.toInt()}%';
      // Fall back to on/off
      final switchKey = '${prefix}_switch_status';
      final sw = deviceState[switchKey];
      if (sw == true) return 'OPEN';
      if (sw == false) return 'SHUT';
      return '?';
    }

    // Switch/dimmer actuator
    final switchKey = '${prefix}_switch_status';
    final sw = deviceState[switchKey];
    if (sw == true) return 'ON';
    if (sw == false) return 'OFF';
    return '?';
  }

  /// Whether this channel is currently active (on/open/non-zero).
  bool _isActive() {
    if (channel.isSpare) return false;

    final prefix = 'ch_${channel.letter.toLowerCase()}';

    if (isValveType) {
      final valveKey = '${prefix}_valve_status';
      final val = deviceState[valveKey];
      if (val is num) return val > 0;
      final switchKey = '${prefix}_switch_status';
      return deviceState[switchKey] == true;
    }

    final switchKey = '${prefix}_switch_status';
    return deviceState[switchKey] == true;
  }

  /// Truncate load name for compact display.
  String _truncateLoadName(String name) {
    // Remove common suffixes to save space
    return name
        .replaceAll(' Light', '')
        .replaceAll(' Valve', '')
        .replaceAll(' Lamp', '');
  }
}
