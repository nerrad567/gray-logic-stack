import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../config/device_types.dart';
import '../../models/device.dart';
import '../../providers/auth_provider.dart';
import '../../providers/discovery_provider.dart';
import '../../providers/location_provider.dart';

/// Device management provider - fetches all devices.
final allDevicesProvider = FutureProvider.autoDispose<List<Device>>((ref) async {
  final apiClient = ref.watch(apiClientProvider);
  final response = await apiClient.getDevices();
  return response.devices;
});

/// Devices management tab.
///
/// Lists all devices with search, filter, edit, delete, and add capabilities.
class DevicesTab extends ConsumerStatefulWidget {
  const DevicesTab({super.key});

  @override
  ConsumerState<DevicesTab> createState() => _DevicesTabState();
}

class _DevicesTabState extends ConsumerState<DevicesTab> {
  String _searchQuery = '';
  String? _domainFilter;

  @override
  Widget build(BuildContext context) {
    final devicesAsync = ref.watch(allDevicesProvider);
    final theme = Theme.of(context);

    return Stack(
      children: [
        Column(
          children: [
            // Search and filter bar
            Container(
              padding: const EdgeInsets.all(16),
              child: Column(
                children: [
                  // Search field
                  TextField(
                    decoration: InputDecoration(
                      hintText: 'Search devices...',
                      prefixIcon: const Icon(Icons.search),
                      border: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(12),
                      ),
                      contentPadding: const EdgeInsets.symmetric(
                        horizontal: 16,
                        vertical: 12,
                      ),
                      suffixIcon: _searchQuery.isNotEmpty
                          ? IconButton(
                              icon: const Icon(Icons.clear),
                              onPressed: () =>
                                  setState(() => _searchQuery = ''),
                            )
                          : null,
                    ),
                    onChanged: (value) =>
                        setState(() => _searchQuery = value),
                  ),
                  const SizedBox(height: 12),

                  // Domain filter chips
                  SingleChildScrollView(
                    scrollDirection: Axis.horizontal,
                    child: Row(
                      children: [
                        _FilterChip(
                          label: 'All',
                          selected: _domainFilter == null,
                          onSelected: () =>
                              setState(() => _domainFilter = null),
                        ),
                        const SizedBox(width: 8),
                        _FilterChip(
                          label: 'Lighting',
                          selected: _domainFilter == 'lighting',
                          onSelected: () =>
                              setState(() => _domainFilter = 'lighting'),
                          icon: Icons.lightbulb_outline,
                        ),
                        const SizedBox(width: 8),
                        _FilterChip(
                          label: 'Blinds',
                          selected: _domainFilter == 'blinds',
                          onSelected: () =>
                              setState(() => _domainFilter = 'blinds'),
                          icon: Icons.blinds_outlined,
                        ),
                        const SizedBox(width: 8),
                        _FilterChip(
                          label: 'Climate',
                          selected: _domainFilter == 'climate',
                          onSelected: () =>
                              setState(() => _domainFilter = 'climate'),
                          icon: Icons.thermostat_outlined,
                        ),
                        const SizedBox(width: 8),
                        _FilterChip(
                          label: 'Sensor',
                          selected: _domainFilter == 'sensor',
                          onSelected: () =>
                              setState(() => _domainFilter = 'sensor'),
                          icon: Icons.sensors_outlined,
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),

            const Divider(height: 1),

            // Device list
            Expanded(
              child: devicesAsync.when(
                data: (devices) {
                  final filtered = _filterDevices(devices);
                  if (filtered.isEmpty) {
                    return Center(
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Icon(
                            Icons.devices_outlined,
                            size: 64,
                            color: theme.colorScheme.onSurfaceVariant,
                          ),
                          const SizedBox(height: 16),
                          Text(
                            devices.isEmpty
                                ? 'No devices configured'
                                : 'No devices match your filters',
                            style: theme.textTheme.bodyLarge?.copyWith(
                              color: theme.colorScheme.onSurfaceVariant,
                            ),
                          ),
                        ],
                      ),
                    );
                  }

                  return RefreshIndicator(
                    onRefresh: () async =>
                        ref.invalidate(allDevicesProvider),
                    child: ListView.builder(
                      padding: const EdgeInsets.only(bottom: 80),
                      itemCount: filtered.length,
                      itemBuilder: (context, index) {
                        return _DeviceTile(
                          device: filtered[index],
                          onEdit: () => _showEditDialog(filtered[index]),
                          onDelete: () =>
                              _showDeleteDialog(filtered[index]),
                        );
                      },
                    ),
                  );
                },
                loading: () =>
                    const Center(child: CircularProgressIndicator()),
                error: (error, _) => Center(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(Icons.error_outline,
                          size: 64, color: theme.colorScheme.error),
                      const SizedBox(height: 16),
                      Text('Failed to load devices'),
                      const SizedBox(height: 8),
                      FilledButton(
                        onPressed: () =>
                            ref.invalidate(allDevicesProvider),
                        child: const Text('Retry'),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ],
        ),

        // FAB - Add Device
        Positioned(
          right: 16,
          bottom: 16,
          child: FloatingActionButton.extended(
            onPressed: _showAddDeviceSheet,
            icon: const Icon(Icons.add),
            label: const Text('Add Device'),
          ),
        ),
      ],
    );
  }

  List<Device> _filterDevices(List<Device> devices) {
    return devices.where((d) {
      if (_domainFilter != null && d.domain != _domainFilter) {
        return false;
      }
      if (_searchQuery.isNotEmpty) {
        final query = _searchQuery.toLowerCase();
        if (!d.name.toLowerCase().contains(query) &&
            !d.id.toLowerCase().contains(query)) {
          return false;
        }
      }
      return true;
    }).toList();
  }

  Future<void> _showAddDeviceSheet() async {
    final result = await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      builder: (context) => const _AddDeviceSheet(),
    );

    if (result == true) {
      ref.invalidate(allDevicesProvider);
    }
  }

  Future<void> _showEditDialog(Device device) async {
    final result = await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      builder: (context) => _EditDeviceSheet(device: device),
    );

    if (result == true) {
      ref.invalidate(allDevicesProvider);
    }
  }

  Future<void> _showDeleteDialog(Device device) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Delete Device'),
        content: Text(
            'Are you sure you want to delete "${device.name}"?\n\nThis action cannot be undone.'),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.of(context).pop(true),
            style: FilledButton.styleFrom(
              backgroundColor: Theme.of(context).colorScheme.error,
            ),
            child: const Text('Delete'),
          ),
        ],
      ),
    );

    if (confirmed == true && mounted) {
      try {
        final apiClient = ref.read(apiClientProvider);
        await apiClient.deleteDevice(device.id);
        ref.invalidate(allDevicesProvider);

        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Deleted "${device.name}"')),
          );
        }
      } catch (e) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Failed to delete: $e')),
          );
        }
      }
    }
  }
}

