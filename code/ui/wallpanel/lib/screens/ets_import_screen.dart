import 'dart:async';
import 'dart:typed_data';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/foundation.dart' show kIsWeb;
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

// Web-only import (conditionally loaded)
// ignore: avoid_web_libraries_in_flutter
import 'dart:html' as html if (dart.library.io) 'dart:io';

import '../models/ets_import.dart';
import '../providers/ets_import_provider.dart';
import '../providers/location_provider.dart';

/// Screen for importing devices from ETS project files.
/// Handles the full flow: upload -> preview -> import -> results.
class ETSImportScreen extends ConsumerWidget {
  /// Optional callback invoked when import completes successfully.
  final VoidCallback? onImportComplete;

  const ETSImportScreen({super.key, this.onImportComplete});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(etsImportProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Import KNX Devices'),
        leading: IconButton(
          icon: const Icon(Icons.close),
          onPressed: () {
            ref.read(etsImportProvider.notifier).reset();
            Navigator.of(context).pop();
          },
        ),
      ),
      body: _buildBody(context, ref, state),
    );
  }

  Widget _buildBody(BuildContext context, WidgetRef ref, ETSImportState state) {
    switch (state.status) {
      case ETSImportStatus.idle:
        return _UploadView(ref: ref);

      case ETSImportStatus.uploading:
      case ETSImportStatus.parsing:
        return _LoadingView(state: state);

      case ETSImportStatus.previewing:
        return _PreviewView(ref: ref, state: state);

      case ETSImportStatus.importing:
        return const _LoadingView(
          state: ETSImportState(status: ETSImportStatus.importing),
        );

      case ETSImportStatus.completed:
        return _CompletedView(ref: ref, state: state, onImportComplete: onImportComplete);

      case ETSImportStatus.error:
        return _ErrorView(ref: ref, state: state);
    }
  }
}

/// Initial view for selecting and uploading a file.
class _UploadView extends StatelessWidget {
  final WidgetRef ref;

  const _UploadView({required this.ref});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Center(
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(32),
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 500),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(
                Icons.upload_file_outlined,
                size: 80,
                color: theme.colorScheme.primary,
              ),
              const SizedBox(height: 24),
              Text(
                'Import from ETS Project',
                style: theme.textTheme.headlineSmall,
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 12),
              Text(
                'Upload your ETS project file (.knxproj) or a group address export (.xml, .csv) to automatically detect and import devices.',
                style: theme.textTheme.bodyLarge?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 32),

              // Supported formats info
              Container(
                padding: const EdgeInsets.all(16),
                decoration: BoxDecoration(
                  color: theme.colorScheme.surfaceContainerHighest,
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Supported formats:',
                      style: theme.textTheme.titleSmall,
                    ),
                    const SizedBox(height: 8),
                    _FormatRow(
                      icon: Icons.folder_zip_outlined,
                      label: '.knxproj',
                      description: 'ETS project file (recommended)',
                    ),
                    const SizedBox(height: 4),
                    _FormatRow(
                      icon: Icons.code_outlined,
                      label: '.xml',
                      description: 'Group address export',
                    ),
                    const SizedBox(height: 4),
                    _FormatRow(
                      icon: Icons.table_chart_outlined,
                      label: '.csv',
                      description: 'Spreadsheet export',
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 32),

              // Upload button
              SizedBox(
                width: double.infinity,
                height: 56,
                child: FilledButton.icon(
                  onPressed: () => _pickFile(context),
                  icon: const Icon(Icons.file_open_outlined),
                  label: const Text('Select File'),
                ),
              ),
              const SizedBox(height: 16),
              Text(
                'Maximum file size: 50 MB',
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _pickFile(BuildContext context) async {
    try {
      if (kIsWeb) {
        await _pickFileWeb(context);
      } else {
        await _pickFileMobile(context);
      }
    } catch (e) {
      debugPrint('_pickFile error: $e');
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Error: $e')),
        );
      }
    }
  }

