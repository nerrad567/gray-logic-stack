import 'dart:typed_data';

import 'package:dio/dio.dart';

import '../config/constants.dart';
import '../models/area.dart';
import '../models/audit_log.dart';
import '../models/auth.dart';
import '../models/device.dart';
import '../models/ets_import.dart';
import '../models/discovery.dart';
import '../models/factory_reset.dart';
import '../models/site.dart';
import '../models/metrics.dart';
import '../models/room.dart';
import '../models/scene.dart';
import '../models/user.dart';
import 'token_storage.dart';

/// HTTP client for the Gray Logic Core REST API.
/// Handles authentication, retries, and error mapping.
class ApiClient {
  late final Dio _dio;
  final TokenStorage _tokenStorage;
  String? _baseUrl;

  ApiClient({required TokenStorage tokenStorage}) : _tokenStorage = tokenStorage {
    _dio = Dio(BaseOptions(
      connectTimeout: const Duration(milliseconds: AppConstants.apiConnectTimeoutMs),
      receiveTimeout: const Duration(milliseconds: AppConstants.apiReceiveTimeoutMs),
      headers: {'Content-Type': 'application/json'},
    ));

    _dio.interceptors.add(_AuthInterceptor(_tokenStorage));
  }

  /// Set the base URL for the API (called after configuration).
  Future<void> configure(String baseUrl) async {
    // Ensure trailing slash consistency
    _baseUrl = baseUrl.endsWith('/') ? baseUrl.substring(0, baseUrl.length - 1) : baseUrl;
    _dio.options.baseUrl = '$_baseUrl/api/${AppConstants.apiVersion}';
  }

  /// Whether the client has been configured with a base URL.
  bool get isConfigured => _baseUrl != null;

  // --- Auth ---

  Future<LoginResponse> login(String username, String password) async {
    final response = await _dio.post('/auth/login', data: {
      'username': username,
      'password': password,
    });
    return LoginResponse.fromJson(response.data as Map<String, dynamic>);
  }

  Future<TicketResponse> getWsTicket() async {
    final response = await _dio.post('/auth/ws-ticket');
    return TicketResponse.fromJson(response.data as Map<String, dynamic>);
  }

  /// Revoke a refresh token family on the server.
  Future<void> logout(String? refreshToken) async {
    await _dio.post('/auth/logout', data: {
      'refresh_token': refreshToken ?? '',
    });
  }

  /// Change the current user's password. Server revokes all sessions on success.
  Future<void> changePassword(String currentPassword, String newPassword) async {
    await _dio.post('/auth/change-password', data: {
      'current_password': currentPassword,
      'new_password': newPassword,
    });
  }

  /// Get the current caller's identity, accessible rooms, and permissions.
  Future<Identity> getMe() async {
    final response = await _dio.get('/auth/me');
    return Identity.fromJson(response.data as Map<String, dynamic>);
  }

  // --- Devices ---

  Future<DeviceListResponse> getDevices({String? roomId, String? domain}) async {
    final params = <String, dynamic>{};
    if (roomId != null) params['room_id'] = roomId;
    if (domain != null) params['domain'] = domain;

    final response = await _dio.get('/devices', queryParameters: params);
    return DeviceListResponse.fromJson(response.data as Map<String, dynamic>);
  }

  Future<Device> getDevice(String id) async {
    final response = await _dio.get('/devices/$id');
    return Device.fromJson(response.data as Map<String, dynamic>);
  }

  Future<CommandResponse> setDeviceState(
    String id, {
    required String command,
    Map<String, dynamic>? parameters,
  }) async {
    final response = await _dio.put('/devices/$id/state', data: {
      'command': command,
      'parameters': parameters ?? {},
    });
    return CommandResponse.fromJson(response.data as Map<String, dynamic>);
  }

  // --- Scenes ---

  Future<SceneListResponse> getScenes({String? roomId}) async {
    final params = <String, dynamic>{};
    if (roomId != null) params['room_id'] = roomId;

    final response = await _dio.get('/scenes', queryParameters: params);
    return SceneListResponse.fromJson(response.data as Map<String, dynamic>);
  }

  Future<Scene> getScene(String id) async {
    final response = await _dio.get('/scenes/$id');
    return Scene.fromJson(response.data as Map<String, dynamic>);
  }

  Future<Scene> createScene(Map<String, dynamic> data) async {
    final response = await _dio.post('/scenes', data: data);
    return Scene.fromJson(response.data as Map<String, dynamic>);
  }

  Future<Scene> updateScene(String id, Map<String, dynamic> data) async {
    final response = await _dio.patch('/scenes/$id', data: data);
    return Scene.fromJson(response.data as Map<String, dynamic>);
  }

  Future<void> deleteScene(String id) async {
    await _dio.delete('/scenes/$id');
  }

  Future<ActivateResponse> activateScene(String id, {String triggerSource = 'wall_panel'}) async {
    final response = await _dio.post('/scenes/$id/activate', data: {
      'trigger_type': 'manual',
      'trigger_source': triggerSource,
    });
    return ActivateResponse.fromJson(response.data as Map<String, dynamic>);
  }

