import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/device.dart';
import '../providers/device_provider.dart';

/// A tile widget for on/off devices (light switches, relays).
/// Shows device name + large toggle button. Displays a subtle pulsing opacity
/// while awaiting bridge/WebSocket confirmation of state change.
class SwitchTile extends ConsumerWidget {
  final Device device;

  const SwitchTile({super.key, required this.device});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final isOn = device.isOn;
    final isOnline = device.isOnline;
    final isPending = ref.watch(pendingDevicesProvider).contains(device.id);
    final theme = Theme.of(context);
    final activeColour = theme.colorScheme.primary;

    return Card(
      child: InkWell(
        borderRadius: BorderRadius.circular(16),
        splashFactory: NoSplash.splashFactory,
        highlightColor: Colors.transparent,
        hoverColor: theme.colorScheme.primary.withValues(alpha: 0.05),
        mouseCursor: isOnline && !isPending
            ? SystemMouseCursors.click
            : SystemMouseCursors.basic,
        onTap: isOnline && !isPending
            ? () => ref.read(roomDevicesProvider.notifier).toggleDevice(device.id)
            : null,
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              // Device name
              Text(
                device.name,
                style: theme.textTheme.titleMedium,
                textAlign: TextAlign.center,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
              ),
              const Spacer(),
              // Toggle indicator — pulses opacity when pending, keeps same look
              isPending
                  ? _PendingIcon(isOn: isOn, activeColour: activeColour)
                  : _DeviceIcon(isOn: isOn, activeColour: activeColour, theme: theme),
              const Spacer(),
              // Status text
              Text(
                isPending
                    ? '...'
                    : !isOnline
                        ? 'Offline'
                        : (isOn ? 'ON' : 'OFF'),
                style: TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.w600,
                  color: isPending
                      ? activeColour
                      : !isOnline
                          ? Colors.grey.shade600
                          : isOn
                              ? activeColour
                              : Colors.grey.shade400,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// The normal device icon circle.
class _DeviceIcon extends StatelessWidget {
  final bool isOn;
  final Color activeColour;
  final ThemeData theme;

  const _DeviceIcon({
    required this.isOn,
    required this.activeColour,
    required this.theme,
  });

  @override
  Widget build(BuildContext context) {
    return AnimatedContainer(
      duration: const Duration(milliseconds: 200),
      width: 64,
      height: 64,
      decoration: BoxDecoration(
        shape: BoxShape.circle,
        color: isOn
            ? activeColour.withValues(alpha: 0.2)
            : theme.colorScheme.surface,
        border: Border.all(
          color: isOn ? activeColour : Colors.grey.shade600,
          width: 2,
        ),
      ),
      child: Icon(
        Icons.power_settings_new,
        size: 28,
        color: isOn ? activeColour : Colors.grey.shade500,
      ),
    );
  }
}

/// The same device icon but with a gentle opacity pulse to indicate pending.
/// Keeps the visual context (on/off state) visible while signalling activity.
class _PendingIcon extends StatefulWidget {
  final bool isOn;
  final Color activeColour;

  const _PendingIcon({required this.isOn, required this.activeColour});

  @override
  State<_PendingIcon> createState() => _PendingIconState();
}

class _PendingIconState extends State<_PendingIcon>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;
  late final Animation<double> _opacity;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 800),
    )..repeat(reverse: true);
    _opacity = Tween<double>(begin: 0.2, end: 1.0).animate(
      CurvedAnimation(parent: _controller, curve: Curves.easeInOut),
    );
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return FadeTransition(
      opacity: _opacity,
      child: Container(
        width: 64,
        height: 64,
        decoration: BoxDecoration(
          shape: BoxShape.circle,
          color: widget.isOn
              ? widget.activeColour.withValues(alpha: 0.2)
              : theme.colorScheme.surface,
          border: Border.all(
            color: widget.activeColour.withValues(alpha: 0.6),
            width: 2,
          ),
        ),
        child: Icon(
          Icons.power_settings_new,
          size: 28,
          color: widget.activeColour.withValues(alpha: 0.7),
        ),
      ),
    );
  }
}
