package sync

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Debug instrumentation: append every auth failure (invalid one-time token
// swap, invalid permanent token, IP-blocked refusals) to /tmp/auth so we can
// diagnose the "invalid token after server runs a while" report.
var (
	authDbgFile *os.File
	authDbgMu   sync.Mutex
)

func init() {
	f, err := os.OpenFile("/tmp/auth", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("auth debug: cannot open /tmp/auth", "error", err)
		return
	}
	authDbgFile = f
}

func tokenFingerprint(t string) string {
	if t == "" {
		return "<empty>"
	}
	prefix := t
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}
	suffix := ""
	if len(t) > 12 {
		suffix = t[len(t)-4:]
	}
	h := sha256.Sum256([]byte(t))
	return fmt.Sprintf("len=%d prefix=%s suffix=%s sha256_8=%s", len(t), prefix, suffix, hex.EncodeToString(h[:])[:8])
}

func logAuthFailure(reason string, r *http.Request, extras map[string]any) {
	if authDbgFile == nil {
		return
	}

	cookieVal := ""
	cookiePresent := false
	if c, err := r.Cookie(AuthCookieName); err == nil {
		cookieVal = c.Value
		cookiePresent = true
	}
	authHeader := r.Header.Get("Authorization")

	mu.RLock()
	onetimeMapSize := len(oneTimeTokens)
	mu.RUnlock()

	blockedIPsMutex.RLock()
	blockedMapSize := len(blockedIPs)
	blockedUntil, ipBlocked := blockedIPs[getIPFromRemoteAddr(r.RemoteAddr)]
	blockedIPsMutex.RUnlock()

	parts := []string{
		"ts=" + time.Now().Format(time.RFC3339Nano),
		"reason=" + reason,
		"method=" + r.Method,
		"path=" + r.URL.Path,
		"remote=" + r.RemoteAddr,
		"ip=" + getIPFromRemoteAddr(r.RemoteAddr),
		"ua=" + strconv.Quote(r.UserAgent()),
		"x_forwarded_for=" + strconv.Quote(r.Header.Get("X-Forwarded-For")),
		"x_real_ip=" + strconv.Quote(r.Header.Get("X-Real-IP")),
		"cf_connecting_ip=" + strconv.Quote(r.Header.Get("CF-Connecting-IP")),
		"referer=" + strconv.Quote(r.Referer()),
		"cookie_present=" + strconv.FormatBool(cookiePresent),
		"cookie=" + tokenFingerprint(cookieVal),
		"auth_header_present=" + strconv.FormatBool(authHeader != ""),
		"auth_header=" + tokenFingerprint(authHeader),
		"onetime_map_size=" + strconv.Itoa(onetimeMapSize),
		"blocked_map_size=" + strconv.Itoa(blockedMapSize),
		"ip_currently_blocked=" + strconv.FormatBool(ipBlocked && time.Now().Before(blockedUntil)),
	}
	if ipBlocked {
		parts = append(parts,
			"ip_block_until="+blockedUntil.Format(time.RFC3339Nano),
			"ip_block_remaining="+time.Until(blockedUntil).String(),
		)
	}
	for k, v := range extras {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}

	line := strings.Join(parts, " ") + "\n"
	authDbgMu.Lock()
	defer authDbgMu.Unlock()
	authDbgFile.WriteString(line)
}
