import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../models/area.dart';
import '../../models/room.dart';
import '../../providers/auth_provider.dart';
import '../../providers/location_provider.dart';

/// Area types available in the system.
const _areaTypes = ['floor', 'building', 'wing', 'zone', 'outdoor', 'utility'];

/// Room types available in the system.
const _roomTypes = [
  'living', 'bedroom', 'bathroom', 'kitchen', 'dining', 'hallway',
  'office', 'garage', 'utility', 'other',
];

/// Location management tab — view, create, edit, and delete areas and rooms
/// in a hierarchical tree layout.
class LocationsTab extends ConsumerStatefulWidget {
  const LocationsTab({super.key});

  @override
  ConsumerState<LocationsTab> createState() => _LocationsTabState();
}

class _LocationsTabState extends ConsumerState<LocationsTab> {
  bool _loading = true;
  String? _error;
  List<Area> _areas = [];
  List<Room> _rooms = [];

  @override
  void initState() {
    super.initState();
    _loadData();
  }

  Future<void> _loadData() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final api = ref.read(apiClientProvider);
      final areasResp = await api.getAreas();
      final roomsResp = await api.getRooms();
      if (mounted) {
        setState(() {
          _areas = areasResp.areas;
          _rooms = roomsResp.rooms;
          _loading = false;
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          _error = e.toString();
          _loading = false;
        });
      }
    }
  }

  List<Room> _roomsForArea(String areaId) {
    return _rooms.where((r) => r.areaId == areaId).toList()
      ..sort((a, b) => a.sortOrder.compareTo(b.sortOrder));
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    if (_loading) {
      return const Center(child: CircularProgressIndicator());
    }

    if (_error != null) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.error_outline, size: 48, color: theme.colorScheme.error),
            const SizedBox(height: 12),
            Text('Failed to load locations', style: theme.textTheme.titleMedium),
            const SizedBox(height: 4),
            Text(_error!, style: theme.textTheme.bodySmall),
            const SizedBox(height: 16),
            FilledButton.icon(
              onPressed: _loadData,
              icon: const Icon(Icons.refresh),
              label: const Text('Retry'),
            ),
          ],
        ),
      );
    }

    if (_areas.isEmpty) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.location_off_outlined, size: 64,
                color: theme.colorScheme.onSurfaceVariant),
            const SizedBox(height: 16),
            Text('No locations configured',
                style: theme.textTheme.bodyLarge?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant)),
            const SizedBox(height: 8),
            Text('Import from an ETS project file, or create areas and rooms manually.',
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant),
                textAlign: TextAlign.center),
            const SizedBox(height: 24),
            FilledButton.icon(
              onPressed: _showCreateAreaDialog,
              icon: const Icon(Icons.add),
              label: const Text('Create Area'),
            ),
          ],
        ),
      );
    }

    final sortedAreas = List<Area>.from(_areas)
      ..sort((a, b) => a.sortOrder.compareTo(b.sortOrder));

    return Column(
      children: [
        // Toolbar
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
          child: Row(
            children: [
              Text(
                '${_areas.length} area${_areas.length == 1 ? '' : 's'}, '
                '${_rooms.length} room${_rooms.length == 1 ? '' : 's'}',
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant),
              ),
              const Spacer(),
              FilledButton.tonalIcon(
                onPressed: _showCreateAreaDialog,
                icon: const Icon(Icons.add, size: 18),
                label: const Text('Add Area'),
              ),
            ],
          ),
        ),
        const Divider(height: 1),
        // Hierarchy list
        Expanded(
          child: RefreshIndicator(
            onRefresh: _loadData,
            child: ListView.builder(
              padding: const EdgeInsets.only(bottom: 80),
              itemCount: sortedAreas.length,
              itemBuilder: (context, index) {
                final area = sortedAreas[index];
                final rooms = _roomsForArea(area.id);
                return _AreaSection(
                  area: area,
                  rooms: rooms,
                  onEditArea: () => _showEditAreaDialog(area),
                  onDeleteArea: () => _confirmDeleteArea(area),
                  onAddRoom: () => _showCreateRoomDialog(area),
                  onEditRoom: (room) => _showEditRoomDialog(room),
                  onDeleteRoom: (room) => _confirmDeleteRoom(room),
                );
              },
            ),
          ),
        ),
      ],
    );
  }

  // --- Area dialogs ---

  Future<void> _showCreateAreaDialog() async {
    final result = await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      builder: (ctx) => _AreaFormSheet(
        siteId: 'site-001', // Default site ID
        onSave: (data) async {
          final api = ref.read(apiClientProvider);
          await api.createArea(data);
        },
      ),
    );
    if (result == true) {
      _loadData();
      _invalidateLocationProvider();
    }
  }

  Future<void> _showEditAreaDialog(Area area) async {
    final result = await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      builder: (ctx) => _AreaFormSheet(
        area: area,
        onSave: (data) async {
          final api = ref.read(apiClientProvider);
          await api.updateArea(area.id, data);
        },
      ),
    );
    if (result == true) {
      _loadData();
      _invalidateLocationProvider();
    }
  }

  Future<void> _confirmDeleteArea(Area area) async {
    final rooms = _roomsForArea(area.id);
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Delete Area'),
        content: rooms.isNotEmpty
            ? Text('Cannot delete "${area.name}" — it has ${rooms.length} room${rooms.length == 1 ? '' : 's'}. Delete the rooms first.')
            : Text('Delete "${area.name}"? This cannot be undone.'),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: Text(rooms.isNotEmpty ? 'OK' : 'Cancel'),
          ),
          if (rooms.isEmpty)
            FilledButton(
              onPressed: () => Navigator.of(ctx).pop(true),
              style: FilledButton.styleFrom(
                backgroundColor: Theme.of(ctx).colorScheme.error),
              child: const Text('Delete'),
            ),
        ],
      ),
    );

    if (confirmed == true && mounted) {
      try {
        final api = ref.read(apiClientProvider);
        await api.deleteArea(area.id);
        _loadData();
        _invalidateLocationProvider();
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Deleted area "${area.name}"')),
          );
        }
      } catch (e) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Failed to delete area: $e')),
          );
        }
      }
    }
  }

  // --- Room dialogs ---

  Future<void> _showCreateRoomDialog(Area parentArea) async {
    final result = await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      builder: (ctx) => _RoomFormSheet(
        areaId: parentArea.id,
        areas: _areas,
        onSave: (data) async {
          final api = ref.read(apiClientProvider);
          await api.createRoom(data);
        },
      ),
    );
    if (result == true) {
      _loadData();
      _invalidateLocationProvider();
    }
  }

  Future<void> _showEditRoomDialog(Room room) async {
    final result = await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      builder: (ctx) => _RoomFormSheet(
        room: room,
        areaId: room.areaId,
        areas: _areas,
        onSave: (data) async {
          final api = ref.read(apiClientProvider);
          await api.updateRoom(room.id, data);
        },
      ),
    );
    if (result == true) {
      _loadData();
      _invalidateLocationProvider();
    }
  }

  Future<void> _confirmDeleteRoom(Room room) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Delete Room'),
        content: Text('Delete "${room.name}"? Devices in this room will become unassigned.'),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            style: FilledButton.styleFrom(
              backgroundColor: Theme.of(ctx).colorScheme.error),
            child: const Text('Delete'),
          ),
        ],
      ),
    );

    if (confirmed == true && mounted) {
      try {
        final api = ref.read(apiClientProvider);
        await api.deleteRoom(room.id);
        _loadData();
        _invalidateLocationProvider();
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Deleted room "${room.name}"')),
          );
        }
      } catch (e) {
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Failed to delete room: $e')),
          );
        }
      }
    }
  }

  void _invalidateLocationProvider() {
    ref.invalidate(locationDataProvider);
  }
}

