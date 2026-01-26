import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/ets_import.dart';
import '../services/api_client.dart';
import 'auth_provider.dart';

/// State for the ETS import flow.
enum ETSImportStatus {
  idle,
  uploading,
  parsing,
  previewing,
  importing,
  completed,
  error,
}

/// State class for the ETS import process.
class ETSImportState {
  final ETSImportStatus status;
  final ETSParseResult? parseResult;
  final ETSImportResponse? importResponse;
  final String? errorMessage;
  final double uploadProgress;

  const ETSImportState({
    this.status = ETSImportStatus.idle,
    this.parseResult,
    this.importResponse,
    this.errorMessage,
    this.uploadProgress = 0.0,
  });

  ETSImportState copyWith({
    ETSImportStatus? status,
    ETSParseResult? parseResult,
    ETSImportResponse? importResponse,
    String? errorMessage,
    double? uploadProgress,
  }) {
    return ETSImportState(
      status: status ?? this.status,
      parseResult: parseResult ?? this.parseResult,
      importResponse: importResponse ?? this.importResponse,
      errorMessage: errorMessage,
      uploadProgress: uploadProgress ?? this.uploadProgress,
    );
  }

  /// Whether we're in a loading state.
  bool get isLoading =>
      status == ETSImportStatus.uploading ||
      status == ETSImportStatus.parsing ||
      status == ETSImportStatus.importing;

  /// Whether we have parsed results to preview.
  bool get hasParsedData => parseResult != null;

  /// Number of devices selected for import.
  int get selectedDeviceCount {
    if (parseResult == null) return 0;
    return parseResult!.devices.where((d) => d.import).length;
  }

  /// Total detected devices.
  int get totalDeviceCount => parseResult?.devices.length ?? 0;
}

/// Provider for ETS import state management.
final etsImportProvider =
    StateNotifierProvider<ETSImportNotifier, ETSImportState>((ref) {
  return ETSImportNotifier(ref);
});

/// Manages the ETS import flow: upload -> parse -> preview -> import.
class ETSImportNotifier extends StateNotifier<ETSImportState> {
  final Ref _ref;

  ETSImportNotifier(this._ref) : super(const ETSImportState());

  ApiClient get _apiClient => _ref.read(apiClientProvider);

  /// Upload and parse an ETS project file.
  Future<void> uploadAndParse(List<int> fileBytes, String filename) async {
    state = state.copyWith(
      status: ETSImportStatus.uploading,
      uploadProgress: 0.0,
      errorMessage: null,
    );

    try {
      // Simulate upload progress (actual progress would need Dio callback)
      state = state.copyWith(uploadProgress: 0.5);

      state = state.copyWith(status: ETSImportStatus.parsing);

      final result = await _apiClient.parseETSFile(fileBytes, filename);

      state = state.copyWith(
        status: ETSImportStatus.previewing,
        parseResult: result,
        uploadProgress: 1.0,
      );
    } catch (e) {
      state = state.copyWith(
        status: ETSImportStatus.error,
        errorMessage: _formatError(e),
      );
    }
  }

  /// Toggle whether a device should be imported.
  void toggleDeviceImport(int index) {
    if (state.parseResult == null) return;

    final devices = List<ETSDetectedDevice>.from(state.parseResult!.devices);
    devices[index].import = !devices[index].import;

    // Create a new parse result with updated devices
    state = state.copyWith(
      parseResult: ETSParseResult(
        importId: state.parseResult!.importId,
        format: state.parseResult!.format,
        sourceFile: state.parseResult!.sourceFile,
        devices: devices,
        unmappedAddresses: state.parseResult!.unmappedAddresses,
        warnings: state.parseResult!.warnings,
        statistics: state.parseResult!.statistics,
      ),
    );
  }

  /// Update the edited ID for a device.
  void updateDeviceId(int index, String newId) {
    if (state.parseResult == null) return;

    final devices = List<ETSDetectedDevice>.from(state.parseResult!.devices);
    devices[index].editedId = newId;

    state = state.copyWith(
      parseResult: ETSParseResult(
        importId: state.parseResult!.importId,
        format: state.parseResult!.format,
        sourceFile: state.parseResult!.sourceFile,
        devices: devices,
        unmappedAddresses: state.parseResult!.unmappedAddresses,
        warnings: state.parseResult!.warnings,
        statistics: state.parseResult!.statistics,
      ),
    );
  }

