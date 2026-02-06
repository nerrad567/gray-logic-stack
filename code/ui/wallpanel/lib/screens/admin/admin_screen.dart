import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../providers/auth_provider.dart';
import 'audit_tab.dart';
import 'devices_tab.dart';
import 'discovery_tab.dart';
import 'import_tab.dart';
import 'locations_tab.dart';
import 'metrics_tab.dart';
import 'scenes_tab.dart';
import 'site_tab.dart';
import 'system_tab.dart';
import 'users_tab.dart';

/// Admin screen with tabbed interface for system management.
/// Tabs are filtered by the caller's role:
/// - Admin: all tabs except System
/// - Owner: all tabs including System (factory reset)
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
  late List<_AdminTab> _visibleTabs;

  @override
  void initState() {
    super.initState();
    final identity = ref.read(identityProvider);
    final isOwner = identity?.isOwner ?? false;

    _visibleTabs = _buildTabList(isOwner);
    _tabController = TabController(length: _visibleTabs.length, vsync: this);
  }

  List<_AdminTab> _buildTabList(bool isOwner) {
    final tabs = <_AdminTab>[
      _AdminTab(
        tab: const Tab(icon: Icon(Icons.analytics_outlined), text: 'Metrics'),
        body: const MetricsTab(),
      ),
      _AdminTab(
        tab: const Tab(icon: Icon(Icons.radar_outlined), text: 'Discovery'),
        body: const DiscoveryTab(),
      ),
      _AdminTab(
        tab: const Tab(icon: Icon(Icons.devices_outlined), text: 'Devices'),
        body: const DevicesTab(),
      ),
      _AdminTab(
        tab: const Tab(icon: Icon(Icons.theaters_outlined), text: 'Scenes'),
        body: const ScenesTab(),
      ),
      _AdminTab(
        tab: const Tab(icon: Icon(Icons.people_outline), text: 'Users'),
        body: const UsersTab(),
      ),
      _AdminTab(
        tab: const Tab(icon: Icon(Icons.location_on_outlined), text: 'Locations'),
        body: const LocationsTab(),
      ),
      _AdminTab(
        tab: const Tab(icon: Icon(Icons.upload_file_outlined), text: 'Import'),
        body: ImportTab(onImportComplete: widget.onRefresh),
      ),
      _AdminTab(
        tab: const Tab(icon: Icon(Icons.home_work_outlined), text: 'Site'),
        body: const SiteTab(),
      ),
      _AdminTab(
        tab: const Tab(icon: Icon(Icons.receipt_long_outlined), text: 'Logs'),
        body: const AuditTab(),
      ),
    ];

    // System tab (factory reset) — owner only
    if (isOwner) {
      tabs.add(_AdminTab(
        tab: const Tab(icon: Icon(Icons.restart_alt_outlined), text: 'System'),
        body: SystemTab(onReset: widget.onRefresh),
      ));
    }

    return tabs;
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
          isScrollable: true,
          tabAlignment: TabAlignment.start,
          tabs: _visibleTabs.map((t) => t.tab).toList(),
          labelColor: theme.colorScheme.primary,
          unselectedLabelColor: theme.colorScheme.onSurfaceVariant,
          indicatorColor: theme.colorScheme.primary,
        ),
      ),
      body: TabBarView(
        controller: _tabController,
        children: _visibleTabs.map((t) => t.body).toList(),
      ),
    );
  }
}

/// Helper class grouping a tab header with its body widget.
class _AdminTab {
  final Tab tab;
  final Widget body;

  const _AdminTab({required this.tab, required this.body});
}
