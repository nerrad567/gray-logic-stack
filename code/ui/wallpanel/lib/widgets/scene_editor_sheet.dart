import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/device.dart';
import '../models/scene.dart';
import '../providers/device_provider.dart';
import '../providers/location_provider.dart';
import '../providers/scene_provider.dart';
import 'scene_action_row.dart';

/// Bottom sheet for creating or editing a scene.
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
  late List<SceneActionData> _actions;
  bool _saving = false;

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
    _actions = s?.actions
            .map((a) => SceneActionData(
                  deviceId: a.deviceId,
                  command: a.command,
                  parameters: Map<String, dynamic>.from(a.parameters ?? {}),
                  delayMs: a.delayMs,
                  fadeMs: a.fadeMs,
                  parallel: a.parallel,
                  continueOnError: a.continueOnError,
                ))
            .toList() ??
        [];
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

  Future<void> _save() async {
    if (!_formKey.currentState!.validate()) return;
    if (_actions.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Add at least one action'),
          behavior: SnackBarBehavior.floating,
        ),
      );
      return;
    }

    setState(() => _saving = true);

    final actions = _actions.asMap().entries.map((e) => {
          'device_id': e.value.deviceId,
          'command': e.value.command,
          if (e.value.parameters.isNotEmpty) 'parameters': e.value.parameters,
          'delay_ms': e.value.delayMs,
          'fade_ms': e.value.fadeMs,
          'parallel': e.value.parallel,
          'continue_on_error': e.value.continueOnError,
          'sort_order': e.key,
        }).toList();

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
                    padding: const EdgeInsets.all(16),
                    children: [
                      // Name
                      TextFormField(
                        controller: _nameController,
                        decoration: const InputDecoration(
                          labelText: 'Name',
                          border: OutlineInputBorder(),
                        ),
                        validator: (v) =>
                            (v == null || v.trim().isEmpty) ? 'Required' : null,
                      ),
                      const SizedBox(height: 16),

                      // Description
                      TextFormField(
                        controller: _descController,
                        decoration: const InputDecoration(
                          labelText: 'Description (optional)',
                          border: OutlineInputBorder(),
                        ),
                        maxLines: 2,
                      ),
                      const SizedBox(height: 16),

                      // Room dropdown
                      locationAsync.when(
                        data: (data) => DropdownButtonFormField<String?>(
                          value: _roomId,
                          decoration: const InputDecoration(
                            labelText: 'Room (optional)',
                            border: OutlineInputBorder(),
                          ),
                          items: [
                            const DropdownMenuItem<String?>(
                              value: null,
                              child: Text('Global (no room)'),
                            ),
                            ...data.sortedRooms.map((room) =>
                                DropdownMenuItem<String?>(
                                  value: room.id,
                                  child: Text(room.name),
                                )),
                          ],
                          onChanged: (v) => setState(() => _roomId = v),
                        ),
                        loading: () => const LinearProgressIndicator(),
                        error: (_, _) => const Text('Failed to load rooms'),
                      ),
                      const SizedBox(height: 16),

                      // Icon + Colour + Category row
                      Row(
                        children: [
                          Expanded(
                            child: DropdownButtonFormField<String?>(
                              value: _icon,
                              decoration: const InputDecoration(
                                labelText: 'Icon',
                                border: OutlineInputBorder(),
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
                                        Icon(entry.value, size: 18),
                                        const SizedBox(width: 8),
                                        Text(entry.key),
                                      ],
                                    ),
                                  ),
                              ],
                              onChanged: (v) => setState(() => _icon = v),
                            ),
                          ),
                          const SizedBox(width: 12),
                          Expanded(
                            child: DropdownButtonFormField<String?>(
                              value: _category,
                              decoration: const InputDecoration(
                                labelText: 'Category',
                                border: OutlineInputBorder(),
                              ),
                              items: const [
                                DropdownMenuItem<String?>(
                                    value: null, child: Text('None')),
                                DropdownMenuItem(
                                    value: 'lighting', child: Text('Lighting')),
                                DropdownMenuItem(
                                    value: 'comfort', child: Text('Comfort')),
                                DropdownMenuItem(
                                    value: 'media', child: Text('Media')),
                                DropdownMenuItem(
                                    value: 'security', child: Text('Security')),
                                DropdownMenuItem(
                                    value: 'custom', child: Text('Custom')),
                              ],
                              onChanged: (v) =>
                                  setState(() => _category = v),
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 16),

                      // Colour picker (simple hex input)
                      TextFormField(
                        initialValue: _colour ?? '',
                        decoration: InputDecoration(
                          labelText: 'Colour (hex, e.g. #FF9800)',
                          border: const OutlineInputBorder(),
                          prefixIcon: _colour != null
                              ? Padding(
                                  padding: const EdgeInsets.all(12),
                                  child: Container(
                                    width: 20,
                                    height: 20,
                                    decoration: BoxDecoration(
                                      color: _parseColour(_colour),
                                      borderRadius: BorderRadius.circular(4),
                                      border: Border.all(
                                          color: theme.colorScheme.outline),
                                    ),
                                  ),
                                )
                              : null,
                        ),
                        onChanged: (v) {
                          final cleaned = v.trim();
                          setState(() => _colour =
                              cleaned.isEmpty ? null : cleaned);
                        },
                      ),
                      const SizedBox(height: 16),

                      // Enabled + Priority row
                      Row(
                        children: [
                          Expanded(
                            child: SwitchListTile(
                              title: const Text('Enabled'),
                              value: _enabled,
                              onChanged: (v) => setState(() => _enabled = v),
                              contentPadding: EdgeInsets.zero,
                            ),
                          ),
                          const SizedBox(width: 16),
                          SizedBox(
                            width: 100,
                            child: TextFormField(
                              initialValue: _priority.toString(),
                              decoration: const InputDecoration(
                                labelText: 'Priority',
                                border: OutlineInputBorder(),
                              ),
                              keyboardType: TextInputType.number,
                              onChanged: (v) {
                                final p = int.tryParse(v);
                                if (p != null) _priority = p;
                              },
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 24),

                      // Actions section
                      Row(
                        children: [
                          Text('Actions', style: theme.textTheme.titleMedium),
                          const Spacer(),
                          TextButton.icon(
                            icon: const Icon(Icons.add, size: 18),
                            label: const Text('Add Action'),
                            onPressed: _addAction,
                          ),
                        ],
                      ),
                      const SizedBox(height: 8),

                      if (_actions.isEmpty)
                        Container(
                          padding: const EdgeInsets.all(24),
                          decoration: BoxDecoration(
                            border: Border.all(
                                color: theme.colorScheme.outlineVariant),
                            borderRadius: BorderRadius.circular(12),
                          ),
                          child: Center(
                            child: Text(
                              'No actions yet. Add at least one action.',
                              style: theme.textTheme.bodySmall?.copyWith(
                                color: theme.colorScheme.onSurfaceVariant,
                              ),
                            ),
                          ),
                        )
                      else
                        ReorderableListView.builder(
                          shrinkWrap: true,
                          physics: const NeverScrollableScrollPhysics(),
                          itemCount: _actions.length,
                          onReorder: (oldIndex, newIndex) {
                            setState(() {
                              if (newIndex > oldIndex) newIndex--;
                              final item = _actions.removeAt(oldIndex);
                              _actions.insert(newIndex, item);
                            });
                          },
                          itemBuilder: (context, index) {
                            return SceneActionRow(
                              key: ValueKey(
                                  '${_actions[index].deviceId}_$index'),
                              action: _actions[index],
                              index: index,
                              roomId: _roomId,
                              onChanged: (updated) {
                                setState(() => _actions[index] = updated);
                              },
                              onDelete: () {
                                setState(() => _actions.removeAt(index));
                              },
                            );
                          },
                        ),

                      const SizedBox(height: 80),
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
    // Get room devices if available
    final devicesAsync = ref.read(roomDevicesProvider);
    final devices = devicesAsync.value ?? <Device>[];

    final actions = <SceneActionData>[];

    for (final device in devices) {
      final template = preset.actionForDomain(device.domain);
      if (template != null) {
        actions.add(SceneActionData(
          deviceId: device.id,
          command: template.command,
          parameters: Map<String, dynamic>.from(template.parameters),
          delayMs: 0,
          fadeMs: template.fadeMs,
          parallel: true,
          continueOnError: true,
        ));
      }
    }

    setState(() {
      _nameController.text = preset.name;
      _icon = preset.icon;
      _colour = preset.colour;
      _category = preset.category;
      _actions = actions;
    });
  }

  void _addAction() {
    setState(() {
      _actions.add(SceneActionData(
        deviceId: '',
        command: 'on',
        parameters: {},
        delayMs: 0,
        fadeMs: 0,
        parallel: false,
        continueOnError: false,
      ));
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

/// Mutable action data used in the editor.
class SceneActionData {
  String deviceId;
  String command;
  Map<String, dynamic> parameters;
  int delayMs;
  int fadeMs;
  bool parallel;
  bool continueOnError;

  SceneActionData({
    required this.deviceId,
    required this.command,
    required this.parameters,
    required this.delayMs,
    required this.fadeMs,
    required this.parallel,
    required this.continueOnError,
  });
}
