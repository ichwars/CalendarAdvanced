import 'package:flutter/material.dart';

import '../../state/app_state.dart';

class SettingsPage extends StatelessWidget {
  const SettingsPage({required this.state, super.key});

  final AppState state;

  @override
  Widget build(BuildContext context) {
    final user = state.user;

    return Scaffold(
      appBar: AppBar(title: const Text('Einstellungen')),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(16, 8, 16, 32),
        children: [
          Card(
            child: ListTile(
              leading: const Icon(Icons.person_outline),
              title: Text(user?.displayName.isNotEmpty == true
                  ? user!.displayName
                  : user?.username ?? ''),
              subtitle: Text(user?.email ?? ''),
            ),
          ),
          const SizedBox(height: 12),
          Card(
            child: ListTile(
              leading: const Icon(Icons.dns_outlined),
              title: const Text('Server'),
              subtitle: Text(state.serverUrl ?? 'Nicht gesetzt'),
            ),
          ),
          const SizedBox(height: 12),
          Card(
            child: Column(
              children: [
                ListTile(
                  leading: const Icon(Icons.sync),
                  title: const Text('Daten aktualisieren'),
                  trailing: state.busy
                      ? const SizedBox.square(
                          dimension: 20,
                          child: CircularProgressIndicator(strokeWidth: 2))
                      : const Icon(Icons.chevron_right),
                  onTap: state.busy ? null : state.refreshAll,
                ),
                const Divider(height: 1),
                ListTile(
                  leading: const Icon(Icons.logout),
                  title: const Text('Abmelden'),
                  onTap: state.logout,
                ),
              ],
            ),
          ),
          const SizedBox(height: 18),
          Text(
            'Push-Benachrichtigungen und Widgets werden als native Android/iOS-Erweiterungen auf dieser Grundlage angebunden.',
            style: Theme.of(context).textTheme.bodySmall,
          ),
        ],
      ),
    );
  }
}
