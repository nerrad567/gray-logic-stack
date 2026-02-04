import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../models/audit_log.dart';
import '../../providers/auth_provider.dart';

/// Action filter options for the audit log viewer.
const _actionFilters = <String, String>{
  '': 'All',
  'create': 'Create',
  'update': 'Update',
  'delete': 'Delete',
  'command': 'Command',
  'login': 'Login',
};

/// Colour associated with each action type.
Color _actionColor(String action) {
  switch (action) {
    case 'create':
      return Colors.green;
    case 'update':
      return Colors.blue;
    case 'delete':
      return Colors.red;
    case 'command':
      return Colors.amber.shade700;
    case 'login':
      return Colors.purple;
    default:
      return Colors.grey;
  }
}

/// Format a [DateTime] as a relative time string (e.g. "2m ago", "3h ago").
String _relativeTime(DateTime dt) {
  final diff = DateTime.now().difference(dt);
  if (diff.inSeconds < 60) return '${diff.inSeconds}s ago';
  if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
  if (diff.inHours < 24) return '${diff.inHours}h ago';
  if (diff.inDays < 7) return '${diff.inDays}d ago';
  return '${dt.year}-${dt.month.toString().padLeft(2, '0')}-${dt.day.toString().padLeft(2, '0')}';
}

/// Audit log viewer tab — read-only browser with filter chips and pagination.
class AuditTab extends ConsumerStatefulWidget {
  const AuditTab({super.key});

  @override
  ConsumerState<AuditTab> createState() => _AuditTabState();
}

class _AuditTabState extends ConsumerState<AuditTab> {
  static const _pageSize = 50;

  bool _loading = true;
  String? _error;
  List<AuditLog> _logs = [];
  int _total = 0;
  int _offset = 0;
  String _actionFilter = '';

  @override
  void initState() {
    super.initState();
    _loadData();
  }

  Future<void> _loadData({bool append = false}) async {
    if (!append) {
      setState(() {
        _loading = true;
        _error = null;
      });
    }
    try {
      final api = ref.read(apiClientProvider);
      final result = await api.getAuditLogs(
        action: _actionFilter.isEmpty ? null : _actionFilter,
        limit: _pageSize,
        offset: _offset,
      );
      if (mounted) {
        setState(() {
          if (append) {
            _logs.addAll(result.logs);
          } else {
            _logs = result.logs;
          }
          _total = result.total;
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

  void _setFilter(String action) {
    if (_actionFilter == action) return;
    _actionFilter = action;
    _offset = 0;
    _loadData();
  }

  void _loadMore() {
    _offset += _pageSize;
    _loadData(append: true);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final cs = theme.colorScheme;

    return Column(
      children: [
        // -- Filter chips row --
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 12, 16, 4),
          child: SizedBox(
            height: 36,
            child: ListView(
              scrollDirection: Axis.horizontal,
              children: _actionFilters.entries.map((entry) {
                final selected = _actionFilter == entry.key;
                return Padding(
                  padding: const EdgeInsets.only(right: 8),
                  child: FilterChip(
                    label: Text(entry.value),
                    selected: selected,
                    selectedColor: entry.key.isEmpty
                        ? cs.primaryContainer
                        : _actionColor(entry.key).withValues(alpha: 0.25),
                    onSelected: (_) => _setFilter(entry.key),
                    visualDensity: VisualDensity.compact,
                  ),
                );
              }).toList(),
            ),
          ),
        ),

        // -- Count --
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
          child: Align(
            alignment: Alignment.centerLeft,
            child: Text(
              _loading ? 'Loading...' : '$_total log entries',
              style: theme.textTheme.bodySmall?.copyWith(color: cs.onSurfaceVariant),
            ),
          ),
        ),

        const Divider(height: 1),

        // -- Content --
        Expanded(
          child: _buildContent(theme, cs),
        ),
      ],
    );
  }

  Widget _buildContent(ThemeData theme, ColorScheme cs) {
    if (_loading && _logs.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }

    if (_error != null && _logs.isEmpty) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.error_outline, size: 48, color: cs.error),
            const SizedBox(height: 12),
            Text('Failed to load audit logs', style: theme.textTheme.titleMedium),
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

    if (_logs.isEmpty) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.receipt_long_outlined, size: 48, color: cs.onSurfaceVariant.withValues(alpha: 0.5)),
            const SizedBox(height: 12),
            Text('No audit logs yet', style: theme.textTheme.titleMedium),
            const SizedBox(height: 4),
            Text(
              'Activity will appear here as the system is used.',
              style: theme.textTheme.bodySmall?.copyWith(color: cs.onSurfaceVariant),
            ),
          ],
        ),
      );
    }

    final hasMore = _logs.length < _total;

    return RefreshIndicator(
      onRefresh: () async {
        _offset = 0;
        await _loadData();
      },
      child: ListView.builder(
        itemCount: _logs.length + (hasMore ? 1 : 0),
        itemBuilder: (context, index) {
          if (index == _logs.length) {
            // "Load more" button
            return Padding(
              padding: const EdgeInsets.all(16),
              child: Center(
                child: OutlinedButton.icon(
                  onPressed: _loadMore,
                  icon: const Icon(Icons.expand_more),
                  label: Text('Load more (${_logs.length} of $_total)'),
                ),
              ),
            );
          }

          final log = _logs[index];
          return _AuditLogTile(log: log);
        },
      ),
    );
  }
}

