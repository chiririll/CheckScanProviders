import 'dart:convert';
import 'dart:ffi';
import 'dart:io';
import 'dart:isolate';

import 'package:ffi/ffi.dart';

import 'decode.dart';
import 'envelope.dart';
import 'native_log.dart';

typedef _CStrFn = Pointer<Utf8> Function();
typedef _CStr2Fn = Pointer<Utf8> Function(Pointer<Utf8>, Pointer<Utf8>);
typedef _CStr4Fn = Pointer<Utf8> Function(Pointer<Utf8>, Pointer<Utf8>, Pointer<Utf8>, Pointer<Utf8>);
typedef _CSetConfigFn = Void Function(Pointer<Utf8>);
typedef _CFreeFn = Void Function(Pointer<Utf8>);

class NativeProvidersLib {
  NativeProvidersLib._(DynamicLibrary lib)
      : _match = lib.lookupFunction<_CStr2Fn, _CStr2Fn>('checkscan_match'),
        _resolve = lib.lookupFunction<_CStr4Fn, _CStr4Fn>('checkscan_resolve'),
        _settings = lib.lookupFunction<_CStrFn, _CStrFn>('checkscan_settings'),
        _setConfig = lib.lookupFunction<_CSetConfigFn, void Function(Pointer<Utf8>)>('checkscan_set_config'),
        _free = lib.lookupFunction<_CFreeFn, void Function(Pointer<Utf8>)>('checkscan_free');
  final _CStr2Fn _match;
  final _CStr4Fn _resolve;
  final _CStrFn _settings;
  final void Function(Pointer<Utf8>) _setConfig;
  final void Function(Pointer<Utf8>) _free;

  static NativeProvidersLib open() {
    if (!Platform.isAndroid) {
      throw UnsupportedError('CheckScanProviders native library is Android-only in this build');
    }
    return NativeProvidersLib._(DynamicLibrary.open('libcheckscan.so'));
  }

  void setConfig(Map<String, String> config) {
    final raw = jsonEncode(config);
    final ptr = raw.toNativeUtf8();
    try {
      _setConfig(ptr);
    } finally {
      malloc.free(ptr);
    }
  }

  String match(String rawQr, {String hint = ''}) => _call2(_match, rawQr, hint);

  String resolve(String rawQr, {String hint = '', bool remote = false, bool wait = false, String current = ''}) {
    final mode = wait ? 'wait' : (remote ? 'remote' : '');
    return _call4(_resolve, rawQr, hint, mode, current);
  }

  String settings() {
    final ptr = _settings();
    if (ptr == nullptr) {
      throw StateError('checkscan_settings returned null');
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
  IsolatedNativeProviders({Map<String, String>? config}) : _config = Map<String, String>.from(config ?? const {});

  Map<String, String> _config;

  void configure(Map<String, String> snapshot) {
    _config = Map<String, String>.from(snapshot);
  }

  Future<NativeEnvelope<NativeMatch>> match(String rawQr, {String hint = ''}) async {
    final config = Map<String, String>.from(_config);
    final raw = await Isolate.run(() => _withLog(config, (lib) => lib.match(rawQr, hint: hint)));
    return decodeEnvelope(raw, decodeMatch);
  }

  Future<NativeEnvelope<NativeResolve>> resolve(
    String rawQr, {
    String hint = '',
    bool remote = false,
    bool wait = false,
    String current = '',
  }) async {
    final config = Map<String, String>.from(_config);
    final raw = await Isolate.run(
      () => _withLog(config, (lib) => lib.resolve(rawQr, hint: hint, remote: remote, wait: wait, current: current)),
    );
    return decodeEnvelope(raw, decodeResolve);
  }

  Future<NativeEnvelope<List<SettingField>>> settings() async {
    final config = Map<String, String>.from(_config);
    final raw = await Isolate.run(() => _withLog(config, (lib) => lib.settings()));
    return decodeEnvelope(raw, decodeSettings);
  }

  static T _withLog<T>(Map<String, String> config, T Function(NativeProvidersLib lib) body) {
    final lib = NativeProvidersLib.open();
    lib.setConfig(config);
    final log = NativeHostLog.attach();
    try {
      return body(lib);
    } finally {
      log?.close();
    }
  }
}
