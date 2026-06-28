import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

import '../../models.dart';
import '../../state/app_state.dart';
import 'task_editor.dart';

class TasksPage extends StatelessWidget {
  const TasksPage({required this.state, super.key});

  final AppState state;

  @override
  Widget build(BuildContext context) {
    final open = state.tasks.where((task) => !task.completed).toList();
    final done = state.tasks.where((task) => task.completed).toList();

    return Scaffold(
      appBar: AppBar(
        title: const Text('Tasks'),
        actions: [
          IconButton(
              tooltip: 'Aktualisieren',
              onPressed: state.busy ? null : state.refreshAll,
              icon: const Icon(Icons.refresh)),
        ],
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: () => _openEditor(context),
        child: const Icon(Icons.add),
      ),
      body: RefreshIndicator(
        onRefresh: state.refreshAll,
        child: ListView(
          padding: const EdgeInsets.fromLTRB(16, 8, 16, 96),
          children: [
            if (state.error != null)
              Padding(
                padding: const EdgeInsets.only(bottom: 12),
                child: Text(state.error!,
                    style:
                        TextStyle(color: Theme.of(context).colorScheme.error)),
              ),
            if (open.isEmpty && done.isEmpty)
              const Padding(
                padding: EdgeInsets.only(top: 96),
                child: Center(child: Text('Keine Tasks vorhanden.')),
              ),
            for (final task in open) _TaskTile(task: task, state: state),
            if (done.isNotEmpty) ...[
              const SizedBox(height: 18),
              Text('Erledigt', style: Theme.of(context).textTheme.titleMedium),
              const SizedBox(height: 8),
              for (final task in done) _TaskTile(task: task, state: state),
            ],
          ],
        ),
      ),
    );
  }

  Future<void> _openEditor(BuildContext context, [TaskItem? task]) async {
    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      builder: (context) => TaskEditor(state: state, task: task),
    );
  }
}

class _TaskTile extends StatelessWidget {
  const _TaskTile({required this.task, required this.state});

  final TaskItem task;
  final AppState state;

  @override
  Widget build(BuildContext context) {
    final dueAt = task.dueAt;
    final subtitle = [
      if (dueAt != null) DateFormat.yMd('de').add_Hm().format(dueAt),
      _priorityLabel(task.priority),
      if (task.showInCalendar) 'im Kalender',
    ].join(' - ');

    return Card(
      child: CheckboxListTile(
        value: task.completed,
        onChanged: (value) =>
            state.saveTask(task.copyWith(completed: value ?? false)),
        title: Text(task.title, maxLines: 2, overflow: TextOverflow.ellipsis),
        subtitle: subtitle.isEmpty ? null : Text(subtitle),
        secondary: IconButton(
          tooltip: 'Bearbeiten',
          icon: const Icon(Icons.edit_outlined),
          onPressed: () => showModalBottomSheet<void>(
            context: context,
            isScrollControlled: true,
            useSafeArea: true,
            builder: (context) => TaskEditor(state: state, task: task),
          ),
        ),
      ),
    );
  }
}

String _priorityLabel(String value) {
  return switch (value) {
    'high' => 'hoch',
    'low' => 'niedrig',
    _ => 'normal',
  };
}
