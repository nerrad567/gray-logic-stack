import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../models/site.dart';
import '../../providers/auth_provider.dart';

/// Site management tab — view/edit the property record, or create one
/// if the system is in "setup needed" state (no site exists yet).
class SiteTab extends ConsumerStatefulWidget {
  const SiteTab({super.key});

  @override
  ConsumerState<SiteTab> createState() => _SiteTabState();
}

class _SiteTabState extends ConsumerState<SiteTab> {
  bool _loading = true;
  bool _saving = false;
  bool _isNew = false;
  String? _error;
  Site? _site;

  // Form controllers
  final _formKey = GlobalKey<FormState>();
  late TextEditingController _nameCtl;
  late TextEditingController _addressCtl;
  late TextEditingController _latCtl;
  late TextEditingController _lonCtl;
  late TextEditingController _elevCtl;
  late TextEditingController _timezoneCtl;

  @override
  void initState() {
    super.initState();
    _nameCtl = TextEditingController();
    _addressCtl = TextEditingController();
    _latCtl = TextEditingController();
    _lonCtl = TextEditingController();
    _elevCtl = TextEditingController();
    _timezoneCtl = TextEditingController();
    _loadSite();
  }

  @override
  void dispose() {
    _nameCtl.dispose();
    _addressCtl.dispose();
    _latCtl.dispose();
    _lonCtl.dispose();
    _elevCtl.dispose();
    _timezoneCtl.dispose();
    super.dispose();
  }

  Future<void> _loadSite() async {
    setState(() {
      _loading = true;
      _error = null;
    });

    try {
      final api = ref.read(apiClientProvider);
      final site = await api.getSite();
      if (mounted) {
        setState(() {
          _site = site;
          _isNew = site == null;
          _loading = false;
          _populateFields(site);
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          _error = e.toString();
          _loading = false;
        });
      }
    }
  }

  void _populateFields(Site? site) {
    _nameCtl.text = site?.name ?? '';
    _addressCtl.text = site?.address ?? '';
    _latCtl.text = site?.latitude?.toString() ?? '';
    _lonCtl.text = site?.longitude?.toString() ?? '';
    _elevCtl.text = site?.elevationM?.toString() ?? '';
    _timezoneCtl.text = site?.timezone ?? 'UTC';
  }

