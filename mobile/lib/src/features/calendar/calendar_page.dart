import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

import '../../models.dart';
import '../../state/app_state.dart';
import 'event_editor.dart';

class CalendarPage extends StatelessWidget {
  const CalendarPage({required this.state, super.key});

  final AppState state;

  @override
  Widget build(BuildContext context) {
    final grouped = _groupEvents(state.events);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Kalender'),
        actions: [
          IconButton(
            tooltip: 'Aktualisieren',
            onPressed: state.busy ? null : state.refreshAll,
            icon: const Icon(Icons.refresh),
          ),
        ],
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: state.calendars.isEmpty ? null : () => _openEditor(context),
        child: const Icon(Icons.add),
      ),
      body: RefreshIndicator(
        onRefresh: state.refreshAll,
        child: state.calendars.isEmpty
            ? const _EmptyCalendar()
            : ListView(
                padding: const EdgeInsets.fromLTRB(16, 8, 16, 96),
                children: [
                  if (state.error != null) _ErrorBanner(message: state.error!),
                  if (grouped.isEmpty)
                    const Padding(
                      padding: EdgeInsets.only(top: 96),
                      child: Center(
                          child: Text('Keine Termine im aktuellen Zeitraum.')),
                    )
                  else
                    for (final entry in grouped.entries) ...[
                      Padding(
                        padding: const EdgeInsets.only(top: 18, bottom: 8),
                        child: Text(_dateTitle(entry.key),
                            style: Theme.of(context).textTheme.titleMedium),
                      ),
                      for (final event in entry.value)
                        _EventTile(event: event, state: state),
                    ],
                ],
              ),
      ),
    );
  }

  Future<void> _openEditor(BuildContext context, [EventItem? event]) async {
    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      builder: (context) => EventEditor(state: state, event: event),
    );
  }
}

class _EventTile extends StatelessWidget {
  const _EventTile({required this.event, required this.state});

  final EventItem event;
  final AppState state;

  @override
  Widget build(BuildContext context) {
    final formatter = DateFormat.Hm('de');
    CalendarSource? calendar;
    for (final item in state.calendars) {
      if (item.id == event.calendarId) {
        calendar = item;
        break;
      }
    }
    final color = _parseColor(calendar?.color);

    return Card(
      child: ListTile(
        contentPadding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
        leading: Container(
          width: 5,
          height: 46,
          decoration: BoxDecoration(
              color: color, borderRadius: BorderRadius.circular(8)),
        ),
        title: Text(event.title, maxLines: 2, overflow: TextOverflow.ellipsis),
        subtitle: Text([
          event.allDay
              ? 'Ganztag'
              : '${formatter.format(event.startsAt)} - ${formatter.format(event.endsAt)}',
          if (event.location.isNotEmpty) event.location,
        ].join(' - ')),
        trailing: IconButton(
          tooltip: 'Bearbeiten',
          icon: const Icon(Icons.edit_outlined),
          onPressed: () => showModalBottomSheet<void>(
            context: context,
            isScrollControlled: true,
            useSafeArea: true,
            builder: (context) => EventEditor(state: state, event: event),
          ),
        ),
      ),
    );
  }
}

class _EmptyCalendar extends StatelessWidget {
  const _EmptyCalendar();

  @override
  Widget build(BuildContext context) {
    return ListView(
      padding: const EdgeInsets.all(24),
      children: const [
        SizedBox(height: 96),
        Icon(Icons.calendar_month_outlined, size: 44),
        SizedBox(height: 16),
        Center(
            child: Text(
                'Noch kein Kalender vorhanden. Bitte zuerst in der Webversion einen Kalender anlegen.')),
      ],
    );
  }
}

class _ErrorBanner extends StatelessWidget {
  const _ErrorBanner({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Card(
      color: scheme.errorContainer,
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Text(message, style: TextStyle(color: scheme.onErrorContainer)),
      ),
    );
  }
}

Map<DateTime, List<EventItem>> _groupEvents(List<EventItem> events) {
  final grouped = <DateTime, List<EventItem>>{};
  for (final event in events) {
    final key =
        DateTime(event.startsAt.year, event.startsAt.month, event.startsAt.day);
    grouped.putIfAbsent(key, () => []).add(event);
  }
  return grouped;
}

String _dateTitle(DateTime date) {
  final today = DateTime.now();
  final todayKey = DateTime(today.year, today.month, today.day);
  if (date == todayKey) {
    return 'Heute';
  }
  return DateFormat.yMMMMEEEEd('de').format(date);
}

Color _parseColor(String? hex) {
  if (hex == null || hex.length != 7) {
    return const Color(0xff2f6f73);
  }
  return Color(int.parse('ff${hex.substring(1)}', radix: 16));
}