// ---------------------------------------------------------------------------
// Filter chip
// ---------------------------------------------------------------------------

class _FilterChip extends StatelessWidget {
  final String label;
  final bool selected;
  final VoidCallback onSelected;
  final IconData? icon;

  const _FilterChip({
    required this.label,
    required this.selected,
    required this.onSelected,
    this.icon,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return FilterChip(
      label: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (icon != null) ...[
            Icon(icon, size: 16),
            const SizedBox(width: 4),
          ],
          Text(label),
        ],
      ),
      selected: selected,
      onSelected: (_) => onSelected(),
      selectedColor: theme.colorScheme.primaryContainer,
      checkmarkColor: theme.colorScheme.onPrimaryContainer,
    );
  }
}

// ---------------------------------------------------------------------------
// Device list tile
// ---------------------------------------------------------------------------

class _DeviceTile extends StatelessWidget {
  final Device device;
  final VoidCallback onEdit;
  final VoidCallback onDelete;

  const _DeviceTile({
    required this.device,
    required this.onEdit,
    required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
      child: ListTile(
        leading: CircleAvatar(
          backgroundColor:
              _domainColor(device.domain).withValues(alpha: 0.15),
          child: Icon(
            _domainIcon(device.domain),
            color: _domainColor(device.domain),
          ),
        ),
        title: Text(device.name),
        subtitle: Text(
          '${_formatType(device.type)} • ${device.domain}',
          style: theme.textTheme.bodySmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
        trailing: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            IconButton(
              icon: const Icon(Icons.edit_outlined),
              onPressed: onEdit,
              tooltip: 'Edit',
            ),
            IconButton(
              icon: Icon(Icons.delete_outline,
                  color: theme.colorScheme.error),
              onPressed: onDelete,
              tooltip: 'Delete',
            ),
          ],
        ),
      ),
    );
  }

  IconData _domainIcon(String domain) {
    switch (domain.toLowerCase()) {
      case 'lighting':
        return Icons.lightbulb_outline;
      case 'blinds':
        return Icons.blinds_outlined;
      case 'climate':
        return Icons.thermostat_outlined;
      case 'sensor':
        return Icons.sensors_outlined;
      case 'audio':
        return Icons.speaker_outlined;
      case 'video':
        return Icons.tv_outlined;
      case 'security':
        return Icons.security_outlined;
      default:
        return Icons.devices_outlined;
    }
  }

  Color _domainColor(String domain) {
    switch (domain.toLowerCase()) {
      case 'lighting':
        return Colors.amber;
      case 'blinds':
        return Colors.blue;
      case 'climate':
        return Colors.teal;
      case 'sensor':
        return Colors.purple;
      default:
        return Colors.grey;
    }
  }

  String _formatType(String type) {
    return type.replaceAll('_', ' ').split(' ').map((w) {
      if (w.isEmpty) return w;
      return w[0].toUpperCase() + w.substring(1);
    }).join(' ');
  }
}

