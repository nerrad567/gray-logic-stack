import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/metrics.dart';
import '../providers/auth_provider.dart';

/// Provider for fetching system metrics from the API.
///
/// Auto-disposes when no longer in use, and can be refreshed
/// by invalidating the provider.
final metricsProvider = FutureProvider.autoDispose<SystemMetrics>((ref) async {
  final apiClient = ref.watch(apiClientProvider);
  return apiClient.getMetrics();
});
