import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/user.dart';
import '../providers/user_provider.dart';

/// Bottom sheet for creating or editing a user.
/// Pass [user] = null for create mode, or an existing user for edit mode.
class UserEditorSheet extends ConsumerStatefulWidget {
  final User? user;

  const UserEditorSheet({super.key, this.user});

  @override
  ConsumerState<UserEditorSheet> createState() => _UserEditorSheetState();
}

class _UserEditorSheetState extends ConsumerState<UserEditorSheet> {
  final _formKey = GlobalKey<FormState>();
  late final TextEditingController _usernameController;
  late final TextEditingController _displayNameController;
  late final TextEditingController _emailController;
  late final TextEditingController _passwordController;
  late String _role;
  late bool _isActive;
  bool _saving = false;
  bool _obscurePassword = true;

  bool get _isEdit => widget.user != null;

  @override
  void initState() {
    super.initState();
    final u = widget.user;
    _usernameController = TextEditingController(text: u?.username ?? '');
    _displayNameController = TextEditingController(text: u?.displayName ?? '');
    _emailController = TextEditingController(text: u?.email ?? '');
    _passwordController = TextEditingController();
    _role = u?.role ?? 'user';
    _isActive = u?.isActive ?? true;
  }

  @override
  void dispose() {
    _usernameController.dispose();
    _displayNameController.dispose();
    _emailController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    if (!_formKey.currentState!.validate()) return;

    setState(() => _saving = true);

    try {
      final notifier = ref.read(allUsersProvider.notifier);

      if (_isEdit) {
        final data = <String, dynamic>{
          'display_name': _displayNameController.text.trim(),
          'email': _emailController.text.trim().isEmpty
              ? ''
              : _emailController.text.trim(),
          'role': _role,
          'is_active': _isActive,
        };
        await notifier.updateUser(widget.user!.id, data);
      } else {
        final data = <String, dynamic>{
          'username': _usernameController.text.trim(),
          'display_name': _displayNameController.text.trim(),
          'password': _passwordController.text,
          'role': _role,
        };
        if (_emailController.text.trim().isNotEmpty) {
          data['email'] = _emailController.text.trim();
        }
        await notifier.createUser(data);
      }

      if (!mounted) return;
      Navigator.of(context).pop(true);
    } catch (e) {
      if (!mounted) return;
      setState(() => _saving = false);

      String msg;
      final err = e.toString();
      if (err.contains('409')) {
        msg = 'Username already exists';
      } else if (err.contains('400')) {
        msg = 'Invalid data — check all fields';
      } else if (err.contains('self')) {
        msg = 'Cannot modify your own role or status';
      } else {
        msg = 'Failed to save user';
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

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return DraggableScrollableSheet(
      initialChildSize: 0.7,
      minChildSize: 0.4,
      maxChildSize: 0.9,
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
                            _isEdit ? 'Edit User' : 'New User',
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
                              : Text(_isEdit ? 'Save' : 'Create'),
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
                      // Username (only editable on create)
                      TextFormField(
                        controller: _usernameController,
                        enabled: !_isEdit,
                        decoration: const InputDecoration(
                          labelText: 'Username',
                          border: OutlineInputBorder(),
                          helperText: 'Letters, numbers, dots, hyphens, underscores',
                        ),
                        validator: (v) {
                          if (v == null || v.trim().isEmpty) return 'Required';
                          if (v.trim().length > 64) return 'Max 64 characters';
                          if (!RegExp(r'^[a-zA-Z0-9._-]+$').hasMatch(v.trim())) {
                            return 'Invalid characters';
                          }
                          return null;
                        },
                      ),
                      const SizedBox(height: 16),

                      // Display name
                      TextFormField(
                        controller: _displayNameController,
                        decoration: const InputDecoration(
                          labelText: 'Display Name',
                          border: OutlineInputBorder(),
                        ),
                        validator: (v) {
                          if (v == null || v.trim().isEmpty) return 'Required';
                          if (v.trim().length > 128) return 'Max 128 characters';
                          return null;
                        },
                      ),
                      const SizedBox(height: 16),

                      // Email
                      TextFormField(
                        controller: _emailController,
                        decoration: const InputDecoration(
                          labelText: 'Email (optional)',
                          border: OutlineInputBorder(),
                        ),
                        keyboardType: TextInputType.emailAddress,
                      ),
                      const SizedBox(height: 16),

                      // Password (required on create, hidden on edit)
                      if (!_isEdit) ...[
                        TextFormField(
                          controller: _passwordController,
                          obscureText: _obscurePassword,
                          decoration: InputDecoration(
                            labelText: 'Password',
                            border: const OutlineInputBorder(),
                            suffixIcon: IconButton(
                              icon: Icon(_obscurePassword
                                  ? Icons.visibility_off
                                  : Icons.visibility),
                              onPressed: () => setState(
                                  () => _obscurePassword = !_obscurePassword),
                            ),
                          ),
                          validator: (v) {
                            if (v == null || v.isEmpty) return 'Required';
                            if (v.length < 8) return 'Minimum 8 characters';
                            if (v.length > 128) return 'Maximum 128 characters';
                            return null;
                          },
                        ),
                        const SizedBox(height: 16),
                      ],

                      // Role dropdown
                      DropdownButtonFormField<String>(
                        value: _role,
                        decoration: const InputDecoration(
                          labelText: 'Role',
                          border: OutlineInputBorder(),
                        ),
                        items: const [
                          DropdownMenuItem(value: 'user', child: Text('User')),
                          DropdownMenuItem(value: 'admin', child: Text('Admin')),
                          DropdownMenuItem(value: 'owner', child: Text('Owner')),
                        ],
                        onChanged: (v) {
                          if (v != null) setState(() => _role = v);
                        },
                      ),
                      const SizedBox(height: 16),

                      // Active toggle (edit only)
                      if (_isEdit)
                        SwitchListTile(
                          title: const Text('Active'),
                          subtitle: Text(
                            _isActive
                                ? 'User can log in'
                                : 'User is locked out',
                            style: theme.textTheme.bodySmall,
                          ),
                          value: _isActive,
                          onChanged: (v) => setState(() => _isActive = v),
                          contentPadding: EdgeInsets.zero,
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
