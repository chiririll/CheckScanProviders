import 'dart:ffi';
import 'dart:io';
import 'dart:isolate';

import 'package:ffi/ffi.dart';

import 'native_log.dart';

typedef _CStrFn = Pointer<Utf8> Function();
typedef _CStr2Fn = Pointer<Utf8> Function(Pointer<Utf8>, Pointer<Utf8>);
typedef _CStr4Fn = Pointer<Utf8> Function(Pointer<Utf8>, Pointer<Utf8>, Pointer<Utf8>, Pointer<Utf8>);
typedef _CFreeFn = Void Function(Pointer<Utf8>);

class NativeProvidersLib {
  NativeProvidersLib._(DynamicLibrary lib)
      : _match = lib.lookupFunction<_CStr2Fn, _CStr2Fn>('checkscan_match'),
        _resolve = lib.lookupFunction<_CStr4Fn, _CStr4Fn>('checkscan_resolve'),
        _providers = lib.lookupFunction<_CStrFn, _CStrFn>('checkscan_providers'),
        _free = lib.lookupFunction<_CFreeFn, void Function(Pointer<Utf8>)>('checkscan_free');
  final _CStr2Fn _match;
  final _CStr4Fn _resolve;
  final _CStrFn _providers;
  final void Function(Pointer<Utf8>) _free;

  static NativeProvidersLib open() {
    if (!Platform.isAndroid) {
      throw UnsupportedError('CheckScanProviders native library is Android-only in this build');
    }
    return NativeProvidersLib._(DynamicLibrary.open('libcheckscan.so'));
  }

  String match(String rawQr, {String hint = ''}) => _call2(_match, rawQr, hint);

  String resolve(String rawQr, {String hint = '', bool remote = false, bool wait = false, String current = ''}) {
    final mode = wait ? 'wait' : (remote ? 'remote' : '');
    return _call4(_resolve, rawQr, hint, mode, current);
  }

  String providers() {
    final ptr = _providers();
    if (ptr == nullptr) {
      throw StateError('checkscan_providers returned null');
    }
    try {
      return ptr.toDartString();
    } finally {
      _free(ptr);
    }
  }

  String _call2(_CStr2Fn fn, String rawQr, String hint) {
    final rawPtr = rawQr.toNativeUtf8();
    final hintPtr = hint.toNativeUtf8();
    try {
      return _read(fn(rawPtr, hintPtr));
    } finally {
      malloc.free(rawPtr);
      malloc.free(hintPtr);
    }
  }

  String _call4(_CStr4Fn fn, String rawQr, String hint, String mode, String current) {
    final rawPtr = rawQr.toNativeUtf8();
    final hintPtr = hint.toNativeUtf8();
    final modePtr = mode.toNativeUtf8();
    final currentPtr = current.toNativeUtf8();
    try {
      return _read(fn(rawPtr, hintPtr, modePtr, currentPtr));
    } finally {
      malloc.free(rawPtr);
      malloc.free(hintPtr);
      malloc.free(modePtr);
      malloc.free(currentPtr);
    }
  }

  String _read(Pointer<Utf8> result) {
    if (result == nullptr) {
      throw StateError('native call returned null');
    }
    try {
      return result.toDartString();
    } finally {
      _free(result);
    }
  }
}

class IsolatedNativeProviders {
  IsolatedNativeProviders();

  Future<String> match(String rawQr, {String hint = ''}) {
    return Isolate.run(() => _withLog((lib) => lib.match(rawQr, hint: hint)));
  }

  Future<String> resolve(
    String rawQr, {
    String hint = '',
    bool remote = false,
    bool wait = false,
    String current = '',
  }) {
    return Isolate.run(
      () => _withLog((lib) => lib.resolve(rawQr, hint: hint, remote: remote, wait: wait, current: current)),
    );
  }

  Future<String> providers() {
    return Isolate.run(() => _withLog((lib) => lib.providers()));
  }

  static T _withLog<T>(T Function(NativeProvidersLib lib) body) {
    final lib = NativeProvidersLib.open();
    final log = NativeHostLog.attach();
    try {
      return body(lib);
    } finally {
      log?.close();
    }
  }
}
