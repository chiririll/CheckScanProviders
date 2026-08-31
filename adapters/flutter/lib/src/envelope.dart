class NativeEnvelope<T> {
  const NativeEnvelope({required this.status, this.message = '', this.data});

  final int status;
  final String message;
  final T? data;
}

class NativeMatch {
  const NativeMatch({required this.adapterId, required this.hash, this.label = ''});

  final String adapterId;
  final String hash;
  final String label;
}

class NativeResolve {
  const NativeResolve({
    required this.adapterId,
    required this.hash,
    required this.receipt,
    this.label = '',
  });

  final String adapterId;
  final String hash;
  final String label;
  final Map<String, dynamic> receipt;
}

class SettingField {
  const SettingField({required this.key, required this.type, required this.label});

  final String key;
  final String type;
  final String label;
}
