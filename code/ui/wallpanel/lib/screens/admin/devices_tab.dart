import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../models/device.dart';
import '../../providers/auth_provider.dart';
import '../../providers/location_provider.dart';

/// Device management provider - fetches all devices.
final allDevicesProvider = FutureProvider.autoDispose<List<Device>>((ref) async {
  final apiClient = ref.watch(apiClientProvider);
  final response = await apiClient.getDevices();
  return response.devices;
});

/// Devices management tab.
///
/// Lists all devices with search, filter, edit, and delete capabilities.
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

    return Column(
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
                          onPressed: () => setState(() => _searchQuery = ''),
                        )
                      : null,
                ),
                onChanged: (value) => setState(() => _searchQuery = value),
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
                      onSelected: () => setState(() => _domainFilter = null),
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
                      onSelected: () => setState(() => _domainFilter = 'blinds'),
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
                      onSelected: () => setState(() => _domainFilter = 'sensor'),
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
                onRefresh: () async => ref.invalidate(allDevicesProvider),
                child: ListView.builder(
                  padding: const EdgeInsets.only(bottom: 80),
                  itemCount: filtered.length,
                  itemBuilder: (context, index) {
                    return _DeviceTile(
                      device: filtered[index],
                      onEdit: () => _showEditDialog(filtered[index]),
                      onDelete: () => _showDeleteDialog(filtered[index]),
                    );
                  },
                ),
              );
            },
            loading: () => const Center(child: CircularProgressIndicator()),
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
                    onPressed: () => ref.invalidate(allDevicesProvider),
                    child: const Text('Retry'),
                  ),
                ],
              ),
            ),
          ),
        ),
      ],
    );
  }

  List<Device> _filterDevices(List<Device> devices) {
    return devices.where((d) {
      // Domain filter
      if (_domainFilter != null && d.domain != _domainFilter) {
        return false;
      }

      // Search filter (name or ID)
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
        content: Text('Are you sure you want to delete "${device.name}"?\n\nThis action cannot be undone.'),
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
          backgroundColor: _domainColor(device.domain).withValues(alpha: 0.15),
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
              icon: Icon(Icons.delete_outline, color: theme.colorScheme.error),
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

class _EditDeviceSheet extends ConsumerStatefulWidget {
  final Device device;

  const _EditDeviceSheet({required this.device});

  @override
  ConsumerState<_EditDeviceSheet> createState() => _EditDeviceSheetState();
}

class _EditDeviceSheetState extends ConsumerState<_EditDeviceSheet> {
  late TextEditingController _nameController;
  String? _selectedRoomId;
  bool _saving = false;

  @override
  void initState() {
    super.initState();
    _nameController = TextEditingController(text: widget.device.name);
    _selectedRoomId = widget.device.roomId;
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
    final rooms = locationData.valueOrNull?.rooms ?? [];

    return Padding(
      padding: EdgeInsets.only(
        left: 16,
        right: 16,
        top: 16,
        bottom: MediaQuery.of(context).viewInsets.bottom + 16,
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Header
          Row(
            children: [
              Text(
                'Edit Device',
                style: theme.textTheme.titleLarge,
              ),
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
                    size: 16, color: theme.colorScheme.onSurfaceVariant),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    'ID: ${widget.device.id} • Type: ${widget.device.type}',
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

          // Room dropdown
          DropdownButtonFormField<String?>(
            initialValue: _selectedRoomId,
            decoration: const InputDecoration(
              labelText: 'Room',
              border: OutlineInputBorder(),
            ),
            items: [
              const DropdownMenuItem(value: null, child: Text('No room')),
              ...rooms.map((room) => DropdownMenuItem(
                    value: room.id,
                    child: Text(room.name),
                  )),
            ],
            onChanged: (value) => setState(() => _selectedRoomId = value),
          ),
          const SizedBox(height: 16),

          // Addresses (read-only)
          if (widget.device.address.isNotEmpty) ...[
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
                widget.device.address.entries
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
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Text('Save Changes'),
            ),
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
