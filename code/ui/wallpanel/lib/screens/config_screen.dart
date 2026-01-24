import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/auth.dart';
import '../providers/auth_provider.dart';

/// One-time configuration screen for setting up the panel.
/// Collects: Core URL, Room ID, and login credentials.
class ConfigScreen extends ConsumerStatefulWidget {
  const ConfigScreen({super.key});

  @override
  ConsumerState<ConfigScreen> createState() => _ConfigScreenState();
}

class _ConfigScreenState extends ConsumerState<ConfigScreen> {
  final _formKey = GlobalKey<FormState>();
  final _coreUrlController = TextEditingController(text: 'http://192.168.4.100:8081');
  final _roomIdController = TextEditingController();
  final _usernameController = TextEditingController(text: 'admin');
  final _passwordController = TextEditingController(text: 'admin');
  bool _obscurePassword = true;

  @override
  void initState() {
    super.initState();
    _loadSavedConfig();
  }

  Future<void> _loadSavedConfig() async {
    final tokenStorage = ref.read(tokenStorageProvider);
    final url = await tokenStorage.getCoreUrl();
    final room = await tokenStorage.getRoomId();

    if (url != null && url.isNotEmpty) {
      _coreUrlController.text = url;
    }
    if (room != null && room.isNotEmpty) {
      _roomIdController.text = room;
    }
  }

  @override
  void dispose() {
    _coreUrlController.dispose();
    _roomIdController.dispose();
    _usernameController.dispose();
    _passwordController.dispose();
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
                      'Configure your wall panel connection',
                      textAlign: TextAlign.center,
                      style: Theme.of(context).textTheme.bodyLarge,
                    ),
                    const SizedBox(height: 32),

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

                    // Default Room (optional)
                    TextFormField(
                      controller: _roomIdController,
                      decoration: const InputDecoration(
                        labelText: 'Default Room (optional)',
                        hintText: 'e.g., room-living',
                        helperText: 'Leave blank to show all rooms',
                        prefixIcon: Icon(Icons.room_outlined),
                        border: OutlineInputBorder(),
                      ),
                    ),
                    const SizedBox(height: 16),

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
                            : const Text('Connect'),
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
    final roomId = _roomIdController.text.trim();
    final username = _usernameController.text.trim();
    final password = _passwordController.text;

    // Persist room ID
    final tokenStorage = ref.read(tokenStorageProvider);
    await tokenStorage.setRoomId(roomId);

    // Attempt login
    await ref.read(authProvider.notifier).login(coreUrl, username, password);
  }
}
