import 'dart:ffi';
import 'dart:io';

import 'package:ffi/ffi.dart';

typedef _CLogNative = Void Function(Int32, Pointer<Utf8>);
typedef _CSetLogNative = Void Function(Pointer<NativeFunction<_CLogNative>>);
typedef _CFreeFn = Void Function(Pointer<Utf8>);

/// Host adapter for one isolate: library calls [checkscan_set_log], we [print].
final class NativeHostLog {
  NativeHostLog._(this._setLog, this._callable, this._free);

  final void Function(Pointer<NativeFunction<_CLogNative>>) _setLog;
  final NativeCallable<_CLogNative> _callable;
  final void Function(Pointer<Utf8>) _free;

  static NativeHostLog? attach() {
    if (!Platform.isAndroid) return null;
    final lib = DynamicLibrary.open('libcheckscan.so');
    final setLog = lib.lookupFunction<_CSetLogNative, void Function(Pointer<NativeFunction<_CLogNative>>)>(
      'checkscan_set_log',
    );
    final free = lib.lookupFunction<_CFreeFn, void Function(Pointer<Utf8>)>('checkscan_free');
    late final NativeHostLog host;
    void onLog(int _, Pointer<Utf8> message) {
      try {
        print('[checkscan] ${message.toDartString()}');
      } finally {
        host._free(message);
      }
    }

    final callable = NativeCallable<_CLogNative>.isolateLocal(onLog);
    host = NativeHostLog._(setLog, callable, free);
    setLog(callable.nativeFunction);
    return host;
  }

  void close() {
    _setLog(nullptr);
    _callable.close();
  }
}
