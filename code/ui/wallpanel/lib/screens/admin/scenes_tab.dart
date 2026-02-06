import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../models/scene.dart';
import '../../providers/location_provider.dart';
import '../../providers/scene_provider.dart';
import '../../widgets/scene_editor_sheet.dart';

/// Scenes management tab in admin screen.
///
/// Lists all scenes with search, filter by room/category, and CRUD actions.
class ScenesTab extends ConsumerStatefulWidget {
  const ScenesTab({super.key});

  @override
  ConsumerState<ScenesTab> createState() => _ScenesTabState();
}

class _ScenesTabState extends ConsumerState<ScenesTab> {
  String _searchQuery = '';
  String? _roomFilter;
  String? _categoryFilter;

  @override
  void initState() {
    super.initState();
    Future.microtask(() => ref.read(allScenesProvider.notifier).load());
  }

  @override
  Widget build(BuildContext context) {
    final scenesAsync = ref.watch(allScenesProvider);
    final locationAsync = ref.watch(locationDataProvider);
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
                      hintText: 'Search scenes...',
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
                  // Room filter chips
                  SingleChildScrollView(
                    scrollDirection: Axis.horizontal,
                    child: Row(
                      children: [
                        _FilterChip(
                          label: 'All Rooms',
                          selected: _roomFilter == null,
                          onSelected: () =>
                              setState(() => _roomFilter = null),
                        ),
                        const SizedBox(width: 8),
                        ...locationAsync.when(
                          data: (data) => data.sortedRooms.map((room) =>
                            Padding(
                              padding: const EdgeInsets.only(right: 8),
                              child: _FilterChip(
                                label: room.name,
                                selected: _roomFilter == room.id,
                                onSelected: () =>
                                    setState(() => _roomFilter = room.id),
                              ),
                            ),
                          ),
                          loading: () => const [SizedBox.shrink()],
                          error: (_, _) => const [SizedBox.shrink()],
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(height: 8),
                  // Category filter chips
                  SingleChildScrollView(
                    scrollDirection: Axis.horizontal,
                    child: Row(
                      children: [
                        _FilterChip(
                          label: 'All Categories',
                          selected: _categoryFilter == null,
                          onSelected: () =>
                              setState(() => _categoryFilter = null),
                        ),
                        const SizedBox(width: 8),
                        for (final cat in const [
                          'lighting', 'comfort', 'media', 'security', 'custom'
                        ])
                          Padding(
                            padding: const EdgeInsets.only(right: 8),
                            child: _FilterChip(
                              label: cat[0].toUpperCase() + cat.substring(1),
                              selected: _categoryFilter == cat,
                              onSelected: () =>
                                  setState(() => _categoryFilter = cat),
                            ),
                          ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
            // Scene list
            Expanded(
              child: scenesAsync.when(
                data: (scenes) {
                  final filtered = _applyFilters(scenes);
                  if (filtered.isEmpty) {
                    return Center(
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Icon(Icons.theaters_outlined,
                              size: 48, color: theme.colorScheme.onSurfaceVariant),
                          const SizedBox(height: 8),
                          Text(
                            scenes.isEmpty ? 'No scenes yet' : 'No matching scenes',
                            style: theme.textTheme.bodyLarge?.copyWith(
                              color: theme.colorScheme.onSurfaceVariant,
                            ),
                          ),
                        ],
                      ),
                    );
                  }
                  return RefreshIndicator(
                    onRefresh: () => ref.read(allScenesProvider.notifier).load(),
                    child: ListView.builder(
                      padding: const EdgeInsets.only(bottom: 80),
                      itemCount: filtered.length,
                      itemBuilder: (context, index) => _SceneTile(
                        scene: filtered[index],
                        onEdit: () => _openEditor(filtered[index]),
                        onDelete: () => _confirmDelete(filtered[index]),
                        onHistory: () => _showHistory(filtered[index]),
                      ),
                    ),
                  );
                },
                loading: () => const Center(child: CircularProgressIndicator()),
                error: (e, _) => Center(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(Icons.error_outline,
                          size: 48, color: theme.colorScheme.error),
                      const SizedBox(height: 8),
                      Text('Failed to load scenes',
                          style: TextStyle(color: theme.colorScheme.error)),
                      const SizedBox(height: 8),
                      FilledButton.tonal(
                        onPressed: () =>
                            ref.read(allScenesProvider.notifier).load(),
                        child: const Text('Retry'),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ],
        ),
        // FAB for creating scenes
        Positioned(
          right: 16,
          bottom: 16,
          child: FloatingActionButton(
            heroTag: 'scene_fab',
            onPressed: () => _openEditor(null),
            child: const Icon(Icons.add),
          ),
        ),
      ],
    );
  }

  List<Scene> _applyFilters(List<Scene> scenes) {
    var filtered = scenes;
    if (_roomFilter != null) {
      filtered = filtered.where((s) => s.roomId == _roomFilter).toList();
    }
    if (_categoryFilter != null) {
      filtered = filtered.where((s) => s.category == _categoryFilter).toList();
    }
    if (_searchQuery.isNotEmpty) {
      final q = _searchQuery.toLowerCase();
      filtered = filtered.where((s) =>
          s.name.toLowerCase().contains(q) ||
          (s.description?.toLowerCase().contains(q) ?? false)).toList();
    }
    return filtered;
  }

  Future<void> _openEditor(Scene? scene) async {
    final result = await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      builder: (context) => SceneEditorSheet(scene: scene),
    );
    if (result == true) {
      ref.read(allScenesProvider.notifier).load();
    }
  }

  Future<void> _confirmDelete(Scene scene) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Delete Scene'),
        content: Text('Delete "${scene.name}"? This cannot be undone.'),
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
      await ref.read(allScenesProvider.notifier).deleteScene(scene.id);
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text('Deleted "${scene.name}"'),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text('Failed to delete: $e'),
          behavior: SnackBarBehavior.floating,
          backgroundColor: Theme.of(context).colorScheme.error,
        ),
      );
    }
  }

  Future<void> _showHistory(Scene scene) async {
    final sceneRepo = ref.read(sceneRepositoryProvider);
    try {
      final response = await sceneRepo.getExecutions(scene.id);
      if (!mounted) return;
      showModalBottomSheet(
        context: context,
        builder: (context) => _ExecutionHistorySheet(
          sceneName: scene.name,
          executions: response.executions,
        ),
      );
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text('Failed to load history: $e'),
          behavior: SnackBarBehavior.floating,
          backgroundColor: Theme.of(context).colorScheme.error,
        ),
      );
    }
  }
}

