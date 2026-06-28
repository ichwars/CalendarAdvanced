class User {
  const User({
    required this.id,
    required this.email,
    required this.username,
    required this.displayName,
    required this.roles,
  });

  final int id;
  final String email;
  final String username;
  final String displayName;
  final List<String> roles;

  factory User.fromJson(Map<String, dynamic> json) {
    return User(
      id: json['id'] as int,
      email: json['email'] as String? ?? '',
      username: json['username'] as String? ?? '',
      displayName: json['displayName'] as String? ?? '',
      roles: (json['roles'] as List<dynamic>? ?? const []).cast<String>(),
    );
  }
}

class CalendarSource {
  const CalendarSource({
    required this.id,
    required this.name,
    required this.color,
    required this.timezone,
    required this.visible,
  });

  final int id;
  final String name;
  final String color;
  final String timezone;
  final bool visible;

  factory CalendarSource.fromJson(Map<String, dynamic> json) {
    return CalendarSource(
      id: json['id'] as int,
      name: json['name'] as String? ?? 'Kalender',
      color: json['color'] as String? ?? '#2f6f73',
      timezone: json['timezone'] as String? ?? 'Europe/Berlin',
      visible: json['visible'] as bool? ?? true,
    );
  }
}

class EventItem {
  const EventItem({
    required this.id,
    required this.calendarId,
    required this.title,
    required this.startsAt,
    required this.endsAt,
    required this.timezone,
    required this.allDay,
    required this.private,
    required this.completed,
    this.description = '',
    this.location = '',
    this.reminders = const [],
  });

  final int id;
  final int calendarId;
  final String title;
  final String description;
  final String location;
  final DateTime startsAt;
  final DateTime endsAt;
  final String timezone;
  final bool allDay;
  final bool private;
  final bool completed;
  final List<int> reminders;

  factory EventItem.fromJson(Map<String, dynamic> json) {
    return EventItem(
      id: json['id'] as int,
      calendarId: json['calendarId'] as int,
      title: json['title'] as String? ?? '',
      description: json['description'] as String? ?? '',
      location: json['location'] as String? ?? '',
      startsAt: DateTime.parse(json['startsAt'] as String).toLocal(),
      endsAt: DateTime.parse(json['endsAt'] as String).toLocal(),
      timezone: json['timezone'] as String? ?? 'Europe/Berlin',
      allDay: json['allDay'] as bool? ?? false,
      private: json['private'] as bool? ?? false,
      completed: json['completed'] as bool? ?? false,
      reminders: (json['reminders'] as List<dynamic>? ?? const [])
          .map((item) =>
              (item as Map<String, dynamic>)['minutesBefore'] as int? ?? 0)
          .where((value) => value > 0)
          .toList(),
    );
  }

  Map<String, dynamic> toPayload() {
    return {
      'calendarId': calendarId,
      'title': title.trim(),
      'description': description.trim(),
      'location': location.trim(),
      'startsAt': startsAt.toUtc().toIso8601String(),
      'endsAt': endsAt.toUtc().toIso8601String(),
      'timezone': timezone,
      'allDay': allDay,
      'private': private,
      'completed': completed,
      'reminders':
          reminders.map((minutes) => {'minutesBefore': minutes}).toList(),
      'attendees': <Object>[],
    };
  }

  EventItem copyWith({
    int? calendarId,
    String? title,
    String? description,
    String? location,
    DateTime? startsAt,
    DateTime? endsAt,
    String? timezone,
    bool? allDay,
    bool? private,
    bool? completed,
    List<int>? reminders,
  }) {
    return EventItem(
      id: id,
      calendarId: calendarId ?? this.calendarId,
      title: title ?? this.title,
      description: description ?? this.description,
      location: location ?? this.location,
      startsAt: startsAt ?? this.startsAt,
      endsAt: endsAt ?? this.endsAt,
      timezone: timezone ?? this.timezone,
      allDay: allDay ?? this.allDay,
      private: private ?? this.private,
      completed: completed ?? this.completed,
      reminders: reminders ?? this.reminders,
    );
  }
}

class TaskItem {
  const TaskItem({
    required this.id,
    required this.title,
    required this.priority,
    required this.completed,
    required this.showInCalendar,
    this.description = '',
    this.dueAt,
    this.reminderAt,
  });

  final int id;
  final String title;
  final String description;
  final DateTime? dueAt;
  final DateTime? reminderAt;
  final String priority;
  final bool completed;
  final bool showInCalendar;

  factory TaskItem.fromJson(Map<String, dynamic> json) {
    return TaskItem(
      id: json['id'] as int,
      title: json['title'] as String? ?? '',
      description: json['description'] as String? ?? '',
      dueAt: _optionalDate(json['dueAt'] as String?),
      reminderAt: _optionalDate(json['reminderAt'] as String?),
      priority: json['priority'] as String? ?? 'normal',
      completed: json['completed'] as bool? ?? false,
      showInCalendar: json['showInCalendar'] as bool? ?? false,
    );
  }

  Map<String, dynamic> toPayload() {
    return {
      'title': title.trim(),
      'description': description.trim(),
      'dueAt': dueAt?.toUtc().toIso8601String(),
      'reminderAt': reminderAt?.toUtc().toIso8601String(),
      'priority': priority,
      'completed': completed,
      'showInCalendar': showInCalendar,
    };
  }

  TaskItem copyWith({
    String? title,
    String? description,
    DateTime? dueAt,
    DateTime? reminderAt,
    String? priority,
    bool? completed,
    bool? showInCalendar,
    bool clearDueAt = false,
    bool clearReminderAt = false,
  }) {
    return TaskItem(
      id: id,
      title: title ?? this.title,
      description: description ?? this.description,
      dueAt: clearDueAt ? null : dueAt ?? this.dueAt,
      reminderAt: clearReminderAt ? null : reminderAt ?? this.reminderAt,
      priority: priority ?? this.priority,
      completed: completed ?? this.completed,
      showInCalendar: showInCalendar ?? this.showInCalendar,
    );
  }
}

DateTime? _optionalDate(String? value) {
  if (value == null || value.isEmpty || value.startsWith('0001-')) {
    return null;
  }
  return DateTime.parse(value).toLocal();
}