  /// Mobile file picker using file_picker package
  Future<void> _pickFileMobile(BuildContext context) async {
    final result = await FilePicker.platform.pickFiles(
      type: FileType.custom,
      allowedExtensions: ['knxproj', 'xml', 'csv'],
      withData: true,
    );

    if (result == null || result.files.isEmpty) return;

    final file = result.files.first;
    if (file.bytes == null) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Could not read file')),
        );
      }
      return;
    }

    await ref.read(etsImportProvider.notifier).uploadAndParse(
          file.bytes!,
          file.name,
        );
  }

  /// Web-specific file picker using dart:html
  Future<void> _pickFileWeb(BuildContext context) async {
    final completer = Completer<void>();

    final input = html.FileUploadInputElement()
      ..accept = '.knxproj,.xml,.csv';

    input.onChange.listen((event) async {
      final files = input.files;
      if (files == null || files.isEmpty) {
        completer.complete();
        return;
      }

      final file = files.first;
      final fileName = file.name;

      // Validate extension
      final extension = fileName.split('.').last.toLowerCase();
      final allowedExtensions = ['knxproj', 'xml', 'csv'];
      if (!allowedExtensions.contains(extension)) {
        if (context.mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(
              content: Text('Unsupported file type: .$extension. Please select a .knxproj, .xml, or .csv file.'),
            ),
          );
        }
        completer.complete();
        return;
      }

      // Read file bytes
      final reader = html.FileReader();
      reader.readAsArrayBuffer(file);

      reader.onLoadEnd.listen((event) async {
        final result = reader.result;
        if (result == null) {
          if (context.mounted) {
            ScaffoldMessenger.of(context).showSnackBar(
              const SnackBar(content: Text('Could not read file')),
            );
          }
          completer.complete();
          return;
        }

        final bytes = Uint8List.fromList(result as List<int>);

        await ref.read(etsImportProvider.notifier).uploadAndParse(
              bytes,
              fileName,
            );

        completer.complete();
      });

      reader.onError.listen((event) {
        if (context.mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('Error reading file')),
          );
        }
        completer.complete();
      });
    });

    // Trigger file dialog
    input.click();

    // Don't wait for completion - the dialog is async
  }
}

/// Format row for the supported formats list.
class _FormatRow extends StatelessWidget {
  final IconData icon;
  final String label;
  final String description;

  const _FormatRow({
    required this.icon,
    required this.label,
    required this.description,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Row(
      children: [
        Icon(icon, size: 20, color: theme.colorScheme.primary),
        const SizedBox(width: 8),
        Text(
          label,
          style: theme.textTheme.bodyMedium?.copyWith(
            fontWeight: FontWeight.w600,
          ),
        ),
        const SizedBox(width: 8),
        Expanded(
          child: Text(
            description,
            style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
        ),
      ],
    );
  }
}

/// Loading view during upload/parse.
class _LoadingView extends StatelessWidget {
  final ETSImportState state;

  const _LoadingView({required this.state});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final message = state.status == ETSImportStatus.uploading
        ? 'Uploading file...'
        : state.status == ETSImportStatus.parsing
            ? 'Analysing project...'
            : 'Importing devices...';

    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const SizedBox(
            width: 60,
            height: 60,
            child: CircularProgressIndicator(strokeWidth: 3),
          ),
          const SizedBox(height: 24),
          Text(
            message,
            style: theme.textTheme.titleMedium,
          ),
          if (state.status == ETSImportStatus.parsing) ...[
            const SizedBox(height: 8),
            Text(
              'Detecting devices from group addresses...',
              style: theme.textTheme.bodyMedium?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ],
      ),
    );
  }
}

/// Preview view showing detected devices before import.
class _PreviewView extends StatelessWidget {
  final WidgetRef ref;
  final ETSImportState state;

  const _PreviewView({required this.ref, required this.state});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final result = state.parseResult!;