// ---------------------------------------------------------------------------
// Add Device sheet
// ---------------------------------------------------------------------------

/// An address row in the form: function key + GA value.
class _AddressRow {
  String functionKey;
  final TextEditingController gaController;

  _AddressRow({required this.functionKey, String ga = ''})
      : gaController = TextEditingController(text: ga);

  void dispose() => gaController.dispose();
}

class _AddDeviceSheet extends ConsumerStatefulWidget {
  const _AddDeviceSheet();

  @override
  ConsumerState<_AddDeviceSheet> createState() => _AddDeviceSheetState();
}

class _AddDeviceSheetState extends ConsumerState<_AddDeviceSheet> {
  final _formKey = GlobalKey<FormState>();
  final _nameController = TextEditingController();
  final _manufacturerController = TextEditingController();
  final _modelController = TextEditingController();

  DeviceTypeInfo? _selectedType;
  String _domain = '';
  String? _selectedAreaId;
  String? _selectedRoomId;
  List<String> _capabilities = [];
  final List<_AddressRow> _addressRows = [];
  bool _saving = false;

  @override
  void dispose() {
    _nameController.dispose();
    _manufacturerController.dispose();
    _modelController.dispose();
    for (final row in _addressRows) {
      row.dispose();
    }
    super.dispose();
  }

