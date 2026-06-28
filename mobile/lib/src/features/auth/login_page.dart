import 'package:flutter/material.dart';

import '../../state/app_state.dart';

class LoginPage extends StatefulWidget {
  const LoginPage({required this.state, super.key});

  final AppState state;

  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  final _formKey = GlobalKey<FormState>();
  final _serverController = TextEditingController();
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();
  final _totpController = TextEditingController();

  @override
  void initState() {
    super.initState();
    _serverController.text = widget.state.serverUrl ?? 'http://10.0.2.2:8080';
  }

  @override
  void dispose() {
    _serverController.dispose();
    _emailController.dispose();
    _passwordController.dispose();
    _totpController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final state = widget.state;
    final scheme = Theme.of(context).colorScheme;

    return Scaffold(
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(24),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 420),
              child: Form(
                key: _formKey,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    Icon(Icons.calendar_month, size: 44, color: scheme.primary),
                    const SizedBox(height: 18),
                    Text('CalendarAdvanced',
                        style: Theme.of(context).textTheme.headlineMedium,
                        textAlign: TextAlign.center),
                    const SizedBox(height: 6),
                    Text('Kalender und Aufgaben',
                        style: Theme.of(context).textTheme.bodyLarge,
                        textAlign: TextAlign.center),
                    const SizedBox(height: 28),
                    TextFormField(
                      controller: _serverController,
                      decoration: const InputDecoration(
                          labelText: 'Server-URL',
                          prefixIcon: Icon(Icons.dns_outlined)),
                      keyboardType: TextInputType.url,
                      validator: (value) =>
                          value == null || value.trim().isEmpty
                              ? 'Server-URL fehlt.'
                              : null,
                    ),
                    const SizedBox(height: 12),
                    TextFormField(
                      controller: _emailController,
                      decoration: const InputDecoration(
                          labelText: 'E-Mail oder Benutzername',
                          prefixIcon: Icon(Icons.person_outline)),
                      textInputAction: TextInputAction.next,
                      validator: (value) =>
                          value == null || value.trim().isEmpty
                              ? 'Login fehlt.'
                              : null,
                    ),
                    const SizedBox(height: 12),
                    TextFormField(
                      controller: _passwordController,
                      decoration: const InputDecoration(
                          labelText: 'Passwort',
                          prefixIcon: Icon(Icons.lock_outline)),
                      obscureText: true,
                      validator: (value) => value == null || value.isEmpty
                          ? 'Passwort fehlt.'
                          : null,
                    ),
                    const SizedBox(height: 12),
                    TextFormField(
                      controller: _totpController,
                      decoration: const InputDecoration(
                          labelText: '2FA-Code optional',
                          prefixIcon: Icon(Icons.pin_outlined)),
                      keyboardType: TextInputType.number,
                    ),
                    if (state.error != null) ...[
                      const SizedBox(height: 14),
                      Text(state.error!, style: TextStyle(color: scheme.error)),
                    ],
                    const SizedBox(height: 20),
                    FilledButton.icon(
                      onPressed: state.busy ? null : _submit,
                      icon: state.busy
                          ? const SizedBox.square(
                              dimension: 18,
                              child: CircularProgressIndicator(strokeWidth: 2))
                          : const Icon(Icons.login),
                      label: const Text('Anmelden'),
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
    if (!_formKey.currentState!.validate()) {
      return;
    }
    await widget.state.login(
      _serverController.text,
      _emailController.text,
      _passwordController.text,
      _totpController.text,
    );
  }
}
