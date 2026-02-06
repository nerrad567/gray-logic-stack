import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/device.dart';
import '../models/scene.dart';
import '../providers/device_provider.dart';
import '../providers/location_provider.dart';
import '../providers/scene_provider.dart';
import 'device_mode_selector.dart';

/// Bottom sheet for creating or editing a scene.
/// Shows all room devices with mode selectors (default "Leave as-is").
/// Pass [scene] = null for create mode, or an existing scene for edit mode.
/// Pass [preselectedRoomId] to pre-fill the room when creating from room view.
class SceneEditorSheet extends ConsumerStatefulWidget {
  final Scene? scene;
  final String? preselectedRoomId;

  const SceneEditorSheet({super.key, this.scene, this.preselectedRoomId});

  @override
  ConsumerState<SceneEditorSheet> createState() => _SceneEditorSheetState();
}

class _SceneEditorSheetState extends ConsumerState<SceneEditorSheet> {
  final _formKey = GlobalKey<FormState>();
  late final TextEditingController _nameController;
  late final TextEditingController _descController;
  late String? _roomId;
  late String? _icon;
  late String? _colour;
  late String? _category;
  late bool _enabled;
  late int _priority;
  bool _saving = false;
  List<Device>? _devices;

  /// Device modes: deviceId -> DeviceSceneMode.
  /// Devices not in this map or with mode 'leave_as_is' are excluded from actions.
  late Map<String, DeviceSceneMode> _deviceModes;

  bool get _isEdit => widget.scene != null;

  @override
  void initState() {
    super.initState();
    final s = widget.scene;
    _nameController = TextEditingController(text: s?.name ?? '');
    _descController = TextEditingController(text: s?.description ?? '');
    _roomId = s?.roomId ?? widget.preselectedRoomId;
    _icon = s?.icon;
    _colour = s?.colour;
    _category = s?.category;
    _enabled = s?.enabled ?? true;
    _priority = s?.priority ?? 50;

    // Reconstruct device modes from existing scene actions
    _deviceModes = {};
    if (s != null) {
      for (final action in s.actions) {
        _deviceModes[action.deviceId] = DeviceSceneMode(
          mode: action.command,
          parameters: Map<String, dynamic>.from(action.parameters ?? {}),
          delayMs: action.delayMs,
          fadeMs: action.fadeMs,
          parallel: action.parallel,
          continueOnError: action.continueOnError,
        );
      }
    }

    _loadDevices();
  }

  Future<void> _loadDevices() async {
    try {
      final repo = ref.read(deviceRepositoryProvider);
      final devices = _roomId != null
          ? await repo.getDevicesByRoom(_roomId!)
          : await repo.getAllDevices();
      if (mounted) setState(() => _devices = devices);
    } catch (_) {
      if (mounted) setState(() => _devices = []);
    }
  }

  @override
  void dispose() {
    _nameController.dispose();
    _descController.dispose();
    super.dispose();
  }

  String _generateSlug(String name) {
    return name
        .toLowerCase()
        .replaceAll(RegExp(r'[^a-z0-9]+'), '-')
        .replaceAll(RegExp(r'^-|-$'), '');
  }

  /// Count devices with a non-leave-as-is mode.
  int get _configuredCount =>
      _deviceModes.values.where((m) => !m.isLeaveAsIs).length;