/// Individual audit log entry tile with expandable details.
class _AuditLogTile extends StatelessWidget {
  final AuditLog log;

  const _AuditLogTile({required this.log});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final cs = theme.colorScheme;
    final color = _actionColor(log.action);
    final hasDetails = log.details != null && log.details!.isNotEmpty;

    final tile = ListTile(
      dense: true,
      leading: Container(
        width: 60,
        padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 3),
        decoration: BoxDecoration(
          color: color.withValues(alpha: 0.15),
          borderRadius: BorderRadius.circular(4),
        ),
        child: Text(
          log.action.toUpperCase(),
          textAlign: TextAlign.center,
          style: theme.textTheme.labelSmall?.copyWith(
            color: color,
            fontWeight: FontWeight.w600,
          ),
        ),
      ),
      title: Text(
        '${log.entityType}${log.entityId != null ? ' · ${log.entityId}' : ''}',
        style: theme.textTheme.bodyMedium,
        overflow: TextOverflow.ellipsis,
      ),
      subtitle: Text(
        '${log.source}${log.userId != null && log.userId!.isNotEmpty ? ' · ${log.userId}' : ''}',
        style: theme.textTheme.bodySmall?.copyWith(color: cs.onSurfaceVariant),
      ),
      trailing: Text(
        _relativeTime(log.createdAt),
        style: theme.textTheme.bodySmall?.copyWith(color: cs.onSurfaceVariant),
      ),
    );

    if (!hasDetails) return tile;

    // Expandable with details JSON
    return ExpansionTile(
      dense: true,
      leading: Container(
        width: 60,
        padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 3),
        decoration: BoxDecoration(
          color: color.withValues(alpha: 0.15),
          borderRadius: BorderRadius.circular(4),
        ),
        child: Text(
          log.action.toUpperCase(),
          textAlign: TextAlign.center,
          style: theme.textTheme.labelSmall?.copyWith(
            color: color,
            fontWeight: FontWeight.w600,
          ),
        ),
      ),
      title: Text(
        '${log.entityType}${log.entityId != null ? ' · ${log.entityId}' : ''}',
        style: theme.textTheme.bodyMedium,
        overflow: TextOverflow.ellipsis,
      ),
      subtitle: Text(
        '${log.source}${log.userId != null && log.userId!.isNotEmpty ? ' · ${log.userId}' : ''}',
        style: theme.textTheme.bodySmall?.copyWith(color: cs.onSurfaceVariant),
      ),
      trailing: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            _relativeTime(log.createdAt),
            style: theme.textTheme.bodySmall?.copyWith(color: cs.onSurfaceVariant),
          ),
          const SizedBox(width: 4),
          Icon(Icons.expand_more, size: 16, color: cs.onSurfaceVariant),
        ],
      ),
      children: [
        Container(
          width: double.infinity,
          margin: const EdgeInsets.fromLTRB(16, 0, 16, 12),
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            color: cs.surfaceContainerHighest,
            borderRadius: BorderRadius.circular(8),
          ),
          child: SelectableText(
            const JsonEncoder.withIndent('  ').convert(log.details),
            style: theme.textTheme.bodySmall?.copyWith(
              fontFamily: 'monospace',
              fontSize: 11,
            ),
          ),
        ),
      ],
    );
  }
}