  Future<void> _save() async {
    if (!_formKey.currentState!.validate()) return;

    setState(() => _saving = true);

    try {
      final api = ref.read(apiClientProvider);
      final data = <String, dynamic>{
        'name': _nameCtl.text.trim(),
        'timezone': _timezoneCtl.text.trim().isEmpty
            ? 'UTC'
            : _timezoneCtl.text.trim(),
      };

      // Only include optional fields if they have values.
      final address = _addressCtl.text.trim();
      if (address.isNotEmpty) data['address'] = address;

      final lat = double.tryParse(_latCtl.text.trim());
      if (lat != null) data['latitude'] = lat;

      final lon = double.tryParse(_lonCtl.text.trim());
      if (lon != null) data['longitude'] = lon;

      final elev = double.tryParse(_elevCtl.text.trim());
      if (elev != null) data['elevation_m'] = elev;

      Site result;
      if (_isNew) {
        result = await api.createSite(data);
      } else {
        result = await api.updateSite(data);
      }

      if (mounted) {
        setState(() {
          _site = result;
          _isNew = false;
          _saving = false;
          _populateFields(result);
        });
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(_isNew
                ? 'Site created successfully'
                : 'Site updated successfully'),
            duration: const Duration(seconds: 3),
          ),
        );
      }
    } catch (e) {
      if (mounted) {
        setState(() => _saving = false);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Failed to save site: $e')),
        );
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    if (_loading) {
      return const Center(child: CircularProgressIndicator());
    }

    if (_error != null) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.error_outline,
                size: 48, color: theme.colorScheme.error),
            const SizedBox(height: 12),
            Text('Failed to load site', style: theme.textTheme.titleMedium),
            const SizedBox(height: 4),
            Text(_error!, style: theme.textTheme.bodySmall),
            const SizedBox(height: 16),
            FilledButton.icon(
              onPressed: _loadSite,
              icon: const Icon(Icons.refresh),
              label: const Text('Retry'),
            ),
          ],
        ),
      );
    }

    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Form(
        key: _formKey,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Header
            if (_isNew) _buildSetupHeader(theme),
            if (!_isNew) _buildEditHeader(theme),

            const SizedBox(height: 24),

            // Form fields
            _buildSectionLabel(theme, 'Property Details'),
            const SizedBox(height: 12),

            TextFormField(
              controller: _nameCtl,
              decoration: const InputDecoration(
                labelText: 'Site Name',
                hintText: 'e.g. Villa Marina',
                prefixIcon: Icon(Icons.home_outlined),
                border: OutlineInputBorder(),
              ),
              validator: (v) =>
                  (v == null || v.trim().isEmpty) ? 'Name is required' : null,
              textInputAction: TextInputAction.next,
            ),
            const SizedBox(height: 16),

            TextFormField(
              controller: _addressCtl,
              decoration: const InputDecoration(
                labelText: 'Address',
                hintText: 'Optional',
                prefixIcon: Icon(Icons.location_on_outlined),
                border: OutlineInputBorder(),
              ),
              textInputAction: TextInputAction.next,
            ),
            const SizedBox(height: 16),

            TextFormField(
              controller: _timezoneCtl,
              decoration: const InputDecoration(
                labelText: 'Timezone',
                hintText: 'e.g. Europe/London',
                prefixIcon: Icon(Icons.schedule_outlined),
                border: OutlineInputBorder(),
              ),
              textInputAction: TextInputAction.next,
            ),

            const SizedBox(height: 24),
            _buildSectionLabel(theme, 'GPS Coordinates'),
            const SizedBox(height: 4),
            Text(
              'Used for sunrise/sunset calculations and weather data.',
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
            const SizedBox(height: 12),

            Row(
              children: [
                Expanded(
                  child: TextFormField(
                    controller: _latCtl,
                    decoration: const InputDecoration(
                      labelText: 'Latitude',
                      hintText: 'e.g. 51.5074',
                      border: OutlineInputBorder(),
                    ),
                    keyboardType:
                        const TextInputType.numberWithOptions(decimal: true, signed: true),
                    validator: (v) {
                      if (v == null || v.trim().isEmpty) return null;
                      final n = double.tryParse(v.trim());
                      if (n == null) return 'Invalid number';
                      if (n < -90 || n > 90) return '-90 to 90';
                      return null;
                    },
                    textInputAction: TextInputAction.next,
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: TextFormField(
                    controller: _lonCtl,
                    decoration: const InputDecoration(
                      labelText: 'Longitude',
                      hintText: 'e.g. -0.1278',
                      border: OutlineInputBorder(),
                    ),
                    keyboardType:
                        const TextInputType.numberWithOptions(decimal: true, signed: true),
                    validator: (v) {
                      if (v == null || v.trim().isEmpty) return null;
                      final n = double.tryParse(v.trim());
                      if (n == null) return 'Invalid number';
                      if (n < -180 || n > 180) return '-180 to 180';
                      return null;
                    },
                    textInputAction: TextInputAction.next,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 16),

            TextFormField(
              controller: _elevCtl,
              decoration: const InputDecoration(
                labelText: 'Elevation (metres)',
                hintText: 'Optional',
                prefixIcon: Icon(Icons.terrain_outlined),
                border: OutlineInputBorder(),
              ),
              keyboardType:
                  const TextInputType.numberWithOptions(decimal: true, signed: true),
              validator: (v) {
                if (v == null || v.trim().isEmpty) return null;
                if (double.tryParse(v.trim()) == null) return 'Invalid number';
                return null;
              },
              textInputAction: TextInputAction.done,
            ),

            // Metadata (read-only, shown only for existing sites)
            if (!_isNew && _site != null) ...[
              const SizedBox(height: 24),
              _buildSectionLabel(theme, 'Status'),
              const SizedBox(height: 12),
              _buildReadOnlyField(
                  theme, 'Site ID', _site!.id),
              const SizedBox(height: 8),
              _buildReadOnlyField(
                  theme, 'Current Mode', _site!.modeCurrent),
              const SizedBox(height: 8),
              _buildReadOnlyField(
                  theme, 'Available Modes', _site!.modesAvailable.join(', ')),
              const SizedBox(height: 8),
              _buildReadOnlyField(
                  theme, 'Created', _formatDateTime(_site!.createdAt)),
              const SizedBox(height: 8),
              _buildReadOnlyField(
                  theme, 'Updated', _formatDateTime(_site!.updatedAt)),
            ],

            const SizedBox(height: 32),

            // Save button
            SizedBox(
              width: double.infinity,
              child: FilledButton.icon(
                onPressed: _saving ? null : _save,
                icon: _saving
                    ? const SizedBox(
                        width: 18,
                        height: 18,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          color: Colors.white,
                        ),
                      )
                    : Icon(_isNew ? Icons.add : Icons.save_outlined),
                label: Text(_saving
                    ? 'Saving...'
                    : _isNew
                        ? 'Create Site'
                        : 'Save Changes'),
                style: FilledButton.styleFrom(
                  padding: const EdgeInsets.symmetric(vertical: 16),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildSetupHeader(ThemeData theme) {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.primaryContainer.withValues(alpha: 0.3),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(
          color: theme.colorScheme.primary.withValues(alpha: 0.3),
        ),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(
            Icons.rocket_launch_outlined,
            color: theme.colorScheme.primary,
            size: 28,
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  'Welcome — Set Up Your Site',
                  style: theme.textTheme.titleMedium?.copyWith(
                    color: theme.colorScheme.primary,
                    fontWeight: FontWeight.bold,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  'No site has been configured yet. Fill in the details '
                  'below to create your property record. You can update '
                  'these fields at any time.',
                  style: theme.textTheme.bodyMedium,
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildEditHeader(ThemeData theme) {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHighest.withValues(alpha: 0.3),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(
            Icons.home_work_outlined,
            color: theme.colorScheme.primary,
            size: 28,
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  _site?.name ?? 'Site',
                  style: theme.textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  'Edit your property details. Changes are saved directly '
                  'to the database and take effect immediately.',
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildSectionLabel(ThemeData theme, String text) {
    return Text(
      text,
      style: theme.textTheme.titleSmall?.copyWith(
        color: theme.colorScheme.onSurfaceVariant,
        fontWeight: FontWeight.w600,
      ),
    );
  }

  Widget _buildReadOnlyField(ThemeData theme, String label, String value) {
    return Row(
      children: [
        SizedBox(
          width: 120,
          child: Text(
            label,
            style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
        ),
        Expanded(
          child: Text(
            value,
            style: theme.textTheme.bodyMedium?.copyWith(
              fontWeight: FontWeight.w500,
            ),
          ),
        ),
      ],
    );
  }

  String _formatDateTime(DateTime dt) {
    return '${dt.year}-${_pad(dt.month)}-${_pad(dt.day)} '
        '${_pad(dt.hour)}:${_pad(dt.minute)}';
  }

  String _pad(int n) => n.toString().padLeft(2, '0');
}
