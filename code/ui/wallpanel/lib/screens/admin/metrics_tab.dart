import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../models/metrics.dart';
import '../../providers/metrics_provider.dart';

/// System metrics dashboard tab.
///
/// Displays real-time system health and statistics with auto-refresh.
class MetricsTab extends ConsumerStatefulWidget {
  const MetricsTab({super.key});

  @override
  ConsumerState<MetricsTab> createState() => _MetricsTabState();
}

class _MetricsTabState extends ConsumerState<MetricsTab> {
  Timer? _refreshTimer;

  @override
  void initState() {
    super.initState();
    // Auto-refresh every 5 seconds
    _refreshTimer = Timer.periodic(const Duration(seconds: 5), (_) {
      if (mounted) {
        ref.invalidate(metricsProvider);
      }
    });
  }

  @override
  void dispose() {
    _refreshTimer?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final metricsAsync = ref.watch(metricsProvider);

    return metricsAsync.when(
      data: (metrics) => _MetricsContent(metrics: metrics),
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (error, _) => _ErrorView(
        error: error.toString(),
        onRetry: () => ref.invalidate(metricsProvider),
      ),
    );
  }
}

class _MetricsContent extends StatelessWidget {
  final SystemMetrics metrics;

  const _MetricsContent({required this.metrics});

  @override
  Widget build(BuildContext context) {
    return RefreshIndicator(
      onRefresh: () async {
        // Manual pull-to-refresh is handled by parent
      },
      child: SingleChildScrollView(
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // System overview card
            _SystemOverviewCard(metrics: metrics),
            const SizedBox(height: 16),

            // Connections status
            _ConnectionsCard(metrics: metrics),
            const SizedBox(height: 16),

            // KNX Bridge (if available)
            if (metrics.knxBridge != null) ...[
              _KNXBridgeCard(knx: metrics.knxBridge!),
              const SizedBox(height: 16),
            ],

            // Device statistics
            _DevicesCard(devices: metrics.devices),
            const SizedBox(height: 16),

            // Runtime & Database
            _RuntimeCard(
              runtime: metrics.runtime,
              database: metrics.database,
            ),
          ],
        ),
      ),
    );
  }
}

class _SystemOverviewCard extends StatelessWidget {
  final SystemMetrics metrics;

