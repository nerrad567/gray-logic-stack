import 'package:flutter/foundation.dart' show kIsWeb;
import 'package:shared_preferences/shared_preferences.dart';

import '../config/constants.dart';

/// Persists auth tokens and panel configuration.
/// Uses localStorage on web, SharedPreferences on native.
class TokenStorage {
  SharedPreferences? _prefs;

  Future<SharedPreferences> get _instance async {
    _prefs ??= await SharedPreferences.getInstance();
    return _prefs!;
  }

  // --- Auth Token ---

  Future<String?> getToken() async {
    final prefs = await _instance;
    return prefs.getString(AppConstants.tokenStorageKey);
  }

  Future<void> setToken(String token) async {
    final prefs = await _instance;
    await prefs.setString(AppConstants.tokenStorageKey, token);
  }

  Future<void> clearToken() async {
    final prefs = await _instance;
    await prefs.remove(AppConstants.tokenStorageKey);
  }

  // --- Core URL ---

  Future<String?> getCoreUrl() async {
    final prefs = await _instance;
    final stored = prefs.getString(AppConstants.coreUrlStorageKey);

    // If no stored URL and running as embedded web panel, use current host
    if (stored == null && kIsWeb) {
      return Uri.base.origin;
    }

    return stored;
  }

  Future<void> setCoreUrl(String url) async {
    final prefs = await _instance;
    await prefs.setString(AppConstants.coreUrlStorageKey, url);
  }

  // --- Room ID ---

  Future<String?> getRoomId() async {
    final prefs = await _instance;
    return prefs.getString(AppConstants.roomIdStorageKey);
  }

  Future<void> setRoomId(String roomId) async {
    final prefs = await _instance;
    await prefs.setString(AppConstants.roomIdStorageKey, roomId);
  }

  // --- Check if configured ---

  Future<bool> isConfigured() async {
    // On web, always configured (auto-detects host)
    if (kIsWeb) return true;

    final url = await getCoreUrl();
    return url != null && url.isNotEmpty;
  }

  // --- Clear all ---

  Future<void> clearAll() async {
    final prefs = await _instance;
    await prefs.remove(AppConstants.tokenStorageKey);
    await prefs.remove(AppConstants.coreUrlStorageKey);
    await prefs.remove(AppConstants.roomIdStorageKey);
  }
}
