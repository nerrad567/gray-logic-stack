import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../providers/auth_provider.dart';
import '../../providers/location_provider.dart';
import 'devices_tab.dart';

/// System management tab with factory reset functionality.
///
/// Provides checkboxes to select which data categories to clear,
/// a two-stage confirmation flow, and progress/result feedback.
class SystemTab extends ConsumerStatefulWidget {
  /// Called after a successful reset to refresh the main view.
  final VoidCallback? onReset;

  const SystemTab({super.key, this.onReset});

  @override
  ConsumerState<SystemTab> createState() => _SystemTabState();
}

class _SystemTabState extends ConsumerState<SystemTab> {
  bool _clearDevices = true;
  bool _clearScenes = true;
  bool _clearLocations = true;
  bool _clearDiscovery = false;
  bool _clearSite = false;
  bool _resetting = false;

  bool get _anySelected =>
      _clearDevices ||
      _clearScenes ||
      _clearLocations ||
      _clearDiscovery ||
      _clearSite;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Warning header
          Container(
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              color: theme.colorScheme.errorContainer.withValues(alpha: 0.3),
              borderRadius: BorderRadius.circular(12),
              border: Border.all(
                color: theme.colorScheme.error.withValues(alpha: 0.3),
              ),
            ),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Icon(
                  Icons.warning_amber_rounded,
                  color: theme.colorScheme.error,
                  size: 28,
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Factory Reset',
                        style: theme.textTheme.titleMedium?.copyWith(
                          color: theme.colorScheme.error,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                      const SizedBox(height: 4),
                      Text(
                        'Permanently delete selected data from the system. '
                        'This cannot be undone. Use this before re-importing '
                        'from an updated ETS project file.',
                        style: theme.textTheme.bodyMedium?.copyWith(
                          color: theme.colorScheme.onSurface,
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),

          const SizedBox(height: 24),

          // Data category toggles
          Text(
            'Select data to clear',
            style: theme.textTheme.titleSmall?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 8),

          _ResetOption(
            icon: Icons.devices_outlined,
            title: 'Devices',
            subtitle: 'All registered devices, their state, and GA mappings',
            value: _clearDevices,
            onChanged: _resetting
                ? null
                : (v) => setState(() => _clearDevices = v ?? false),
          ),
          _ResetOption(
            icon: Icons.auto_awesome_outlined,
            title: 'Scenes',
            subtitle: 'Scene definitions and execution history',
            value: _clearScenes,
            onChanged: _resetting
                ? null
                : (v) => setState(() => _clearScenes = v ?? false),
          ),
          _ResetOption(
            icon: Icons.location_on_outlined,
            title: 'Locations',
            subtitle: 'Rooms and areas (re-created on next ETS import)',
            value: _clearLocations,
            onChanged: _resetting
                ? null
                : (v) => setState(() => _clearLocations = v ?? false),
          ),

          const Divider(height: 32),
          Text(
            'Advanced',
            style: theme.textTheme.titleSmall?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 8),

          _ResetOption(
            icon: Icons.radar_outlined,
            title: 'Discovery Data',
            subtitle:
                'Passively recorded KNX bus addresses and device sightings',
            value: _clearDiscovery,
            onChanged: _resetting
                ? null
                : (v) => setState(() => _clearDiscovery = v ?? false),
          ),
          _ResetOption(
            icon: Icons.home_outlined,
            title: 'Site Record',
            subtitle:
                'Property name, timezone, GPS, and house modes. '
                'Must be re-created via Admin after clearing.',
            value: _clearSite,
            onChanged: _resetting
                ? null
                : (v) => setState(() => _clearSite = v ?? false),
          ),

          const SizedBox(height: 32),

          // Reset button
          SizedBox(
            width: double.infinity,
            child: FilledButton.icon(
              onPressed: (!_anySelected || _resetting) ? null : _confirmReset,
              icon: _resetting
                  ? const SizedBox(
                      width: 18,
                      height: 18,
                      child: CircularProgressIndicator(
                        strokeWidth: 2,
                        color: Colors.white,
                      ),
                    )
                  : const Icon(Icons.restart_alt),
              label: Text(_resetting ? 'Resetting...' : 'Factory Reset'),
              style: FilledButton.styleFrom(
                backgroundColor: theme.colorScheme.error,
                foregroundColor: theme.colorScheme.onError,
                padding: const EdgeInsets.symmetric(vertical: 16),
              ),
            ),
          ),

          if (!_anySelected) ...[
            const SizedBox(height: 8),
            Text(
              'Select at least one category to enable reset.',
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ],
      ),
    );
  }

  Future<void> _confirmReset() async {
    // Build a summary of what will be cleared.
    final items = <String>[];
    if (_clearDevices) items.add('devices');
    if (_clearScenes) items.add('scenes');
    if (_clearLocations) items.add('rooms & areas');
    if (_clearDiscovery) items.add('discovery data');
    if (_clearSite) items.add('site record');

    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => _ConfirmResetDialog(items: items),
    );

    if (confirmed == true && mounted) {
      await _performReset();
    }
  }

