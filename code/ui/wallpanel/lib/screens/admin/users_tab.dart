import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../models/user.dart';
import '../../providers/user_provider.dart';
import '../../widgets/user_editor_sheet.dart';
import '../../widgets/user_room_access_sheet.dart';
import '../../widgets/user_sessions_sheet.dart';

/// User management tab in admin screen.
///
/// Lists all users with search, role badges, and CRUD actions.
class UsersTab extends ConsumerStatefulWidget {
  const UsersTab({super.key});

  @override
  ConsumerState<UsersTab> createState() => _UsersTabState();
}

class _UsersTabState extends ConsumerState<UsersTab> {
  String _searchQuery = '';
  String? _roleFilter;

  @override
  void initState() {
    super.initState();
    Future.microtask(() => ref.read(allUsersProvider.notifier).load());
  }

  @override
  Widget build(BuildContext context) {
    final usersAsync = ref.watch(allUsersProvider);
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
                  TextField(
                    decoration: InputDecoration(
                      hintText: 'Search users...',
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
                  SingleChildScrollView(
                    scrollDirection: Axis.horizontal,
                    child: Row(
                      children: [
                        _FilterChip(
                          label: 'All',
                          selected: _roleFilter == null,
                          onSelected: () =>
                              setState(() => _roleFilter = null),
                        ),
                        const SizedBox(width: 8),
                        for (final role in const ['owner', 'admin', 'user'])
                          Padding(
                            padding: const EdgeInsets.only(right: 8),
                            child: _FilterChip(
                              label: role[0].toUpperCase() + role.substring(1),
                              selected: _roleFilter == role,
                              onSelected: () =>
                                  setState(() => _roleFilter = role),
                            ),
                          ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
            // User list
            Expanded(
              child: usersAsync.when(
                data: (users) {
                  final filtered = _applyFilters(users);
                  if (filtered.isEmpty) {
                    return Center(
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Icon(Icons.people_outline,
                              size: 48,
                              color: theme.colorScheme.onSurfaceVariant),
                          const SizedBox(height: 8),
                          Text(
                            users.isEmpty ? 'No users yet' : 'No matching users',
                            style: theme.textTheme.bodyLarge?.copyWith(
                              color: theme.colorScheme.onSurfaceVariant,
                            ),
                          ),
                        ],
                      ),
                    );
                  }
                  return RefreshIndicator(
                    onRefresh: () => ref.read(allUsersProvider.notifier).load(),
                    child: ListView.builder(
                      padding: const EdgeInsets.only(bottom: 80),
                      itemCount: filtered.length,
                      itemBuilder: (context, index) => _UserTile(
                        user: filtered[index],
                        onEdit: () => _openEditor(filtered[index]),
                        onRooms: () => _openRoomAccess(filtered[index]),
                        onSessions: () => _openSessions(filtered[index]),
                        onDelete: () => _confirmDelete(filtered[index]),
                      ),
                    ),
                  );
                },
                loading: () =>
                    const Center(child: CircularProgressIndicator()),
                error: (e, _) => Center(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(Icons.error_outline,
                          size: 48, color: theme.colorScheme.error),
                      const SizedBox(height: 8),
                      Text('Failed to load users',
                          style: TextStyle(color: theme.colorScheme.error)),
                      const SizedBox(height: 8),
                      FilledButton.tonal(
                        onPressed: () =>
                            ref.read(allUsersProvider.notifier).load(),
                        child: const Text('Retry'),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ],
        ),
        // FAB for creating users
        Positioned(
          right: 16,
          bottom: 16,
          child: FloatingActionButton(
            heroTag: 'user_fab',
            onPressed: () => _openEditor(null),
            child: const Icon(Icons.person_add),
          ),
        ),
      ],
    );
  }

  List<User> _applyFilters(List<User> users) {
    var filtered = users;
    if (_roleFilter != null) {
      filtered = filtered.where((u) => u.role == _roleFilter).toList();
    }
    if (_searchQuery.isNotEmpty) {
      final q = _searchQuery.toLowerCase();
      filtered = filtered.where((u) =>
          u.username.toLowerCase().contains(q) ||
          u.displayName.toLowerCase().contains(q) ||
          (u.email?.toLowerCase().contains(q) ?? false)).toList();
    }
    return filtered;
  }

  Future<void> _openEditor(User? user) async {
    final result = await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      builder: (context) => UserEditorSheet(user: user),
    );
    if (result == true) {
      ref.read(allUsersProvider.notifier).load();
    }
  }

  Future<void> _openRoomAccess(User user) async {
    await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      builder: (context) => UserRoomAccessSheet(user: user),
    );
  }

  Future<void> _openSessions(User user) async {
    await showModalBottomSheet(
      context: context,
      builder: (context) => UserSessionsSheet(user: user),
    );
  }

  Future<void> _confirmDelete(User user) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Delete User'),
        content: Text(
          'Delete "${user.displayName}" (@${user.username})? '
          'This will revoke all their sessions and cannot be undone.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: Theme.of(context).colorScheme.error,
            ),
            onPressed: () => Navigator.pop(context, true),
            child: const Text('Delete'),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;

    try {
      await ref.read(allUsersProvider.notifier).deleteUser(user.id);
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text('Deleted "${user.displayName}"'),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } catch (e) {
      if (!mounted) return;
      final msg = e.toString().contains('self')
          ? 'Cannot delete your own account'
          : 'Failed to delete user';
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(msg),
          behavior: SnackBarBehavior.floating,
          backgroundColor: Theme.of(context).colorScheme.error,
        ),
      );
    }
  }
}

