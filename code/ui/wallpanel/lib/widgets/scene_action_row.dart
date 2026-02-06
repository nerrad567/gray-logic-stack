import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/device.dart';
import '../providers/auth_provider.dart';
import 'scene_editor_sheet.dart';

/// A single action row within the scene editor.
/// Shows device picker, command, parameters, and timing options.
class SceneActionRow extends ConsumerStatefulWidget {
  final SceneActionData action;
  final int index;
  final String? roomId;
  final ValueChanged<SceneActionData> onChanged;
  final VoidCallback onDelete;

  const SceneActionRow({
    super.key,
    required this.action,
    required this.index,
    required this.roomId,
    required this.onChanged,
    required this.onDelete,
  });

  @override
  ConsumerState<SceneActionRow> createState() => _SceneActionRowState();
}

class _SceneActionRowState extends ConsumerState<SceneActionRow> {
  bool _expanded = false;
  List<Device>? _devices;
  bool _loadingDevices = false;

  @override
  void initState() {
    super.initState();
    _loadDevices();
  }

  Future<void> _loadDevices() async {
    setState(() => _loadingDevices = true);
    try {
      final apiClient = ref.read(apiClientProvider);
      final response = widget.roomId != null
          ? await apiClient.getDevices(roomId: widget.roomId)
          : await apiClient.getDevices();
      if (mounted) {
        setState(() {
          _devices = response.devices;
          _loadingDevices = false;
        });
      }
    } catch (_) {
      if (mounted) setState(() => _loadingDevices = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final a = widget.action;

    // Determine commands based on selected device
    final selectedDevice = _devices?.where((d) => d.id == a.deviceId).firstOrNull;
    final commands = _commandsForDevice(selectedDevice);

    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          children: [
            // Header row: device + command + delete
            Row(
              children: [
                // Drag handle
                Icon(Icons.drag_handle,
                    size: 20, color: theme.colorScheme.onSurfaceVariant),
                const SizedBox(width: 8),
                // Action number
                CircleAvatar(
                  radius: 12,
                  backgroundColor: theme.colorScheme.primaryContainer,
                  child: Text(
                    '${widget.index + 1}',
                    style: theme.textTheme.labelSmall?.copyWith(
                      color: theme.colorScheme.onPrimaryContainer,
                    ),
                  ),
                ),
                const SizedBox(width: 8),
                // Device dropdown
                Expanded(
                  flex: 3,
                  child: _loadingDevices
                      ? const LinearProgressIndicator()
                      : DropdownButtonFormField<String>(
                          value: a.deviceId.isEmpty ? null : a.deviceId,
                          decoration: const InputDecoration(
                            labelText: 'Device',
                            border: OutlineInputBorder(),
                            contentPadding: EdgeInsets.symmetric(
                                horizontal: 12, vertical: 8),
                            isDense: true,
                          ),
                          items: _devices?.map((d) => DropdownMenuItem(
                                value: d.id,
                                child: Text(d.name, overflow: TextOverflow.ellipsis),
                              )).toList() ?? [],
                          onChanged: (v) {
                            if (v == null) return;
                            widget.onChanged(SceneActionData(
                              deviceId: v,
                              command: a.command,
                              parameters: a.parameters,
                              delayMs: a.delayMs,
                              fadeMs: a.fadeMs,
                              parallel: a.parallel,
                              continueOnError: a.continueOnError,
                            ));
                          },
                          validator: (v) =>
                              (v == null || v.isEmpty) ? 'Required' : null,
                        ),
                ),
                const SizedBox(width: 8),
                // Command dropdown
                Expanded(
                  flex: 2,
                  child: DropdownButtonFormField<String>(
                    value: commands.contains(a.command) ? a.command : commands.first,
                    decoration: const InputDecoration(
                      labelText: 'Command',
                      border: OutlineInputBorder(),
                      contentPadding:
                          EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                      isDense: true,
                    ),
                    items: commands
                        .map((c) => DropdownMenuItem(value: c, child: Text(c)))
                        .toList(),
                    onChanged: (v) {
                      if (v == null) return;
                      widget.onChanged(SceneActionData(
                        deviceId: a.deviceId,
                        command: v,
                        parameters: a.parameters,
                        delayMs: a.delayMs,
                        fadeMs: a.fadeMs,
                        parallel: a.parallel,
                        continueOnError: a.continueOnError,
                      ));
                    },
                  ),
                ),
                const SizedBox(width: 4),
                // Expand/collapse
                IconButton(
                  icon: Icon(
                    _expanded ? Icons.expand_less : Icons.expand_more,
                    size: 20,
                  ),
                  onPressed: () => setState(() => _expanded = !_expanded),
                  visualDensity: VisualDensity.compact,
                ),
                // Delete
                IconButton(
                  icon: Icon(Icons.close, size: 20, color: theme.colorScheme.error),
                  onPressed: widget.onDelete,
                  visualDensity: VisualDensity.compact,
                ),
              ],
            ),

            // Parameter row (for dim/position/setpoint)
            if (a.command == 'dim' || a.command == 'set_level')
              Padding(
                padding: const EdgeInsets.only(top: 8),
                child: _LevelSlider(
                  label: 'Level',
                  value: (a.parameters['level'] as num?)?.toDouble() ?? 50,
                  onChanged: (v) {
                    final params = Map<String, dynamic>.from(a.parameters);
                    params['level'] = v.round();
                    widget.onChanged(SceneActionData(
                      deviceId: a.deviceId,
                      command: a.command,
                      parameters: params,
                      delayMs: a.delayMs,
                      fadeMs: a.fadeMs,
                      parallel: a.parallel,
                      continueOnError: a.continueOnError,
                    ));
                  },
                ),
              ),
            if (a.command == 'set_position')
              Padding(
                padding: const EdgeInsets.only(top: 8),
                child: _LevelSlider(
                  label: 'Position',
                  value: (a.parameters['position'] as num?)?.toDouble() ?? 0,
                  onChanged: (v) {
                    final params = Map<String, dynamic>.from(a.parameters);
                    params['position'] = v.round();
                    widget.onChanged(SceneActionData(
                      deviceId: a.deviceId,
                      command: a.command,
                      parameters: params,
                      delayMs: a.delayMs,
                      fadeMs: a.fadeMs,
                      parallel: a.parallel,
                      continueOnError: a.continueOnError,
                    ));
                  },
                ),
              ),
            if (a.command == 'set_setpoint')
              Padding(
                padding: const EdgeInsets.only(top: 8),
                child: _SetpointField(
                  value: (a.parameters['setpoint'] as num?)?.toDouble() ?? 21,
                  onChanged: (v) {
                    final params = Map<String, dynamic>.from(a.parameters);
                    params['setpoint'] = v;
                    widget.onChanged(SceneActionData(
                      deviceId: a.deviceId,
                      command: a.command,
                      parameters: params,
                      delayMs: a.delayMs,
                      fadeMs: a.fadeMs,
                      parallel: a.parallel,
                      continueOnError: a.continueOnError,
                    ));
                  },
                ),
              ),

            // Expanded timing options
            if (_expanded) ...[
              const Divider(height: 16),
              Row(
                children: [
                  Expanded(
                    child: TextFormField(
                      initialValue: a.delayMs.toString(),
                      decoration: const InputDecoration(
                        labelText: 'Delay (ms)',
                        border: OutlineInputBorder(),
                        isDense: true,
                      ),
                      keyboardType: TextInputType.number,
                      onChanged: (v) {
                        final ms = int.tryParse(v) ?? 0;
                        widget.onChanged(SceneActionData(
                          deviceId: a.deviceId,
                          command: a.command,
                          parameters: a.parameters,
                          delayMs: ms,
                          fadeMs: a.fadeMs,
                          parallel: a.parallel,
                          continueOnError: a.continueOnError,
                        ));
                      },
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: TextFormField(
                      initialValue: a.fadeMs.toString(),
                      decoration: const InputDecoration(
                        labelText: 'Fade (ms)',
                        border: OutlineInputBorder(),
                        isDense: true,
                      ),
                      keyboardType: TextInputType.number,
                      onChanged: (v) {
                        final ms = int.tryParse(v) ?? 0;
                        widget.onChanged(SceneActionData(
                          deviceId: a.deviceId,
                          command: a.command,
                          parameters: a.parameters,
                          delayMs: a.delayMs,
                          fadeMs: ms,
                          parallel: a.parallel,
                          continueOnError: a.continueOnError,
                        ));
                      },
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 8),
              Row(
                children: [
                  Expanded(
                    child: CheckboxListTile(
                      title: const Text('Parallel'),
                      subtitle: const Text('Run with previous'),
                      value: a.parallel,
                      dense: true,
                      contentPadding: EdgeInsets.zero,
                      controlAffinity: ListTileControlAffinity.leading,
                      onChanged: (v) {
                        widget.onChanged(SceneActionData(
                          deviceId: a.deviceId,
                          command: a.command,
                          parameters: a.parameters,
                          delayMs: a.delayMs,
                          fadeMs: a.fadeMs,
                          parallel: v ?? false,
                          continueOnError: a.continueOnError,
                        ));
                      },
                    ),
                  ),
                  Expanded(
                    child: CheckboxListTile(
                      title: const Text('Continue on error'),
                      value: a.continueOnError,
                      dense: true,
                      contentPadding: EdgeInsets.zero,
                      controlAffinity: ListTileControlAffinity.leading,
                      onChanged: (v) {
                        widget.onChanged(SceneActionData(
                          deviceId: a.deviceId,
                          command: a.command,
                          parameters: a.parameters,
                          delayMs: a.delayMs,
                          fadeMs: a.fadeMs,
                          parallel: a.parallel,
                          continueOnError: v ?? false,
                        ));
                      },
                    ),
                  ),
                ],
              ),
            ],
          ],
        ),
      ),
    );
  }

  /// Return available commands based on device domain/type.
  List<String> _commandsForDevice(Device? device) {
    if (device == null) return ['on', 'off', 'toggle'];

    switch (device.domain) {
      case 'lighting':
        if (device.hasDim) return ['on', 'off', 'toggle', 'dim', 'set_level'];
        return ['on', 'off', 'toggle'];
      case 'blinds':
        return ['on', 'off', 'set_position', 'stop'];
      case 'climate':
        return ['set_setpoint'];
      default:
        return ['on', 'off', 'toggle'];
    }
  }
}

class _LevelSlider extends StatelessWidget {
  final String label;
  final double value;
  final ValueChanged<double> onChanged;

  const _LevelSlider({
    required this.label,
    required this.value,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        SizedBox(
          width: 60,
          child: Text('$label: ${value.round()}%',
              style: Theme.of(context).textTheme.bodySmall),
        ),
        Expanded(
          child: Slider(
            value: value.clamp(0, 100),
            min: 0,
            max: 100,
            divisions: 20,
            onChanged: onChanged,
          ),
        ),
      ],
    );
  }
}

class _SetpointField extends StatelessWidget {
  final double value;
  final ValueChanged<double> onChanged;

  const _SetpointField({
    required this.value,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        SizedBox(
          width: 100,
          child: TextFormField(
            initialValue: value.toString(),
            decoration: const InputDecoration(
              labelText: 'Setpoint',
              suffixText: '\u00B0C',
              border: OutlineInputBorder(),
              isDense: true,
            ),
            keyboardType: const TextInputType.numberWithOptions(decimal: true),
            onChanged: (v) {
              final sp = double.tryParse(v);
              if (sp != null) onChanged(sp);
            },
          ),
        ),
      ],
    );
  }
}
