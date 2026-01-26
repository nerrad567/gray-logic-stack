import 'dart:typed_data';

import 'package:dio/dio.dart';

import '../config/constants.dart';
import '../models/area.dart';
import '../models/auth.dart';
import '../models/device.dart';
import '../models/ets_import.dart';
import '../models/room.dart';
import '../models/scene.dart';
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

  Future<ActivateResponse> activateScene(String id, {String triggerSource = 'wall_panel'}) async {
    final response = await _dio.post('/scenes/$id/activate', data: {
      'trigger_type': 'manual',
      'trigger_source': triggerSource,
    });
    return ActivateResponse.fromJson(response.data as Map<String, dynamic>);
  }

  // --- Areas ---

  Future<AreaListResponse> getAreas({String? siteId}) async {
    final params = <String, dynamic>{};
    if (siteId != null) params['site_id'] = siteId;
    final response = await _dio.get('/areas', queryParameters: params);
    return AreaListResponse.fromJson(response.data as Map<String, dynamic>);
  }

  // --- Rooms ---

  Future<RoomListResponse> getRooms({String? areaId}) async {
    final params = <String, dynamic>{};
    if (areaId != null) params['area_id'] = areaId;
    final response = await _dio.get('/rooms', queryParameters: params);
    return RoomListResponse.fromJson(response.data as Map<String, dynamic>);
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

/// Interceptor that adds the Bearer token to all requests (except login).
class _AuthInterceptor extends Interceptor {
  final TokenStorage _tokenStorage;

  _AuthInterceptor(this._tokenStorage);

  @override
  Future<void> onRequest(RequestOptions options, RequestInterceptorHandler handler) async {
    // Skip auth for login endpoint
    if (options.path.contains('/auth/login')) {
      handler.next(options);
      return;
    }

    final token = await _tokenStorage.getToken();
    if (token != null) {
      options.headers['Authorization'] = 'Bearer $token';
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
