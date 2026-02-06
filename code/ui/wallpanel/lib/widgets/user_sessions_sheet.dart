import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/user.dart';
import '../providers/user_provider.dart';

/// Bottom sheet showing active sessions for a user with a "Revoke All" button.
class UserSessionsSheet extends ConsumerStatefulWidget {
  final User user;

  const UserSessionsSheet({super.key, required this.user});

  @override
  ConsumerState<UserSessionsSheet> createState() =>
      _UserSessionsSheetState();
}

class _UserSessionsSheetState extends ConsumerState<UserSessionsSheet> {
  List<UserSession>? _sessions;
  bool _loading = true;
  bool _revoking = false;

  @override
  void initState() {
    super.initState();
    _loadSessions();
  }

  Future<void> _loadSessions() async {
    try {
      final userRepo = ref.read(userRepositoryProvider);
      final sessions = await userRepo.getUserSessions(widget.user.id);
      if (mounted) {
        setState(() {
          _sessions = sessions;
          _loading = false;
        });
      }
    } catch (_) {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _revokeAll() async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Revoke All Sessions'),
        content: Text(
          'Revoke all active sessions for ${widget.user.displayName}? '
          'They will be logged out of all devices.',
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
            child: const Text('Revoke All'),
          ),
        ],
      ),
    );

    if (confirmed != true || !mounted) return;

    setState(() => _revoking = true);
    try {
      final userRepo = ref.read(userRepositoryProvider);
      await userRepo.revokeUserSessions(widget.user.id);
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('All sessions revoked'),
          behavior: SnackBarBehavior.floating,
        ),
      );
      // Reload to show cleared state
      setState(() {
        _sessions = [];
        _revoking = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() => _revoking = false);
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text('Failed to revoke sessions: $e'),
          behavior: SnackBarBehavior.floating,
          backgroundColor: Theme.of(context).colorScheme.error,
        ),
      );
    }
  }

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
                color: theme.colorScheme.onSurfaceVariant
                    .withValues(alpha: 0.4),
                borderRadius: BorderRadius.circular(2),
              ),
            ),
          ),
          const SizedBox(height: 16),
          Row(
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('Sessions', style: theme.textTheme.titleMedium),
                    Text(
                      widget.user.displayName,
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
              ),
              if (_sessions != null && _sessions!.isNotEmpty)
                FilledButton.tonal(
                  onPressed: _revoking ? null : _revokeAll,
                  style: FilledButton.styleFrom(
                    backgroundColor: theme.colorScheme.errorContainer,
                    foregroundColor: theme.colorScheme.onErrorContainer,
                  ),
                  child: _revoking
                      ? const SizedBox(
                          width: 16,
                          height: 16,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : const Text('Revoke All'),
                ),
            ],
          ),
          const SizedBox(height: 12),
          if (_loading)
            const Padding(
              padding: EdgeInsets.symmetric(vertical: 24),
              child: Center(child: CircularProgressIndicator()),
            )
          else if (_sessions == null || _sessions!.isEmpty)
            const Padding(
              padding: EdgeInsets.symmetric(vertical: 24),
              child: Center(child: Text('No active sessions')),
            )
          else
            ConstrainedBox(
              constraints: BoxConstraints(
                maxHeight: MediaQuery.of(context).size.height * 0.4,
              ),
              child: ListView.builder(
                shrinkWrap: true,
                itemCount: _sessions!.length,
                itemBuilder: (context, index) {
                  final session = _sessions![index];
                  final isExpired = session.expiresAt.isBefore(DateTime.now());

                  return ListTile(
                    dense: true,
                    leading: Icon(
                      session.revoked || isExpired
                          ? Icons.cancel_outlined
                          : Icons.devices,
                      color: session.revoked || isExpired
                          ? theme.colorScheme.error
                          : theme.colorScheme.primary,
                      size: 20,
                    ),
                    title: Text(
                      session.deviceInfo ?? 'Unknown device',
                      style: theme.textTheme.bodySmall,
                    ),
                    subtitle: Text(
                      session.revoked
                          ? 'Revoked'
                          : isExpired
                              ? 'Expired'
                              : 'Expires ${_formatTime(session.expiresAt)}',
                      style: theme.textTheme.labelSmall?.copyWith(
                        color: session.revoked || isExpired
                            ? theme.colorScheme.error
                            : null,
                      ),
                    ),
                    trailing: Text(
                      _formatTime(session.createdAt),
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
    if (diff.inMinutes.abs() < 1) return 'just now';
    if (diff.inMinutes < 0) {
      // Future (expiry time)
      final absDiff = diff.abs();
      if (absDiff.inMinutes < 60) return 'in ${absDiff.inMinutes}m';
      if (absDiff.inHours < 24) return 'in ${absDiff.inHours}h';
      return '${local.day}/${local.month}';
    }
    if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
    if (diff.inHours < 24) return '${diff.inHours}h ago';
    return '${local.day}/${local.month} ${local.hour.toString().padLeft(2, '0')}:${local.minute.toString().padLeft(2, '0')}';
  }
}
