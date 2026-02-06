import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/panel.dart';
import '../providers/location_provider.dart';
import '../providers/panel_provider.dart';

/// Bottom sheet for managing a panel's room assignments.
/// Uses a checklist of all rooms with Select All / Clear All actions.
class PanelRoomsSheet extends ConsumerStatefulWidget {
  final Panel panel;

  const PanelRoomsSheet({super.key, required this.panel});

  @override
  ConsumerState<PanelRoomsSheet> createState() => _PanelRoomsSheetState();
}

class _PanelRoomsSheetState extends ConsumerState<PanelRoomsSheet> {
  final Map<String, bool> _accessMap = {}; // roomId -> assigned
  bool _loading = true;
  bool _saving = false;

  @override
  void initState() {
    super.initState();
    _loadCurrentRooms();
  }

  Future<void> _loadCurrentRooms() async {
    try {
      final panelRepo = ref.read(panelRepositoryProvider);
      final roomIds = await panelRepo.getPanelRooms(widget.panel.id);
      if (!mounted) return;
      setState(() {
        for (final id in roomIds) {
          _accessMap[id] = true;
        }
        _loading = false;
      });
    } catch (_) {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _save() async {
    setState(() => _saving = true);

    final roomIds = <String>[];
    for (final entry in _accessMap.entries) {
      if (entry.value) {
        roomIds.add(entry.key);
      }
    }

    try {
      final panelRepo = ref.read(panelRepositoryProvider);
      await panelRepo.setPanelRooms(widget.panel.id, roomIds);

      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Panel rooms updated'),
          behavior: SnackBarBehavior.floating,
        ),
      );
      Navigator.of(context).pop(true);
    } catch (e) {
      if (!mounted) return;
      setState(() => _saving = false);
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text('Failed to update rooms: $e'),
          behavior: SnackBarBehavior.floating,
          backgroundColor: Theme.of(context).colorScheme.error,
        ),
      );
    }
  }

  void _selectAll(List<String> allRoomIds) {
    setState(() {
      for (final id in allRoomIds) {
        _accessMap[id] = true;
      }
    });
  }

  void _clearAll() {
    setState(() {
      _accessMap.updateAll((_, __) => false);
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final locationAsync = ref.watch(locationDataProvider);

    return DraggableScrollableSheet(
      initialChildSize: 0.7,
      minChildSize: 0.4,
      maxChildSize: 0.9,
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
              // Handle + header
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
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text('Room Assignments',
                                  style: theme.textTheme.titleLarge),
                              Text(
                                widget.panel.name,
                                style: theme.textTheme.bodySmall?.copyWith(
                                  color: theme.colorScheme.onSurfaceVariant,
                                ),
                              ),
                            ],
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
                              : const Text('Save'),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
              const Divider(),
              // Room checklist
              Expanded(
                child: _loading
                    ? const Center(child: CircularProgressIndicator())
                    : locationAsync.when(
                        data: (data) {
                          final rooms = data.sortedRooms;
                          if (rooms.isEmpty) {
                            return const Center(
                              child: Text('No rooms configured yet'),
                            );
                          }
                          final allRoomIds =
                              rooms.map((r) => r.id).toList();
                          return Column(
                            children: [
                              // Select All / Clear All
                              Padding(
                                padding: const EdgeInsets.symmetric(
                                    horizontal: 16, vertical: 4),
                                child: Row(
                                  children: [
                                    TextButton(
                                      onPressed: () =>
                                          _selectAll(allRoomIds),
                                      child: const Text('Select All'),
                                    ),
                                    const SizedBox(width: 8),
                                    TextButton(
                                      onPressed: _clearAll,
                                      child: const Text('Clear All'),
                                    ),
                                    const Spacer(),
                                    Text(
                                      '${_accessMap.values.where((v) => v).length} selected',
                                      style: theme.textTheme.bodySmall
                                          ?.copyWith(
                                        color: theme
                                            .colorScheme.onSurfaceVariant,
                                      ),
                                    ),
                                  ],
                                ),
                              ),
                              Expanded(
                                child: ListView.builder(
                                  controller: scrollController,
                                  padding: const EdgeInsets.symmetric(
                                      horizontal: 16),
                                  itemCount: rooms.length,
                                  itemBuilder: (context, index) {
                                    final room = rooms[index];
                                    final assigned =
                                        _accessMap[room.id] ?? false;

                                    return Card(
                                      margin:
                                          const EdgeInsets.only(bottom: 8),
                                      child: CheckboxListTile(
                                        title: Text(room.name),
                                        value: assigned,
                                        onChanged: (v) {
                                          setState(() {
                                            _accessMap[room.id] =
                                                v ?? false;
                                          });
                                        },
                                      ),
                                    );
                                  },
                                ),
                              ),
                            ],
                          );
                        },
                        loading: () => const Center(
                            child: CircularProgressIndicator()),
                        error: (_, _) =>
                            const Center(child: Text('Failed to load rooms')),
                      ),
              ),
            ],
          ),
        );
      },
    );
  }
}
