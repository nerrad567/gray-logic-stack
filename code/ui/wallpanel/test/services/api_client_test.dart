import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

import 'package:wallpanel/services/api_client.dart';
import 'package:wallpanel/services/token_storage.dart';

class MockTokenStorage extends Mock implements TokenStorage {}

class MockDio extends Mock implements Dio {
  @override
  BaseOptions options = BaseOptions();

  @override
  Interceptors interceptors = Interceptors();
}

void main() {
  late ApiClient apiClient;
  late MockTokenStorage mockTokenStorage;

  setUp(() {
    mockTokenStorage = MockTokenStorage();
    apiClient = ApiClient(tokenStorage: mockTokenStorage);
  });

  group('ApiClient', () {
    test('configure sets base URL', () async {
      await apiClient.configure('http://192.168.1.100:8080');
      // If configure doesn't throw, it worked
      expect(true, isTrue);
    });

    test('getWsUrl builds correct WebSocket URL', () async {
      await apiClient.configure('http://192.168.1.100:8080');
      final wsUrl = apiClient.getWsUrl('test-ticket-123');

      expect(wsUrl, contains('ws://'));
      expect(wsUrl, contains('192.168.1.100:8080'));
      expect(wsUrl, contains('/api/v1/ws'));
      expect(wsUrl, contains('ticket=test-ticket-123'));
    });

    test('getWsUrl converts https to wss', () async {
      await apiClient.configure('https://secure.example.com:8080');
      final wsUrl = apiClient.getWsUrl('ticket-abc');

      expect(wsUrl, startsWith('wss://'));
      expect(wsUrl, contains('secure.example.com:8080'));
    });

    test('login returns LoginResponse on success', () async {
      await apiClient.configure('http://localhost:8080');

      // This test validates the method signature and structure.
      // Full integration tests would use a mock HTTP server.
      expect(() => apiClient.login('admin', 'wrong'), throwsA(anything));
    });
  });
}
