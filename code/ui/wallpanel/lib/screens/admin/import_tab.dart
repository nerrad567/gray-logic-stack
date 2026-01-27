import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../ets_import_screen.dart';

/// Import tab that wraps the ETS import functionality.
///
/// Embeds the ETSImportScreen content within the admin tab structure.
class ImportTab extends ConsumerWidget {
  /// Callback when import completes successfully.
  final VoidCallback? onImportComplete;

  const ImportTab({super.key, this.onImportComplete});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // For simplicity, we show a card that launches the full import screen.
    // This keeps the import flow unchanged while making it accessible from admin.
    return _ImportLauncherView(onImportComplete: onImportComplete);
  }
}

class _ImportLauncherView extends StatelessWidget {
  final VoidCallback? onImportComplete;

  const _ImportLauncherView({this.onImportComplete});

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
              // ETS Import card
              _ImportOptionCard(
                icon: Icons.upload_file_outlined,
                title: 'Import from ETS',
                description:
                    'Import KNX devices from an ETS project file (.knxproj), group address XML export, or CSV file.',
                buttonLabel: 'Start Import',
                onPressed: () {
                  Navigator.of(context).push(
                    MaterialPageRoute(
                      builder: (context) => ETSImportScreen(
                        onImportComplete: () {
                          onImportComplete?.call();
                          Navigator.of(context).pop();
                        },
                      ),
                    ),
                  );
                },
              ),
              const SizedBox(height: 16),

              // Sample file hint
              Container(
                padding: const EdgeInsets.all(16),
                decoration: BoxDecoration(
                  color: theme.colorScheme.surfaceContainerHighest,
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(
                    color: theme.colorScheme.outlineVariant,
                  ),
                ),
                child: Row(
                  children: [
                    Icon(
                      Icons.lightbulb_outline,
                      color: theme.colorScheme.primary,
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            'Tip: Sample file available',
                            style: theme.textTheme.titleSmall,
                          ),
                          const SizedBox(height: 4),
                          Text(
                            'A sample CSV file matching the KNXSim devices is available at sim/knxsim/sample-ets-export.csv',
                            style: theme.textTheme.bodySmall?.copyWith(
                              color: theme.colorScheme.onSurfaceVariant,
                            ),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 24),

              // Manual setup (coming soon)
              _ImportOptionCard(
                icon: Icons.edit_outlined,
                title: 'Manual Device Setup',
                description:
                    'Add devices manually by entering their KNX group addresses. Best for small installations or custom configurations.',
                buttonLabel: 'Coming Soon',
                onPressed: null,
                isPrimary: false,
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _ImportOptionCard extends StatelessWidget {
  final IconData icon;
  final String title;
  final String description;
  final String buttonLabel;
  final VoidCallback? onPressed;
  final bool isPrimary;

  const _ImportOptionCard({
    required this.icon,
    required this.title,
    required this.description,
    required this.buttonLabel,
    required this.onPressed,
    this.isPrimary = true,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Card(
      elevation: isPrimary ? 2 : 0,
      color: isPrimary ? theme.colorScheme.surface : theme.colorScheme.surfaceContainerLow,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: isPrimary
            ? BorderSide.none
            : BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Container(
                  width: 48,
                  height: 48,
                  decoration: BoxDecoration(
                    color: isPrimary
                        ? theme.colorScheme.primary.withValues(alpha: 0.1)
                        : theme.colorScheme.surfaceContainerHighest,
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Icon(
                    icon,
                    color: isPrimary
                        ? theme.colorScheme.primary
                        : theme.colorScheme.onSurfaceVariant,
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: Text(
                    title,
                    style: theme.textTheme.titleMedium?.copyWith(
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 16),
            Text(
              description,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
            const SizedBox(height: 20),
            SizedBox(
              width: double.infinity,
              child: isPrimary
                  ? FilledButton(
                      onPressed: onPressed,
                      child: Text(buttonLabel),
                    )
                  : OutlinedButton(
                      onPressed: onPressed,
                      child: Text(buttonLabel),
                    ),
            ),
          ],
        ),
      ),
    );
  }
}
