#ifndef CHECKSCAN_H
#define CHECKSCAN_H

#ifdef __cplusplus
extern "C" {
#endif

/* All returned char* are UTF-8 JSON allocated by the library. Free with checkscan_free. */
/* hint may be NULL or empty. */

char* checkscan_match(const char* raw_qr, const char* adapter_hint);
/* mode: empty = local QR/vl only; "remote" = fetch if the host gate is ready; "wait" = fetch and wait for the min interval */
char* checkscan_resolve(const char* raw_qr, const char* adapter_hint, const char* mode);
char* checkscan_providers(void);
void  checkscan_free(char* p);

#ifdef __cplusplus
}
#endif

#endif
