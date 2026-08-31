class NativeStatus {
  static const ok = 200;
  static const incomplete = 206;
  static const parseError = 400;
  static const needsSecret = 401;
  static const unknownFormat = 415;
  static const rateLimited = 429;
  static const unavailable = 503;

  static int clazz(int status) => status < 100 ? 0 : status ~/ 100;

  static bool isSuccess(int status) => clazz(status) == 2;
  static bool isClient(int status) => clazz(status) == 4;
  static bool isRemote(int status) => clazz(status) == 5;

  static bool canPersist(int status, bool hasReceipt) => hasReceipt && status != parseError && status != unknownFormat;

  static bool canRetry(int status) {
    if (status == rateLimited || status == needsSecret || status == ok) return false;
    return status == incomplete || isRemote(status);
  }
}