  /// When the user picks a device type, auto-fill domain, capabilities,
  /// and set up the default address function rows.
  void _onTypeSelected(DeviceTypeInfo info) {
    // Dispose old address rows.
    for (final row in _addressRows) {
      row.dispose();
    }

    setState(() {
      _selectedType = info;
      _domain = info.domain;
      _capabilities = List<String>.from(info.defaultCapabilities);
      _addressRows.clear();
      for (final fn in info.addressFunctions) {
        _addressRows.add(_AddressRow(functionKey: fn.key));
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final locationData = ref.watch(locationDataProvider);
    final areas = locationData.value?.areas ?? [];
    final allRooms = locationData.value?.rooms ?? [];
    final suggestionsAsync = ref.watch(gaSuggestionsProvider);

    // Filter rooms by selected area.
    final rooms = _selectedAreaId != null
        ? allRooms.where((r) => r.areaId == _selectedAreaId).toList()
        : allRooms;

    return DraggableScrollableSheet(
      initialChildSize: 0.9,
      minChildSize: 0.5,
      maxChildSize: 0.95,
      expand: false,
      builder: (context, scrollController) {
        return Padding(
          padding: EdgeInsets.only(
            left: 16,
            right: 16,
            top: 8,
            bottom: MediaQuery.of(context).viewInsets.bottom + 16,
          ),
          child: Form(
            key: _formKey,
            child: ListView(
              controller: scrollController,
              children: [
                // Drag handle
                Center(
                  child: Container(
                    width: 40,
                    height: 4,
                    margin: const EdgeInsets.only(bottom: 16),
                    decoration: BoxDecoration(
                      color: theme.colorScheme.onSurfaceVariant
                          .withValues(alpha: 0.3),
                      borderRadius: BorderRadius.circular(2),
                    ),
                  ),
                ),

                // Header
                Row(
                  children: [
                    Text('Add Device',
                        style: theme.textTheme.titleLarge),
                    const Spacer(),
                    IconButton(
                      icon: const Icon(Icons.close),
                      onPressed: () => Navigator.of(context).pop(false),
                    ),
                  ],
                ),
                const SizedBox(height: 16),

                // ── Section 1: Basic Info ──
                _sectionHeader(theme, 'Basic Info'),
                const SizedBox(height: 12),

                // Name
                TextFormField(
                  controller: _nameController,
                  decoration: const InputDecoration(
                    labelText: 'Device Name *',
                    hintText: 'e.g. Living Room Downlights',
                    border: OutlineInputBorder(),
                  ),
                  validator: (v) => (v == null || v.trim().isEmpty)
                      ? 'Name is required'
                      : null,
                  textInputAction: TextInputAction.next,
                ),
                const SizedBox(height: 16),

                // Type (grouped by category)
                _DeviceTypePicker(
                  selectedType: _selectedType,
                  onSelected: _onTypeSelected,
                ),
                const SizedBox(height: 16),

                // Domain (auto-filled, editable)
                TextFormField(
                  key: ValueKey('domain_$_domain'),
                  initialValue: _domain,
                  decoration: const InputDecoration(
                    labelText: 'Domain *',
                    hintText: 'e.g. lighting',
                    border: OutlineInputBorder(),
                  ),
                  validator: (v) => (v == null || v.trim().isEmpty)
                      ? 'Domain is required'
                      : null,
                  onChanged: (v) => _domain = v.trim(),
                ),
                const SizedBox(height: 24),

                // ── Section 2: Location ──
                _sectionHeader(theme, 'Location'),
                const SizedBox(height: 12),

                // Area dropdown
                DropdownButtonFormField<String?>(
                  value: _selectedAreaId,
                  decoration: const InputDecoration(
                    labelText: 'Area',
                    border: OutlineInputBorder(),
                  ),
                  items: [
                    const DropdownMenuItem(
                        value: null, child: Text('No area')),
                    ...areas.map((a) => DropdownMenuItem(
                          value: a.id,
                          child: Text(a.name),
                        )),
                  ],
                  onChanged: (v) => setState(() {
                    _selectedAreaId = v;
                    // Reset room if area changed and current room doesn't
                    // belong to new area.
                    if (v != null &&
                        _selectedRoomId != null &&
                        !allRooms.any((r) =>
                            r.id == _selectedRoomId &&
                            r.areaId == v)) {
                      _selectedRoomId = null;
                    }
                  }),
                ),
                const SizedBox(height: 16),

                // Room dropdown (filtered by area)
                DropdownButtonFormField<String?>(
                  key: ValueKey('room_$_selectedAreaId'),
                  value: _selectedRoomId,
                  decoration: const InputDecoration(
                    labelText: 'Room',
                    border: OutlineInputBorder(),
                  ),
                  items: [
                    const DropdownMenuItem(
                        value: null, child: Text('No room')),
                    ...rooms.map((r) => DropdownMenuItem(
                          value: r.id,
                          child: Text(r.name),
                        )),
                  ],
                  onChanged: (v) => setState(() => _selectedRoomId = v),
                ),
                const SizedBox(height: 24),

                // ── Section 3: KNX Addresses ──
                _sectionHeader(theme, 'KNX Addresses'),
                const SizedBox(height: 8),

                if (_selectedType == null)
                  Padding(
                    padding: const EdgeInsets.symmetric(vertical: 8),
                    child: Text(
                      'Select a device type above to configure addresses.',
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  )
                else ...[
                  // Address rows
                  ...List.generate(_addressRows.length, (i) {
                    final row = _addressRows[i];
                    final fnInfo = _selectedType!.addressFunctions
                        .where((f) => f.key == row.functionKey)
                        .firstOrNull;
                    final isRequired = fnInfo?.required ?? false;

                    return Padding(
                      padding: const EdgeInsets.only(bottom: 12),
                      child: Row(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          // Function label
                          SizedBox(
                            width: 120,
                            child: _AddressFunctionDropdown(
                              value: row.functionKey,
                              deviceType: _selectedType!,
                              usedKeys: _addressRows
                                  .where((r) => r != row)
                                  .map((r) => r.functionKey)
                                  .toSet(),
                              onChanged: (v) =>
                                  setState(() => row.functionKey = v),
                            ),
                          ),
                          const SizedBox(width: 8),

                          // GA field with autocomplete
                          Expanded(
                            child: _GAAutocompleteField(
                              controller: row.gaController,
                              suggestions: suggestionsAsync.value ?? [],
                              label:
                                  isRequired ? 'GA *' : 'GA',
                              validator: isRequired
                                  ? (v) => (v == null || v.trim().isEmpty)
                                      ? 'Required'
                                      : null
                                  : null,
                            ),
                          ),

                          // Remove button (only for non-default rows)
                          if (i >= _selectedType!.addressFunctions.length)
                            IconButton(
                              icon: Icon(Icons.remove_circle_outline,
                                  color: theme.colorScheme.error, size: 20),
                              onPressed: () {
                                _addressRows[i].dispose();
                                setState(() => _addressRows.removeAt(i));
                              },
                            ),
                        ],
                      ),
                    );
                  }),

                  // Add extra address row
                  Align(
                    alignment: Alignment.centerLeft,
                    child: TextButton.icon(
                      onPressed: () => setState(() {
                        _addressRows.add(
                            _AddressRow(functionKey: 'custom'));
                      }),
                      icon: const Icon(Icons.add, size: 18),
                      label: const Text('Add Address'),
                    ),
                  ),
                ],
                const SizedBox(height: 24),

                // ── Section 4: Optional ──
                _sectionHeader(theme, 'Optional'),
                const SizedBox(height: 12),

                TextFormField(
                  controller: _manufacturerController,
                  decoration: const InputDecoration(
                    labelText: 'Manufacturer',
                    hintText: 'e.g. ABB, MDT, Theben',
                    border: OutlineInputBorder(),
                  ),
                  textInputAction: TextInputAction.next,
                ),
                const SizedBox(height: 16),

                TextFormField(
                  controller: _modelController,
                  decoration: const InputDecoration(
                    labelText: 'Model',
                    hintText: 'e.g. SA/S 8.16.5.1',
                    border: OutlineInputBorder(),
                  ),
                  textInputAction: TextInputAction.done,
                ),
                const SizedBox(height: 16),

                // Capabilities chips
                if (_capabilities.isNotEmpty) ...[
                  Text('Capabilities',
                      style: theme.textTheme.labelMedium?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      )),
                  const SizedBox(height: 8),
                  Wrap(
                    spacing: 6,
                    runSpacing: 6,
                    children: _capabilities
                        .map((cap) => Chip(
                              label: Text(cap,
                                  style: const TextStyle(fontSize: 12)),
                              deleteIcon:
                                  const Icon(Icons.close, size: 16),
                              onDeleted: () => setState(
                                  () => _capabilities.remove(cap)),
                              materialTapTargetSize:
                                  MaterialTapTargetSize.shrinkWrap,
                              visualDensity: VisualDensity.compact,
                            ))
                        .toList(),
                  ),
                  const SizedBox(height: 16),
                ],

                // Submit
                SizedBox(
                  width: double.infinity,
                  child: FilledButton.icon(
                    onPressed: _saving ? null : _submit,
                    icon: _saving
                        ? const SizedBox(
                            width: 20,
                            height: 20,
                            child: CircularProgressIndicator(
                                strokeWidth: 2),
                          )
                        : const Icon(Icons.check),
                    label: Text(_saving ? 'Creating...' : 'Create Device'),
                  ),
                ),
                const SizedBox(height: 16),
              ],
            ),
          ),
        );
      },
    );
  }

  Widget _sectionHeader(ThemeData theme, String title) {
    return Text(
      title,
      style: theme.textTheme.titleSmall?.copyWith(
        color: theme.colorScheme.primary,
        fontWeight: FontWeight.w600,
      ),
    );
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;
    if (_selectedType == null) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Please select a device type')),
      );
      return;
    }

    setState(() => _saving = true);

    try {
      final name = _nameController.text.trim();
      final slug = name
          .toLowerCase()
          .replaceAll(' ', '-')
          .replaceAll(RegExp(r'[^a-z0-9-]'), '');

      // Build structured functions map from rows (skip empty GAs).
      final functions = <String, Map<String, dynamic>>{};
      for (final row in _addressRows) {
        final ga = row.gaController.text.trim();
        if (ga.isNotEmpty) {
          // Look up DPT and flags from the device type metadata.
          final fnInfo = _selectedType!.addressFunctions
              .where((f) => f.key == row.functionKey)
              .firstOrNull;
          functions[row.functionKey] = {
            'ga': ga,
            'dpt': fnInfo?.dpt ?? '',
            'flags': fnInfo?.flags ?? <String>[],
          };
        }
      }

      // Require at least one function.
      if (functions.isEmpty) {
        setState(() => _saving = false);
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(
                content: Text('At least one GA address is required')),
          );
        }
        return;
      }