  Future<SceneExecutionResponse> getSceneExecutions(String id) async {
    final response = await _dio.get('/scenes/$id/executions');
    return SceneExecutionResponse.fromJson(response.data as Map<String, dynamic>);
  }

  // --- Areas ---

  Future<AreaListResponse> getAreas({String? siteId}) async {
    final params = <String, dynamic>{};
    if (siteId != null) params['site_id'] = siteId;
    final response = await _dio.get('/areas', queryParameters: params);
    return AreaListResponse.fromJson(response.data as Map<String, dynamic>);
  }

  /// Create a new area.
  Future<Area> createArea(Map<String, dynamic> data) async {
    final response = await _dio.post('/areas', data: data);
    return Area.fromJson(response.data as Map<String, dynamic>);
  }

  /// Update an area (PATCH semantics).
  Future<Area> updateArea(String id, Map<String, dynamic> data) async {
    final response = await _dio.patch('/areas/$id', data: data);
    return Area.fromJson(response.data as Map<String, dynamic>);
  }

  /// Delete an area. Fails with 409 if rooms still reference it.
  Future<void> deleteArea(String id) async {
    await _dio.delete('/areas/$id');
  }

  // --- Rooms ---

  Future<RoomListResponse> getRooms({String? areaId}) async {
    final params = <String, dynamic>{};
    if (areaId != null) params['area_id'] = areaId;
    final response = await _dio.get('/rooms', queryParameters: params);
    return RoomListResponse.fromJson(response.data as Map<String, dynamic>);
  }

  /// Create a new room.
  Future<Room> createRoom(Map<String, dynamic> data) async {
    final response = await _dio.post('/rooms', data: data);
    return Room.fromJson(response.data as Map<String, dynamic>);
  }

  /// Update a room (PATCH semantics).
  Future<Room> updateRoom(String id, Map<String, dynamic> data) async {
    final response = await _dio.patch('/rooms/$id', data: data);
    return Room.fromJson(response.data as Map<String, dynamic>);
  }

  /// Delete a room.
  Future<void> deleteRoom(String id) async {
    await _dio.delete('/rooms/$id');
  }

  // --- Users ---

  Future<UserListResponse> getUsers() async {
    final response = await _dio.get('/users');
    return UserListResponse.fromJson(response.data as Map<String, dynamic>);
  }

  Future<User> createUser(Map<String, dynamic> data) async {
    final response = await _dio.post('/users', data: data);
    return User.fromJson(response.data as Map<String, dynamic>);
  }

  Future<User> getUser(String id) async {
    final response = await _dio.get('/users/$id');
    return User.fromJson(response.data as Map<String, dynamic>);
  }

  Future<User> updateUser(String id, Map<String, dynamic> data) async {
    final response = await _dio.patch('/users/$id', data: data);
    return User.fromJson(response.data as Map<String, dynamic>);
  }

  Future<void> deleteUser(String id) async {
    await _dio.delete('/users/$id');
  }

  Future<UserSessionListResponse> getUserSessions(String id) async {
    final response = await _dio.get('/users/$id/sessions');
    return UserSessionListResponse.fromJson(response.data as Map<String, dynamic>);
  }

  Future<void> revokeUserSessions(String id) async {
    await _dio.delete('/users/$id/sessions');
  }

  Future<RoomAccessResponse> getUserRooms(String id) async {
    final response = await _dio.get('/users/$id/rooms');
    return RoomAccessResponse.fromJson(response.data as Map<String, dynamic>);
  }

  Future<RoomAccessResponse> setUserRooms(
      String id, List<RoomAccessGrant> rooms) async {
    final response = await _dio.put('/users/$id/rooms', data: {
      'rooms': rooms.map((r) => r.toJson()).toList(),
    });
    return RoomAccessResponse.fromJson(response.data as Map<String, dynamic>);
  }

  // --- Health ---

  Future<bool> healthCheck() async {
    try {
      final response = await _dio.get('/health');
      return response.statusCode == 200;
    } catch (_) {
      return false;
    }
  }

  // --- System Metrics ---

  /// Get comprehensive system metrics for the admin dashboard.
  Future<SystemMetrics> getMetrics() async {
    final response = await _dio.get('/metrics');
    return SystemMetrics.fromJson(response.data as Map<String, dynamic>);
  }

  /// Get passive KNX bus discovery data (group addresses and devices seen).
  Future<DiscoveryData> getDiscovery() async {
    final response = await _dio.get('/discovery');
    return DiscoveryData.fromJson(response.data as Map<String, dynamic>);
  }

  // --- Device Management ---

  /// Create a new device.
  Future<Device> createDevice(Map<String, dynamic> data) async {
    final response = await _dio.post('/devices', data: data);
    return Device.fromJson(response.data as Map<String, dynamic>);
  }

  /// Update a device's properties (name, room, etc.).
  Future<Device> updateDevice(String id, Map<String, dynamic> updates) async {
    final response = await _dio.patch('/devices/$id', data: updates);
    return Device.fromJson(response.data as Map<String, dynamic>);
  }

