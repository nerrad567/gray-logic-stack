import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'admin/admin_screen.dart';
import 'ets_import_screen.dart';

/// Onboarding screen shown when no devices/locations are configured.
/// Guides users to import their KNX devices from ETS.
class OnboardingScreen extends ConsumerWidget {
  final VoidCallback onRefresh;

  const OnboardingScreen({super.key, required this.onRefresh});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    return Scaffold(
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(32),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 500),
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  // Welcome icon
                  Container(
                    width: 120,
                    height: 120,
                    decoration: BoxDecoration(
                      color: colorScheme.primaryContainer,
                      shape: BoxShape.circle,
                    ),
                    child: Icon(
                      Icons.home_outlined,
                      size: 64,
                      color: colorScheme.onPrimaryContainer,
                    ),
                  ),
                  const SizedBox(height: 32),

                  // Welcome text
                  Text(
                    'Welcome to Gray Logic',
                    style: theme.textTheme.headlineMedium?.copyWith(
                      fontWeight: FontWeight.bold,
                    ),
                    textAlign: TextAlign.center,
                  ),
                  const SizedBox(height: 16),
                  Text(
                    'Your building intelligence system is ready.\nLet\'s add your devices to get started.',
                    style: theme.textTheme.bodyLarge?.copyWith(
                      color: colorScheme.onSurfaceVariant,
                    ),
                    textAlign: TextAlign.center,
                  ),
                  const SizedBox(height: 48),

                  // Import from ETS card
                  _OnboardingCard(
                    icon: Icons.upload_file_outlined,
                    title: 'Import from ETS',
                    description:
                        'Import your KNX devices directly from an ETS project file (.knxproj). This is the fastest way to set up your system.',
                    buttonLabel: 'Import ETS Project',
                    isPrimary: true,
                    onPressed: () {
                      Navigator.of(context).push(
                        MaterialPageRoute(
                          builder: (context) => ETSImportScreen(
                            onImportComplete: onRefresh,
                          ),
                        ),
                      );
                    },
                  ),
                  const SizedBox(height: 16),

                  // Manual setup card — opens Admin panel for manual device creation
                  _OnboardingCard(
                    icon: Icons.edit_outlined,
                    title: 'Manual Setup',
                    description:
                        'Add devices, rooms, and areas manually via the Admin panel. Best for small installations or custom configurations.',
                    buttonLabel: 'Continue to Admin',
                    isPrimary: false,
                    onPressed: () {
                      Navigator.of(context).push(
                        MaterialPageRoute(
                          builder: (context) => AdminScreen(
                            onRefresh: onRefresh,
                          ),
                        ),
                      );
                    },
                  ),
                  const SizedBox(height: 32),

                  // Refresh button
                  TextButton.icon(
                    onPressed: onRefresh,
                    icon: const Icon(Icons.refresh),
                    label: const Text('Refresh'),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

/// Card widget for onboarding options.
class _OnboardingCard extends StatelessWidget {
  final IconData icon;
  final String title;
  final String description;
  final String buttonLabel;
  final bool isPrimary;
  final VoidCallback? onPressed;

  const _OnboardingCard({
    required this.icon,
    required this.title,
    required this.description,
    required this.buttonLabel,
    required this.isPrimary,
    required this.onPressed,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    return Card(
      elevation: isPrimary ? 2 : 0,
      color: isPrimary ? colorScheme.surface : colorScheme.surfaceContainerLow,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: isPrimary
            ? BorderSide.none
            : BorderSide(color: colorScheme.outlineVariant),
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
                        ? colorScheme.primary.withValues(alpha: 0.1)
                        : colorScheme.surfaceContainerHighest,
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Icon(
                    icon,
                    color: isPrimary
                        ? colorScheme.primary
                        : colorScheme.onSurfaceVariant,
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
                color: colorScheme.onSurfaceVariant,
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
