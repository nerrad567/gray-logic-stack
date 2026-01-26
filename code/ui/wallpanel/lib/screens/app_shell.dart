import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/auth.dart';
import '../providers/auth_provider.dart';
import '../providers/connection_provider.dart';
import '../providers/location_provider.dart';
import '../widgets/room_nav_bar.dart';
import 'config_screen.dart';
import 'onboarding_screen.dart';
import 'room_view.dart';

/// Top-level shell that gates the app behind authentication.
///
/// Flow:
/// 1. On startup, attempts to restore a previous session (stored token).
/// 2. If no valid session → shows ConfigScreen (login + setup).
/// 3. Once authenticated → connects WebSocket, shows RoomView.
class AppShell extends ConsumerStatefulWidget {
  const AppShell({super.key});

  @override
  ConsumerState<AppShell> createState() => _AppShellState();
}

class _AppShellState extends ConsumerState<AppShell> {
  bool _initializing = true;

  @override
  void initState() {
    super.initState();
    _restoreSession();
  }

  Future<void> _restoreSession() async {
    await ref.read(authProvider.notifier).restoreSession();
    if (mounted) {
      setState(() => _initializing = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_initializing) {
      return const Scaffold(
        body: Center(child: CircularProgressIndicator()),
      );
    }

    final authState = ref.watch(authProvider);

    if (authState.status == AuthStatus.authenticated) {
      return _AuthenticatedShell(authState: authState);
    }

    return const ConfigScreen();
  }
}

/// Wraps RoomView with WebSocket connection and room navigation.
/// Loads location data, connects WebSocket, shows room nav bar + device view.
class _AuthenticatedShell extends ConsumerStatefulWidget {
  final AuthState authState;

  const _AuthenticatedShell({required this.authState});

  @override
  ConsumerState<_AuthenticatedShell> createState() =>
      _AuthenticatedShellState();
}

class _AuthenticatedShellState extends ConsumerState<_AuthenticatedShell> {
  bool _initialized = false;
  bool _connecting = false;
  bool _isEmpty = false;

  @override
  void initState() {
    super.initState();
    _initConnection();
  }

  Future<void> _initConnection() async {
    // Guard against double-initialization on widget rebuild.
    if (_connecting) return;
    _connecting = true;

    // Establish WebSocket connection
    ref.read(connectionManagerProvider).connect();

    // Load location data (areas + rooms)
    await ref.read(locationDataProvider.notifier).load();

    // Check if we have any devices - if not, show onboarding
    final apiClient = ref.read(apiClientProvider);
    final deviceResponse = await apiClient.getDevices();
    final hasDevices = deviceResponse.count > 0;
    final locationData = ref.read(locationDataProvider).valueOrNull;

    if (!hasDevices) {
      // No devices configured - show onboarding
      if (mounted) {
        setState(() {
          _isEmpty = true;
          _initialized = true;
        });
      }
      return;
    }

    // Determine initial room
    final tokenStorage = ref.read(tokenStorageProvider);
    final defaultRoom = await tokenStorage.getRoomId();

    String? initialRoom;
    if (defaultRoom != null && defaultRoom.isNotEmpty) {
      initialRoom = defaultRoom;
    } else if (locationData != null) {
      final sorted = locationData.sortedRooms;
      if (sorted.isNotEmpty) {
        initialRoom = sorted.first.id;
      }
    }

    // If no rooms but we have devices, use a placeholder
    // (devices can exist without room assignment)
    if (initialRoom == null && hasDevices) {
      initialRoom = '__all__'; // Special value for "all devices" view
    }

    if (mounted && initialRoom != null) {
      ref.read(selectedRoomProvider.notifier).state = initialRoom;
    }

    if (mounted) {
      setState(() {
        _isEmpty = false;
        _initialized = true;
      });
    }
  }

  /// Reload data after import or other changes.
  Future<void> _refresh() async {
    setState(() {
      _initialized = false;
      _connecting = false;
    });
    await _initConnection();
  }

  @override
  void dispose() {
    // Do NOT disconnect here — ConnectionManager is a global singleton.
    // It will be disposed when the provider container is disposed (app exit).
    // Disconnecting here would kill the WS during widget rebuilds.
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (!_initialized) {
      return const Scaffold(
        body: Center(child: CircularProgressIndicator()),
      );
    }

    // Show onboarding if no devices configured
    if (_isEmpty) {
      return OnboardingScreen(onRefresh: _refresh);
    }

    final selectedRoom = ref.watch(selectedRoomProvider);

    if (selectedRoom == null) {
      return const Scaffold(
        body: Center(child: CircularProgressIndicator()),
      );
    }

    return Scaffold(
      body: SafeArea(
        child: Column(
          children: [
            const RoomNavBar(),
            Expanded(
              child: RoomView(key: ValueKey(selectedRoom), roomId: selectedRoom),
            ),
          ],
        ),
      ),
    );
  }
}
