import 'dart:convert';

import 'envelope.dart';
import 'status.dart';

NativeEnvelope<T> decodeEnvelope<T>(String raw, T? Function(Object? data) map) {
  final value = jsonDecode(raw);
  if (value is! Map) {
    return NativeEnvelope(status: NativeStatus.parseError, message: 'invalid_native_json');
  }
  final status = _asInt(value['status']) ?? NativeStatus.parseError;
  final message = '${value['message'] ?? ''}';
  return NativeEnvelope(status: status, message: message, data: map(value['data']));
}

NativeMatch? decodeMatch(Object? data) {
  if (data is! Map) return null;
  final adapterId = '${data['adapter_id'] ?? ''}';
  final hash = '${data['hash'] ?? ''}';
  if (adapterId.isEmpty || hash.isEmpty) return null;
  return NativeMatch(adapterId: adapterId, hash: hash, label: '${data['label'] ?? ''}');
}

NativeResolve? decodeResolve(Object? data) {
  if (data is! Map) return null;
  final adapterId = '${data['adapter_id'] ?? ''}';
  final hash = '${data['hash'] ?? ''}';
  final receiptRaw = data['receipt'];
  if (adapterId.isEmpty || hash.isEmpty || receiptRaw is! Map) return null;
  return NativeResolve(
    adapterId: adapterId,
    hash: hash,
    label: '${data['label'] ?? ''}',
    receipt: Map<String, dynamic>.from(receiptRaw),
  );
}

List<SettingField> decodeSettings(Object? data) {
  if (data is! Map) return const [];
  final fields = data['fields'];
  if (fields is! List) return const [];
  return [
    for (final item in fields)
      if (item is Map && '${item['key'] ?? ''}'.isNotEmpty)
        SettingField(
          key: '${item['key']}',
          type: '${item['type'] ?? 'secret'}',
          label: '${item['label'] ?? ''}',
        ),
  ];
}

int? _asInt(Object? value) {
  if (value is int) return value;
  if (value is num) return value.toInt();
  return int.tryParse('$value');
}
