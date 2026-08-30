#ifndef CHECKSCAN_H
#define CHECKSCAN_H

#ifdef __cplusplus
extern "C" {
#endif

/* All returned char* are UTF-8 JSON allocated by the library. Free with checkscan_free. */
/* hint may be NULL or empty. */

char* checkscan_match(const char* raw_qr, const char* adapter_hint);
/* mode: empty = local QR/vl only; "remote" = fetch if the host gate is ready; "wait" = fetch and wait for the min interval */
/* current_json may be NULL or empty; when set, the library keeps it unless the new receipt is richer. */
char* checkscan_resolve(const char* raw_qr, const char* adapter_hint, const char* mode, const char* current_json);
char* checkscan_providers(void);
void  checkscan_free(char* p);
/* Host sink. level: 3 debug, 4 info, 5 warn, 6 error. message is library-owned; host must checkscan_free it. */
typedef void (*checkscan_log_fn)(int level, const char* message);
void  checkscan_set_log(checkscan_log_fn fn);

#ifdef __cplusplus
}
#endif

#endif
