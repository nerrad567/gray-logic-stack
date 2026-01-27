import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../models/discovery.dart';
import '../../providers/auth_provider.dart';

/// Provider for discovery data with auto-refresh.
final discoveryProvider = FutureProvider.autoDispose<DiscoveryData>((ref) async {
  final apiClient = ref.watch(apiClientProvider);
  return apiClient.getDiscovery();
});

/// KNX domain types with colors and icons.
enum KnxDomain {
  lighting(Icons.lightbulb_outline, Colors.amber, 'Lighting'),
  blinds(Icons.blinds_outlined, Colors.teal, 'Blinds'),
  hvac(Icons.thermostat_outlined, Colors.orange, 'HVAC'),
  sensor(Icons.sensors_outlined, Colors.cyan, 'Sensor'),
  system(Icons.settings_outlined, Colors.grey, 'System'),
  unknown(Icons.help_outline, Colors.blueGrey, 'Unknown');

  final IconData icon;
  final Color color;
  final String label;

  const KnxDomain(this.icon, this.color, this.label);

  /// Detect domain from group address pattern.
  static KnxDomain fromGroupAddress(String ga) {
    final parts = ga.split('/');
    if (parts.isEmpty) return unknown;

    final main = int.tryParse(parts[0]) ?? 0;

    // Common KNX GA conventions:
    // 0/x/x = System
    // 1/x/x = Lighting
    // 2/x/x = Blinds/Shading
    // 3/x/x = HVAC
    // 4/x/x = Sensors
    // 5/x/x = Audio/Video
    switch (main) {
      case 0:
        return system;
      case 1:
        return lighting;
      case 2:
        return blinds;
      case 3:
        return hvac;
      case 4:
        return sensor;
      default:
        return unknown;
    }
  }

  /// Detect domain from individual address pattern.
  static KnxDomain fromIndividualAddress(String addr) {
    // 0.0.x addresses are system/health check
    if (addr.startsWith('0.0.')) return system;

    // 1.1.x are typically actuators (lights, blinds)
    // 1.2.x are typically input devices (switches)
    // We can't determine type from individual address alone
    return unknown;
  }
}

/// Tab showing passively discovered KNX bus data.
class DiscoveryTab extends ConsumerStatefulWidget {
  const DiscoveryTab({super.key});

  @override
  ConsumerState<DiscoveryTab> createState() => _DiscoveryTabState();
}

class _DiscoveryTabState extends ConsumerState<DiscoveryTab> {
  Timer? _refreshTimer;
  String _filter = 'all'; // 'all', 'ga', 'devices', 'lighting', 'blinds', 'sensor'
  String _searchQuery = '';
  bool _hideSystemAddresses = true;

  @override
  void initState() {
    super.initState();
    _refreshTimer = Timer.periodic(const Duration(seconds: 10), (_) {
      ref.invalidate(discoveryProvider);
    });
  }

