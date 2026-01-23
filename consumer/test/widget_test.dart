// This is a basic Flutter widget test.
//
// To perform an interaction with a widget in your test, use the WidgetTester
// utility in the flutter_test package. For example, you can send tap and scroll
// gestures. You can also use WidgetTester to find child widgets in the widget
// tree, read text, and verify that the values of widget properties are correct.

import 'package:flutter_test/flutter_test.dart';

import 'package:consumer/main.dart';

void main() {
  testWidgets('Consumer app smoke test', (WidgetTester tester) async {
    // Build our app and trigger a frame.
    await tester.pumpWidget(const ConsumerApp(envLoaded: false));

    // Verify that our app starts.
    expect(find.text('Node is Active'), findsOneWidget);
    expect(find.text('Environment: Missing .env'), findsOneWidget);
  });
}
