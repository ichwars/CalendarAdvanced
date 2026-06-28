import 'package:calendaradvanced_mobile/src/models.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('event payload matches CalendarAdvanced API shape', () {
    final event = EventItem(
      id: 0,
      calendarId: 7,
      title: 'Planung',
      startsAt: DateTime.utc(2026, 6, 28, 8),
      endsAt: DateTime.utc(2026, 6, 28, 9),
      timezone: 'Europe/Berlin',
      allDay: false,
      private: false,
      completed: false,
      reminders: const [30],
    );

    final payload = event.toPayload();

    expect(payload['calendarId'], 7);
    expect(payload['title'], 'Planung');
    expect(payload['timezone'], 'Europe/Berlin');
    expect(payload['reminders'], [
      {'minutesBefore': 30},
    ]);
  });
}
