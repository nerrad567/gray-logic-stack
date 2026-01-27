import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'devices_tab.dart';
import 'discovery_tab.dart';
import 'import_tab.dart';
import 'metrics_tab.dart';

/// Admin screen with tabbed interface for system management.
///
/// Tabs:
/// - Metrics: System monitoring dashboard
/// - Discovery: Passive KNX bus scan results
/// - Devices: Device list with edit/delete capabilities
/// - Import: ETS import functionality
class AdminScreen extends ConsumerStatefulWidget {
  /// Optional callback when returning to refresh main view.
  final VoidCallback? onRefresh;

  const AdminScreen({super.key, this.onRefresh});

  @override
  ConsumerState<AdminScreen> createState() => _AdminScreenState();
}

class _AdminScreenState extends ConsumerState<AdminScreen>
    with SingleTickerProviderStateMixin {
  late TabController _tabController;

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 4, vsync: this);
  }

  @override
  void dispose() {
    _tabController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Admin'),
        leading: IconButton(
          icon: const Icon(Icons.close),
          onPressed: () {
            widget.onRefresh?.call();
            Navigator.of(context).pop();
          },
        ),
        bottom: TabBar(
          controller: _tabController,
          tabs: const [
            Tab(
              icon: Icon(Icons.analytics_outlined),
              text: 'Metrics',
            ),
            Tab(
              icon: Icon(Icons.radar_outlined),
              text: 'Discovery',
            ),
            Tab(
              icon: Icon(Icons.devices_outlined),
              text: 'Devices',
            ),
            Tab(
              icon: Icon(Icons.upload_file_outlined),
              text: 'Import',
            ),
          ],
          labelColor: theme.colorScheme.primary,
          unselectedLabelColor: theme.colorScheme.onSurfaceVariant,
          indicatorColor: theme.colorScheme.primary,
        ),
      ),
      body: TabBarView(
        controller: _tabController,
        children: [
          const MetricsTab(),
          const DiscoveryTab(),
          const DevicesTab(),
          ImportTab(onImportComplete: widget.onRefresh),
        ],
      ),
    );
  }
}
