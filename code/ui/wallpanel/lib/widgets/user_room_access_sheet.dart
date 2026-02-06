import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/user.dart';
import '../providers/location_provider.dart';
import '../providers/user_provider.dart';

/// Bottom sheet for managing a user's room access grants.
/// Uses a checklist of all rooms with a can_manage_scenes toggle per room.
class UserRoomAccessSheet extends ConsumerStatefulWidget {
  final User user;

  const UserRoomAccessSheet({super.key, required this.user});

  @override
  ConsumerState<UserRoomAccessSheet> createState() =>
      _UserRoomAccessSheetState();
}

class _UserRoomAccessSheetState extends ConsumerState<UserRoomAccessSheet> {
  final Map<String, bool> _accessMap = {}; // roomId -> granted
  final Map<String, bool> _manageMap = {}; // roomId -> canManageScenes
  bool _loading = true;
  bool _saving = false;

  @override
  void initState() {
    super.initState();
    _loadCurrentGrants();
  }

  Future<void> _loadCurrentGrants() async {
    try {
      final userRepo = ref.read(userRepositoryProvider);
      final grants = await userRepo.getUserRooms(widget.user.id);
      if (!mounted) return;
      setState(() {
        for (final g in grants) {
          _accessMap[g.roomId] = true;
          _manageMap[g.roomId] = g.canManageScenes;
        }
        _loading = false;
      });
    } catch (_) {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _save() async {
    setState(() => _saving = true);

    final rooms = <RoomAccessGrant>[];
    for (final entry in _accessMap.entries) {
      if (entry.value) {
        rooms.add(RoomAccessGrant(
          roomId: entry.key,
          canManageScenes: _manageMap[entry.key] ?? false,
        ));
      }
    }

    try {
      final userRepo = ref.read(userRepositoryProvider);
      await userRepo.setUserRooms(widget.user.id, rooms);

      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Room access updated'),
          behavior: SnackBarBehavior.floating,
        ),
      );
      Navigator.of(context).pop(true);
    } catch (e) {
      if (!mounted) return;
      setState(() => _saving = false);
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text('Failed to update room access: $e'),
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
                              Text('Room Access',
                                  style: theme.textTheme.titleLarge),
                              Text(
                                widget.user.displayName,
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
                          return ListView.builder(
                            controller: scrollController,
                            padding: const EdgeInsets.all(16),
                            itemCount: rooms.length,
                            itemBuilder: (context, index) {
                              final room = rooms[index];
                              final hasAccess =
                                  _accessMap[room.id] ?? false;
                              final canManage =
                                  _manageMap[room.id] ?? false;

                              return Card(
                                margin: const EdgeInsets.only(bottom: 8),
                                child: Padding(
                                  padding: const EdgeInsets.symmetric(
                                      vertical: 4, horizontal: 8),
                                  child: Row(
                                    children: [
                                      Checkbox(
                                        value: hasAccess,
                                        onChanged: (v) {
                                          setState(() {
                                            _accessMap[room.id] =
                                                v ?? false;
                                            if (v != true) {
                                              _manageMap[room.id] = false;
                                            }
                                          });
                                        },
                                      ),
                                      Expanded(
                                        child: Text(room.name),
                                      ),
                                      if (hasAccess) ...[
                                        Text(
                                          'Manage scenes',
                                          style: theme.textTheme.bodySmall
                                              ?.copyWith(
                                            color: theme.colorScheme
                                                .onSurfaceVariant,
                                          ),
                                        ),
                                        Switch(
                                          value: canManage,
                                          onChanged: (v) {
                                            setState(() =>
                                                _manageMap[room.id] = v);
                                          },
                                        ),
                                      ],
                                    ],
                                  ),
                                ),
                              );
                            },
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