// ---------------------------------------------------------------------------
// Area expansion tile
// ---------------------------------------------------------------------------

class _AreaSection extends StatelessWidget {
  final Area area;
  final List<Room> rooms;
  final VoidCallback onEditArea;
  final VoidCallback onDeleteArea;
  final VoidCallback onAddRoom;
  final void Function(Room) onEditRoom;
  final void Function(Room) onDeleteRoom;

  const _AreaSection({
    required this.area,
    required this.rooms,
    required this.onEditArea,
    required this.onDeleteArea,
    required this.onAddRoom,
    required this.onEditRoom,
    required this.onDeleteRoom,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
      clipBehavior: Clip.antiAlias,
      child: ExpansionTile(
        initiallyExpanded: true,
        leading: CircleAvatar(
          backgroundColor: theme.colorScheme.primaryContainer,
          child: Icon(Icons.layers_outlined,
              color: theme.colorScheme.onPrimaryContainer, size: 20),
        ),
        title: Text(area.name, style: const TextStyle(fontWeight: FontWeight.w600)),
        subtitle: Text(
          '${_formatType(area.type)} • ${rooms.length} room${rooms.length == 1 ? '' : 's'}',
          style: theme.textTheme.bodySmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant),
        ),
        trailing: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            IconButton(
              icon: const Icon(Icons.edit_outlined, size: 20),
              onPressed: onEditArea,
              tooltip: 'Edit area',
            ),
            IconButton(
              icon: Icon(Icons.delete_outline, size: 20,
                  color: theme.colorScheme.error),
              onPressed: onDeleteArea,
              tooltip: 'Delete area',
            ),
          ],
        ),
        children: [
          ...rooms.map((room) => _RoomTile(
                room: room,
                onEdit: () => onEditRoom(room),
                onDelete: () => onDeleteRoom(room),
              )),
          // Add room button
          ListTile(
            leading: const SizedBox(width: 40),
            title: TextButton.icon(
              onPressed: onAddRoom,
              icon: const Icon(Icons.add, size: 18),
              label: const Text('Add Room'),
            ),
          ),
        ],
      ),
    );
  }

  String _formatType(String type) {
    if (type.isEmpty) return type;
    return type[0].toUpperCase() + type.substring(1);
  }
}

