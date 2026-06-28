import 'package:flutter/material.dart';
import 'package:intl/date_symbol_data_local.dart';

import 'src/app.dart';
import 'src/services/api_client.dart';
import 'src/state/app_state.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await initializeDateFormatting('de');

  final state = AppState(api: ApiClient());
  await state.restoreSession();

  runApp(CalendarAdvancedApp(state: state));
}