    return Column(
      children: [
        // Statistics header
        Container(
          width: double.infinity,
          padding: const EdgeInsets.all(16),
          color: theme.colorScheme.surfaceContainerHighest,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Icon(
                    Icons.analytics_outlined,
                    color: theme.colorScheme.primary,
                  ),
                  const SizedBox(width: 8),
                  Text(
                    'Parse Results',
                    style: theme.textTheme.titleMedium,
                  ),
                  const Spacer(),
                  Text(
                    'Format: ${result.format.toUpperCase()}',
                    style: theme.textTheme.bodySmall,
                  ),
                ],
              ),
              const SizedBox(height: 12),
              Wrap(
                spacing: 16,
                runSpacing: 8,
                children: [
                  _StatChip(
                    label: 'Detected',
                    value: '${result.devices.length}',
                    icon: Icons.devices_outlined,
                  ),
                  _StatChip(
                    label: 'Selected',
                    value: '${state.selectedDeviceCount}',
                    icon: Icons.check_circle_outline,
                  ),
                  _StatChip(
                    label: 'Addresses',
                    value: '${result.statistics.totalGroupAddresses}',
                    icon: Icons.tag_outlined,
                  ),
                  if (result.warnings.isNotEmpty)
                    _StatChip(
                      label: 'Warnings',
                      value: '${result.warnings.length}',
                      icon: Icons.warning_amber_outlined,
                      isWarning: true,
                    ),
                ],
              ),
            ],
          ),
        ),

        // Select all row
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
          child: Row(
            children: [
              Checkbox(
                value: state.selectedDeviceCount == state.totalDeviceCount,
                tristate: true,
                onChanged: (value) {
                  ref.read(etsImportProvider.notifier).selectAll(value ?? false);
                },
              ),
              const Text('Select All'),
              const Spacer(),
              TextButton.icon(
                onPressed: () => _showWarnings(context, result.warnings),
                icon: const Icon(Icons.warning_amber_outlined, size: 18),
                label: Text('${result.warnings.length} warnings'),
              ),
            ],
          ),
        ),

        const Divider(height: 1),

        // Device list
        Expanded(
          child: ListView.builder(
            padding: const EdgeInsets.only(bottom: 100),
            itemCount: result.devices.length,
            itemBuilder: (context, index) {
              return _DevicePreviewTile(
                ref: ref,
                device: result.devices[index],
                index: index,
              );
            },
          ),
        ),

        // Import button
        Container(
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: theme.colorScheme.surface,
            boxShadow: [
              BoxShadow(
                color: Colors.black.withValues(alpha: 0.1),
                blurRadius: 8,
                offset: const Offset(0, -2),
              ),
            ],
          ),
          child: SafeArea(
            child: Row(
              children: [
                Expanded(
                  child: OutlinedButton(
                    onPressed: () => ref.read(etsImportProvider.notifier).reset(),
                    child: const Text('Cancel'),
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  flex: 2,
                  child: FilledButton.icon(
                    onPressed: state.selectedDeviceCount > 0
                        ? () => _showImportOptions(context, ref)
                        : null,
                    icon: const Icon(Icons.download_outlined),
                    label: Text('Import ${state.selectedDeviceCount} Devices'),
                  ),
                ),
              ],
            ),
          ),
        ),
      ],
    );
  }

  void _showWarnings(BuildContext context, List<ETSWarning> warnings) {
    showModalBottomSheet(
      context: context,
      builder: (context) {
        return DraggableScrollableSheet(
          initialChildSize: 0.5,
          minChildSize: 0.3,
          maxChildSize: 0.9,
          expand: false,
          builder: (context, scrollController) {
            return Column(
              children: [
                Padding(
                  padding: const EdgeInsets.all(16),
                  child: Text(
                    'Warnings (${warnings.length})',
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                ),
                const Divider(height: 1),
                Expanded(
                  child: ListView.builder(
                    controller: scrollController,
                    itemCount: warnings.length,
                    itemBuilder: (context, index) {
                      final warning = warnings[index];
                      return ListTile(
                        leading: const Icon(
                          Icons.warning_amber_outlined,
                          color: Colors.orange,
                        ),
                        title: Text(warning.message),
                        subtitle: warning.address != null
                            ? Text('Address: ${warning.address}')
                            : null,
                      );
                    },
                  ),
                ),
              ],
            );
          },
        );
      },
    );
  }

  void _showImportOptions(BuildContext context, WidgetRef ref) {
    showDialog(
      context: context,
      builder: (context) => _ImportOptionsDialog(ref: ref),
    );
  }
}

