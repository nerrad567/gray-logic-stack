/// Represents a single audit trail entry from the system log.
class AuditLog {
  final String id;
  final String action;
  final String entityType;
  final String? entityId;
  final String? userId;
  final String source;
  final Map<String, dynamic>? details;
  final DateTime createdAt;

  const AuditLog({
    required this.id,
    required this.action,
    required this.entityType,
    this.entityId,
    this.userId,
    required this.source,
    this.details,
    required this.createdAt,
  });

  factory AuditLog.fromJson(Map<String, dynamic> json) {
    return AuditLog(
      id: json['id'] as String,
      action: json['action'] as String,
      entityType: json['entity_type'] as String,
      entityId: json['entity_id'] as String?,
      userId: json['user_id'] as String?,
      source: json['source'] as String,
      details: json['details'] as Map<String, dynamic>?,
      createdAt: DateTime.parse(json['created_at'] as String),
    );
  }
}

/// Paginated response from the audit logs endpoint.
class AuditLogResponse {
  final List<AuditLog> logs;
  final int total;
  final int limit;
  final int offset;

  const AuditLogResponse({
    required this.logs,
    required this.total,
    required this.limit,
    required this.offset,
  });

  factory AuditLogResponse.fromJson(Map<String, dynamic> json) {
    return AuditLogResponse(
      logs: (json['logs'] as List<dynamic>)
          .map((e) => AuditLog.fromJson(e as Map<String, dynamic>))
          .toList(),
      total: json['total'] as int,
      limit: json['limit'] as int,
      offset: json['offset'] as int,
    );
  }
}