class _RoomTile extends StatelessWidget {
  final Room room;
  final VoidCallback onEdit;
  final VoidCallback onDelete;

  const _RoomTile({
    required this.room,
    required this.onEdit,
    required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return ListTile(
      contentPadding: const EdgeInsets.only(left: 72, right: 16),
      leading: Icon(_roomIcon(room.type),
          color: theme.colorScheme.onSurfaceVariant, size: 20),
      title: Text(room.name),
      subtitle: Text(
        _formatType(room.type),
        style: theme.textTheme.bodySmall?.copyWith(
          color: theme.colorScheme.onSurfaceVariant),
      ),
      trailing: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          IconButton(
            icon: const Icon(Icons.edit_outlined, size: 18),
            onPressed: onEdit,
            tooltip: 'Edit room',
          ),
          IconButton(
            icon: Icon(Icons.delete_outline, size: 18,
                color: theme.colorScheme.error),
            onPressed: onDelete,
            tooltip: 'Delete room',
          ),
        ],
      ),
    );
  }

  IconData _roomIcon(String type) {
    switch (type) {
      case 'living': return Icons.weekend_outlined;
      case 'bedroom': return Icons.bed_outlined;
      case 'bathroom': return Icons.bathtub_outlined;
      case 'kitchen': return Icons.kitchen_outlined;
      case 'dining': return Icons.dining_outlined;
      case 'hallway': return Icons.door_sliding_outlined;
      case 'office': return Icons.computer_outlined;
      case 'garage': return Icons.garage_outlined;
      case 'utility': return Icons.handyman_outlined;
      default: return Icons.room_outlined;
    }
  }

  String _formatType(String type) {
    if (type.isEmpty) return type;
    return type[0].toUpperCase() + type.substring(1);
  }
}

// ---------------------------------------------------------------------------
// Area form bottom sheet
// ---------------------------------------------------------------------------

class _AreaFormSheet extends StatefulWidget {
  final Area? area;
  final String? siteId;
  final Future<void> Function(Map<String, dynamic>) onSave;

  const _AreaFormSheet({this.area, this.siteId, required this.onSave});

  @override
  State<_AreaFormSheet> createState() => _AreaFormSheetState();
}