/// Chip for displaying statistics.
class _StatChip extends StatelessWidget {
  final String label;
  final String value;
  final IconData icon;
  final bool isWarning;

  const _StatChip({
    required this.label,
    required this.value,
    required this.icon,
    this.isWarning = false,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final color = isWarning ? Colors.orange : theme.colorScheme.primary;

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(16),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 16, color: color),
          const SizedBox(width: 4),
          Text(
            value,
            style: theme.textTheme.titleSmall?.copyWith(
              color: color,
              fontWeight: FontWeight.bold,
            ),
          ),
          const SizedBox(width: 4),
          Text(
            label,
            style: theme.textTheme.bodySmall?.copyWith(
              color: color,
            ),
          ),
        ],
      ),
    );
  }
}

/// Tile for previewing a detected device.
class _DevicePreviewTile extends ConsumerWidget {
  final WidgetRef ref;
  final ETSDetectedDevice device;
  final int index;

  const _DevicePreviewTile({
    required this.ref,
    required this.device,
    required this.index,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final locationData = ref.watch(locationDataProvider);
    final rooms = locationData.value?.rooms ?? [];

    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
      child: ExpansionTile(
        leading: Checkbox(
          value: device.import,
          onChanged: (_) {
            ref.read(etsImportProvider.notifier).toggleDeviceImport(index);
          },
        ),
        title: Text(
          device.editedName,
          style: TextStyle(
            color: device.import ? null : theme.disabledColor,
          ),
        ),
        subtitle: Row(
          children: [
            _ConfidenceBadge(confidence: device.confidence),
            const SizedBox(width: 8),
            Text(
              device.detectedType.replaceAll('_', ' '),
              style: theme.textTheme.bodySmall,
            ),
            const SizedBox(width: 8),
            Text(
              '${device.addresses.length} addresses',
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
        children: [
          Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                // ID field
                TextFormField(
                  initialValue: device.editedId,
                  decoration: const InputDecoration(
                    labelText: 'Device ID',
                    border: OutlineInputBorder(),
                    isDense: true,
                  ),
                  onChanged: (value) {
                    ref.read(etsImportProvider.notifier).updateDeviceId(index, value);
                  },
                ),
                const SizedBox(height: 12),

                // Name field
                TextFormField(
                  initialValue: device.editedName,
                  decoration: const InputDecoration(
                    labelText: 'Device Name',
                    border: OutlineInputBorder(),
                    isDense: true,
                  ),
                  onChanged: (newValue) {
                    ref.read(etsImportProvider.notifier).updateDeviceName(index, newValue);
                  },
                ),
                const SizedBox(height: 12),

                // Room dropdown - include suggested room if not in existing rooms
                Builder(builder: (context) {
                  // Build room options: existing rooms + suggested room if different
                  final roomOptions = <DropdownMenuItem<String?>>[];

                  // Add "No room" option
                  roomOptions.add(const DropdownMenuItem(
                    value: null,
                    child: Text('No room'),
                  ));

                  // Add existing rooms from the system
                  final existingIds = <String>{};
                  for (final room in rooms) {
                    existingIds.add(room.id);
                    roomOptions.add(DropdownMenuItem(
                      value: room.id,
                      child: Text(room.name),
                    ));
                  }

                  // Add suggested room if it's not already in the list
                  if (device.suggestedRoom != null &&
                      device.suggestedRoom!.isNotEmpty &&
                      !existingIds.contains(device.suggestedRoom)) {
                    // Format the suggested room nicely (slug to title case)
                    final displayName = device.suggestedRoom!
                        .split('-')
                        .map((w) => w.isNotEmpty ? '${w[0].toUpperCase()}${w.substring(1)}' : '')
                        .join(' ');
                    roomOptions.add(DropdownMenuItem(
                      value: device.suggestedRoom,
                      child: Text('$displayName (suggested)'),
                    ));
                  }

                  return DropdownButtonFormField<String?>(
                    value: device.selectedRoomId,
                    decoration: const InputDecoration(
                      labelText: 'Room',
                      border: OutlineInputBorder(),
                      isDense: true,
                    ),
                    items: roomOptions,
                    onChanged: (selectedValue) {
                      ref.read(etsImportProvider.notifier).updateDeviceRoom(index, selectedValue);
                    },
                  );
                }),
                const SizedBox(height: 16),

                // Address list
                Text(
                  'Group Addresses',
                  style: theme.textTheme.titleSmall,
                ),
                const SizedBox(height: 8),
                ...device.addresses.map((addr) => Padding(
                      padding: const EdgeInsets.only(bottom: 4),
                      child: Row(
                        children: [
                          Container(
                            padding: const EdgeInsets.symmetric(
                              horizontal: 8,
                              vertical: 2,
                            ),
                            decoration: BoxDecoration(
                              color: theme.colorScheme.primaryContainer,
                              borderRadius: BorderRadius.circular(4),
                            ),
                            child: Text(
                              addr.ga,
                              style: theme.textTheme.bodySmall?.copyWith(
                                fontFamily: 'monospace',
                              ),
                            ),
                          ),
                          const SizedBox(width: 8),
                          Expanded(
                            child: Text(
                              addr.suggestedFunction,
                              style: theme.textTheme.bodySmall,
                            ),
                          ),
                          Text(
                            addr.dpt,
                            style: theme.textTheme.bodySmall?.copyWith(
                              color: theme.colorScheme.onSurfaceVariant,
                            ),
                          ),
                        ],
                      ),
                    )),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

/// Badge showing confidence level.
class _ConfidenceBadge extends StatelessWidget {
  final double confidence;

  const _ConfidenceBadge({required this.confidence});

  @override
  Widget build(BuildContext context) {
    Color color;
    String label;

    if (confidence >= 0.8) {
      color = Colors.green;
      label = 'High';
    } else if (confidence >= 0.5) {
      color = Colors.orange;
      label = 'Med';
    } else {
      color = Colors.red;
      label = 'Low';
    }

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.2),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        label,
        style: TextStyle(
          fontSize: 10,
          fontWeight: FontWeight.bold,
          color: color,
        ),
      ),
    );
  }
}

/// Dialog for import options.
class _ImportOptionsDialog extends ConsumerStatefulWidget {
  final WidgetRef ref;

  const _ImportOptionsDialog({required this.ref});

  @override
  ConsumerState<_ImportOptionsDialog> createState() => _ImportOptionsDialogState();
}

class _ImportOptionsDialogState extends ConsumerState<_ImportOptionsDialog> {
  bool _skipExisting = true;
  bool _updateExisting = false;
  bool _dryRun = false;

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('Import Options'),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          CheckboxListTile(
            title: const Text('Skip existing devices'),
            subtitle: const Text('Devices with matching IDs will be skipped'),
            value: _skipExisting,
            onChanged: (value) {
              setState(() {
                _skipExisting = value ?? true;
                if (_skipExisting) _updateExisting = false;
              });
            },
          ),
          CheckboxListTile(
            title: const Text('Update existing devices'),
            subtitle: const Text('Overwrite devices with matching IDs'),
            value: _updateExisting,
            onChanged: (value) {
              setState(() {
                _updateExisting = value ?? false;
                if (_updateExisting) _skipExisting = false;
              });
            },
          ),
          const Divider(),
          CheckboxListTile(
            title: const Text('Dry run'),
            subtitle: const Text('Validate without making changes'),
            value: _dryRun,
            onChanged: (value) => setState(() => _dryRun = value ?? false),
          ),
        ],
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: const Text('Cancel'),
        ),
        FilledButton(
          onPressed: () {
            Navigator.of(context).pop();
            widget.ref.read(etsImportProvider.notifier).importDevices(
                  skipExisting: _skipExisting,
                  updateExisting: _updateExisting,
                  dryRun: _dryRun,
                );
          },
          child: Text(_dryRun ? 'Validate' : 'Import'),
        ),
      ],
    );
  }
}