      final data = <String, dynamic>{
        'name': name,
        'slug': slug,
        'type': _selectedType!.type,
        'domain': _domain,
        'protocol': 'knx',
        'address': {
          'functions': functions,
        },
        'capabilities': _capabilities,
      };

      if (_selectedAreaId != null) data['area_id'] = _selectedAreaId;
      if (_selectedRoomId != null) data['room_id'] = _selectedRoomId;

      final manufacturer = _manufacturerController.text.trim();
      final model = _modelController.text.trim();
      if (manufacturer.isNotEmpty) data['manufacturer'] = manufacturer;
      if (model.isNotEmpty) data['model'] = model;

      final api = ref.read(apiClientProvider);
      await api.createDevice(data);

      if (mounted) {
        Navigator.of(context).pop(true);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Created "$name"')),
        );
      }
    } catch (e) {
      if (mounted) {
        setState(() => _saving = false);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Failed to create device: $e')),
        );
      }
    }
  }
}

// ---------------------------------------------------------------------------
// Device type picker (grouped by category)
// ---------------------------------------------------------------------------

class _DeviceTypePicker extends StatelessWidget {
  final DeviceTypeInfo? selectedType;
  final ValueChanged<DeviceTypeInfo> onSelected;

  const _DeviceTypePicker({
    required this.selectedType,
    required this.onSelected,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return InkWell(
      onTap: () => _showTypePicker(context),
      borderRadius: BorderRadius.circular(4),
      child: InputDecorator(
        decoration: const InputDecoration(
          labelText: 'Device Type *',
          border: OutlineInputBorder(),
        ),
        child: Row(
          children: [
            Expanded(
              child: Text(
                selectedType?.label ?? 'Select a type...',
                style: selectedType == null
                    ? theme.textTheme.bodyLarge
                        ?.copyWith(color: theme.hintColor)
                    : theme.textTheme.bodyLarge,
              ),
            ),
            if (selectedType != null)
              Chip(
                label: Text(selectedType!.category,
                    style: const TextStyle(fontSize: 11)),
                materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
                visualDensity: VisualDensity.compact,
              ),
            const SizedBox(width: 4),
            const Icon(Icons.arrow_drop_down),
          ],
        ),
      ),
    );
  }

  Future<void> _showTypePicker(BuildContext context) async {
    final categories = getDeviceCategories();

    final result = await showModalBottomSheet<DeviceTypeInfo>(
      context: context,
      isScrollControlled: true,
      builder: (ctx) {
        String search = '';

        return StatefulBuilder(
          builder: (ctx, setModalState) {
            return DraggableScrollableSheet(
              initialChildSize: 0.7,
              minChildSize: 0.4,
              maxChildSize: 0.9,
              expand: false,
              builder: (ctx, scrollCtl) {
                return Column(
                  children: [
                    Padding(
                      padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
                      child: TextField(
                        decoration: InputDecoration(
                          hintText: 'Search device types...',
                          prefixIcon: const Icon(Icons.search),
                          border: OutlineInputBorder(
                            borderRadius: BorderRadius.circular(12)),
                          contentPadding: const EdgeInsets.symmetric(
                              horizontal: 16, vertical: 12),
                        ),
                        onChanged: (v) =>
                            setModalState(() => search = v.toLowerCase()),
                      ),
                    ),
                    Expanded(
                      child: ListView(
                        controller: scrollCtl,
                        children: _buildTypePickerItems(
                          categories, search, ctx),
                      ),
                    ),
                  ],
                );
              },
            );
          },
        );
      },
    );

    if (result != null) {
      onSelected(result);
    }
  }

  List<Widget> _buildTypePickerItems(
      List<String> categories, String search, BuildContext ctx) {
    final widgets = <Widget>[];
    for (final cat in categories) {
      final types = getDeviceTypesForCategory(cat).where((t) =>
          search.isEmpty ||
          t.label.toLowerCase().contains(search) ||
          t.type.toLowerCase().contains(search));
      if (types.isEmpty) continue;

      widgets.add(Padding(
        padding: const EdgeInsets.fromLTRB(16, 16, 16, 4),
        child: Text(cat,
            style: Theme.of(ctx).textTheme.labelLarge?.copyWith(
                  color: Theme.of(ctx).colorScheme.primary,
                )),
      ));
      for (final t in types) {
        widgets.add(ListTile(
          title: Text(t.label),
          subtitle:
              Text(t.domain, style: Theme.of(ctx).textTheme.bodySmall),
          trailing: selectedType?.type == t.type
              ? Icon(Icons.check,
                  color: Theme.of(ctx).colorScheme.primary)
              : null,
          onTap: () => Navigator.of(ctx).pop(t),
        ));
      }
    }
    return widgets;
  }
}

// ---------------------------------------------------------------------------
// Address function dropdown
// ---------------------------------------------------------------------------

class _AddressFunctionDropdown extends StatelessWidget {
  final String value;
  final DeviceTypeInfo deviceType;
  final Set<String> usedKeys;
  final ValueChanged<String> onChanged;

