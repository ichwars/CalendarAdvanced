# UI- und Designleitlinien

Diese Leitlinien gelten für die gesamte CalendarAdvanced-Oberfläche.

## Grundhaltung

CalendarAdvanced soll ruhig, professionell und vertrauenswürdig wirken. Die App ist ein Arbeitswerkzeug für private Kalenderdaten und sollte deshalb klar, reduziert und gut lesbar bleiben.

## Icons und Aktionen

- Icons werden dort bevorzugt, wo die Bedeutung eindeutig ist oder durch Tooltip und `aria-label` eindeutig gemacht wird.
- Icon-Aktionen haben transparenten Hintergrund und keinen sichtbaren Rand.
- Hover-, Fokus- und Aktivzustände bei Icon-Aktionen werden primär über Farbwechsel dargestellt.
- Textbuttons bleiben fuer klare Befehle, Formularaktionen und riskante Aktionen erlaubt.
- Loesch- und Sicherheitsaktionen sollen optisch zurueckhaltend bleiben und erst durch Kontext, Text oder Farbe klar werden.

## Typografie

- Die Hauptschrift ist Inter mit System-Fallbacks.
- Schriftgrößen und Gewichte bleiben moderat; UI-Elemente sollen nicht laut oder schwer wirken.
- Normale Aktionsbuttons verwenden maximal `font-weight: 600`.
- Überschriften dürfen stärker sein, bleiben aber ohne negative Laufweite.
- Umlaute und deutsche Begriffe werden korrekt ausgeschrieben.

## Struktur und Dateiumfang

- CSS wird nach Zweck getrennt: Tokens, Basis, Layout, Komponenten, Kalender und Responsive-Regeln.
- Neue größere UI-Bereiche erhalten eigene Komponenten oder Styles, statt bestehende Dateien unbegrenzt zu erweitern.
- Wiederverwendbare UI-Elemente gehören nach `frontend/src/shared/components`.
