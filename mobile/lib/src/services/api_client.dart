import 'dart:async';
import 'dart:convert';

import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:http/http.dart' as http;

import '../models.dart';

class ApiClient {
  ApiClient({FlutterSecureStorage? storage})
      : _storage = storage ?? const FlutterSecureStorage();

  final FlutterSecureStorage _storage;

  String? _baseUrl;
  String? _sessionCookie;
  String? _csrfToken;

  Future<String?> getStoredServerUrl() => _storage.read(key: _serverUrlKey);

  Future<void> restore() async {
    _baseUrl = await _storage.read(key: _serverUrlKey);
    _sessionCookie = await _storage.read(key: _sessionCookieKey);
    _csrfToken = await _storage.read(key: _csrfTokenKey);
  }

  Future<void> configureServer(String rawUrl) async {
    final normalized = _normalizeBaseUrl(rawUrl);
    _baseUrl = normalized;
    await _storage.write(key: _serverUrlKey, value: normalized);
  }

  Future<User> login({
    required String serverUrl,
    required String email,
    required String password,
    String? totpCode,
  }) async {
    await configureServer(serverUrl);
    final response = await _send(
      'POST',
      '/api/v1/auth/login',
      body: {
        'email': email,
        'password': password,
        if (totpCode != null && totpCode.isNotEmpty) 'totpCode': totpCode,
      },
      includeAuth: false,
    );

    await _captureCookies(response);
    final payload = _decode(response);
    final csrfToken = payload['csrfToken'] as String?;
    if (csrfToken != null && csrfToken.isNotEmpty) {
      _csrfToken = csrfToken;
      await _storage.write(key: _csrfTokenKey, value: csrfToken);
    }
    final user = User.fromJson(payload['user'] as Map<String, dynamic>);
    return user;
  }

  Future<void> logout() async {
    try {
      await _send('POST', '/api/v1/auth/logout', body: <String, dynamic>{});
    } finally {
      await clearSession();
    }
  }

  Future<void> clearSession() async {
    _sessionCookie = null;
    _csrfToken = null;
    await _storage.delete(key: _sessionCookieKey);
    await _storage.delete(key: _csrfTokenKey);
  }

  Future<User> me() async {
    final response = await _send('GET', '/api/v1/auth/me');
    final payload = _decode(response);
    final csrfToken = payload['csrfToken'] as String?;
    if (csrfToken != null && csrfToken.isNotEmpty) {
      _csrfToken = csrfToken;
      await _storage.write(key: _csrfTokenKey, value: csrfToken);
    }
    return User.fromJson(payload['user'] as Map<String, dynamic>);
  }

  Future<List<CalendarSource>> calendars() async {
    final response = await _send('GET', '/api/v1/calendars');
    final payload = _decode(response);
    return (payload['items'] as List<dynamic>? ?? const [])
        .map((item) => CalendarSource.fromJson(item as Map<String, dynamic>))
        .toList();
  }

  Future<List<EventItem>> events(
      {required DateTime from, required DateTime to}) async {
    final query = Uri(queryParameters: {
      'from': from.toUtc().toIso8601String(),
      'to': to.toUtc().toIso8601String(),
      'expand': 'true',
      'limit': '500',
    }).query;
    final response = await _send('GET', '/api/v1/events?$query');
    final payload = _decode(response);
    return (payload['items'] as List<dynamic>? ?? const [])
        .map((item) => EventItem.fromJson(item as Map<String, dynamic>))
        .toList()
      ..sort((a, b) => a.startsAt.compareTo(b.startsAt));
  }

  Future<EventItem> createEvent(EventItem event) async {
    final response =
        await _send('POST', '/api/v1/events', body: event.toPayload());
    return EventItem.fromJson(_decode(response));
  }

  Future<EventItem> updateEvent(EventItem event) async {
    final response = await _send('PUT', '/api/v1/events/${event.id}',
        body: event.toPayload());
    return EventItem.fromJson(_decode(response));
  }

  Future<void> deleteEvent(int id) async {
    await _send('DELETE', '/api/v1/events/$id');
  }

  Future<List<TaskItem>> tasks() async {
    final query = Uri(queryParameters: {'limit': '500'}).query;
    final response = await _send('GET', '/api/v1/tasks?$query');
    final payload = _decode(response);
    return (payload['items'] as List<dynamic>? ?? const [])
        .map((item) => TaskItem.fromJson(item as Map<String, dynamic>))
        .toList()
      ..sort(_sortTasks);
  }

  Future<TaskItem> createTask(TaskItem task) async {
    final response =
        await _send('POST', '/api/v1/tasks', body: task.toPayload());
    return TaskItem.fromJson(_decode(response));
  }

  Future<TaskItem> updateTask(TaskItem task) async {
    final response =
        await _send('PUT', '/api/v1/tasks/${task.id}', body: task.toPayload());
    return TaskItem.fromJson(_decode(response));
  }