class _AreaFormSheetState extends State<_AreaFormSheet> {
  final _formKey = GlobalKey<FormState>();
  late TextEditingController _nameCtl;
  late TextEditingController _sortCtl;
  late String _type;
  bool _saving = false;

  bool get _isNew => widget.area == null;

  @override
  void initState() {
    super.initState();
    _nameCtl = TextEditingController(text: widget.area?.name ?? '');
    _sortCtl = TextEditingController(
        text: widget.area?.sortOrder.toString() ?? '0');
    _type = widget.area?.type ?? 'floor';
  }

  @override
  void dispose() {
    _nameCtl.dispose();
    _sortCtl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Padding(
      padding: EdgeInsets.only(
        left: 16, right: 16, top: 16,
        bottom: MediaQuery.of(context).viewInsets.bottom + 16,
      ),
      child: Form(
        key: _formKey,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Text(_isNew ? 'Create Area' : 'Edit Area',
                    style: theme.textTheme.titleLarge),
                const Spacer(),
                IconButton(
                  icon: const Icon(Icons.close),
                  onPressed: () => Navigator.of(context).pop(false),
                ),
              ],
            ),
            const SizedBox(height: 16),
            TextFormField(
              controller: _nameCtl,
              decoration: const InputDecoration(
                labelText: 'Name',
                hintText: 'e.g. Ground Floor',
                border: OutlineInputBorder(),
              ),
              validator: (v) =>
                  (v == null || v.trim().isEmpty) ? 'Name is required' : null,
              textInputAction: TextInputAction.next,
            ),
            const SizedBox(height: 16),
            DropdownButtonFormField<String>(
              value: _type,
              decoration: const InputDecoration(
                labelText: 'Type',
                border: OutlineInputBorder(),
              ),
              items: _areaTypes.map((t) => DropdownMenuItem(
                    value: t,
                    child: Text(t[0].toUpperCase() + t.substring(1)),
                  )).toList(),
              onChanged: (v) {
                if (v != null) setState(() => _type = v);
              },
            ),
            const SizedBox(height: 16),
            TextFormField(
              controller: _sortCtl,
              decoration: const InputDecoration(
                labelText: 'Sort Order',
                border: OutlineInputBorder(),
              ),
              keyboardType: TextInputType.number,
              validator: (v) {
                if (v == null || v.trim().isEmpty) return null;
                if (int.tryParse(v.trim()) == null) return 'Must be a number';
                return null;
              },
              textInputAction: TextInputAction.done,
            ),
            const SizedBox(height: 16),
            SizedBox(
              width: double.infinity,
              child: FilledButton(
                onPressed: _saving ? null : _save,
                child: _saving
                    ? const SizedBox(
                        width: 20, height: 20,
                        child: CircularProgressIndicator(strokeWidth: 2))
                    : Text(_isNew ? 'Create' : 'Save'),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _save() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() => _saving = true);

    try {
      final name = _nameCtl.text.trim();
      final slug = name.toLowerCase().replaceAll(' ', '-').replaceAll(RegExp(r'[^a-z0-9-]'), '');
      final data = <String, dynamic>{
        'name': name,
        'type': _type,
        'sort_order': int.tryParse(_sortCtl.text.trim()) ?? 0,
      };
      if (_isNew) {
        data['id'] = slug;
        data['slug'] = slug;
        data['site_id'] = widget.siteId ?? 'site-001';
      }
      await widget.onSave(data);
      if (mounted) Navigator.of(context).pop(true);
    } catch (e) {
      if (mounted) {
        setState(() => _saving = false);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Failed: $e')),
        );
      }
    }
  }
}

// ---------------------------------------------------------------------------
// Room form bottom sheet
// ---------------------------------------------------------------------------

class _RoomFormSheet extends StatefulWidget {
  final Room? room;
  final String areaId;
  final List<Area> areas;
  final Future<void> Function(Map<String, dynamic>) onSave;

  const _RoomFormSheet({
    this.room,
    required this.areaId,
    required this.areas,
    required this.onSave,
  });

  @override
  State<_RoomFormSheet> createState() => _RoomFormSheetState();
}

