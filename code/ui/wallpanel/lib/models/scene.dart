/// Scene model matching the Go `automation.Scene` struct.
/// See: code/core/internal/automation/types.go
class Scene {
  final String id;
  final String name;
  final String slug;
  final String? description;
  final String? roomId;
  final String? areaId;
  final bool enabled;
  final int priority;
  final String? icon;
  final String? colour;
  final String? category;
  final List<SceneAction> actions;
  final int sortOrder;
  final DateTime createdAt;
  final DateTime updatedAt;

  const Scene({
    required this.id,
    required this.name,
    required this.slug,
    this.description,
    this.roomId,
    this.areaId,
    this.enabled = true,
    this.priority = 50,
    this.icon,
    this.colour,
    this.category,
    this.actions = const [],
    this.sortOrder = 0,
    required this.createdAt,
    required this.updatedAt,
  });

  factory Scene.fromJson(Map<String, dynamic> json) {
    final actionsList = (json['actions'] as List<dynamic>?) ?? [];
    return Scene(
      id: json['id'] as String,
      name: json['name'] as String,
      slug: json['slug'] as String,
      description: json['description'] as String?,
      roomId: json['room_id'] as String?,
      areaId: json['area_id'] as String?,
      enabled: (json['enabled'] as bool?) ?? true,
      priority: (json['priority'] as num?)?.toInt() ?? 50,
      icon: json['icon'] as String?,
      colour: json['colour'] as String?,
      category: json['category'] as String?,
      actions: actionsList
          .map((a) => SceneAction.fromJson(a as Map<String, dynamic>))
          .toList(),
      sortOrder: (json['sort_order'] as num?)?.toInt() ?? 0,
      createdAt: DateTime.parse(json['created_at'] as String),
      updatedAt: DateTime.parse(json['updated_at'] as String),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'name': name,
      'slug': slug,
      if (description != null) 'description': description,
      if (roomId != null) 'room_id': roomId,
      if (areaId != null) 'area_id': areaId,
      'enabled': enabled,
      'priority': priority,
      if (icon != null) 'icon': icon,
      if (colour != null) 'colour': colour,
      if (category != null) 'category': category,
      'actions': actions.map((a) => a.toJson()).toList(),
      'sort_order': sortOrder,
      'created_at': createdAt.toUtc().toIso8601String(),
      'updated_at': updatedAt.toUtc().toIso8601String(),
    };
  }
}

/// A single device command within a scene.
class SceneAction {
  final String deviceId;
  final String command;
  final Map<String, dynamic>? parameters;
  final int delayMs;
  final int fadeMs;
  final bool parallel;
  final bool continueOnError;
  final int sortOrder;

  const SceneAction({
    required this.deviceId,
    required this.command,
    this.parameters,
    this.delayMs = 0,
    this.fadeMs = 0,
    this.parallel = false,
    this.continueOnError = false,
    this.sortOrder = 0,
  });

  factory SceneAction.fromJson(Map<String, dynamic> json) {
    return SceneAction(
      deviceId: json['device_id'] as String,
      command: json['command'] as String,
      parameters: json['parameters'] as Map<String, dynamic>?,
      delayMs: (json['delay_ms'] as num?)?.toInt() ?? 0,
      fadeMs: (json['fade_ms'] as num?)?.toInt() ?? 0,
      parallel: (json['parallel'] as bool?) ?? false,
      continueOnError: (json['continue_on_error'] as bool?) ?? false,
      sortOrder: (json['sort_order'] as num?)?.toInt() ?? 0,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'device_id': deviceId,
      'command': command,
      if (parameters != null) 'parameters': parameters,
      'delay_ms': delayMs,
      'fade_ms': fadeMs,
      'parallel': parallel,
      'continue_on_error': continueOnError,
      'sort_order': sortOrder,
    };
  }
}

/// Response wrapper for GET /scenes.
class SceneListResponse {
  final List<Scene> scenes;
  final int count;

  const SceneListResponse({required this.scenes, required this.count});

  factory SceneListResponse.fromJson(Map<String, dynamic> json) {
    final list = (json['scenes'] as List<dynamic>?) ?? [];
    return SceneListResponse(
      scenes: list
          .map((s) => Scene.fromJson(s as Map<String, dynamic>))
          .toList(),
      count: (json['count'] as num?)?.toInt() ?? list.length,
    );
  }
}

/// Response for POST /scenes/{id}/activate (202 Accepted).
class ActivateResponse {
  final String executionId;
  final String status;
  final String message;

  const ActivateResponse({
    required this.executionId,
    required this.status,
    required this.message,
  });

  factory ActivateResponse.fromJson(Map<String, dynamic> json) {
    return ActivateResponse(
      executionId: json['execution_id'] as String,
      status: json['status'] as String,
      message: json['message'] as String,
    );
  }
}
