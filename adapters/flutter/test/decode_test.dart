import 'package:flutter_test/flutter_test.dart';
import 'package:providers_native/src/decode.dart';
import 'package:providers_native/src/status.dart';

void main() {
  test('match envelope', () {
    const raw = '{"status":200,"message":"","data":{"adapter_id":"ru_fns","hash":"h","label":"RU"}}';
    final env = decodeEnvelope(raw, decodeMatch);
    expect(env.status, NativeStatus.ok);
    expect(env.data!.adapterId, 'ru_fns');
  });

  test('unknown format has null data', () {
    const raw = '{"status":415,"message":"unknown_format","data":null}';
    final env = decodeEnvelope(raw, decodeMatch);
    expect(env.status, NativeStatus.unknownFormat);
    expect(env.data, isNull);
  });

  test('settings fields', () {
    const raw = '{"status":200,"message":"","data":{"fields":[{"key":"ru_fns.token","type":"secret","label":"RU"}]}}';
    final env = decodeEnvelope(raw, decodeSettings);
    expect(env.data, hasLength(1));
    expect(env.data!.single.key, 'ru_fns.token');
  });

  test('retry classes', () {
    expect(NativeStatus.canRetry(NativeStatus.incomplete), isTrue);
    expect(NativeStatus.canRetry(NativeStatus.unavailable), isTrue);
    expect(NativeStatus.canRetry(NativeStatus.rateLimited), isFalse);
    expect(NativeStatus.canRetry(NativeStatus.needsSecret), isFalse);
    expect(NativeStatus.canRetry(NativeStatus.ok), isFalse);
  });
}
