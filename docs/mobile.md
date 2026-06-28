# Mobile app plan

CalendarAdvanced Mobile is a Flutter app in `mobile/`. The first implementation talks to the existing JSON API and keeps the server as the source of truth. Direct CalDAV support should remain server-side first, because the server can validate, audit, synchronize and later trigger push events in one place.

## Architecture

- Flutter app with Material 3 UI.
- Server API access through `ApiClient`.
- Session cookie and CSRF token stored with `flutter_secure_storage`.
- `AppState` keeps the first MVP simple while leaving room for Riverpod/BLoC later if the app grows.
- Event and task editors use the same payload shape as the web frontend.

## Security notes

- Prefer HTTPS for real devices and any remote server.
- `http://10.0.2.2:8080` is only for Android emulator development.
- The current backend session model works for MVP use, but mobile-specific bearer/app tokens would be cleaner for Play Store release.
- Do not log session cookies, CSRF tokens, passwords or CalDAV credentials.

## Push and widgets

Planned native work after the Flutter SDK is installed:

- Add `firebase_core`, `firebase_messaging` and `flutter_local_notifications`.
- Add backend endpoint for registering a device push token.
- Schedule local reminders for events/tasks cached on device.
- Add Android App Widget for today's agenda and open tasks.
- Add iOS WidgetKit extension once the iOS target is generated.
