import 'package:flutter/material.dart';

import '../models/device.dart';

/// Represents the mode assigned to a device within a scene.
/// 'leave_as_is' means the device won't be included in scene actions.
class DeviceSceneMode {
  final String mode; // 'leave_as_is', 'on', 'off', 'dim', 'set_level',
  //                    'set_position', 'set_tilt', 'set_setpoint', 'stop', 'toggle'
  final Map<String, dynamic> parameters;
  final int delayMs;
  final int fadeMs;
  final bool parallel;
  final bool continueOnError;

  const DeviceSceneMode({
    this.mode = 'leave_as_is',
    this.parameters = const {},
    this.delayMs = 0,
    this.fadeMs = 0,
    this.parallel = true,
    this.continueOnError = true,
  });

  DeviceSceneMode copyWith({
    String? mode,
    Map<String, dynamic>? parameters,
    int? delayMs,
    int? fadeMs,
    bool? parallel,
    bool? continueOnError,
  }) {
    return DeviceSceneMode(
      mode: mode ?? this.mode,
      parameters: parameters ?? this.parameters,
      delayMs: delayMs ?? this.delayMs,
      fadeMs: fadeMs ?? this.fadeMs,
      parallel: parallel ?? this.parallel,
      continueOnError: continueOnError ?? this.continueOnError,
    );
  }

  bool get isLeaveAsIs => mode == 'leave_as_is';
}

/// One row per device in the scene editor showing the device name
/// and a mode dropdown. Inline parameter controls appear based on mode.
/// Timing options (delay, fade, parallel, continueOnError) behind an expand chevron.
class DeviceModeSelector extends StatefulWidget {
  final Device device;
  final DeviceSceneMode mode;
  final ValueChanged<DeviceSceneMode> onChanged;

  const DeviceModeSelector({
    super.key,
    required this.device,
    required this.mode,
    required this.onChanged,
  });

  @override
  State<DeviceModeSelector> createState() => _DeviceModeSelectorState();
}

class _DeviceModeSelectorState extends State<DeviceModeSelector> {
  bool _expanded = false;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final device = widget.device;
    final mode = widget.mode;
    final isConfigured = !mode.isLeaveAsIs;