  const _AddressFunctionDropdown({
    required this.value,
    required this.deviceType,
    required this.usedKeys,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    // Build items: the type's known functions + a "custom" option.
    final items = <DropdownMenuItem<String>>[];
    for (final fn in deviceType.addressFunctions) {
      items.add(DropdownMenuItem(
        value: fn.key,
        child: Text(fn.label, style: const TextStyle(fontSize: 12)),
      ));
    }
    items.add(const DropdownMenuItem(
      value: 'custom',
      child: Text('Custom', style: TextStyle(fontSize: 12)),
    ));

    // If current value isn't in items, add it.
    if (!items.any((i) => i.value == value)) {
      items.add(DropdownMenuItem(
        value: value,
        child: Text(value, style: const TextStyle(fontSize: 12)),
      ));
    }

    return DropdownButtonFormField<String>(
      value: value,
      isDense: true,
      decoration: const InputDecoration(
        border: OutlineInputBorder(),
        contentPadding: EdgeInsets.symmetric(horizontal: 8, vertical: 10),
      ),
      items: items,
      onChanged: (v) {
        if (v != null) onChanged(v);
      },
    );
  }
}

// ---------------------------------------------------------------------------
// GA autocomplete text field
// ---------------------------------------------------------------------------

class _GAAutocompleteField extends StatelessWidget {
  final TextEditingController controller;
  final List<GASuggestion> suggestions;
  final String label;
  final FormFieldValidator<String>? validator;

