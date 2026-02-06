import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/device.dart';
import '../providers/device_provider.dart';

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
    // Watch the provider to trigger rebuilds on WebSocket state changes.
    // Find our updated device from the provider's list.
    final devicesAsync = ref.watch(roomDevicesProvider);
    final liveDevice = devicesAsync.value
            ?.firstWhere((d) => d.id == device.id, orElse: () => device) ??
        device;

    final theme = Theme.of(context);
    final channels = _parseChannels(liveDevice.config);
    final isOnline = liveDevice.isOnline;

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
                deviceState: liveDevice.state,
                isValveType: liveDevice.type.contains('heating'),
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
    final valvePercent = isValveType ? _getValvePercent() : 0.0;

    // Heating channels use a warm colour gradient based on valve position;
    // switch channels keep the existing green/grey scheme.
    final Color accentColor;
    if (isValveType && isActive) {
      accentColor = _heatColor(valvePercent);
    } else if (isActive) {
      accentColor = Colors.green;
    } else {
      accentColor = Colors.grey.shade700;
    }

    final bgColor = isSpare
        ? Colors.grey.shade800
        : isActive
            ? accentColor.withValues(alpha: 0.2)
            : Colors.grey.shade900;

    final borderColor = isSpare ? Colors.grey.shade700 : accentColor;

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
                          ? (isValveType ? accentColor : Colors.green.shade400)
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
  /// State keys use the Go StateKeyForFunction convention:
  ///   ch_a_switch → ch_a_on, ch_a_valve → ch_a_valve
  String _getStatusText() {
    if (channel.isSpare) return '--';

    final prefix = 'ch_${channel.letter.toLowerCase()}';

    if (isValveType) {
      // Valve state key: ch_X_valve (matches Go StateKeyForFunction)
      for (final suffix in ['_valve', '_valve_status']) {
        final val = deviceState['$prefix$suffix'];
        if (val is num) return '${val.toInt()}%';
      }
      return '?';
    }

    // Switch state key: ch_X_on (Go maps "switch" → state key "on")
    final sw = deviceState['${prefix}_on'];
    if (sw == true) return 'ON';
    if (sw == false) return 'OFF';
    return '?';
  }

  /// Get the valve position as a 0.0–1.0 fraction for colour interpolation.
  double _getValvePercent() {
    final prefix = 'ch_${channel.letter.toLowerCase()}';
    for (final suffix in ['_valve', '_valve_status']) {
      final val = deviceState['$prefix$suffix'];
      if (val is num) return (val / 100).clamp(0.0, 1.0);
    }
    return 0.0;
  }

  /// Map a 0.0–1.0 valve fraction to a warm colour:
  ///   0% = steel blue (cold/closed)
  ///  25% = amber (warming)
  ///  50% = deep orange
  /// 100% = red (fully open / max heat)
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

  /// Whether this channel is currently active (on/open/non-zero).
  bool _isActive() {
    if (channel.isSpare) return false;

    final prefix = 'ch_${channel.letter.toLowerCase()}';

    if (isValveType) {
      for (final suffix in ['_valve', '_valve_status']) {
        final val = deviceState['$prefix$suffix'];
        if (val is num) return val > 0;
      }
      return false;
    }

    // Switch state key: ch_X_on (Go maps "switch" → state key "on")
    final sw = deviceState['${prefix}_on'];
    if (sw is bool) return sw;
    return false;
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