    return Container(
      margin: const EdgeInsets.only(bottom: 2),
      decoration: BoxDecoration(
        color: isConfigured
            ? theme.colorScheme.primaryContainer.withValues(alpha: 0.15)
            : null,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          // Main row: icon + name + mode dropdown + expand chevron
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
            child: Row(
              children: [
                Icon(
                  _domainIcon(device.domain),
                  size: 16,
                  color: isConfigured
                      ? theme.colorScheme.primary
                      : theme.colorScheme.onSurfaceVariant,
                ),
                const SizedBox(width: 8),
                Expanded(
                  flex: 3,
                  child: Text(
                    device.name,
                    overflow: TextOverflow.ellipsis,
                    style: theme.textTheme.bodyMedium?.copyWith(
                      fontWeight: isConfigured ? FontWeight.w600 : FontWeight.normal,
                    ),
                  ),
                ),
                const SizedBox(width: 8),
                Expanded(
                  flex: 2,
                  child: DropdownButtonFormField<String>(
                    value: mode.mode,
                    decoration: const InputDecoration(
                      border: OutlineInputBorder(),
                      contentPadding: EdgeInsets.symmetric(
                          horizontal: 8, vertical: 4),
                      isDense: true,
                    ),
                    items: _modesForDevice(device)
                        .map((m) => DropdownMenuItem(
                              value: m.value,
                              child: Text(m.label,
                                  style: const TextStyle(fontSize: 13)),
                            ))
                        .toList(),
                    onChanged: (v) {
                      if (v == null) return;
                      // When switching mode, reset parameters to defaults
                      final params = _defaultParams(v, device);
                      widget.onChanged(mode.copyWith(
                        mode: v,
                        parameters: params,
                      ));
                    },
                  ),
                ),
                if (isConfigured)
                  IconButton(
                    icon: Icon(
                      _expanded ? Icons.expand_less : Icons.expand_more,
                      size: 18,
                    ),
                    onPressed: () => setState(() => _expanded = !_expanded),
                    visualDensity: VisualDensity.compact,
                    padding: EdgeInsets.zero,
                    constraints:
                        const BoxConstraints(minWidth: 28, minHeight: 28),
                  ),
              ],
            ),
          ),

          // Inline parameter controls based on mode
          if (mode.mode == 'dim' || mode.mode == 'set_level')
            _buildSliderRow('Level', 'level', 50),
          if (mode.mode == 'set_position')
            _buildSliderRow('Position', 'position', 0),
          if (mode.mode == 'set_tilt')
            _buildSliderRow('Tilt', 'tilt', 50),
          if (mode.mode == 'set_setpoint')
            _buildSetpointRow(),

          // Expanded timing options
          if (_expanded && isConfigured) ...[
            const Divider(height: 8, indent: 8, endIndent: 8),
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 8),
              child: Row(
                children: [
                  Expanded(
                    child: TextFormField(
                      initialValue: mode.delayMs.toString(),
                      decoration: const InputDecoration(
                        labelText: 'Delay (ms)',
                        border: OutlineInputBorder(),
                        isDense: true,
                      ),
                      keyboardType: TextInputType.number,
                      onChanged: (v) => widget.onChanged(
                          mode.copyWith(delayMs: int.tryParse(v) ?? 0)),
                    ),
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: TextFormField(
                      initialValue: mode.fadeMs.toString(),
                      decoration: const InputDecoration(
                        labelText: 'Fade (ms)',
                        border: OutlineInputBorder(),
                        isDense: true,
                      ),
                      keyboardType: TextInputType.number,
                      onChanged: (v) => widget.onChanged(
                          mode.copyWith(fadeMs: int.tryParse(v) ?? 0)),
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 4),
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 8),
              child: Row(
                children: [
                  Expanded(
                    child: CheckboxListTile(
                      title: const Text('Parallel'),
                      subtitle: const Text('Run with others'),
                      value: mode.parallel,
                      dense: true,
                      contentPadding: EdgeInsets.zero,
                      controlAffinity: ListTileControlAffinity.leading,
                      onChanged: (v) => widget.onChanged(
                          mode.copyWith(parallel: v ?? true)),
                    ),
                  ),
                  Expanded(
                    child: CheckboxListTile(
                      title: const Text('Continue on error'),
                      value: mode.continueOnError,
                      dense: true,
                      contentPadding: EdgeInsets.zero,
                      controlAffinity: ListTileControlAffinity.leading,
                      onChanged: (v) => widget.onChanged(
                          mode.copyWith(continueOnError: v ?? true)),
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 4),
          ],
        ],
      ),
    );
  }

  Widget _buildSliderRow(String label, String paramKey, int defaultVal) {
    final mode = widget.mode;
    final value =
        ((mode.parameters[paramKey] as num?)?.toDouble() ?? defaultVal.toDouble())
            .clamp(0.0, 100.0);
    return Padding(
      padding: const EdgeInsets.only(left: 8, right: 8, top: 2),
      child: Row(
        children: [
          SizedBox(
            width: 56,
            child: Text('$label: ${value.round()}%',
                style: Theme.of(context).textTheme.bodySmall),
          ),
          Expanded(
            child: Slider(
              value: value,
              min: 0,
              max: 100,
              divisions: 20,
              onChanged: (v) {
                final params = Map<String, dynamic>.from(mode.parameters);
                params[paramKey] = v.round();
                widget.onChanged(mode.copyWith(parameters: params));
              },
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildSetpointRow() {
    final mode = widget.mode;
    final value =
        ((mode.parameters['setpoint'] as num?)?.toDouble() ?? 21.0)
            .clamp(5.0, 35.0);
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.only(top: 4, bottom: 4),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Material(
            color: theme.colorScheme.primaryContainer,
            borderRadius: BorderRadius.circular(8),
            child: InkWell(
              borderRadius: BorderRadius.circular(8),
              onTap: value > 5.0
                  ? () {
                      final params =
                          Map<String, dynamic>.from(mode.parameters);
                      params['setpoint'] = value - 0.5;
                      widget.onChanged(mode.copyWith(parameters: params));
                    }
                  : null,
              child: SizedBox(
                width: 44,
                height: 44,
                child: Icon(Icons.remove,
                    color: theme.colorScheme.onPrimaryContainer),
              ),
            ),
          ),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: Text(
              '${value.toStringAsFixed(1)}\u00B0C',
              style: theme.textTheme.headlineSmall?.copyWith(
                fontWeight: FontWeight.bold,
              ),
            ),
          ),
          Material(
            color: theme.colorScheme.primaryContainer,
            borderRadius: BorderRadius.circular(8),
            child: InkWell(
              borderRadius: BorderRadius.circular(8),
              onTap: value < 35.0
                  ? () {
                      final params =
                          Map<String, dynamic>.from(mode.parameters);
                      params['setpoint'] = value + 0.5;
                      widget.onChanged(mode.copyWith(parameters: params));
                    }
                  : null,
              child: SizedBox(
                width: 44,
                height: 44,
                child: Icon(Icons.add,
                    color: theme.colorScheme.onPrimaryContainer),
              ),
            ),
          ),
        ],
      ),
    );
  }

  /// Return available modes based on device capabilities.
  List<_ModeOption> _modesForDevice(Device device) {
    final modes = <_ModeOption>[
      const _ModeOption('leave_as_is', 'Leave as-is'),
    ];

    if (device.hasOnOff) {
      modes.add(const _ModeOption('on', 'On'));
      modes.add(const _ModeOption('off', 'Off'));
    }

    if (device.hasDim) {
      modes.add(const _ModeOption('set_level', 'Dim to %'));
    }

    // blind_switch has position capability but no position GA — only on/off
    if (device.hasPosition && device.type != 'blind_switch') {
      modes.add(const _ModeOption('set_position', 'Position %'));
    }

    if (device.hasTilt) {
      modes.add(const _ModeOption('set_tilt', 'Tilt %'));
    }

    if (device.domain == 'blinds') {
      modes.add(const _ModeOption('stop', 'Stop'));
      if (!device.hasOnOff && device.type == 'blind_switch') {
        modes.insert(1, const _ModeOption('on', 'Up'));
        modes.insert(2, const _ModeOption('off', 'Down'));
      }
    }

    if (device.hasTemperatureSet) {
      modes.add(const _ModeOption('set_setpoint', 'Set temp'));
    }

    return modes;
  }

  /// Default parameters when switching to a new mode.
  Map<String, dynamic> _defaultParams(String mode, Device device) {
    switch (mode) {
      case 'set_level':
      case 'dim':
        return {'level': 50};
      case 'set_position':
        return {'position': 0};
      case 'set_tilt':
        return {'tilt': 50};
      case 'set_setpoint':
        return {'setpoint': 21.0};
      default:
        return {};
    }
  }

  IconData _domainIcon(String domain) {
    switch (domain) {
      case 'lighting':
        return Icons.lightbulb_outline;
      case 'blinds':
        return Icons.blinds;
      case 'climate':
        return Icons.thermostat;
      case 'security':
        return Icons.security;
      case 'audio':
        return Icons.volume_up;
      default:
        return Icons.device_hub;
    }
  }
}

class _ModeOption {
  final String value;
  final String label;
  const _ModeOption(this.value, this.label);
}
