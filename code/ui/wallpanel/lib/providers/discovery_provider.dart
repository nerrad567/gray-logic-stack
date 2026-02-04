import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/discovery.dart';
import '../models/device.dart';
import 'auth_provider.dart';

/// A GA suggestion combining discovery data with device assignment info.
class GASuggestion {
  /// The group address string, e.g. "1/2/3".
  final String groupAddress;

  /// Number of messages seen on the bus (0 if not discovered).
  final int messageCount;

  /// Human-readable "last seen" time, e.g. "2m ago".
  final String lastSeenAgo;

  /// Whether this GA responded to a read request.
  final bool hasReadResponse;

  /// If assigned, the device name that uses this GA.
  final String? assignedDeviceName;

  /// If assigned, the function key within the device address map.
  final String? assignedFunction;

  /// True if this GA was seen on the bus (vs only known from device config).
  bool get isDiscovered => messageCount > 0;

  /// True if this GA is already assigned to a device.
  bool get isAssigned => assignedDeviceName != null;

  const GASuggestion({
    required this.groupAddress,
    this.messageCount = 0,
    this.lastSeenAgo = '',
    this.hasReadResponse = false,
    this.assignedDeviceName,
    this.assignedFunction,
  });
}

/// Provides GA suggestions by merging Discovery data with existing device
/// address assignments. Used by the Add Device form for autocomplete.
///
/// Results are sorted: unassigned+active first (by message count desc),
/// then assigned GAs (greyed out in UI).
final gaSuggestionsProvider =
    FutureProvider.autoDispose<List<GASuggestion>>((ref) async {
  final api = ref.watch(apiClientProvider);

  // Fetch both datasets in parallel.
  final results = await Future.wait([
    api.getDiscovery(),
    api.getDevices(),
  ]);

  final discovery = results[0] as DiscoveryData;
  final deviceResponse = results[1] as DeviceListResponse;

  // Build a map of GA → (deviceName, functionKey) from all existing devices.
  final assignedGAs = <String, _Assignment>{};
  for (final device in deviceResponse.devices) {
    for (final entry in device.address.entries) {
      final ga = entry.value?.toString() ?? '';
      if (ga.isNotEmpty) {
        assignedGAs[ga] = _Assignment(device.name, entry.key);
      }
    }
  }

  // Build suggestion list from discovered GAs.
  final suggestions = <String, GASuggestion>{};

  for (final ga in discovery.groupAddresses) {
    final assignment = assignedGAs[ga.groupAddress];
    suggestions[ga.groupAddress] = GASuggestion(
      groupAddress: ga.groupAddress,
      messageCount: ga.messageCount,
      lastSeenAgo: ga.lastSeenAgo,
      hasReadResponse: ga.hasReadResponse,
      assignedDeviceName: assignment?.deviceName,
      assignedFunction: assignment?.functionKey,
    );
  }

  // Also include assigned GAs that weren't in discovery (device has a GA
  // that hasn't been seen on the bus — still useful to show as "taken").
  for (final entry in assignedGAs.entries) {
    if (!suggestions.containsKey(entry.key)) {
      suggestions[entry.key] = GASuggestion(
        groupAddress: entry.key,
        assignedDeviceName: entry.value.deviceName,
        assignedFunction: entry.value.functionKey,
      );
    }
  }

  // Sort: unassigned+active first (high message count), then assigned.
  final sorted = suggestions.values.toList()
    ..sort((a, b) {
      // Unassigned before assigned.
      if (a.isAssigned != b.isAssigned) {
        return a.isAssigned ? 1 : -1;
      }
      // Within same assignment status, sort by message count descending.
      return b.messageCount.compareTo(a.messageCount);
    });

  return sorted;
});

/// Internal helper for tracking GA-to-device assignments.
class _Assignment {
  final String deviceName;
  final String functionKey;
  const _Assignment(this.deviceName, this.functionKey);
}