  /// Update the edited name for a device.
  void updateDeviceName(int index, String newName) {
    if (state.parseResult == null) return;

    final devices = List<ETSDetectedDevice>.from(state.parseResult!.devices);
    devices[index].editedName = newName;

    state = state.copyWith(
      parseResult: ETSParseResult(
        importId: state.parseResult!.importId,
        format: state.parseResult!.format,
        sourceFile: state.parseResult!.sourceFile,
        devices: devices,
        unmappedAddresses: state.parseResult!.unmappedAddresses,
        warnings: state.parseResult!.warnings,
        statistics: state.parseResult!.statistics,
      ),
    );
  }

  /// Update the room assignment for a device.
  void updateDeviceRoom(int index, String? roomId) {
    if (state.parseResult == null) return;

    final devices = List<ETSDetectedDevice>.from(state.parseResult!.devices);
    devices[index].selectedRoomId = roomId;

    state = state.copyWith(
      parseResult: ETSParseResult(
        importId: state.parseResult!.importId,
        format: state.parseResult!.format,
        sourceFile: state.parseResult!.sourceFile,
        devices: devices,
        unmappedAddresses: state.parseResult!.unmappedAddresses,
        warnings: state.parseResult!.warnings,
        statistics: state.parseResult!.statistics,
      ),
    );
  }

  /// Select or deselect all devices.
  void selectAll(bool selected) {
    if (state.parseResult == null) return;

    final devices = List<ETSDetectedDevice>.from(state.parseResult!.devices);
    for (final device in devices) {
      device.import = selected;
    }

    state = state.copyWith(
      parseResult: ETSParseResult(
        importId: state.parseResult!.importId,
        format: state.parseResult!.format,
        sourceFile: state.parseResult!.sourceFile,
        devices: devices,
        unmappedAddresses: state.parseResult!.unmappedAddresses,
        warnings: state.parseResult!.warnings,
        statistics: state.parseResult!.statistics,
      ),
    );
  }

  /// Import the selected devices.
  Future<void> importDevices({
    bool skipExisting = false,
    bool updateExisting = false,
    bool dryRun = false,
  }) async {
    if (state.parseResult == null) return;

    state = state.copyWith(
      status: ETSImportStatus.importing,
      errorMessage: null,
    );

    try {
      final request = ETSImportRequest(
        importId: state.parseResult!.importId,
        devices: state.parseResult!.devices.where((d) => d.import).toList(),
        options: ETSImportOptions(
          skipExisting: skipExisting,
          updateExisting: updateExisting,
          dryRun: dryRun,
        ),
      );

      final response = await _apiClient.importETSDevices(request);

      state = state.copyWith(
        status: ETSImportStatus.completed,
        importResponse: response,
      );
    } catch (e) {
      state = state.copyWith(
        status: ETSImportStatus.error,
        errorMessage: _formatError(e),
      );
    }
  }

  /// Reset the import state to start over.
  void reset() {
    state = const ETSImportState();
  }

  /// Go back to preview state (from error or completed).
  void backToPreview() {
    if (state.parseResult != null) {
      state = state.copyWith(
        status: ETSImportStatus.previewing,
        errorMessage: null,
        importResponse: null,
      );
    } else {
      reset();
    }
  }

  String _formatError(dynamic error) {
    final msg = error.toString();
    if (msg.contains('file_too_large')) return 'File exceeds 50MB limit';
    if (msg.contains('invalid file format')) return 'Invalid file format';
    if (msg.contains('corrupt archive')) return 'Corrupt .knxproj file';
    if (msg.contains('no group addresses')) return 'No group addresses found';
    if (msg.contains('SocketException')) return 'Cannot reach Core server';
    if (msg.contains('TimeoutException')) return 'Upload timed out';
    return 'Import failed: ${error.runtimeType}';
  }
}