/// View shown after import completes.
class _CompletedView extends StatelessWidget {
  final WidgetRef ref;
  final ETSImportState state;
  final VoidCallback? onImportComplete;

  const _CompletedView({
    required this.ref,
    required this.state,
    this.onImportComplete,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final response = state.importResponse!;

    return Center(
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(32),
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 400),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(
                response.isSuccess
                    ? Icons.check_circle_outline
                    : Icons.warning_amber_outlined,
                size: 80,
                color: response.isSuccess ? Colors.green : Colors.orange,
              ),
              const SizedBox(height: 24),
              Text(
                response.isSuccess
                    ? 'Import Completed!'
                    : 'Import Completed with Errors',
                style: theme.textTheme.headlineSmall,
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 24),

              // Results summary
              Container(
                padding: const EdgeInsets.all(16),
                decoration: BoxDecoration(
                  color: theme.colorScheme.surfaceContainerHighest,
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Column(
                  children: [
                    _ResultRow(
                      icon: Icons.add_circle_outline,
                      label: 'Created',
                      value: response.created,
                      color: Colors.green,
                    ),
                    const SizedBox(height: 8),
                    _ResultRow(
                      icon: Icons.update_outlined,
                      label: 'Updated',
                      value: response.updated,
                      color: Colors.blue,
                    ),
                    const SizedBox(height: 8),
                    _ResultRow(
                      icon: Icons.skip_next_outlined,
                      label: 'Skipped',
                      value: response.skipped,
                      color: Colors.grey,
                    ),
                    if (response.errors.isNotEmpty) ...[
                      const SizedBox(height: 8),
                      _ResultRow(
                        icon: Icons.error_outline,
                        label: 'Errors',
                        value: response.errors.length,
                        color: Colors.red,
                      ),
                    ],
                  ],
                ),
              ),

              // Error details
              if (response.errors.isNotEmpty) ...[
                const SizedBox(height: 16),
                ExpansionTile(
                  title: Text(
                    'Error Details',
                    style: theme.textTheme.titleSmall,
                  ),
                  children: response.errors
                      .map((e) => ListTile(
                            leading: const Icon(Icons.error_outline, color: Colors.red),
                            title: Text(e.deviceId),
                            subtitle: Text(e.message),
                          ))
                      .toList(),
                ),
              ],

              const SizedBox(height: 32),
              SizedBox(
                width: double.infinity,
                child: FilledButton(
                  onPressed: () {
                    ref.read(etsImportProvider.notifier).reset();
                    onImportComplete?.call();
                    Navigator.of(context).pop();
                  },
                  child: const Text('Done'),
                ),
              ),
              const SizedBox(height: 12),
              SizedBox(
                width: double.infinity,
                child: OutlinedButton(
                  onPressed: () => ref.read(etsImportProvider.notifier).reset(),
                  child: const Text('Import More'),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// Row for displaying import results.
class _ResultRow extends StatelessWidget {
  final IconData icon;
  final String label;
  final int value;
  final Color color;

  const _ResultRow({
    required this.icon,
    required this.label,
    required this.value,
    required this.color,
  });

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Icon(icon, color: color, size: 20),
        const SizedBox(width: 8),
        Expanded(child: Text(label)),
        Text(
          '$value',
          style: Theme.of(context).textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.bold,
                color: color,
              ),
        ),
      ],
    );
  }
}

/// Error view.
class _ErrorView extends StatelessWidget {
  final WidgetRef ref;
  final ETSImportState state;

  const _ErrorView({required this.ref, required this.state});

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
              size: 80,
              color: theme.colorScheme.error,
            ),
            const SizedBox(height: 24),
            Text(
              'Import Failed',
              style: theme.textTheme.headlineSmall,
            ),
            const SizedBox(height: 12),
            Text(
              state.errorMessage ?? 'An unknown error occurred',
              style: theme.textTheme.bodyLarge?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 32),
            Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                OutlinedButton(
                  onPressed: () => ref.read(etsImportProvider.notifier).reset(),
                  child: const Text('Start Over'),
                ),
                const SizedBox(width: 16),
                if (state.hasParsedData)
                  FilledButton(
                    onPressed: () =>
                        ref.read(etsImportProvider.notifier).backToPreview(),
                    child: const Text('Back to Preview'),
                  ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
