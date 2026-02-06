import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/panel.dart';
import '../providers/panel_provider.dart';

/// Bottom sheet for creating or editing a panel.
/// Pass [panel] = null for create mode, or an existing panel for edit mode.
///
/// On create, shows a one-time token dialog after success.
class PanelEditorSheet extends ConsumerStatefulWidget {
  final Panel? panel;

  const PanelEditorSheet({super.key, this.panel});

  @override
  ConsumerState<PanelEditorSheet> createState() => _PanelEditorSheetState();
}

class _PanelEditorSheetState extends ConsumerState<PanelEditorSheet> {
  final _formKey = GlobalKey<FormState>();
  late final TextEditingController _nameController;
  bool _saving = false;

  bool get _isEdit => widget.panel != null;

  @override
  void initState() {
    super.initState();
    _nameController = TextEditingController(text: widget.panel?.name ?? '');
  }

  @override
  void dispose() {
    _nameController.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    if (!_formKey.currentState!.validate()) return;

    setState(() => _saving = true);

    try {
      final notifier = ref.read(allPanelsProvider.notifier);

      if (_isEdit) {
        await notifier.updatePanel(widget.panel!.id, {
          'name': _nameController.text.trim(),
        });
        if (!mounted) return;
        Navigator.of(context).pop(true);
      } else {
        final response = await notifier.createPanel({
          'name': _nameController.text.trim(),
        });
        if (!mounted) return;
        // Show one-time token dialog before closing
        await _showTokenDialog(response.panelToken);
        if (!mounted) return;
        Navigator.of(context).pop(true);
      }
    } catch (e) {
      if (!mounted) return;
      setState(() => _saving = false);

      String msg;
      final err = e.toString();
      if (err.contains('409')) {
        msg = 'Panel name already exists';
      } else if (err.contains('400')) {
        msg = 'Invalid data — check all fields';
      } else {
        msg = 'Failed to save panel';
      }

      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(msg),
          behavior: SnackBarBehavior.floating,
          backgroundColor: Theme.of(context).colorScheme.error,
        ),
      );
    }
  }

  Future<void> _showTokenDialog(String token) async {
    await showDialog<void>(
      context: context,
      barrierDismissible: false,
      builder: (context) => AlertDialog(
        title: Row(
          children: [
            Icon(Icons.warning_amber_rounded,
                color: Theme.of(context).colorScheme.error),
            const SizedBox(width: 8),
            const Expanded(child: Text('Panel Token')),
          ],
        ),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              'Save this token — it cannot be shown again.',
              style: TextStyle(
                color: Theme.of(context).colorScheme.error,
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 16),
            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: Theme.of(context)
                    .colorScheme
                    .surfaceContainerHighest,
                borderRadius: BorderRadius.circular(8),
                border: Border.all(
                  color: Theme.of(context).colorScheme.outline,
                ),
              ),
              child: SelectableText(
                token,
                style: const TextStyle(
                  fontFamily: 'monospace',
                  fontSize: 13,
                  letterSpacing: 0.5,
                ),
              ),
            ),
            const SizedBox(height: 12),
            SizedBox(
              width: double.infinity,
              child: OutlinedButton.icon(
                icon: const Icon(Icons.copy, size: 18),
                label: const Text('Copy to Clipboard'),
                onPressed: () {
                  Clipboard.setData(ClipboardData(text: token));
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(
                      content: Text('Token copied to clipboard'),
                      behavior: SnackBarBehavior.floating,
                      duration: Duration(seconds: 2),
                    ),
                  );
                },
              ),
            ),
          ],
        ),
        actions: [
          FilledButton(
            onPressed: () => Navigator.of(context).pop(),
            child: const Text('Done'),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return DraggableScrollableSheet(
      initialChildSize: 0.5,
      minChildSize: 0.3,
      maxChildSize: 0.7,
      expand: false,
      builder: (context, scrollController) {
        return Container(
          decoration: BoxDecoration(
            color: theme.colorScheme.surface,
            borderRadius:
                const BorderRadius.vertical(top: Radius.circular(16)),
          ),
          child: Column(
            children: [
              // Handle + header
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
                child: Column(
                  children: [
                    Center(
                      child: Container(
                        width: 32,
                        height: 4,
                        decoration: BoxDecoration(
                          color: theme.colorScheme.onSurfaceVariant
                              .withValues(alpha: 0.4),
                          borderRadius: BorderRadius.circular(2),
                        ),
                      ),
                    ),
                    const SizedBox(height: 12),
                    Row(
                      children: [
                        Expanded(
                          child: Text(
                            _isEdit ? 'Edit Panel' : 'Register Panel',
                            style: theme.textTheme.titleLarge,
                          ),
                        ),
                        TextButton(
                          onPressed: () => Navigator.pop(context),
                          child: const Text('Cancel'),
                        ),
                        const SizedBox(width: 8),
                        FilledButton(
                          onPressed: _saving ? null : _save,
                          child: _saving
                              ? const SizedBox(
                                  width: 20,
                                  height: 20,
                                  child: CircularProgressIndicator(
                                      strokeWidth: 2),
                                )
                              : Text(_isEdit ? 'Save' : 'Register'),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
              const Divider(),
              // Form
              Expanded(
                child: Form(
                  key: _formKey,
                  child: ListView(
                    controller: scrollController,
                    padding: const EdgeInsets.all(16),
                    children: [
                      TextFormField(
                        controller: _nameController,
                        autofocus: !_isEdit,
                        decoration: const InputDecoration(
                          labelText: 'Panel Name',
                          border: OutlineInputBorder(),
                          helperText: 'e.g. "Kitchen Panel", "Master Bedroom"',
                        ),
                        validator: (v) {
                          if (v == null || v.trim().isEmpty) return 'Required';
                          if (v.trim().length > 128) return 'Max 128 characters';
                          return null;
                        },
                      ),
                    ],
                  ),
                ),
              ),
            ],
          ),
        );
      },
    );
  }
}