  const _SystemOverviewCard({required this.metrics});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(Icons.info_outline, color: theme.colorScheme.primary),
                const SizedBox(width: 8),
                Text(
                  'System Overview',
                  style: theme.textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 16),
            Row(
              children: [
                Expanded(
                  child: _MetricItem(
                    label: 'Version',
                    value: metrics.version,
                    icon: Icons.tag,
                  ),
                ),
                Expanded(
                  child: _MetricItem(
                    label: 'Uptime',
                    value: metrics.uptimeFormatted,
                    icon: Icons.schedule,
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _ConnectionsCard extends StatelessWidget {
  final SystemMetrics metrics;

  const _ConnectionsCard({required this.metrics});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(Icons.cable_outlined, color: theme.colorScheme.primary),
                const SizedBox(width: 8),
                Text(
                  'Connections',
                  style: theme.textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 16),
            Row(
              children: [
                Expanded(
                  child: _StatusItem(
                    label: 'MQTT Broker',
                    isConnected: metrics.mqtt.connected,
                  ),
                ),
                Expanded(
                  child: _MetricItem(
                    label: 'WebSocket Clients',
                    value: '${metrics.websocket.connectedClients}',
                    icon: Icons.people_outline,
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _KNXBridgeCard extends StatelessWidget {
  final KNXBridgeMetrics knx;

  const _KNXBridgeCard({required this.knx});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(Icons.router_outlined, color: theme.colorScheme.primary),
                const SizedBox(width: 8),
                Text(
                  'KNX Bridge',
                  style: theme.textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                ),
                const Spacer(),
                _StatusBadge(
                  label: knx.status,
                  isHealthy: knx.connected,
                ),
              ],
            ),
            const SizedBox(height: 16),
            Row(
              children: [
                Expanded(
                  child: _MetricItem(
                    label: 'Telegrams TX',
                    value: _formatNumber(knx.telegramsTx),
                    icon: Icons.arrow_upward,
                  ),
                ),
                Expanded(
                  child: _MetricItem(
                    label: 'Telegrams RX',
                    value: _formatNumber(knx.telegramsRx),
                    icon: Icons.arrow_downward,
                  ),
                ),
                Expanded(
                  child: _MetricItem(
                    label: 'Devices',
                    value: '${knx.devicesManaged}',
                    icon: Icons.devices,
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  String _formatNumber(int n) {
    if (n >= 1000000) {
      return '${(n / 1000000).toStringAsFixed(1)}M';
    } else if (n >= 1000) {
      return '${(n / 1000).toStringAsFixed(1)}K';
    }
    return '$n';
  }
}

class _DevicesCard extends StatelessWidget {
  final DeviceMetrics devices;

  const _DevicesCard({required this.devices});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(Icons.devices_outlined, color: theme.colorScheme.primary),
                const SizedBox(width: 8),
                Text(
                  'Devices',
                  style: theme.textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                ),
                const Spacer(),
                Text(
                  '${devices.total} total',
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 16),

            // Health status row
            if (devices.byHealth.isNotEmpty) ...[
              Text(
                'By Health',
                style: theme.textTheme.labelMedium?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 8),
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: devices.byHealth.entries.map((e) {
                  return _StatChip(
                    label: e.key,
                    value: e.value,
                    color: _healthColor(e.key),
                  );
                }).toList(),
              ),
              const SizedBox(height: 12),
            ],

            // Domain row
            if (devices.byDomain.isNotEmpty) ...[
              Text(
                'By Domain',
                style: theme.textTheme.labelMedium?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 8),
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: devices.byDomain.entries.map((e) {
                  return _StatChip(
                    label: e.key,
                    value: e.value,
                    color: _domainColor(e.key, theme),
                  );
                }).toList(),
              ),
            ],
          ],
        ),
      ),
    );
  }

  Color _healthColor(String health) {
    switch (health.toLowerCase()) {
      case 'online':
        return Colors.green;
      case 'offline':
        return Colors.red;
      case 'degraded':
        return Colors.orange;
      default:
        return Colors.grey;
    }
  }

  Color _domainColor(String domain, ThemeData theme) {
    switch (domain.toLowerCase()) {
      case 'lighting':
        return Colors.amber;
      case 'blinds':
        return Colors.blue;
      case 'climate':
        return Colors.teal;
      case 'sensor':
        return Colors.purple;
      default:
        return theme.colorScheme.primary;
    }
  }
}

class _RuntimeCard extends StatelessWidget {
  final RuntimeMetrics runtime;
  final DatabaseMetrics database;

  const _RuntimeCard({required this.runtime, required this.database});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(Icons.memory_outlined, color: theme.colorScheme.primary),
                const SizedBox(width: 8),
                Text(
                  'Runtime',
                  style: theme.textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 16),
            Row(
              children: [
                Expanded(
                  child: _MetricItem(
                    label: 'Memory',
                    value: '${runtime.memoryAllocMB.toStringAsFixed(1)} MB',
                    icon: Icons.sd_storage_outlined,
                  ),
                ),
                Expanded(
                  child: _MetricItem(
                    label: 'Goroutines',
                    value: '${runtime.goroutines}',
                    icon: Icons.account_tree_outlined,
                  ),
                ),
                Expanded(
                  child: _MetricItem(
                    label: 'DB Connections',
                    value: '${database.openConnections}',
                    icon: Icons.storage_outlined,
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

// --- Helper Widgets ---

class _MetricItem extends StatelessWidget {
  final String label;
  final String value;
  final IconData icon;

  const _MetricItem({
    required this.label,
    required this.value,
    required this.icon,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 14, color: theme.colorScheme.onSurfaceVariant),
            const SizedBox(width: 4),
            Text(
              label,
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
        const SizedBox(height: 4),
        Text(
          value,
          style: theme.textTheme.titleMedium?.copyWith(
            fontWeight: FontWeight.bold,
          ),
        ),
      ],
    );
  }
}

class _StatusItem extends StatelessWidget {
  final String label;
  final bool isConnected;

  const _StatusItem({required this.label, required this.isConnected});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final color = isConnected ? Colors.green : Colors.red;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label,
          style: theme.textTheme.bodySmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
        const SizedBox(height: 4),
        Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              width: 8,
              height: 8,
              decoration: BoxDecoration(
                color: color,
                shape: BoxShape.circle,
              ),
            ),
            const SizedBox(width: 6),
            Text(
              isConnected ? 'Connected' : 'Disconnected',
              style: theme.textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.bold,
                color: color,
              ),
            ),
          ],
        ),
      ],
    );
  }
}

class _StatusBadge extends StatelessWidget {
  final String label;
  final bool isHealthy;

  const _StatusBadge({required this.label, required this.isHealthy});

  @override
  Widget build(BuildContext context) {
    final color = isHealthy ? Colors.green : Colors.red;

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.15),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 6,
            height: 6,
            decoration: BoxDecoration(
              color: color,
              shape: BoxShape.circle,
            ),
          ),
          const SizedBox(width: 4),
          Text(
            label,
            style: TextStyle(
              fontSize: 12,
              fontWeight: FontWeight.w600,
              color: color,
            ),
          ),
        ],
      ),
    );
  }
}

class _StatChip extends StatelessWidget {
  final String label;
  final int value;
  final Color color;

  const _StatChip({
    required this.label,
    required this.value,
    required this.color,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.15),
        borderRadius: BorderRadius.circular(16),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            '$value',
            style: TextStyle(
              fontWeight: FontWeight.bold,
              color: color,
            ),
          ),
          const SizedBox(width: 4),
          Text(
            label,
            style: TextStyle(
              fontSize: 12,
              color: color,
            ),
          ),
        ],
      ),
    );
  }
}

class _ErrorView extends StatelessWidget {
  final String error;
  final VoidCallback onRetry;

  const _ErrorView({required this.error, required this.onRetry});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.error_outline,
              size: 64,
              color: theme.colorScheme.error,
            ),
            const SizedBox(height: 16),
            Text(
              'Failed to load metrics',
              style: theme.textTheme.titleMedium,
            ),
            const SizedBox(height: 8),
            Text(
              error,
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 24),
            FilledButton.icon(
              onPressed: onRetry,
              icon: const Icon(Icons.refresh),
              label: const Text('Retry'),
            ),
          ],
        ),
      ),
    );
  }
}
