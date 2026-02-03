import 'dart:convert';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_riverpod/legacy.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../config/constants.dart';
import '../models/area.dart';
import '../models/room.dart';
import '../repositories/location_repository.dart';
import 'auth_provider.dart';

/// Provides the LocationRepository.
final locationRepositoryProvider = Provider<LocationRepository>((ref) {
  return LocationRepository(apiClient: ref.watch(apiClientProvider));
});

/// Holds the loaded location data (areas + rooms).
class LocationData {
  final List<Area> areas;
  final List<Room> rooms;

  const LocationData({required this.areas, required this.rooms});

  /// Get rooms grouped by area, sorted by area.sortOrder then room.sortOrder.
  Map<Area, List<Room>> get roomsByArea {
    final sortedAreas = List<Area>.from(areas)
      ..sort((a, b) => a.sortOrder.compareTo(b.sortOrder));
    final map = <Area, List<Room>>{};
    for (final area in sortedAreas) {
      final areaRooms = rooms.where((r) => r.areaId == area.id).toList()
        ..sort((a, b) => a.sortOrder.compareTo(b.sortOrder));
      if (areaRooms.isNotEmpty) {
        map[area] = areaRooms;
      }
    }
    return map;
  }

  /// Flat list of all rooms sorted by area then room sort_order.
  List<Room> get sortedRooms {
    final result = <Room>[];
    for (final entry in roomsByArea.entries) {
      result.addAll(entry.value);
    }
    return result;
  }

  /// Serialize to JSON for offline caching.
  String toJsonString() {
    return jsonEncode({
      'areas': areas.map((a) => a.toJson()).toList(),
      'rooms': rooms.map((r) => r.toJson()).toList(),
    });
  }

  /// Deserialize from cached JSON.
  static LocationData? fromJsonString(String? jsonStr) {
    if (jsonStr == null || jsonStr.isEmpty) return null;
    try {
      final map = jsonDecode(jsonStr) as Map<String, dynamic>;
      final areas = (map['areas'] as List<dynamic>)
          .map((a) => Area.fromJson(a as Map<String, dynamic>))
          .toList();
      final rooms = (map['rooms'] as List<dynamic>)
          .map((r) => Room.fromJson(r as Map<String, dynamic>))
          .toList();
      return LocationData(areas: areas, rooms: rooms);
    } catch (_) {
      return null;
    }
  }
}

/// Fetches and caches area+room data. Loaded once on app start.
final locationDataProvider =
    NotifierProvider<LocationDataNotifier, AsyncValue<LocationData>>(
  LocationDataNotifier.new,
);

class LocationDataNotifier extends Notifier<AsyncValue<LocationData>> {
  @override
  AsyncValue<LocationData> build() => const AsyncValue.loading();

  LocationRepository get _repo => ref.read(locationRepositoryProvider);

  /// Load areas and rooms from the API, falling back to cache on failure.
  Future<void> load() async {
    state = const AsyncValue.loading();
    try {
      final areas = await _repo.getAreas();
      final rooms = await _repo.getRooms();
      final data = LocationData(areas: areas, rooms: rooms);
      state = AsyncValue.data(data);
      // Cache for offline use
      await _cacheLocally(data);
    } catch (e, st) {
      // Try loading from cache
      final cached = await _loadFromCache();
      if (cached != null) {
        state = AsyncValue.data(cached);
      } else {
        state = AsyncValue.error(e, st);
      }
    }
  }

  Future<void> _cacheLocally(LocationData data) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(AppConstants.locationCacheKey, data.toJsonString());
    } catch (_) {
      // Non-critical: caching failure shouldn't break the app
    }
  }

  Future<LocationData?> _loadFromCache() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final jsonStr = prefs.getString(AppConstants.locationCacheKey);
      return LocationData.fromJsonString(jsonStr);
    } catch (_) {
      return null;
    }
  }
}

/// The currently selected room ID.
final selectedRoomProvider = StateProvider<String?>((ref) => null);
