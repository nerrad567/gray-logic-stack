/// Application-wide constants for the Gray Logic Wall Panel.
class AppConstants {
  AppConstants._();

  // API
  static const int apiConnectTimeoutMs = 5000;
  static const int apiReceiveTimeoutMs = 10000;
  static const String apiVersion = 'v1';

  // WebSocket
  static const int wsPingIntervalMs = 30000;
  static const int wsReconnectBaseMs = 1000;
  static const int wsReconnectMaxMs = 30000;
  static const int wsSendBufferSize = 256;

  // Optimistic updates
  static const int optimisticRollbackMs = 3000;
  static const int commandDebounceMs = 300;
  static const int sliderDebounceMs = 250;

  // Auth
  static const int ticketTtlSeconds = 60;
  static const String tokenStorageKey = 'auth_token';
  static const String refreshTokenStorageKey = 'auth_refresh_token';
  static const String coreUrlStorageKey = 'core_base_url';
  static const String roomIdStorageKey = 'configured_room_id';
  static const String locationCacheKey = 'cached_location_data';

  // UI
  static const double tileSize = 150.0;
  static const double tilePadding = 12.0;
  static const double touchTargetMin = 56.0;
}
