#include <nss.h>
#include <netdb.h>
#include <syslog.h>    // ← must include this for LOG_INFO, LOG_USER, openlog, syslog
#include <stddef.h>    // for size_t

// Simple wrapper that takes a single C string
void go_syslog_fmt(int logtype, const char *msg) {
    syslog(logtype, "%s", msg);
}

// Forward the symbol to Go’s implementation:
extern enum nss_status go_gethostbyname_r(const char *name, int af,
                                                 struct hostent *result_buf,
                                                 char *buf, size_t buflen,
                                                 int *errnop, int *h_errnop);

// This is the actual symbol glibc will look for:
enum nss_status _nss_dnd_gethostbyname_r (
    const char *name,
    struct hostent *result_buf,
    char *buf, size_t buflen,
    int *errnop, int *h_errnop)
{
    return go_gethostbyname_r(name, AF_INET, result_buf, buf, buflen, errnop, h_errnop);
}


enum nss_status _nss_dnd_gethostbyname2_r (
    const char *name, int af,
    struct hostent *result_buf, char *buf,
    size_t buflen, int *errnop, int *h_errnop
) {
    return go_gethostbyname_r(name, af, result_buf, buf, buflen, errnop, h_errnop);
}
