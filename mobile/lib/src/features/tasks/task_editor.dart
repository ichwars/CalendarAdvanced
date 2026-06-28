import 'package:flutter/material.dart';

import '../../models.dart';
import '../../state/app_state.dart';

class TaskEditor extends StatefulWidget {
  const TaskEditor({required this.state, this.task, super.key});

  final AppState state;
  final TaskItem? task;

  @override
  State<TaskEditor> createState() => _TaskEditorState();
}

class _TaskEditorState extends State<TaskEditor> {
  final _formKey = GlobalKey<FormState>();
  late final TextEditingController _title;
  late final TextEditingController _description;
  DateTime? _dueAt;
  DateTime? _reminderAt;
  late String _priority;
  late bool _completed;
  late bool _showInCalendar;

  @override
  void initState() {
    super.initState();
    final task = widget.task;
    _title = TextEditingController(text: task?.title ?? '');
    _description = TextEditingController(text: task?.description ?? '');
    _dueAt = task?.dueAt;
    _reminderAt = task?.reminderAt;
    _priority = task?.priority ?? 'normal';
    _completed = task?.completed ?? false;
    _showInCalendar = task?.showInCalendar ?? true;
  }

  @override
  void dispose() {
    _title.dispose();
    _description.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final bottom = MediaQuery.of(context).viewInsets.bottom;

    return Padding(
      padding: EdgeInsets.fromLTRB(16, 16, 16, bottom + 16),
      child: Form(
        key: _formKey,
        child: ListView(
          shrinkWrap: true,
          children: [
            Row(
              children: [
                Expanded(
                    child: Text(
                        widget.task == null
                            ? 'Task anlegen'
                            : 'Task bearbeiten',
                        style: Theme.of(context).textTheme.titleLarge)),
                IconButton(
                    tooltip: 'Schliessen',
                    onPressed: () => Navigator.pop(context),
                    icon: const Icon(Icons.close)),
              ],
            ),
            const SizedBox(height: 12),
            TextFormField(
              controller: _title,
              decoration: const InputDecoration(
                  labelText: 'Titel', prefixIcon: Icon(Icons.title)),
              validator: (value) =>
                  value == null || value.trim().isEmpty ? 'Titel fehlt.' : null,
            ),
            const SizedBox(height: 12),
            TextFormField(
              controller: _description,
              minLines: 3,
              maxLines: 5,
              decoration: const InputDecoration(
                  labelText: 'Notizen', prefixIcon: Icon(Icons.notes_outlined)),
            ),
            const SizedBox(height: 12),
            DropdownButtonFormField<String>(
              initialValue: _priority,
              decoration: const InputDecoration(
                  labelText: 'Prioritaet',
                  prefixIcon: Icon(Icons.flag_outlined)),
              items: const [
                DropdownMenuItem(value: 'low', child: Text('Niedrig')),
                DropdownMenuItem(value: 'normal', child: Text('Normal')),
                DropdownMenuItem(value: 'high', child: Text('Hoch')),
              ],
              onChanged: (value) =>
                  setState(() => _priority = value ?? 'normal'),
            ),
            const SizedBox(height: 12),
            _OptionalDateTimeRow(
              label: 'Faelligkeit',
              value: _dueAt,
              onChanged: (value) => setState(() => _dueAt = value),
              onClear: () => setState(() => _dueAt = null),
            ),
            const SizedBox(height: 8),
            _OptionalDateTimeRow(
              label: 'Erinnerung',
              value: _reminderAt,
              onChanged: (value) => setState(() => _reminderAt = value),
              onClear: () => setState(() => _reminderAt = null),
            ),
            SwitchListTile(
              contentPadding: EdgeInsets.zero,
              title: const Text('Im Kalender anzeigen'),
              value: _showInCalendar,
              onChanged: (value) => setState(() => _showInCalendar = value),
            ),
            SwitchListTile(
              contentPadding: EdgeInsets.zero,
              title: const Text('Erledigt'),
              value: _completed,
              onChanged: (value) => setState(() => _completed = value),
            ),
            const SizedBox(height: 18),
            Row(
              children: [
                if (widget.task != null)
                  IconButton.filledTonal(
                    tooltip: 'Loeschen',
                    onPressed: _delete,
                    icon: const Icon(Icons.delete_outline),
                  ),
                const Spacer(),
                FilledButton.icon(
                    onPressed: _save,
                    icon: const Icon(Icons.check),
                    label: const Text('Speichern')),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _save() async {
    if (!_formKey.currentState!.validate()) {
      return;
    }
    final task = TaskItem(
      id: widget.task?.id ?? 0,
      title: _title.text,
      description: _description.text,
      dueAt: _dueAt,
      reminderAt: _reminderAt,
      priority: _priority,
      completed: _completed,
      showInCalendar: _showInCalendar,
    );
    await widget.state.saveTask(task);
    if (mounted && widget.state.error == null) {
      Navigator.pop(context);
    }
  }

  Future<void> _delete() async {
    final task = widget.task;
    if (task == null) {
      return;
    }
    await widget.state.deleteTask(task.id);
    if (mounted && widget.state.error == null) {
      Navigator.pop(context);
    }
  }
}

class _OptionalDateTimeRow extends StatelessWidget {
  const _OptionalDateTimeRow({
    required this.label,
    required this.value,
    required this.onChanged,
    required this.onClear,
  });

  final String label;
  final DateTime? value;
  final ValueChanged<DateTime> onChanged;
  final VoidCallback onClear;

  @override
  Widget build(BuildContext context) {
    final current = value;
    return Row(
      children: [
        Expanded(
          child: OutlinedButton.icon(
            onPressed: () => _pick(context),
            icon: const Icon(Icons.schedule),
            label: Align(
              alignment: Alignment.centerLeft,
              child: Text(current == null
                  ? label
                  : '$label: ${MaterialLocalizations.of(context).formatFullDate(current)} ${TimeOfDay.fromDateTime(current).format(context)}'),
            ),
          ),
        ),
        if (current != null)
          IconButton(
            tooltip: 'Entfernen',
            onPressed: onClear,
            icon: const Icon(Icons.close),
          ),
      ],
    );
  }

  Future<void> _pick(BuildContext context) async {
    final now = DateTime.now();
    final initial =
        value ?? DateTime(now.year, now.month, now.day, now.hour + 1);
    final date = await showDatePicker(
        context: context,
        initialDate: initial,
        firstDate: DateTime(2000),
        lastDate: DateTime(2100));
    if (date == null || !context.mounted) {
      return;
    }
    final time = await showTimePicker(
        context: context, initialTime: TimeOfDay.fromDateTime(initial));
    if (time == null) {
      return;
    }
    onChanged(
        DateTime(date.year, date.month, date.day, time.hour, time.minute));
  }
}
