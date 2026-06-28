import 'package:flutter/foundation.dart';

import '../models.dart';
import '../services/api_client.dart';

class AppState extends ChangeNotifier {
  AppState({required this.api});

  final ApiClient api;

  User? user;
  String? serverUrl;
  String? error;
  bool busy = false;
  List<CalendarSource> calendars = const [];
  List<EventItem> events = const [];
  List<TaskItem> tasks = const [];

  Future<void> restoreSession() async {
    await api.restore();
    serverUrl = await api.getStoredServerUrl();
    if (serverUrl == null) {
      notifyListeners();
      return;
    }
    await _guard(() async {
      user = await api.me();
      await refreshAll();
    }, clearOnUnauthorized: true);
  }

  Future<void> login(
      String serverUrl, String email, String password, String totpCode) async {
    await _guard(() async {
      user = await api.login(
          serverUrl: serverUrl,
          email: email,
          password: password,
          totpCode: totpCode);
      this.serverUrl = serverUrl;
      await refreshAll();
    });
  }

  Future<void> logout() async {
    await api.logout();
    user = null;
    calendars = const [];
    events = const [];
    tasks = const [];
    notifyListeners();
  }

  Future<void> refreshAll() async {
    final now = DateTime.now();
    calendars = await api.calendars();
    events = await api.events(
        from: now.subtract(const Duration(days: 14)),
        to: now.add(const Duration(days: 60)));
    tasks = await api.tasks();
    notifyListeners();
  }

  Future<void> saveEvent(EventItem event) async {
    await _guard(() async {
      if (event.id == 0) {
        await api.createEvent(event);
      } else {
        await api.updateEvent(event);
      }
      await refreshAll();
    });
  }

  Future<void> deleteEvent(int id) async {
    await _guard(() async {
      await api.deleteEvent(id);
      await refreshAll();
    });
  }

  Future<void> saveTask(TaskItem task) async {
    await _guard(() async {
      if (task.id == 0) {
        await api.createTask(task);
      } else {
        await api.updateTask(task);
      }
      await refreshAll();
    });
  }

  Future<void> deleteTask(int id) async {
    await _guard(() async {
      await api.deleteTask(id);
      await refreshAll();
    });
  }

  Future<void> _guard(Future<void> Function() action,
      {bool clearOnUnauthorized = false}) async {
    busy = true;
    error = null;
    notifyListeners();
    try {
      await action();
    } on ApiException catch (exception) {
      error = exception.message;
      if (clearOnUnauthorized && exception.code == 'unauthorized') {
        await api.clearSession();
        user = null;
      }
    } catch (_) {
      error = 'Die App konnte den Server nicht erreichen.';
    } finally {
      busy = false;
      notifyListeners();
    }
  }
}