  const _GAAutocompleteField({
    required this.controller,
    required this.suggestions,
    required this.label,
    this.validator,
  });

  @override
  Widget build(BuildContext context) {
    return Autocomplete<GASuggestion>(
      textEditingController: controller,
      fieldViewBuilder: (ctx, textCtl, focusNode, onSubmitted) {
        return TextFormField(
          controller: textCtl,
          focusNode: focusNode,
          decoration: InputDecoration(
            labelText: label,
            border: const OutlineInputBorder(),
            contentPadding:
                const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          ),
          validator: validator,
          onFieldSubmitted: (_) => onSubmitted(),
        );
      },
      optionsBuilder: (textEditingValue) {
        final query = textEditingValue.text.trim();
        if (query.isEmpty) {
          // Show top active unassigned GAs.
          return suggestions.where((s) => !s.isAssigned).take(10);
        }
        return suggestions
            .where((s) => s.groupAddress.contains(query))
            .take(15);
      },
      displayStringForOption: (s) => s.groupAddress,
      optionsViewBuilder: (ctx, onSelected, options) {
        return Align(
          alignment: Alignment.topLeft,
          child: Material(
            elevation: 4,
            borderRadius: BorderRadius.circular(8),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxHeight: 200, maxWidth: 320),
              child: ListView.builder(
                padding: EdgeInsets.zero,
                shrinkWrap: true,
                itemCount: options.length,
                itemBuilder: (ctx, i) {
                  final s = options.elementAt(i);
                  return ListTile(
                    dense: true,
                    leading: _statusDot(s),
                    title: Text(s.groupAddress,
                        style: TextStyle(
                          fontFamily: 'monospace',
                          color: s.isAssigned ? Colors.grey : null,
                        )),
                    subtitle: Text(
                      s.isAssigned
                          ? '${s.assignedDeviceName} (${s.assignedFunction})'
                          : s.isDiscovered
                              ? '${s.messageCount} msgs • ${s.lastSeenAgo}'
                              : 'Not seen on bus',
                      style: const TextStyle(fontSize: 11),
                    ),
                    onTap: () => onSelected(s),
                  );
                },
              ),
            ),
          ),
        );
      },
    );
  }

