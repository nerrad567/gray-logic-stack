import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import 'package:wallpanel/models/scene.dart';
import 'package:wallpanel/providers/scene_provider.dart';
import 'package:wallpanel/repositories/scene_repository.dart';
import 'package:wallpanel/providers/auth_provider.dart';
import 'package:wallpanel/services/api_client.dart';
import 'package:wallpanel/services/token_storage.dart';
import 'package:wallpanel/widgets/scene_button.dart';

class MockSceneRepository extends Mock implements SceneRepository {}

class MockApiClient extends Mock implements ApiClient {}

class MockTokenStorage extends Mock implements TokenStorage {}

void main() {
  group('SceneButton', () {
    late MockSceneRepository mockRepo;

    setUp(() {
      mockRepo = MockSceneRepository();
    });

    Widget buildWidget(Scene scene, {String? activatingSceneId}) {
      return ProviderScope(
        overrides: [
          sceneRepositoryProvider.overrideWithValue(mockRepo),
          tokenStorageProvider.overrideWithValue(MockTokenStorage()),
          apiClientProvider.overrideWithValue(MockApiClient()),
          if (activatingSceneId != null)
            activatingSceneIdProvider.overrideWith((ref) => activatingSceneId),
        ],
        child: MaterialApp(
          home: Scaffold(
            body: Center(child: SceneButton(scene: scene)),
          ),
        ),
      );
    }

    testWidgets('displays scene name', (tester) async {
      final scene = Scene(
        id: 's1',
        name: 'Movie Night',
        slug: 'movie_night',
        icon: 'movie',
        colour: '#4ECDC4',
        createdAt: DateTime(2026),
        updatedAt: DateTime(2026),
      );

      await tester.pumpWidget(buildWidget(scene));
      expect(find.text('Movie Night'), findsOneWidget);
    });

    testWidgets('shows icon when provided', (tester) async {
      final scene = Scene(
        id: 's1',
        name: 'Cinema',
        slug: 'cinema',
        icon: 'movie',
        createdAt: DateTime(2026),
        updatedAt: DateTime(2026),
      );

      await tester.pumpWidget(buildWidget(scene));
      expect(find.byIcon(Icons.movie), findsOneWidget);
    });

    testWidgets('shows spinner when activating', (tester) async {
      final scene = Scene(
        id: 's1',
        name: 'Test Scene',
        slug: 'test_scene',
        createdAt: DateTime(2026),
        updatedAt: DateTime(2026),
      );

      await tester.pumpWidget(buildWidget(scene, activatingSceneId: 's1'));
      await tester.pump();

      expect(find.byType(CircularProgressIndicator), findsOneWidget);
    });

    testWidgets('no spinner when not activating', (tester) async {
      final scene = Scene(
        id: 's1',
        name: 'Test Scene',
        slug: 'test_scene',
        createdAt: DateTime(2026),
        updatedAt: DateTime(2026),
      );

      await tester.pumpWidget(buildWidget(scene));
      expect(find.byType(CircularProgressIndicator), findsNothing);
    });

    testWidgets('applies colour from scene', (tester) async {
      final scene = Scene(
        id: 's1',
        name: 'Relax',
        slug: 'relax',
        colour: '#FF6B6B',
        createdAt: DateTime(2026),
        updatedAt: DateTime(2026),
      );

      await tester.pumpWidget(buildWidget(scene));

      // The button text should use the scene colour
      final textWidget = tester.widget<Text>(find.text('Relax'));
      expect(textWidget.style!.color, const Color(0xFFFF6B6B));
    });

    testWidgets('renders without icon when none provided', (tester) async {
      final scene = Scene(
        id: 's1',
        name: 'No Icon',
        slug: 'no_icon',
        createdAt: DateTime(2026),
        updatedAt: DateTime(2026),
      );

      await tester.pumpWidget(buildWidget(scene));
      expect(find.text('No Icon'), findsOneWidget);
      // No icon should be rendered
      expect(find.byType(Icon), findsNothing);
    });
  });
}
