import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../models/panel.dart';
import '../../providers/panel_provider.dart';
import '../../widgets/panel_editor_sheet.dart';
import '../../widgets/panel_rooms_sheet.dart';

/// Panel management tab in admin screen.
///
/// Lists all registered panels with search, status indicators, and CRUD actions.
class PanelsTab extends ConsumerStatefulWidget {
  const PanelsTab({super.key});

  @override
  ConsumerState<PanelsTab> createState() => _PanelsTabState();
}

class _PanelsTabState extends ConsumerState<PanelsTab> {
  String _searchQuery = '';

  @override
  void initState() {
    super.initState();
    Future.microtask(() => ref.read(allPanelsProvider.notifier).load());
  }

  @override
  Widget build(BuildContext context) {
    final panelsAsync = ref.watch(allPanelsProvider);
    final theme = Theme.of(context);

    return Stack(
      children: [
        Column(
          children: [
            // Search bar
            Container(
              padding: const EdgeInsets.all(16),
              child: TextField(
                decoration: InputDecoration(
                  hintText: 'Search panels...',
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
            ),
            // Panel list
            Expanded(
              child: panelsAsync.when(
                data: (panels) {
                  final filtered = _applyFilters(panels);
                  if (filtered.isEmpty) {
                    return Center(
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Icon(Icons.tablet_outlined,
                              size: 48,
                              color: theme.colorScheme.onSurfaceVariant),
                          const SizedBox(height: 8),
                          Text(
                            panels.isEmpty
                                ? 'No panels registered'
                                : 'No matching panels',
                            style: theme.textTheme.bodyLarge?.copyWith(
                              color: theme.colorScheme.onSurfaceVariant,
                            ),
                          ),
                        ],
                      ),
                    );
                  }
                  return RefreshIndicator(
                    onRefresh: () =>
                        ref.read(allPanelsProvider.notifier).load(),
                    child: ListView.builder(
                      padding: const EdgeInsets.only(bottom: 80),
                      itemCount: filtered.length,
                      itemBuilder: (context, index) => _PanelTile(
                        panel: filtered[index],
                        onEdit: () => _openEditor(filtered[index]),
                        onRooms: () => _openRooms(filtered[index]),
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
                      Text('Failed to load panels',
                          style: TextStyle(color: theme.colorScheme.error)),
                      const SizedBox(height: 8),
                      FilledButton.tonal(
                        onPressed: () =>
                            ref.read(allPanelsProvider.notifier).load(),
                        child: const Text('Retry'),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ],
        ),
        // FAB for registering panels
        Positioned(
          right: 16,
          bottom: 16,
          child: FloatingActionButton(
            heroTag: 'panel_fab',
            onPressed: () => _openEditor(null),
            child: const Icon(Icons.add),
          ),
        ),
      ],
    );
  }

  List<Panel> _applyFilters(List<Panel> panels) {
    if (_searchQuery.isEmpty) return panels;
    final q = _searchQuery.toLowerCase();
    return panels.where((p) => p.name.toLowerCase().contains(q)).toList();
  }

  Future<void> _openEditor(Panel? panel) async {
    final result = await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      builder: (context) => PanelEditorSheet(panel: panel),
    );
    if (result == true) {
      ref.read(allPanelsProvider.notifier).load();
    }
  }

  Future<void> _openRooms(Panel panel) async {
    await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      builder: (context) => PanelRoomsSheet(panel: panel),
    );
  }

  Future<void> _confirmDelete(Panel panel) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Delete Panel'),
        content: Text(
          'Delete "${panel.name}"? '
          'This will revoke its token and remove all room assignments. '
          'This cannot be undone.',
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
      await ref.read(allPanelsProvider.notifier).deletePanel(panel.id);
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text('Deleted "${panel.name}"'),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: const Text('Failed to delete panel'),
          behavior: SnackBarBehavior.floating,
          backgroundColor: Theme.of(context).colorScheme.error,
        ),
      );
    }
  }
}

class _PanelTile extends StatelessWidget {
  final Panel panel;
  final VoidCallback onEdit;
  final VoidCallback onRooms;
  final VoidCallback onDelete;

  const _PanelTile({
    required this.panel,
    required this.onEdit,
    required this.onRooms,
    required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return ListTile(
      leading: CircleAvatar(
        backgroundColor: _statusColor.withValues(alpha: 0.15),
        child: Icon(
          Icons.tablet_outlined,
          color: _statusColor,
          size: 20,
        ),
      ),
      title: Row(
        children: [
          Expanded(child: Text(panel.name)),
          Container(
            width: 8,
            height: 8,
            decoration: BoxDecoration(
              color: _statusColor,
              shape: BoxShape.circle,
            ),
          ),
        ],
      ),
      subtitle: Text(
        _subtitle,
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
            case 'delete':
              onDelete();
          }
        },
        itemBuilder: (context) => [
          const PopupMenuItem(value: 'edit', child: Text('Edit')),
          const PopupMenuItem(value: 'rooms', child: Text('Rooms')),
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

  Color get _statusColor {
    if (!panel.isActive) return Colors.red;
    if (panel.lastSeenAt != null) return Colors.green;
    return Colors.grey;
  }

  String get _subtitle {
    final parts = <String>[];
    if (panel.lastSeenAt != null) {
      parts.add('Last seen ${_timeAgo(panel.lastSeenAt!)}');
    } else {
      parts.add('Never connected');
    }
    return parts.join(' · ');
  }

  String _timeAgo(DateTime dt) {
    final diff = DateTime.now().difference(dt);
    if (diff.inSeconds < 60) return 'just now';
    if (diff.inMinutes < 60) return '${diff.inMinutes} min ago';
    if (diff.inHours < 24) return '${diff.inHours}h ago';
    if (diff.inDays < 7) return '${diff.inDays}d ago';
    return '${dt.day}/${dt.month}/${dt.year}';
  }
}