// --- Private widgets ---

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

class _SceneTile extends StatelessWidget {
  final Scene scene;
  final VoidCallback onEdit;
  final VoidCallback onDelete;
  final VoidCallback onHistory;

  const _SceneTile({
    required this.scene,
    required this.onEdit,
    required this.onDelete,
    required this.onHistory,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colour = _parseColour(scene.colour) ?? theme.colorScheme.secondary;

    return ListTile(
      leading: CircleAvatar(
        backgroundColor: colour.withValues(alpha: 0.15),
        child: Icon(
          scene.icon != null ? _getIconData(scene.icon!) : Icons.play_circle_outline,
          color: colour,
          size: 20,
        ),
      ),
      title: Row(
        children: [
          Expanded(child: Text(scene.name)),
          if (!scene.enabled)
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
              decoration: BoxDecoration(
                color: theme.colorScheme.errorContainer,
                borderRadius: BorderRadius.circular(4),
              ),
              child: Text(
                'Disabled',
                style: theme.textTheme.labelSmall?.copyWith(
                  color: theme.colorScheme.onErrorContainer,
                ),
              ),
            ),
        ],
      ),
      subtitle: Text(
        [
          if (scene.category != null) scene.category!,
          '${scene.actions.length} action${scene.actions.length == 1 ? '' : 's'}',
          if (scene.roomId != null) 'room-scoped',
        ].join(' · '),
        style: theme.textTheme.bodySmall?.copyWith(
          color: theme.colorScheme.onSurfaceVariant,
        ),
      ),
      trailing: PopupMenuButton<String>(
        onSelected: (value) {
          switch (value) {
            case 'edit':
              onEdit();
            case 'history':
              onHistory();
            case 'delete':
              onDelete();
          }
        },
        itemBuilder: (context) => [
          const PopupMenuItem(value: 'edit', child: Text('Edit')),
          const PopupMenuItem(value: 'history', child: Text('History')),
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

  Color? _parseColour(String? hex) {
    if (hex == null || hex.isEmpty) return null;
    final cleaned = hex.replaceFirst('#', '');
    if (cleaned.length != 6) return null;
    final value = int.tryParse(cleaned, radix: 16);
    if (value == null) return null;
    return Color(0xFF000000 | value);
  }

  IconData _getIconData(String iconName) {
    const iconMap = {
      'movie': Icons.movie, 'cinema': Icons.movie,
      'reading': Icons.menu_book, 'book': Icons.menu_book,
      'bright': Icons.wb_sunny, 'sun': Icons.wb_sunny,
      'relax': Icons.spa, 'night': Icons.nightlight_round,
      'off': Icons.power_settings_new, 'all_off': Icons.power_settings_new,
      'morning': Icons.wb_twilight, 'evening': Icons.nights_stay,
      'party': Icons.celebration, 'dinner': Icons.restaurant,
      'welcome': Icons.waving_hand,
    };
    return iconMap[iconName.toLowerCase()] ?? Icons.play_circle_outline;
  }
}

class _ExecutionHistorySheet extends StatelessWidget {
  final String sceneName;
  final List<SceneExecution> executions;

  const _ExecutionHistorySheet({
    required this.sceneName,
    required this.executions,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Center(
            child: Container(
              width: 32,
              height: 4,
              decoration: BoxDecoration(
                color: theme.colorScheme.onSurfaceVariant.withValues(alpha: 0.4),
                borderRadius: BorderRadius.circular(2),
              ),
            ),
          ),
          const SizedBox(height: 16),
          Text('History: $sceneName', style: theme.textTheme.titleMedium),
          const SizedBox(height: 12),
          if (executions.isEmpty)
            const Padding(
              padding: EdgeInsets.symmetric(vertical: 24),
              child: Center(child: Text('No executions yet')),
            )
          else
            ConstrainedBox(
              constraints: BoxConstraints(
                maxHeight: MediaQuery.of(context).size.height * 0.4,
              ),
              child: ListView.builder(
                shrinkWrap: true,
                itemCount: executions.length,
                itemBuilder: (context, index) {
                  final exec = executions[index];
                  final statusIcon = exec.status == 'completed'
                      ? Icons.check_circle
                      : exec.status == 'failed'
                          ? Icons.error
                          : Icons.pending;
                  final statusColor = exec.status == 'completed'
                      ? Colors.green
                      : exec.status == 'failed'
                          ? theme.colorScheme.error
                          : Colors.orange;

                  return ListTile(
                    dense: true,
                    leading: Icon(statusIcon, color: statusColor, size: 20),
                    title: Text(
                      '${exec.triggerType} (${exec.triggerSource})',
                      style: theme.textTheme.bodySmall,
                    ),
                    subtitle: Text(
                      '${exec.successCount}/${exec.actionCount} actions · ${exec.durationMs}ms',
                      style: theme.textTheme.labelSmall,
                    ),
                    trailing: Text(
                      _formatTime(exec.startedAt),
                      style: theme.textTheme.labelSmall,
                    ),
                  );
                },
              ),
            ),
          const SizedBox(height: 8),
        ],
      ),
    );
  }

  String _formatTime(DateTime dt) {
    final local = dt.toLocal();
    final now = DateTime.now();
    final diff = now.difference(local);
    if (diff.inMinutes < 1) return 'just now';
    if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
    if (diff.inHours < 24) return '${diff.inHours}h ago';
    return '${local.day}/${local.month} ${local.hour.toString().padLeft(2, '0')}:${local.minute.toString().padLeft(2, '0')}';
  }
}
