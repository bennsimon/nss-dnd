package main

/*
#cgo CFLAGS: -I/usr/include
#cgo LDFLAGS: -shared
#include <nss.h>
#include <netdb.h>
#include <syslog.h>
#include <stdlib.h>
#include <errno.h>

void go_syslog_fmt(int logtype, const char *arg);
*/
import "C"

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"
	"unsafe"
)

type Rule struct {
	Type    string            `yaml:"type"`
	Pattern string            `yaml:"pattern"`
	Options map[string]string `yaml:"options,omitempty"`
}

// Config represents the top-level YAML structure
// with a list of rules.
type Config struct {
	Rules []Rule `yaml:"rules"`
}

const (
	NSS_DND_CONFIG_FILE_PATH = "NSS_DND_CONFIG_FILE_PATH"
	NSS_DND_STATIC           = "static"
	NSS_DND_API              = "api"
	NSS_DND_CNAME            = "cname"
	NSS_DND_TARGET           = "target"
	NSS_DND_ALIAS_TO         = "alias_to"
	NSS_DND_ENDPOINT         = "endpoint"
)

var (
	configPath  = "/etc/nss_dnd_rules.yaml"
	lastModTime time.Time
	once        sync.Once
	rules       []Rule
)

// initLogger initializes syslog tagging
func initLogger() {
	name := C.CString("nss_dnd")
	defer C.free(unsafe.Pointer(name))
	C.openlog(name, C.LOG_PID|C.LOG_CONS, C.LOG_USER)
}

// logMessage sends a timestamped message to syslog
func logMessage(logType C.int, msg string) {
	cMsg := C.CString(msg)
	defer C.free(unsafe.Pointer(cMsg))
	C.go_syslog_fmt(logType, cMsg)
}

// reloadConfig checks file mod time and reloads rules if changed
func reloadConfig() {
	configPathFromEnv, exists := os.LookupEnv(NSS_DND_CONFIG_FILE_PATH)
	if exists {
		configPath = configPathFromEnv
	}

	fi, err := os.Stat(configPath)
	if err != nil {
		logMessage(C.LOG_ERR, fmt.Sprintf("config reload error: %s", err.Error()))
		return
	}
	if fi.ModTime().After(lastModTime) {
		_rules, err := parseConfig(configPath)
		if err != nil {
			logMessage(C.LOG_ERR, fmt.Sprintf("config reload error: %s", err.Error()))
			return
		}
		rules = _rules
		lastModTime = fi.ModTime()
		logMessage(C.LOG_INFO, "config reloaded")
	}
}

// fillHostent populates the hostent struct with the given IP and name
func fillHostent(result_buf *C.struct_hostent, buf *C.char, ip net.IP, name string, errnop *C.int, h_errnop *C.int) {
	base := uintptr(unsafe.Pointer(buf))
	// IP: store raw bytes
	copy((*[4]byte)(unsafe.Pointer(base))[:], ip)

	ipPtr := base

	// Canonical name
	cname := []byte(name)
	cnamePtr := base + 4
	copy((*[256]byte)(unsafe.Pointer(cnamePtr))[:], cname)

	// Aliases (just NULL)
	aliasesPtr := cnamePtr + uintptr(len(cname))
	*(*uintptr)(unsafe.Pointer(aliasesPtr)) = 0

	// Addr list: [ptr to ip][NULL]
	addrListPtr := aliasesPtr + unsafe.Sizeof(uintptr(0))
	*(*uintptr)(unsafe.Pointer(addrListPtr)) = ipPtr
	*(*uintptr)(unsafe.Pointer(addrListPtr + unsafe.Sizeof(uintptr(0)))) = 0

	// Fill result_buf
	result_buf.h_name = (*C.char)(unsafe.Pointer(cnamePtr))
	result_buf.h_aliases = (**C.char)(unsafe.Pointer(aliasesPtr))
	result_buf.h_addrtype = C.AF_INET
	result_buf.h_length = C.int(len(ip))
	result_buf.h_addr_list = (**C.char)(unsafe.Pointer(addrListPtr))

	*errnop = 0
	*h_errnop = 0
}

// parseConfig reads and unmarshals YAML rules
func parseConfig(path string) ([]Rule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return cfg.Rules, nil
}

// matchRule finds the first matching rule and returns an IP, or nil
func matchRule(host string) net.IP {
	for _, rule := range rules {
		if matched, _ := path.Match(rule.Pattern, host); !matched {
			continue
		}

		op := rule.Options

		switch rule.Type {

		case NSS_DND_STATIC:
			if ipStr, ok := op[NSS_DND_TARGET]; ok {
				return net.ParseIP(ipStr).To4()
			}

		case NSS_DND_API:
			if urlTmpl, ok := op[NSS_DND_ENDPOINT]; ok {
				url := strings.Replace(urlTmpl, "{host}", host, -1)
				if resp, err := http.Get(url); err == nil && resp.StatusCode == 200 {
					body, _ := io.ReadAll(resp.Body)
					return net.ParseIP(strings.TrimSpace(string(body))).To4()
				}
			}

		case NSS_DND_CNAME:
			aliasTo := rule.Options[NSS_DND_ALIAS_TO]
			if host != aliasTo {
				return matchRule(aliasTo)
			}
		}
	}
	return nil
}

//export go_gethostbyname_r
func go_gethostbyname_r(
	name *C.char, af C.int,
	result_buf *C.struct_hostent, buf *C.char, buflen C.size_t,
	errnop *C.int, h_errnop *C.int,
) C.enum_nss_status {
	once.Do(func() {
		initLogger()
		fi, err := os.Stat(configPath)
		if err == nil {
			r, err := parseConfig(configPath)
			if err == nil {
				rules = r
				lastModTime = fi.ModTime()
			}
		}
	})
	reloadConfig()
	host := C.GoString(name)

	if ip := matchRule(host); ip != nil {
		fillHostent(result_buf, buf, ip, host, errnop, h_errnop)
		return C.NSS_STATUS_SUCCESS
	}

	*errnop = C.int(C.ENOENT)
	*h_errnop = C.int(C.HOST_NOT_FOUND)
	return C.NSS_STATUS_NOTFOUND
}

func main() {}