  Future<void> _save() async {
    if (!_formKey.currentState!.validate()) return;
    if (_configuredCount == 0) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Configure at least one device'),
          behavior: SnackBarBehavior.floating,
        ),
      );
      return;
    }

    setState(() => _saving = true);

    // Convert device modes to action list, excluding 'leave_as_is'
    final actions = <Map<String, dynamic>>[];
    int sortOrder = 0;
    for (final entry in _deviceModes.entries) {
      if (entry.value.isLeaveAsIs) continue;
      actions.add({
        'device_id': entry.key,
        'command': entry.value.mode,
        if (entry.value.parameters.isNotEmpty)
          'parameters': entry.value.parameters,
        'delay_ms': entry.value.delayMs,
        'fade_ms': entry.value.fadeMs,
        'parallel': entry.value.parallel,
        'continue_on_error': entry.value.continueOnError,
        'sort_order': sortOrder,
      });
      sortOrder++;
    }

    final data = <String, dynamic>{
      'name': _nameController.text.trim(),
      'slug': _generateSlug(_nameController.text.trim()),
      if (_descController.text.trim().isNotEmpty)
        'description': _descController.text.trim(),
      if (_roomId != null) 'room_id': _roomId,
      'enabled': _enabled,
      'priority': _priority,
      if (_icon != null) 'icon': _icon,
      if (_colour != null) 'colour': _colour,
      if (_category != null) 'category': _category,
      'actions': actions,
    };

    try {
      final notifier = ref.read(allScenesProvider.notifier);
      if (_isEdit) {
        await notifier.updateScene(widget.scene!.id, data);
      } else {
        await notifier.createScene(data);
      }

      if (!mounted) return;
      Navigator.of(context).pop(true);
    } catch (e) {
      if (!mounted) return;
      setState(() => _saving = false);

      final msg = e.toString().contains('409')
          ? 'A scene with this name already exists'
          : e.toString().contains('400')
              ? 'Invalid scene data — check all fields'
              : 'Failed to save scene';

      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(msg),
          behavior: SnackBarBehavior.floating,
          backgroundColor: Theme.of(context).colorScheme.error,
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final locationAsync = ref.watch(locationDataProvider);

    return DraggableScrollableSheet(
      initialChildSize: 0.85,
      minChildSize: 0.5,
      maxChildSize: 0.95,
      expand: false,
      builder: (context, scrollController) {
        return Container(
          decoration: BoxDecoration(
            color: theme.colorScheme.surface,
            borderRadius:
                const BorderRadius.vertical(top: Radius.circular(16)),
          ),
          child: Column(
            children: [
              // Handle bar + header
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
                child: Column(
                  children: [
                    Center(
                      child: Container(
                        width: 32,
                        height: 4,
                        decoration: BoxDecoration(
                          color: theme.colorScheme.onSurfaceVariant
                              .withValues(alpha: 0.4),
                          borderRadius: BorderRadius.circular(2),
                        ),
                      ),
                    ),
                    const SizedBox(height: 12),
                    Row(
                      children: [
                        Expanded(
                          child: Text(
                            _isEdit ? 'Edit Scene' : 'New Scene',
                            style: theme.textTheme.titleLarge,
                          ),
                        ),
                        if (_configuredCount > 0)
                          Padding(
                            padding: const EdgeInsets.only(right: 8),
                            child: Chip(
                              label: Text('$_configuredCount devices'),
                              visualDensity: VisualDensity.compact,
                            ),
                          ),
                        TextButton(
                          onPressed: () => Navigator.pop(context),
                          child: const Text('Cancel'),
                        ),
                        const SizedBox(width: 8),
                        FilledButton(
                          onPressed: _saving ? null : _save,
                          child: _saving
                              ? const SizedBox(
                                  width: 20,
                                  height: 20,
                                  child: CircularProgressIndicator(
                                      strokeWidth: 2),
                                )
                              : Text(_isEdit ? 'Save' : 'Create'),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
              const Divider(),
              // Quick presets (create mode only)
              if (!_isEdit) _buildPresetsRow(context),
              // Scrollable form
              Expanded(
                child: Form(
                  key: _formKey,
                  child: ListView(
                    controller: scrollController,
                    padding: const EdgeInsets.symmetric(
                        horizontal: 16, vertical: 8),
                    children: [
                      // Name + Room row
                      Row(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Expanded(
                            flex: 3,
                            child: TextFormField(
                              controller: _nameController,
                              decoration: const InputDecoration(
                                labelText: 'Name',
                                border: OutlineInputBorder(),
                                isDense: true,
                              ),
                              validator: (v) => (v == null || v.trim().isEmpty)
                                  ? 'Required'
                                  : null,
                            ),
                          ),
                          const SizedBox(width: 8),
                          Expanded(
                            flex: 2,
                            child: locationAsync.when(
                              data: (data) =>
                                  DropdownButtonFormField<String?>(
                                value: _roomId,
                                decoration: const InputDecoration(
                                  labelText: 'Room',
                                  border: OutlineInputBorder(),
                                  isDense: true,
                                ),
                                items: [
                                  const DropdownMenuItem<String?>(
                                    value: null,
                                    child: Text('Global'),
                                  ),
                                  ...data.sortedRooms.map((room) =>
                                      DropdownMenuItem<String?>(
                                        value: room.id,
                                        child: Text(room.name,
                                            overflow: TextOverflow.ellipsis),
                                      )),
                                ],
                                onChanged: (v) {
                                  setState(() {
                                    _roomId = v;
                                    // Reset all modes when room changes
                                    _deviceModes = {};
                                  });
                                  _loadDevices();
                                },
                              ),
                              loading: () =>
                                  const LinearProgressIndicator(),
                              error: (_, _) =>
                                  const Text('Failed to load rooms'),
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 8),

                      // Description (single line)
                      TextFormField(
                        controller: _descController,
                        decoration: const InputDecoration(
                          labelText: 'Description (optional)',
                          border: OutlineInputBorder(),
                          isDense: true,
                        ),
                      ),
                      const SizedBox(height: 8),

                      // Style row: Icon + Category + Colour
                      Row(
                        children: [
                          Expanded(
                            child: DropdownButtonFormField<String?>(
                              value: _icon,
                              decoration: const InputDecoration(
                                labelText: 'Icon',
                                border: OutlineInputBorder(),
                                isDense: true,
                              ),
                              items: [
                                const DropdownMenuItem<String?>(
                                  value: null,
                                  child: Text('None'),
                                ),
                                for (final entry in _iconOptions.entries)
                                  DropdownMenuItem<String?>(
                                    value: entry.key,
                                    child: Row(
                                      children: [
                                        Icon(entry.value, size: 16),
                                        const SizedBox(width: 4),
                                        Text(entry.key),
                                      ],
                                    ),
                                  ),
                              ],
                              onChanged: (v) => setState(() => _icon = v),
                            ),
                          ),
                          const SizedBox(width: 8),
                          Expanded(
                            child: DropdownButtonFormField<String?>(
                              value: _category,
                              decoration: const InputDecoration(
                                labelText: 'Category',
                                border: OutlineInputBorder(),
                                isDense: true,
                              ),
                              items: const [
                                DropdownMenuItem<String?>(
                                    value: null, child: Text('None')),
                                DropdownMenuItem(
                                    value: 'lighting',
                                    child: Text('Lighting')),
                                DropdownMenuItem(
                                    value: 'comfort',
                                    child: Text('Comfort')),
                                DropdownMenuItem(
                                    value: 'media', child: Text('Media')),
                                DropdownMenuItem(
                                    value: 'security',
                                    child: Text('Security')),
                                DropdownMenuItem(
                                    value: 'custom',
                                    child: Text('Custom')),
                              ],
                              onChanged: (v) =>
                                  setState(() => _category = v),
                            ),
                          ),
                          const SizedBox(width: 8),
                          SizedBox(
                            width: 100,
                            child: TextFormField(
                              initialValue: _colour ?? '',
                              decoration: InputDecoration(
                                labelText: 'Colour',
                                hintText: '#FF9800',
                                border: const OutlineInputBorder(),
                                isDense: true,
                                prefixIcon: _colour != null
                                    ? Padding(
                                        padding: const EdgeInsets.all(8),
                                        child: Container(
                                          width: 16,
                                          height: 16,
                                          decoration: BoxDecoration(
                                            color: _parseColour(_colour),
                                            borderRadius:
                                                BorderRadius.circular(3),
                                            border: Border.all(
                                                color: theme
                                                    .colorScheme.outline),
                                          ),
                                        ),
                                      )
                                    : null,
                                prefixIconConstraints:
                                    const BoxConstraints(
                                        minWidth: 32, maxWidth: 32),
                              ),
                              onChanged: (v) {
                                final cleaned = v.trim();
                                setState(() => _colour =
                                    cleaned.isEmpty ? null : cleaned);
                              },
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 12),

                      // Devices section — grouped by domain
                      Text('Devices', style: theme.textTheme.titleMedium),
                      const SizedBox(height: 8),

                      if (_devices == null)
                        const Center(child: CircularProgressIndicator())
                      else if (_devices!.where((d) => d.isSceneTarget).isEmpty)
                        Container(
                          padding: const EdgeInsets.all(16),
                          decoration: BoxDecoration(
                            border: Border.all(
                                color: theme.colorScheme.outlineVariant),
                            borderRadius: BorderRadius.circular(8),
                          ),
                          child: Center(
                            child: Text(
                              _roomId != null
                                  ? 'No controllable devices in this room'
                                  : 'Select a room to see devices',
                              style: theme.textTheme.bodySmall?.copyWith(
                                color: theme.colorScheme.onSurfaceVariant,
                              ),
                            ),
                          ),
                        )
                      else
                        ..._buildDeviceGrid(theme),

                      const SizedBox(height: 40),
                    ],
                  ),
                ),
              ),
            ],
          ),
        );
      },
    );
  }

  /// Build device rows grouped by domain with section headers.
  List<Widget> _buildDeviceGrid(ThemeData theme) {
    final sceneTargets = _devices!.where((d) => d.isSceneTarget).toList();

    // Group by domain
    final grouped = <String, List<Device>>{};
    for (final device in sceneTargets) {
      grouped.putIfAbsent(device.domain, () => []).add(device);
    }

    // Sort domains and devices within each domain
    final sortedDomains = grouped.keys.toList()..sort();
    final widgets = <Widget>[];

    for (final domain in sortedDomains) {
      final devices = grouped[domain]!
        ..sort((a, b) => a.name.compareTo(b.name));

      // Domain header
      widgets.add(Padding(
        padding: const EdgeInsets.only(top: 8, bottom: 4),
        child: Text(
          _domainLabel(domain),
          style: theme.textTheme.labelLarge?.copyWith(
            color: theme.colorScheme.primary,
          ),
        ),
      ));

      // Device rows
      for (final device in devices) {
        final mode =
            _deviceModes[device.id] ?? const DeviceSceneMode();
        widgets.add(DeviceModeSelector(
          key: ValueKey(device.id),
          device: device,
          mode: mode,
          onChanged: (newMode) {
            setState(() {
              _deviceModes[device.id] = newMode;
            });
          },
        ));
      }
    }

    return widgets;
  }

  Widget _buildPresetsRow(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('Quick Presets',
              style: theme.textTheme.labelMedium?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              )),
          const SizedBox(height: 8),
          Wrap(
            spacing: 8,
            runSpacing: 4,
            children: [
              for (final preset in _scenePresets)
                ActionChip(
                  avatar: Icon(preset.iconData, size: 16),
                  label: Text(preset.name),
                  onPressed: () => _applyPreset(preset),
                ),
            ],
          ),
        ],
      ),
    );
  }

  void _applyPreset(_ScenePreset preset) {
    final devices = _devices ?? <Device>[];
    final modes = <String, DeviceSceneMode>{};

    for (final device in devices) {
      if (!device.isSceneTarget) continue;

      final template = preset.actionForDomain(device.domain);
      if (template == null) continue;

      var command = template.command;
      var parameters = Map<String, dynamic>.from(template.parameters);

      // Downgrade commands the device can't execute
      if ((command == 'set_level' || command == 'dim') && !device.hasDim) {
        command = 'on';
        parameters = {};
      } else if (command == 'set_position' &&
          (!device.hasPosition || device.type == 'blind_switch')) {
        final posValue = (template.parameters['position'] as num?) ?? 0;
        command = posValue > 50 ? 'on' : 'off';
        parameters = {};
      } else if (command == 'set_setpoint' && !device.hasTemperatureSet) {
        continue;
      }

      modes[device.id] = DeviceSceneMode(
        mode: command,
        parameters: parameters,
        fadeMs: template.fadeMs,
      );
    }

    setState(() {
      _deviceModes = modes;
      _nameController.text = preset.name;
      _icon = preset.icon;
      _colour = preset.colour;
      _category = preset.category;
    });
  }

  Color? _parseColour(String? hex) {
    if (hex == null || hex.isEmpty) return null;
    final cleaned = hex.replaceFirst('#', '');
    if (cleaned.length != 6) return null;
    final value = int.tryParse(cleaned, radix: 16);
    if (value == null) return null;
    return Color(0xFF000000 | value);
  }

  String _domainLabel(String domain) {
    switch (domain) {
      case 'lighting':
        return 'Lighting';
      case 'blinds':
        return 'Blinds';
      case 'climate':
        return 'Climate';
      case 'security':
        return 'Security';
      case 'audio':
        return 'Audio';
      default:
        return domain[0].toUpperCase() + domain.substring(1);
    }
  }

  static const _iconOptions = <String, IconData>{
    'movie': Icons.movie,
    'reading': Icons.menu_book,
    'bright': Icons.wb_sunny,
    'relax': Icons.spa,
    'night': Icons.nightlight_round,
    'off': Icons.power_settings_new,
    'morning': Icons.wb_twilight,
    'evening': Icons.nights_stay,
    'party': Icons.celebration,
    'dinner': Icons.restaurant,
    'welcome': Icons.waving_hand,
  };
}

/// A quick-create preset that pre-fills the scene editor.
class _ScenePreset {
  final String name;
  final String icon;
  final String colour;
  final String category;
  final IconData iconData;
  final Map<String, _PresetAction> domainActions;

  const _ScenePreset({
    required this.name,
    required this.icon,
    required this.colour,
    required this.category,
    required this.iconData,
    required this.domainActions,
  });

  _PresetAction? actionForDomain(String domain) => domainActions[domain];
}

class _PresetAction {
  final String command;
  final Map<String, dynamic> parameters;
  final int fadeMs;

  const _PresetAction({
    required this.command,
    this.parameters = const {},
    this.fadeMs = 0,
  });
}

const _scenePresets = <_ScenePreset>[
  _ScenePreset(
    name: 'Movie',
    icon: 'movie',
    colour: '#7B1FA2',
    category: 'media',
    iconData: Icons.movie,
    domainActions: {
      'lighting': _PresetAction(command: 'off', fadeMs: 2000),
      'blinds': _PresetAction(
        command: 'set_position',
        parameters: {'position': 0},
      ),
    },
  ),
  _ScenePreset(
    name: 'Reading',
    icon: 'reading',
    colour: '#FFA726',
    category: 'comfort',
    iconData: Icons.menu_book,
    domainActions: {
      'lighting': _PresetAction(
        command: 'set_level',
        parameters: {'level': 80},
        fadeMs: 1000,
      ),
    },
  ),
  _ScenePreset(
    name: 'Night',
    icon: 'night',
    colour: '#1A237E',
    category: 'comfort',
    iconData: Icons.nightlight_round,
    domainActions: {
      'lighting': _PresetAction(command: 'off', fadeMs: 3000),
      'blinds': _PresetAction(
        command: 'set_position',
        parameters: {'position': 0},
      ),
    },
  ),
  _ScenePreset(
    name: 'Morning',
    icon: 'morning',
    colour: '#FFD54F',
    category: 'comfort',
    iconData: Icons.wb_twilight,
    domainActions: {
      'lighting': _PresetAction(
        command: 'set_level',
        parameters: {'level': 100},
        fadeMs: 2000,
      ),
      'blinds': _PresetAction(
        command: 'set_position',
        parameters: {'position': 100},
      ),
    },
  ),
  _ScenePreset(
    name: 'Relax',
    icon: 'relax',
    colour: '#26A69A',
    category: 'comfort',
    iconData: Icons.spa,
    domainActions: {
      'lighting': _PresetAction(
        command: 'set_level',
        parameters: {'level': 40},
        fadeMs: 2000,
      ),
    },
  ),
];