class _RoomFormSheetState extends State<_RoomFormSheet> {
  final _formKey = GlobalKey<FormState>();
  late TextEditingController _nameCtl;
  late TextEditingController _sortCtl;
  late String _type;
  late String _areaId;
  bool _saving = false;

  bool get _isNew => widget.room == null;

  @override
  void initState() {
    super.initState();
    _nameCtl = TextEditingController(text: widget.room?.name ?? '');
    _sortCtl = TextEditingController(
        text: widget.room?.sortOrder.toString() ?? '0');
    _type = widget.room?.type ?? 'other';
    _areaId = widget.room?.areaId ?? widget.areaId;
  }

  @override
  void dispose() {
    _nameCtl.dispose();
    _sortCtl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Padding(
      padding: EdgeInsets.only(
        left: 16, right: 16, top: 16,
        bottom: MediaQuery.of(context).viewInsets.bottom + 16,
      ),
      child: Form(
        key: _formKey,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Text(_isNew ? 'Create Room' : 'Edit Room',
                    style: theme.textTheme.titleLarge),
                const Spacer(),
                IconButton(
                  icon: const Icon(Icons.close),
                  onPressed: () => Navigator.of(context).pop(false),
                ),
              ],
            ),
            const SizedBox(height: 16),
            TextFormField(
              controller: _nameCtl,
              decoration: const InputDecoration(
                labelText: 'Name',
                hintText: 'e.g. Master Bedroom',
                border: OutlineInputBorder(),
              ),
              validator: (v) =>
                  (v == null || v.trim().isEmpty) ? 'Name is required' : null,
              textInputAction: TextInputAction.next,
            ),
            const SizedBox(height: 16),
            DropdownButtonFormField<String>(
              value: _areaId,
              decoration: const InputDecoration(
                labelText: 'Area',
                border: OutlineInputBorder(),
              ),
              items: widget.areas.map((a) => DropdownMenuItem(
                    value: a.id,
                    child: Text(a.name),
                  )).toList(),
              onChanged: (v) {
                if (v != null) setState(() => _areaId = v);
              },
            ),
            const SizedBox(height: 16),
            DropdownButtonFormField<String>(
              value: _roomTypes.contains(_type) ? _type : 'other',
              decoration: const InputDecoration(
                labelText: 'Type',
                border: OutlineInputBorder(),
              ),
              items: _roomTypes.map((t) => DropdownMenuItem(
                    value: t,
                    child: Text(t[0].toUpperCase() + t.substring(1)),
                  )).toList(),
              onChanged: (v) {
                if (v != null) setState(() => _type = v);
              },
            ),
            const SizedBox(height: 16),
            TextFormField(
              controller: _sortCtl,
              decoration: const InputDecoration(
                labelText: 'Sort Order',
                border: OutlineInputBorder(),
              ),
              keyboardType: TextInputType.number,
              validator: (v) {
                if (v == null || v.trim().isEmpty) return null;
                if (int.tryParse(v.trim()) == null) return 'Must be a number';
                return null;
              },
              textInputAction: TextInputAction.done,
            ),
            const SizedBox(height: 16),
            SizedBox(
              width: double.infinity,
              child: FilledButton(
                onPressed: _saving ? null : _save,
                child: _saving
                    ? const SizedBox(
                        width: 20, height: 20,
                        child: CircularProgressIndicator(strokeWidth: 2))
                    : Text(_isNew ? 'Create' : 'Save'),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _save() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() => _saving = true);

    try {
      final name = _nameCtl.text.trim();
      final slug = name.toLowerCase().replaceAll(' ', '-').replaceAll(RegExp(r'[^a-z0-9-]'), '');
      final data = <String, dynamic>{
        'name': name,
        'type': _type,
        'sort_order': int.tryParse(_sortCtl.text.trim()) ?? 0,
        'area_id': _areaId,
      };
      if (_isNew) {
        data['id'] = slug;
        data['slug'] = slug;
      }
      await widget.onSave(data);
      if (mounted) Navigator.of(context).pop(true);
    } catch (e) {
      if (mounted) {
        setState(() => _saving = false);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Failed: $e')),
        );
      }
    }
  }
}