  Future<void> _performReset() async {
    setState(() => _resetting = true);

    try {
      final apiClient = ref.read(apiClientProvider);
      final result = await apiClient.factoryReset(
        clearDevices: _clearDevices,
        clearScenes: _clearScenes,
        clearLocations: _clearLocations,
        clearDiscovery: _clearDiscovery,
        clearSite: _clearSite,
      );

      // Invalidate relevant providers so the UI refreshes.
      ref.invalidate(allDevicesProvider);
      ref.invalidate(locationDataProvider);
      widget.onReset?.call();

      if (mounted) {
        // Build a result summary.
        final parts = result.deleted.entries
            .where((e) => e.value > 0)
            .map((e) => '${e.value} ${e.key.replaceAll('_', ' ')}')
            .toList();
        final summary = parts.isEmpty
            ? 'No data to clear.'
            : 'Cleared: ${parts.join(', ')}';

        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(summary),
            duration: const Duration(seconds: 5),
          ),
        );
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Factory reset failed: $e')),
        );
      }
    } finally {
      if (mounted) {
        setState(() => _resetting = false);
      }
    }
  }
}

/// A single toggle row for a data category.
class _ResetOption extends StatelessWidget {
  final IconData icon;
  final String title;
  final String subtitle;
  final bool value;
  final ValueChanged<bool?>? onChanged;

  const _ResetOption({
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.value,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: CheckboxListTile(
        secondary: Icon(icon, color: theme.colorScheme.onSurfaceVariant),
        title: Text(title),
        subtitle: Text(
          subtitle,
          style: theme.textTheme.bodySmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
        value: value,
        onChanged: onChanged,
        controlAffinity: ListTileControlAffinity.trailing,
      ),
    );
  }
}

/// Two-stage confirmation dialog requiring the user to type "RESET".
class _ConfirmResetDialog extends StatefulWidget {
  final List<String> items;

  const _ConfirmResetDialog({required this.items});

  @override
  State<_ConfirmResetDialog> createState() => _ConfirmResetDialogState();
}

class _ConfirmResetDialogState extends State<_ConfirmResetDialog> {
  final _controller = TextEditingController();
  bool get _confirmed => _controller.text.trim().toUpperCase() == 'RESET';

  @override
  void initState() {
    super.initState();
    _controller.addListener(() => setState(() {}));
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return AlertDialog(
      title: Row(
        children: [
          Icon(Icons.warning_amber_rounded, color: theme.colorScheme.error),
          const SizedBox(width: 8),
          const Text('Confirm Factory Reset'),
        ],
      ),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            'This will permanently delete:',
            style: theme.textTheme.bodyMedium,
          ),
          const SizedBox(height: 8),
          ...widget.items.map(
            (item) => Padding(
              padding: const EdgeInsets.only(left: 8, bottom: 4),
              child: Row(
                children: [
                  Icon(Icons.remove, size: 16, color: theme.colorScheme.error),
                  const SizedBox(width: 8),
                  Text(item),
                ],
              ),
            ),
          ),
          const SizedBox(height: 16),
          Text(
            'Type RESET to confirm:',
            style: theme.textTheme.bodyMedium?.copyWith(
              fontWeight: FontWeight.bold,
            ),
          ),
          const SizedBox(height: 8),
          TextField(
            controller: _controller,
            autofocus: true,
            decoration: InputDecoration(
              hintText: 'RESET',
              border: const OutlineInputBorder(),
              errorText: _controller.text.isNotEmpty && !_confirmed
                  ? 'Type RESET to confirm'
                  : null,
            ),
            textCapitalization: TextCapitalization.characters,
          ),
        ],
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(false),
          child: const Text('Cancel'),
        ),
        FilledButton(
          onPressed: _confirmed
              ? () => Navigator.of(context).pop(true)
              : null,
          style: FilledButton.styleFrom(
            backgroundColor: theme.colorScheme.error,
          ),
          child: const Text('Reset'),
        ),
      ],
    );
  }
}