  @override
  void dispose() {
    _refreshTimer?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final discoveryAsync = ref.watch(discoveryProvider);
    final theme = Theme.of(context);

    return discoveryAsync.when(
      data: (data) => _buildContent(context, data),
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (error, _) => Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.error_outline, size: 48, color: theme.colorScheme.error),
            const SizedBox(height: 12),
            Text('Failed to load discovery data', style: theme.textTheme.titleMedium),
            const SizedBox(height: 8),
            Text(error.toString(), style: theme.textTheme.bodySmall),
            const SizedBox(height: 16),
            FilledButton.icon(
              onPressed: () => ref.invalidate(discoveryProvider),
              icon: const Icon(Icons.refresh),
              label: const Text('Retry'),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildContent(BuildContext context, DiscoveryData data) {
    // Filter out system addresses if enabled
    final filteredGAs = _hideSystemAddresses
        ? data.groupAddresses.where((ga) => !ga.groupAddress.startsWith('0/')).toList()
        : data.groupAddresses;
    final filteredDevices = _hideSystemAddresses
        ? data.devices.where((d) => !d.individualAddress.startsWith('0.0.')).toList()
        : data.devices;

    final hasData = filteredGAs.isNotEmpty || filteredDevices.isNotEmpty;

    return Column(
      children: [
        // Summary cards
        _buildSummaryRow(context, filteredGAs, filteredDevices),
        const SizedBox(height: 12),

        // Filter row
        _buildFilterRow(context, filteredGAs, filteredDevices),
        const SizedBox(height: 8),

        // Search and options row
        _buildSearchRow(context),
        const SizedBox(height: 12),

        // List
        Expanded(
          child: hasData
              ? _buildList(context, filteredGAs, filteredDevices)
              : _buildEmptyState(context),
        ),
      ],
    );
  }

  Widget _buildSummaryRow(
    BuildContext context,
    List<DiscoveredGA> gas,
    List<DiscoveredDevice> devices,
  ) {
    // Count by domain
    final domainCounts = <KnxDomain, int>{};
    for (final ga in gas) {
      final domain = KnxDomain.fromGroupAddress(ga.groupAddress);
      domainCounts[domain] = (domainCounts[domain] ?? 0) + 1;
    }

    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 0),
      child: Row(
        children: [
          Expanded(
            child: _SummaryCard(
              icon: Icons.lan_outlined,
              label: 'Group Addresses',
              value: gas.length.toString(),
              subValue: '${gas.where((g) => g.hasReadResponse).length} responding',
              color: Colors.blue,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: _SummaryCard(
              icon: Icons.memory_outlined,
              label: 'KNX Devices',
              value: devices.length.toString(),
              subValue: 'Physical devices',
              color: Colors.green,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: _SummaryCard(
              icon: Icons.lightbulb_outline,
              label: 'Lighting',
              value: (domainCounts[KnxDomain.lighting] ?? 0).toString(),
              subValue: 'Group addresses',
              color: KnxDomain.lighting.color,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: _SummaryCard(
              icon: Icons.sensors_outlined,
              label: 'Sensors',
              value: (domainCounts[KnxDomain.sensor] ?? 0).toString(),
              subValue: 'Group addresses',
              color: KnxDomain.sensor.color,
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildFilterRow(
    BuildContext context,
    List<DiscoveredGA> gas,
    List<DiscoveredDevice> devices,
  ) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        child: Row(
          children: [
            FilterChip(
              label: const Text('All'),
              selected: _filter == 'all',
              onSelected: (_) => setState(() => _filter = 'all'),
            ),
            const SizedBox(width: 8),
            FilterChip(
              label: Text('GAs (${gas.length})'),
              selected: _filter == 'ga',
              onSelected: (_) => setState(() => _filter = 'ga'),
            ),
            const SizedBox(width: 8),
            FilterChip(
              label: Text('Devices (${devices.length})'),
              selected: _filter == 'devices',
              onSelected: (_) => setState(() => _filter = 'devices'),
            ),
            const SizedBox(width: 16),
            const VerticalDivider(width: 1),
            const SizedBox(width: 16),
            FilterChip(
              avatar: Icon(KnxDomain.lighting.icon, size: 16),
              label: const Text('Lighting'),
              selected: _filter == 'lighting',
              onSelected: (_) => setState(() => _filter = 'lighting'),
              selectedColor: KnxDomain.lighting.color.withValues(alpha: 0.3),
            ),
            const SizedBox(width: 8),
            FilterChip(
              avatar: Icon(KnxDomain.blinds.icon, size: 16),
              label: const Text('Blinds'),
              selected: _filter == 'blinds',
              onSelected: (_) => setState(() => _filter = 'blinds'),
              selectedColor: KnxDomain.blinds.color.withValues(alpha: 0.3),
            ),
            const SizedBox(width: 8),
            FilterChip(
              avatar: Icon(KnxDomain.sensor.icon, size: 16),
              label: const Text('Sensors'),
              selected: _filter == 'sensor',
              onSelected: (_) => setState(() => _filter = 'sensor'),
              selectedColor: KnxDomain.sensor.color.withValues(alpha: 0.3),
            ),
            const SizedBox(width: 16),
            IconButton(
              onPressed: () => ref.invalidate(discoveryProvider),
              icon: const Icon(Icons.refresh),
              tooltip: 'Refresh',
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildSearchRow(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Row(
        children: [
          Expanded(
            child: TextField(
              decoration: InputDecoration(
                hintText: 'Search addresses...',
                prefixIcon: const Icon(Icons.search),
                border: OutlineInputBorder(borderRadius: BorderRadius.circular(8)),
                isDense: true,
                contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 12),
              ),
              onChanged: (value) => setState(() => _searchQuery = value),
            ),
          ),
          const SizedBox(width: 12),
          FilterChip(
            label: const Text('Hide 0.0.x'),
            selected: _hideSystemAddresses,
            onSelected: (v) => setState(() => _hideSystemAddresses = v),
            tooltip: 'Hide system/health check addresses',
          ),
        ],
      ),
    );
  }

  Widget _buildList(
    BuildContext context,
    List<DiscoveredGA> gas,
    List<DiscoveredDevice> devices,
  ) {
    final theme = Theme.of(context);
    final List<_ListItem> items = [];

    // Add group addresses
    if (_filter == 'all' || _filter == 'ga' ||
        _filter == 'lighting' || _filter == 'blinds' || _filter == 'sensor') {
      for (final ga in gas) {
        final domain = KnxDomain.fromGroupAddress(ga.groupAddress);

        // Filter by domain
        if (_filter == 'lighting' && domain != KnxDomain.lighting) continue;
        if (_filter == 'blinds' && domain != KnxDomain.blinds) continue;
        if (_filter == 'sensor' && domain != KnxDomain.sensor) continue;

        // Search filter
        if (_searchQuery.isNotEmpty && !ga.groupAddress.contains(_searchQuery)) continue;

        items.add(_ListItem(
          itemType: 'ga',
          address: ga.groupAddress,
          lastSeenAgo: ga.lastSeenAgo,
          messageCount: ga.messageCount,
          domain: domain,
          extra: ga.hasReadResponse ? 'Responds to reads' : null,
        ));
      }
    }

    // Add devices
    if (_filter == 'all' || _filter == 'devices') {
      for (final dev in devices) {
        if (_searchQuery.isNotEmpty && !dev.individualAddress.contains(_searchQuery)) continue;

        final domain = KnxDomain.fromIndividualAddress(dev.individualAddress);
        final deviceType = _guessDeviceType(dev.individualAddress);

        items.add(_ListItem(
          itemType: 'device',
          address: dev.individualAddress,
          lastSeenAgo: dev.lastSeenAgo,
          messageCount: dev.messageCount,
          domain: domain,
          extra: deviceType,
        ));
      }
    }

    if (items.isEmpty) {
      return Center(
        child: Text(
          _searchQuery.isNotEmpty
              ? 'No matches for "$_searchQuery"'
              : 'No items match the current filter',
          style: theme.textTheme.bodyMedium?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
      );
    }

    return ListView.builder(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      itemCount: items.length,
      itemBuilder: (context, index) => _buildListItem(context, items[index]),
    );
  }

  Widget _buildListItem(BuildContext context, _ListItem item) {
    final theme = Theme.of(context);
    final isGA = item.itemType == 'ga';
    final color = isGA ? item.domain.color : Colors.green;
    final icon = isGA ? item.domain.icon : Icons.memory_outlined;

    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: ListTile(
        leading: CircleAvatar(
          backgroundColor: color.withValues(alpha: 0.2),
          child: Icon(icon, color: color, size: 20),
        ),
        title: Row(
          children: [
            Text(
              item.address,
              style: theme.textTheme.titleSmall?.copyWith(
                fontFamily: 'monospace',
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(width: 8),
            if (isGA)
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                decoration: BoxDecoration(
                  color: color.withValues(alpha: 0.2),
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Text(
                  item.domain.label,
                  style: theme.textTheme.labelSmall?.copyWith(color: color),
                ),
              ),
          ],
        ),
        subtitle: Text(
          '${item.messageCount} msgs • ${item.lastSeenAgo}${item.extra != null ? ' • ${item.extra}' : ''}',
          style: theme.textTheme.bodySmall,
        ),
        trailing: Chip(
          label: Text(isGA ? 'GA' : 'Device'),
          visualDensity: VisualDensity.compact,
          padding: EdgeInsets.zero,
          backgroundColor: isGA ? Colors.blue.withValues(alpha: 0.1) : Colors.green.withValues(alpha: 0.1),
          side: BorderSide(color: isGA ? Colors.blue : Colors.green, width: 0.5),
        ),
      ),
    );
  }

  /// Guess device type from individual address pattern.
  String? _guessDeviceType(String addr) {
    final parts = addr.split('.');
    if (parts.length != 3) return null;

    final area = int.tryParse(parts[0]) ?? 0;
    final line = int.tryParse(parts[1]) ?? 0;

    // Common KNX conventions
    if (area == 1 && line == 1) return 'Actuator';
    if (area == 1 && line == 2) return 'Wall Switch';
    if (area == 0 && line == 0) return 'System';

    return null;
  }

  Widget _buildEmptyState(BuildContext context) {
    final theme = Theme.of(context);

    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.radar_outlined,
              size: 64,
              color: theme.colorScheme.primary.withValues(alpha: 0.5),
            ),
            const SizedBox(height: 16),
            Text('No KNX traffic detected yet', style: theme.textTheme.titleMedium),
            const SizedBox(height: 8),
            Text(
              'The system passively listens to the KNX bus and records\n'
              'all group addresses and device addresses it sees.\n\n'
              'Try operating a light switch or sensor to generate traffic.',
              textAlign: TextAlign.center,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// Helper class for list items
class _ListItem {
  final String itemType; // 'ga' or 'device'
  final String address;
  final String lastSeenAgo;
  final int messageCount;
  final KnxDomain domain;
  final String? extra;

  _ListItem({
    required this.itemType,
    required this.address,
    required this.lastSeenAgo,
    required this.messageCount,
    required this.domain,
    this.extra,
  });
}

/// Compact summary card widget
class _SummaryCard extends StatelessWidget {
  final IconData icon;
  final String label;
  final String value;
  final String subValue;
  final Color color;

  const _SummaryCard({
    required this.icon,
    required this.label,
    required this.value,
    required this.subValue,
    required this.color,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(icon, size: 16, color: color),
                const SizedBox(width: 6),
                Expanded(
                  child: Text(
                    label,
                    style: theme.textTheme.labelSmall?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 8),
            Text(
              value,
              style: theme.textTheme.headlineMedium?.copyWith(
                fontWeight: FontWeight.bold,
                color: color,
              ),
            ),
            Text(
              subValue,
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