  Future<void> deleteTask(int id) async {
    await _send('DELETE', '/api/v1/tasks/$id');
  }

  Future<http.Response> _send(
    String method,
    String path, {
    Map<String, dynamic>? body,
    bool includeAuth = true,
  }) async {
    final baseUrl = _baseUrl;
    if (baseUrl == null || baseUrl.isEmpty) {
      throw const ApiException('server_missing', 'Bitte Server-URL eintragen.');
    }

    final uri = Uri.parse('$baseUrl$path');
    final headers = <String, String>{
      'Accept': 'application/json',
      if (body != null) 'Content-Type': 'application/json',
      if (includeAuth && _sessionCookie != null) 'Cookie': _sessionCookie!,
      if (includeAuth && _csrfToken != null) 'X-CSRF-Token': _csrfToken!,
    };
    final encodedBody = body == null ? null : jsonEncode(body);
    final Future<http.Response> request = switch (method) {
      'GET' => http.get(uri, headers: headers),
      'POST' => http.post(uri, headers: headers, body: encodedBody),
      'PUT' => http.put(uri, headers: headers, body: encodedBody),
      'DELETE' => http.delete(uri, headers: headers),
      _ => throw const ApiException(
          'unsupported_method', 'Nicht unterstuetzte Methode.'),
    };

    late final http.Response response;
    try {
      response = await request.timeout(_requestTimeout);
    } on TimeoutException {
      throw const ApiException(
        'network_timeout',
        'Der Server antwortet nicht. Bitte Server-URL und Erreichbarkeit pruefen.',
      );
    } on http.ClientException {
      throw const ApiException(
        'network_error',
        'Der Server ist nicht erreichbar. Bitte Server-URL und Netzwerk pruefen.',
      );
    }

    await _captureCookies(response);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw _errorFromResponse(response);
    }
    return response;
  }

  Map<String, dynamic> _decode(http.Response response) {
    if (response.body.isEmpty) {
      return <String, dynamic>{};
    }
    return jsonDecode(utf8.decode(response.bodyBytes)) as Map<String, dynamic>;
  }

  Future<void> _captureCookies(http.Response response) async {
    final setCookie = response.headers['set-cookie'];
    if (setCookie == null || setCookie.isEmpty) {
      return;
    }
    final session = _readCookieValue(setCookie, 'ck_session');
    final csrf = _readCookieValue(setCookie, 'ck_csrf');
    final cookies = <String>[];
    if (session != null && session.isNotEmpty) {
      cookies.add('ck_session=$session');
    }
    if (csrf != null && csrf.isNotEmpty) {
      cookies.add('ck_csrf=$csrf');
      _csrfToken = csrf;
      await _storage.write(key: _csrfTokenKey, value: csrf);
    }
    if (cookies.isNotEmpty) {
      _sessionCookie = cookies.join('; ');
      await _storage.write(key: _sessionCookieKey, value: _sessionCookie);
    }
  }

  String? _readCookieValue(String header, String name) {
    final match = RegExp('(?:^|,\\s*)$name=([^;,]*)').firstMatch(header);
    return match?.group(1);
  }

  ApiException _errorFromResponse(http.Response response) {
    try {
      final payload = _decode(response);
      return ApiException(
        payload['code'] as String? ?? 'http_${response.statusCode}',
        payload['message'] as String? ??
            response.reasonPhrase ??
            'Request fehlgeschlagen.',
      );
    } catch (_) {
      return ApiException('http_${response.statusCode}',
          response.reasonPhrase ?? 'Request fehlgeschlagen.');
    }
  }
}

class ApiException implements Exception {
  const ApiException(this.code, this.message);

  final String code;
  final String message;

  @override
  String toString() => message;
}

String _normalizeBaseUrl(String rawUrl) {
  final trimmed = rawUrl.trim();
  final withScheme =
      trimmed.startsWith('http://') || trimmed.startsWith('https://')
          ? trimmed
          : 'https://$trimmed';
  return withScheme.endsWith('/')
      ? withScheme.substring(0, withScheme.length - 1)
      : withScheme;
}

int _sortTasks(TaskItem a, TaskItem b) {
  if (a.completed != b.completed) {
    return a.completed ? 1 : -1;
  }
  final dueA = a.dueAt;
  final dueB = b.dueAt;
  if (dueA != null && dueB != null) {
    return dueA.compareTo(dueB);
  }
  if (dueA != null) {
    return -1;
  }
  if (dueB != null) {
    return 1;
  }
  return a.title.toLowerCase().compareTo(b.title.toLowerCase());
}

const _serverUrlKey = 'calendaradvanced.server_url';
const _sessionCookieKey = 'calendaradvanced.session_cookie';
const _csrfTokenKey = 'calendaradvanced.csrf_token';
const _requestTimeout = Duration(seconds: 10);