  Widget _statusDot(GASuggestion s) {
    Color color;
    if (s.isAssigned) {
      color = Colors.grey;
    } else if (s.isDiscovered) {
      color = Colors.green;
    } else {
      color = Colors.transparent;
    }
    return Container(
      width: 10,
      height: 10,
      decoration: BoxDecoration(
        shape: BoxShape.circle,
        color: color,
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Edit Device sheet (existing)
// ---------------------------------------------------------------------------

class _EditDeviceSheet extends ConsumerStatefulWidget {
  final Device device;

  const _EditDeviceSheet({required this.device});

  @override
  ConsumerState<_EditDeviceSheet> createState() => _EditDeviceSheetState();
}

class _EditDeviceSheetState extends ConsumerState<_EditDeviceSheet> {
  late TextEditingController _nameController;
  String? _selectedRoomId;
  String? _selectedAreaId;
  bool _saving = false;

  @override
  void initState() {
    super.initState();
    _nameController = TextEditingController(text: widget.device.name);
    _selectedRoomId = widget.device.roomId;
    _selectedAreaId = widget.device.areaId;
  }

  @override
  void dispose() {
    _nameController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final locationData = ref.watch(locationDataProvider);
    final rooms = locationData.value?.rooms ?? [];
    final areas = locationData.value?.areas ?? [];
    final dev = widget.device;

    return Padding(
      padding: EdgeInsets.only(
        left: 16,
        right: 16,
        top: 16,
        bottom: MediaQuery.of(context).viewInsets.bottom + 16,
      ),
      child: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Header
            Row(
              children: [
                Text('Edit Device', style: theme.textTheme.titleLarge),
                const Spacer(),
                IconButton(
                  icon: const Icon(Icons.close),
                  onPressed: () => Navigator.of(context).pop(false),
                ),
              ],
            ),
            const SizedBox(height: 16),

            // Device info (read-only)
            Container(
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: theme.colorScheme.surfaceContainerHighest,
                borderRadius: BorderRadius.circular(8),
              ),
              child: Row(
                children: [
                  Icon(Icons.info_outline,
                      size: 16,
                      color: theme.colorScheme.onSurfaceVariant),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      'ID: ${dev.id}',
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 16),

            // Name field
            TextField(
              controller: _nameController,
              decoration: const InputDecoration(
                labelText: 'Device Name',
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 16),

            // Area dropdown
            DropdownButtonFormField<String?>(
              value: _selectedAreaId,
              decoration: const InputDecoration(
                labelText: 'Area',
                border: OutlineInputBorder(),
              ),
              items: [
                const DropdownMenuItem(
                    value: null, child: Text('No area')),
                ...areas.map((area) => DropdownMenuItem(
                      value: area.id,
                      child: Text(area.name),
                    )),
              ],
              onChanged: (value) =>
                  setState(() => _selectedAreaId = value),
            ),
            const SizedBox(height: 16),

            // Room dropdown
            DropdownButtonFormField<String?>(
              value: _selectedRoomId,
              decoration: const InputDecoration(
                labelText: 'Room',
                border: OutlineInputBorder(),
              ),
              items: [
                const DropdownMenuItem(
                    value: null, child: Text('No room')),
                ...rooms.map((room) => DropdownMenuItem(
                      value: room.id,
                      child: Text(room.name),
                    )),
              ],
              onChanged: (value) =>
                  setState(() => _selectedRoomId = value),
            ),
            const SizedBox(height: 16),

            // Metadata (read-only)
            Text(
              'Device Info',
              style: theme.textTheme.labelMedium?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
            const SizedBox(height: 8),
            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: theme.colorScheme.surfaceContainerHighest,
                borderRadius: BorderRadius.circular(8),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  _metaRow(theme, 'Type', _formatType(dev.type)),
                  _metaRow(theme, 'Domain', dev.domain),
                  _metaRow(theme, 'Protocol', dev.protocol),
                  _metaRow(theme, 'Health', dev.healthStatus),
                  if (dev.manufacturer != null)
                    _metaRow(theme, 'Manufacturer', dev.manufacturer!),
                  if (dev.model != null)
                    _metaRow(theme, 'Model', dev.model!),
                  if (dev.capabilities.isNotEmpty) ...[
                    const SizedBox(height: 4),
                    Wrap(
                      spacing: 4,
                      runSpacing: 4,
                      children: dev.capabilities
                          .map((cap) => Chip(
                                label: Text(cap,
                                    style:
                                        const TextStyle(fontSize: 11)),
                                materialTapTargetSize:
                                    MaterialTapTargetSize.shrinkWrap,
                                visualDensity: VisualDensity.compact,
                              ))
                          .toList(),
                    ),
                  ],
                ],
              ),
            ),
            const SizedBox(height: 16),

            // Addresses (read-only)
            if (dev.address.isNotEmpty) ...[
              Text(
                'Addresses',
                style: theme.textTheme.labelMedium?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 8),
              Container(
                width: double.infinity,
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: theme.colorScheme.surfaceContainerHighest,
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Text(
                  dev.address.entries
                      .map((e) => '${e.key}: ${e.value}')
                      .join('\n'),
                  style: theme.textTheme.bodySmall?.copyWith(
                    fontFamily: 'monospace',
                  ),
                ),
              ),
              const SizedBox(height: 16),
            ],

            // Save button
            SizedBox(
              width: double.infinity,
              child: FilledButton(
                onPressed: _saving ? null : _save,
                child: _saving
                    ? const SizedBox(
                        width: 20,
                        height: 20,
                        child:
                            CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Text('Save Changes'),
              ),
            ),
          ],
        ),
      ),
    );
  }

  String _formatType(String type) {
    return type.replaceAll('_', ' ').split(' ').map((w) {
      if (w.isEmpty) return w;
      return w[0].toUpperCase() + w.substring(1);
    }).join(' ');
  }

  Widget _metaRow(ThemeData theme, String label, String value) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 2),
      child: Row(
        children: [
          SizedBox(
            width: 100,
            child: Text(label,
                style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant)),
          ),
          Expanded(
            child: Text(value,
                style: theme.textTheme.bodySmall
                    ?.copyWith(fontWeight: FontWeight.w500)),
          ),
        ],
      ),
    );
  }

  Future<void> _save() async {
    final name = _nameController.text.trim();
    if (name.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Name cannot be empty')),
      );
      return;
    }

    setState(() => _saving = true);

    try {
      final apiClient = ref.read(apiClientProvider);
      await apiClient.updateDevice(widget.device.id, {
        'name': name,
        'room_id': _selectedRoomId,
        'area_id': _selectedAreaId,
      });

      if (mounted) {
        Navigator.of(context).pop(true);
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Device updated')),
        );
      }
    } catch (e) {
      if (mounted) {
        setState(() => _saving = false);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Failed to update: $e')),
        );
      }
    }
  }
}
