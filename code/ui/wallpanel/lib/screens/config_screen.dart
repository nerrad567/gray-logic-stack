import 'package:flutter/foundation.dart' show kIsWeb;
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/auth.dart';
import '../providers/auth_provider.dart';

/// One-time configuration screen for setting up the panel.
/// Supports two modes: User login (username/password) and Panel mode (panel token).
class ConfigScreen extends ConsumerStatefulWidget {
  const ConfigScreen({super.key});

  @override
  ConsumerState<ConfigScreen> createState() => _ConfigScreenState();
}

class _ConfigScreenState extends ConsumerState<ConfigScreen> {
  final _formKey = GlobalKey<FormState>();
  // On web, auto-detect host from browser; on native, use placeholder
  final _coreUrlController = TextEditingController(
    text: kIsWeb ? Uri.base.origin : '',
  );
  final _usernameController = TextEditingController(text: 'admin');
  final _passwordController = TextEditingController(text: 'admin');
  final _panelTokenController = TextEditingController();
  bool _obscurePassword = true;
  bool _panelMode = false;

  @override
  void initState() {
    super.initState();
    _loadSavedConfig();
  }

  Future<void> _loadSavedConfig() async {
    final tokenStorage = ref.read(tokenStorageProvider);
    final url = await tokenStorage.getCoreUrl();
    final mode = await tokenStorage.getAuthMode();

    if (url != null && url.isNotEmpty) {
      _coreUrlController.text = url;
    }

    if (mode == 'panel') {
      setState(() => _panelMode = true);
    }
  }

  @override
  void dispose() {
    _coreUrlController.dispose();
    _usernameController.dispose();
    _passwordController.dispose();
    _panelTokenController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final authState = ref.watch(authProvider);
    final isLoading = authState.status == AuthStatus.authenticating;

    return Scaffold(
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(32),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 400),
              child: Form(
                key: _formKey,
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    // Logo / Title
                    Icon(
                      Icons.home_outlined,
                      size: 64,
                      color: Theme.of(context).colorScheme.primary,
                    ),
                    const SizedBox(height: 16),
                    Text(
                      'Gray Logic Panel',
                      textAlign: TextAlign.center,
                      style: Theme.of(context).textTheme.titleLarge,
                    ),
                    const SizedBox(height: 8),
                    Text(
                      _panelMode
                          ? 'Configure panel device connection'
                          : 'Configure your wall panel connection',
                      textAlign: TextAlign.center,
                      style: Theme.of(context).textTheme.bodyLarge,
                    ),
                    const SizedBox(height: 24),

                    // Panel mode toggle
                    SwitchListTile(
                      title: const Text('Panel Mode'),
                      subtitle: Text(
                        _panelMode
                            ? 'Authenticate with panel token'
                            : 'Authenticate with user credentials',
                        style: Theme.of(context).textTheme.bodySmall,
                      ),
                      value: _panelMode,
                      onChanged: (v) => setState(() => _panelMode = v),
                      contentPadding: EdgeInsets.zero,
                    ),
                    const SizedBox(height: 16),

                    // Core URL
                    TextFormField(
                      controller: _coreUrlController,
                      decoration: const InputDecoration(
                        labelText: 'Core URL',
                        hintText: 'http://192.168.1.100:8080',
                        prefixIcon: Icon(Icons.dns_outlined),
                        border: OutlineInputBorder(),
                      ),
                      keyboardType: TextInputType.url,
                      validator: (v) {
                        if (v == null || v.isEmpty) return 'Required';
                        final uri = Uri.tryParse(v);
                        if (uri == null || !uri.hasScheme) return 'Invalid URL';
                        return null;
                      },
                    ),
                    const SizedBox(height: 16),

                    if (_panelMode) ...[
                      // Panel token field
                      TextFormField(
                        controller: _panelTokenController,
                        decoration: const InputDecoration(
                          labelText: 'Panel Token',
                          hintText: '64-character hex token',
                          prefixIcon: Icon(Icons.vpn_key_outlined),
                          border: OutlineInputBorder(),
                        ),
                        maxLines: 1,
                        style: const TextStyle(fontFamily: 'monospace', fontSize: 13),
                        validator: (v) {
                          if (v == null || v.isEmpty) return 'Required';
                          if (v.length < 16) return 'Token too short';
                          return null;
                        },
                      ),
                    ] else ...[
                      // Username
                      TextFormField(
                        controller: _usernameController,
                        decoration: const InputDecoration(
                          labelText: 'Username',
                          prefixIcon: Icon(Icons.person_outline),
                          border: OutlineInputBorder(),
                        ),
                        validator: (v) {
                          if (v == null || v.isEmpty) return 'Required';
                          return null;
                        },
                      ),
                      const SizedBox(height: 16),

                      // Password
                      TextFormField(
                        controller: _passwordController,
                        obscureText: _obscurePassword,
                        decoration: InputDecoration(
                          labelText: 'Password',
                          prefixIcon: const Icon(Icons.lock_outline),
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
                          return null;
                        },
                      ),
                    ],
                    const SizedBox(height: 8),

                    // Error message
                    if (authState.status == AuthStatus.error)
                      Padding(
                        padding: const EdgeInsets.only(bottom: 8),
                        child: Text(
                          authState.errorMessage ?? 'Login failed',
                          style: TextStyle(
                            color: Theme.of(context).colorScheme.error,
                            fontSize: 13,
                          ),
                          textAlign: TextAlign.center,
                        ),
                      ),

                    const SizedBox(height: 16),

                    // Connect button
                    SizedBox(
                      height: 48,
                      child: FilledButton(
                        onPressed: isLoading ? null : _submit,
                        child: isLoading
                            ? const SizedBox(
                                width: 20,
                                height: 20,
                                child: CircularProgressIndicator(
                                  strokeWidth: 2,
                                  color: Colors.white,
                                ),
                              )
                            : Text(_panelMode ? 'Connect Panel' : 'Connect'),
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;

    final coreUrl = _coreUrlController.text.trim();

    if (_panelMode) {
      final panelToken = _panelTokenController.text.trim();
      await ref.read(authProvider.notifier).panelLogin(coreUrl, panelToken);
    } else {
      final username = _usernameController.text.trim();
      final password = _passwordController.text;
      await ref.read(authProvider.notifier).login(coreUrl, username, password);
    }
  }
}