class _FilterChip extends StatelessWidget {
  final String label;
  final bool selected;
  final VoidCallback onSelected;

  const _FilterChip({
    required this.label,
    required this.selected,
    required this.onSelected,
  });

  @override
  Widget build(BuildContext context) {
    return FilterChip(
      label: Text(label),
      selected: selected,
      onSelected: (_) => onSelected(),
      visualDensity: VisualDensity.compact,
    );
  }
}

class _UserTile extends StatelessWidget {
  final User user;
  final VoidCallback onEdit;
  final VoidCallback onRooms;
  final VoidCallback onSessions;
  final VoidCallback onDelete;

  const _UserTile({
    required this.user,
    required this.onEdit,
    required this.onRooms,
    required this.onSessions,
    required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final initial = user.displayName.isNotEmpty
        ? user.displayName[0].toUpperCase()
        : user.username[0].toUpperCase();

    return ListTile(
      leading: CircleAvatar(
        backgroundColor: _roleColor(user.role).withValues(alpha: 0.15),
        child: Text(
          initial,
          style: TextStyle(
            color: _roleColor(user.role),
            fontWeight: FontWeight.w600,
          ),
        ),
      ),
      title: Row(
        children: [
          Expanded(
            child: Text(user.displayName),
          ),
          _RoleBadge(role: user.role),
          if (!user.isActive) ...[
            const SizedBox(width: 4),
            Container(
              width: 8,
              height: 8,
              decoration: BoxDecoration(
                color: theme.colorScheme.error,
                shape: BoxShape.circle,
              ),
            ),
          ],
        ],
      ),
      subtitle: Text(
        '@${user.username}${user.email != null ? ' · ${user.email}' : ''}',
        style: theme.textTheme.bodySmall?.copyWith(
          color: theme.colorScheme.onSurfaceVariant,
        ),
      ),
      trailing: PopupMenuButton<String>(
        onSelected: (value) {
          switch (value) {
            case 'edit':
              onEdit();
            case 'rooms':
              onRooms();
            case 'sessions':
              onSessions();
            case 'delete':
              onDelete();
          }
        },
        itemBuilder: (context) => [
          const PopupMenuItem(value: 'edit', child: Text('Edit')),
          const PopupMenuItem(value: 'rooms', child: Text('Room Access')),
          const PopupMenuItem(value: 'sessions', child: Text('Sessions')),
          const PopupMenuDivider(),
          PopupMenuItem(
            value: 'delete',
            child: Text('Delete',
                style: TextStyle(color: theme.colorScheme.error)),
          ),
        ],
      ),
      onTap: onEdit,
    );
  }

  Color _roleColor(String role) {
    switch (role) {
      case 'owner':
        return Colors.deepPurple;
      case 'admin':
        return Colors.blue;
      case 'user':
        return Colors.teal;
      default:
        return Colors.grey;
    }
  }
}

class _RoleBadge extends StatelessWidget {
  final String role;

  const _RoleBadge({required this.role});

  @override
  Widget build(BuildContext context) {
    Color bgColor;
    switch (role) {
      case 'owner':
        bgColor = Colors.deepPurple;
      case 'admin':
        bgColor = Colors.blue;
      default:
        bgColor = Colors.teal;
    }

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: bgColor.withValues(alpha: 0.15),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        role,
        style: TextStyle(
          fontSize: 11,
          fontWeight: FontWeight.w600,
          color: bgColor,
        ),
      ),
    );
  }
}
