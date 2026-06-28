# CalendarAdvanced Mobile

Flutter client for the CalendarAdvanced server API.

## Current scope

- Login against `/api/v1/auth/login`
- Secure local storage for server URL, session cookie and CSRF token
- Calendar agenda with event create, edit and delete
- Task list with create, edit, complete and delete
- Lightweight Material 3 UI for Android first, with iOS kept in mind

## Local setup

Install Flutter, then run:

```bash
cd mobile
flutter pub get
flutter create --platforms=android,ios .
flutter run
```

For an Android emulator talking to a backend on the host machine, use:

```text
http://10.0.2.2:8080
```

For a physical device, use the LAN address or HTTPS public URL of the server.

## Next mobile steps

- Add Android/iOS native project files with `flutter create`.
- Add Firebase Cloud Messaging and local notification wiring.
- Add Android home screen widget and iOS widget extension.
- Add optional API bearer tokens on the backend for first-class mobile sessions.
