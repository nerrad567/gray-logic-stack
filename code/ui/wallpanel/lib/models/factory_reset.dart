/// Response from the factory reset API endpoint.
class FactoryResetResponse {
  final String status;
  final Map<String, int> deleted;

  const FactoryResetResponse({
    required this.status,
    required this.deleted,
  });

  factory FactoryResetResponse.fromJson(Map<String, dynamic> json) {
    final deletedRaw = json['deleted'] as Map<String, dynamic>? ?? {};
    return FactoryResetResponse(
      status: json['status'] as String? ?? '',
      deleted: deletedRaw.map((k, v) => MapEntry(k, (v as num).toInt())),
    );
  }

  /// Total number of rows deleted across all tables.
  int get totalDeleted => deleted.values.fold(0, (a, b) => a + b);
}
