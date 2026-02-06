import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/panel.dart';
import '../repositories/panel_repository.dart';
import 'auth_provider.dart';

/// Provides the PanelRepository.
final panelRepositoryProvider = Provider<PanelRepository>((ref) {
  return PanelRepository(apiClient: ref.watch(apiClientProvider));
});

/// Provides all panels (admin view).
final allPanelsProvider =
    NotifierProvider<AllPanelsNotifier, AsyncValue<List<Panel>>>(
  AllPanelsNotifier.new,
);

class AllPanelsNotifier extends Notifier<AsyncValue<List<Panel>>> {
  @override
  AsyncValue<List<Panel>> build() => const AsyncValue.loading();

  PanelRepository get _panelRepo => ref.read(panelRepositoryProvider);

  /// Load all panels.
  Future<void> load() async {
    state = const AsyncValue.loading();
    try {
      final panels = await _panelRepo.getPanels();
      panels.sort((a, b) => a.name.compareTo(b.name));
      state = AsyncValue.data(panels);
    } catch (e, st) {
      state = AsyncValue.error(e, st);
    }
  }

  /// Create a new panel and refresh list. Returns the full response with token.
  Future<PanelCreateResponse> createPanel(Map<String, dynamic> data) async {
    final response = await _panelRepo.createPanel(data);
    await load();
    return response;
  }

  /// Update a panel and refresh list.
  Future<Panel> updatePanel(String id, Map<String, dynamic> data) async {
    final panel = await _panelRepo.updatePanel(id, data);
    await load();
    return panel;
  }

  /// Delete a panel and refresh list.
  Future<void> deletePanel(String id) async {
    await _panelRepo.deletePanel(id);
    await load();
  }
}
