import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../providers/connection_provider.dart';
import '../services/websocket_service.dart';

/// Shows connection status: hidden when connected, amber when connecting,
/// red when disconnected.
class ConnectionIndicator extends ConsumerWidget {
  const ConnectionIndicator({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final statusAsync = ref.watch(connectionStatusProvider);

    return statusAsync.when(
      data: (status) => _buildIndicator(context, status),
      loading: () => _buildIndicator(context, WSConnectionStatus.connecting),
      error: (e, s) => _buildIndicator(context, WSConnectionStatus.disconnected),
    );
  }

  Widget _buildIndicator(BuildContext context, WSConnectionStatus status) {
    if (status == WSConnectionStatus.connected) {
      return const SizedBox.shrink();
    }

    final isConnecting = status == WSConnectionStatus.connecting;
    final colour = isConnecting ? Colors.amber : Colors.red;
    final label = isConnecting ? 'Connecting...' : 'Offline';

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      decoration: BoxDecoration(
        color: colour.withValues(alpha: 0.2),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: colour.withValues(alpha: 0.5)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 8,
            height: 8,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              color: colour,
            ),
          ),
          const SizedBox(width: 6),
          Text(
            label,
            style: TextStyle(
              fontSize: 12,
              color: colour,
              fontWeight: FontWeight.w500,
            ),
          ),
        ],
      ),
    );
  }
}
