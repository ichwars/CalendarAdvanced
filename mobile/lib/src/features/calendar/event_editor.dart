import 'package:flutter/material.dart';

import '../../models.dart';
import '../../state/app_state.dart';

class EventEditor extends StatefulWidget {
  const EventEditor({required this.state, this.event, super.key});

  final AppState state;
  final EventItem? event;

  @override
  State<EventEditor> createState() => _EventEditorState();
}

class _EventEditorState extends State<EventEditor> {
  final _formKey = GlobalKey<FormState>();
  late final TextEditingController _title;
  late final TextEditingController _description;
  late final TextEditingController _location;
  late DateTime _startsAt;
  late DateTime _endsAt;
  late int _calendarId;
  late bool _allDay;
  late bool _private;
  int _reminderMinutes = 0;

  @override
  void initState() {
    super.initState();
    final event = widget.event;
    final now = DateTime.now();
    _title = TextEditingController(text: event?.title ?? '');
    _description = TextEditingController(text: event?.description ?? '');
    _location = TextEditingController(text: event?.location ?? '');
    _startsAt =
        event?.startsAt ?? DateTime(now.year, now.month, now.day, now.hour + 1);
    _endsAt = event?.endsAt ?? _startsAt.add(const Duration(hours: 1));
    _calendarId = event?.calendarId ?? widget.state.calendars.first.id;
    _allDay = event?.allDay ?? false;
    _private = event?.private ?? false;
    _reminderMinutes =
        event == null || event.reminders.isEmpty ? 0 : event.reminders.first;
  }

  @override
  void dispose() {
    _title.dispose();
    _description.dispose();
    _location.dispose();
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
                        widget.event == null
                            ? 'Termin anlegen'
                            : 'Termin bearbeiten',
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
            DropdownButtonFormField<int>(
              initialValue: _calendarId,
              decoration: const InputDecoration(
                  labelText: 'Kalender',
                  prefixIcon: Icon(Icons.calendar_month_outlined)),
              items: widget.state.calendars
                  .map((calendar) => DropdownMenuItem<int>(
                      value: calendar.id, child: Text(calendar.name)))
                  .toList(),
              onChanged: (value) =>
                  setState(() => _calendarId = value ?? _calendarId),
            ),
            const SizedBox(height: 12),
            _DateTimeRow(
                label: 'Beginn',
                value: _startsAt,
                onChanged: (value) => setState(() => _startsAt = value)),
            const SizedBox(height: 8),
            _DateTimeRow(
                label: 'Ende',
                value: _endsAt,
                onChanged: (value) => setState(() => _endsAt = value)),
            const SizedBox(height: 12),
            TextFormField(
              controller: _location,
              decoration: const InputDecoration(
                  labelText: 'Ort', prefixIcon: Icon(Icons.place_outlined)),
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
            SwitchListTile(
              contentPadding: EdgeInsets.zero,
              title: const Text('Ganztag'),
              value: _allDay,
              onChanged: (value) => setState(() => _allDay = value),
            ),
            SwitchListTile(
              contentPadding: EdgeInsets.zero,
              title: const Text('Privat'),
              value: _private,
              onChanged: (value) => setState(() => _private = value),
            ),
            DropdownButtonFormField<int>(
              initialValue: _reminderMinutes,
              decoration: const InputDecoration(
                  labelText: 'Erinnerung',
                  prefixIcon: Icon(Icons.notifications_outlined)),
              items: const [
                DropdownMenuItem(value: 0, child: Text('Keine')),
                DropdownMenuItem(value: 10, child: Text('10 Minuten vorher')),
                DropdownMenuItem(value: 30, child: Text('30 Minuten vorher')),
                DropdownMenuItem(value: 60, child: Text('1 Stunde vorher')),
                DropdownMenuItem(value: 1440, child: Text('1 Tag vorher')),
              ],
              onChanged: (value) =>
                  setState(() => _reminderMinutes = value ?? 0),
            ),
            const SizedBox(height: 18),
            Row(
              children: [
                if (widget.event != null)
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
    if (!_endsAt.isAfter(_startsAt)) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(
          content: Text('Das Ende muss nach dem Beginn liegen.')));
      return;
    }
    final calendar =
        widget.state.calendars.firstWhere((item) => item.id == _calendarId);
    final event = EventItem(
      id: widget.event?.id ?? 0,
      calendarId: _calendarId,
      title: _title.text,
      description: _description.text,
      location: _location.text,
      startsAt: _startsAt,
      endsAt: _endsAt,
      timezone: calendar.timezone,
      allDay: _allDay,
      private: _private,
      completed: widget.event?.completed ?? false,
      reminders: _reminderMinutes > 0 ? [_reminderMinutes] : const [],
    );
    await widget.state.saveEvent(event);
    if (mounted && widget.state.error == null) {
      Navigator.pop(context);
    }
  }

  Future<void> _delete() async {
    final event = widget.event;
    if (event == null) {
      return;
    }
    await widget.state.deleteEvent(event.id);
    if (mounted && widget.state.error == null) {
      Navigator.pop(context);
    }
  }
}

class _DateTimeRow extends StatelessWidget {
  const _DateTimeRow(
      {required this.label, required this.value, required this.onChanged});

  final String label;
  final DateTime value;
  final ValueChanged<DateTime> onChanged;

  @override
  Widget build(BuildContext context) {
    return OutlinedButton.icon(
      onPressed: () => _pick(context),
      icon: const Icon(Icons.schedule),
      label: Align(
          alignment: Alignment.centerLeft,
          child: Text(
              '$label: ${MaterialLocalizations.of(context).formatFullDate(value)} ${TimeOfDay.fromDateTime(value).format(context)}')),
    );
  }

  Future<void> _pick(BuildContext context) async {
    final date = await showDatePicker(
      context: context,
      initialDate: value,
      firstDate: DateTime(2000),
      lastDate: DateTime(2100),
    );
    if (date == null || !context.mounted) {
      return;
    }
    final time = await showTimePicker(
        context: context, initialTime: TimeOfDay.fromDateTime(value));
    if (time == null) {
      return;
    }
    onChanged(
        DateTime(date.year, date.month, date.day, time.hour, time.minute));
  }
}