  /// Delete a device from the registry.
  Future<void> deleteDevice(String id) async {
    await _dio.delete('/devices/$id');
  }

  // --- Site Management ---

  /// Get the site record. Returns null if no site exists (setup needed).
  Future<Site?> getSite() async {
    try {
      final response = await _dio.get('/site');
      return Site.fromJson(response.data as Map<String, dynamic>);
    } on DioException catch (e) {
      if (e.response?.statusCode == 404) return null;
      rethrow;
    }
  }

  /// Create a new site record (used during initial setup).
  Future<Site> createSite(Map<String, dynamic> data) async {
    final response = await _dio.post('/site', data: data);
    return Site.fromJson(response.data as Map<String, dynamic>);
  }

  /// Update the existing site record (PATCH semantics — only send changed fields).
  Future<Site> updateSite(Map<String, dynamic> data) async {
    final response = await _dio.patch('/site', data: data);
    return Site.fromJson(response.data as Map<String, dynamic>);
  }

  // --- Audit Logs ---

  /// Get paginated audit logs with optional filters.
  Future<AuditLogResponse> getAuditLogs({
    String? action,
    String? entityType,
    int limit = 50,
    int offset = 0,
  }) async {
    final params = <String, dynamic>{
      'limit': limit,
      'offset': offset,
    };
    if (action != null) params['action'] = action;
    if (entityType != null) params['entity_type'] = entityType;

    final response = await _dio.get('/audit-logs', queryParameters: params);
    return AuditLogResponse.fromJson(response.data as Map<String, dynamic>);
  }

  // --- System Management ---

  /// Perform a factory reset, clearing selected data categories.
  Future<FactoryResetResponse> factoryReset({
    bool clearDevices = true,
    bool clearScenes = true,
    bool clearLocations = true,
    bool clearDiscovery = false,
    bool clearSite = false,
  }) async {
    final response = await _dio.post('/system/factory-reset', data: {
      'clear_devices': clearDevices,
      'clear_scenes': clearScenes,
      'clear_locations': clearLocations,
      'clear_discovery': clearDiscovery,
      'clear_site': clearSite,
      'confirm': 'FACTORY RESET',
    });
    return FactoryResetResponse.fromJson(response.data as Map<String, dynamic>);
  }

  // --- ETS Import (Commissioning) ---

  /// Parse an ETS project file and return detected devices.
  /// The file is sent as multipart/form-data.
  Future<ETSParseResult> parseETSFile(List<int> fileBytes, String filename) async {
    // Ensure we have a proper Uint8List for web compatibility
    final bytes = fileBytes is Uint8List ? fileBytes : Uint8List.fromList(fileBytes);

    final formData = FormData.fromMap({
      'file': MultipartFile.fromBytes(bytes, filename: filename),
    });

    final response = await _dio.post(
      '/commissioning/ets/parse',
      data: formData,
      options: Options(
        // Don't set contentType - Dio will set it with boundary automatically
        receiveTimeout: const Duration(seconds: 60),
      ),
    );

    return ETSParseResult.fromJson(response.data as Map<String, dynamic>);
  }

  /// Import devices from a parsed ETS file.
  Future<ETSImportResponse> importETSDevices(ETSImportRequest request) async {
    final response = await _dio.post(
      '/commissioning/ets/import',
      data: request.toJson(),
    );

    return ETSImportResponse.fromJson(response.data as Map<String, dynamic>);
  }

  /// Get the WebSocket URL for connecting.
  String getWsUrl(String ticket) {
    if (_baseUrl == null) throw StateError('ApiClient not configured');
    final wsScheme = _baseUrl!.startsWith('https') ? 'wss' : 'ws';
    final host = _baseUrl!.replaceFirst(RegExp(r'^https?://'), '');
    return '$wsScheme://$host/api/${AppConstants.apiVersion}/ws?ticket=$ticket';
  }
}

/// Interceptor that adds auth credentials to all requests (except login/health).
/// Supports dual auth mode: user JWT (Bearer token) or panel token (X-Panel-Token).
class _AuthInterceptor extends Interceptor {
  final TokenStorage _tokenStorage;

  _AuthInterceptor(this._tokenStorage);

  @override
  Future<void> onRequest(RequestOptions options, RequestInterceptorHandler handler) async {
    // Skip auth for login and health endpoints
    if (options.path.contains('/auth/login') || options.path.contains('/health')) {
      handler.next(options);
      return;
    }

    final mode = await _tokenStorage.getAuthMode();
    if (mode == 'panel') {
      final panelToken = await _tokenStorage.getPanelToken();
      if (panelToken != null) {
        options.headers['X-Panel-Token'] = panelToken;
      }
    } else {
      final token = await _tokenStorage.getToken();
      if (token != null) {
        options.headers['Authorization'] = 'Bearer $token';
      }
    }
    handler.next(options);
  }

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) {
    // Clear token on 401 (expired/invalid)
    if (err.response?.statusCode == 401) {
      _tokenStorage.clearToken();
    }
    handler.next(err);
  }
}
