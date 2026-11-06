package docs

import (
	"errors"
	"fmt"
	"hash/maphash"
	"log/slog"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimitMode controls what the per-client limiter does with a request
// over budget.
type RateLimitMode string

const (
	// RateLimitOff disables the limiter.
	RateLimitOff RateLimitMode = "off"
	// RateLimitShadow serves every request but logs the clients that would
	// have been limited, so limits can be set from real traffic before
	// enforcing.
	RateLimitShadow RateLimitMode = "shadow"
	// RateLimitEnforce answers over-budget requests with 429.
	RateLimitEnforce RateLimitMode = "enforce"
)

const (
	limiterShards = 64

	// unidentifiedClient is the shared bucket for requests with no usable
	// client IP.
	unidentifiedClient = "unidentified"
)

// ParseRateLimitMode validates a mode string.
func ParseRateLimitMode(s string) (RateLimitMode, error) {
	switch m := RateLimitMode(strings.ToLower(strings.TrimSpace(s))); m {
	case RateLimitOff, RateLimitShadow, RateLimitEnforce:
		return m, nil
	default:
		return "", fmt.Errorf("invalid rate limit mode %q (want off, shadow, or enforce)", s)
	}
}

// RateLimitConfig configures RateLimit.
type RateLimitConfig struct {
	Mode RateLimitMode
	// RPS is the sustained requests per second allowed per client.
	RPS float64
	// Burst is the number of requests a client may make at once.
	Burst int
	// TrustedIPHeader names a header carrying the client IP set by a trusted
	// edge (e.g. CF-Connecting-IP). Required for enforce mode: behind an edge
	// RemoteAddr is the edge, so every client would share one bucket.
	TrustedIPHeader string
	// MaxClients bounds limiter memory across all shards.
	MaxClients int
	// IdleTTL is how long an unused client bucket is kept.
	IdleTTL time.Duration
}

// Validate rejects configurations that would disable the limiter by
// accident or throttle every client as one.
func (c RateLimitConfig) Validate() error {
	if c.Mode == "" || c.Mode == RateLimitOff {
		return nil
	}
	if math.IsNaN(c.RPS) || math.IsInf(c.RPS, 0) || c.RPS <= 0 {
		return errors.New("rate limit rps must be a finite number greater than 0")
	}
	if c.Burst < 1 {
		return errors.New("rate limit burst must be at least 1")
	}
	if c.Mode == RateLimitEnforce && strings.TrimSpace(c.TrustedIPHeader) == "" {
		return errors.New("enforce mode requires a trusted client IP header; without one every client behind the edge shares a single bucket")
	}
	return nil
}

type clientBucket struct {
	limiter  *rate.Limiter
	lastSeen time.Time
	// limited records whether the client's last request was over budget, so
	// a flood logs once when it starts rather than once per request.
	limited bool
}

type limiterShard struct {
	mu        sync.Mutex
	clients   map[string]*clientBucket
	lastSweep time.Time
}

type rateLimiter struct {
	cfg      RateLimitConfig
	logger   *slog.Logger
	now      func() time.Time
	seed     maphash.Seed
	perShard int
	shards   [limiterShards]limiterShard
}

// RateLimit wraps next with a per-client token bucket. The limiter is
// per-process: with N instances behind a load balancer the effective limit
// is up to N times higher, which is acceptable for abuse protection. Edge
// rules are the place for a precise global limit. An invalid cfg is an
// error rather than a silently disabled limiter.
func RateLimit(next http.Handler, cfg RateLimitConfig, logger *slog.Logger) (http.Handler, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if cfg.Mode == "" || cfg.Mode == RateLimitOff {
		return next, nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	return newRateLimiter(cfg, logger).wrap(next), nil
}

func newRateLimiter(cfg RateLimitConfig, logger *slog.Logger) *rateLimiter {
	if cfg.MaxClients <= 0 {
		cfg.MaxClients = 100_000
	}
	if cfg.IdleTTL <= 0 {
		cfg.IdleTTL = 10 * time.Minute
	}
	rl := &rateLimiter{
		cfg:      cfg,
		logger:   logger,
		now:      time.Now,
		seed:     maphash.MakeSeed(),
		perShard: max(1, cfg.MaxClients/limiterShards),
	}
	for i := range rl.shards {
		rl.shards[i].clients = make(map[string]*clientBucket)
	}
	return rl
}

func (rl *rateLimiter) wrap(next http.Handler) http.Handler {
	// The policy is the same for every client, so it is safe on responses a
	// CDN caches. Per-client counters are only sent on 429, which is never
	// cached, so one client's budget is not served to another.
	policy := fmt.Sprintf("%s;w=1;burst=%d", strconv.FormatFloat(rl.cfg.RPS, 'f', -1, 64), rl.cfg.Burst)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("RateLimit-Policy", policy)

		key, ok := clientKey(r, rl.cfg.TrustedIPHeader)
		if !ok {
			// The trusted edge always sets the header, so a request without a
			// usable value reached the origin some other way. Those share one
			// bucket: bypassing the edge must not bypass the limit.
			key = unidentifiedClient
		}

		allowed, retryAfter, startedLimiting := rl.take(key)
		if startedLimiting {
			rl.logger.Warn("docs search api: client over rate limit", "mode", string(rl.cfg.Mode), "client", key)
		}
		if !allowed && rl.cfg.Mode == RateLimitEnforce {
			h := w.Header()
			h.Set("Access-Control-Allow-Origin", "*")
			h.Set("Retry-After", strconv.Itoa(retryAfter))
			h.Set("RateLimit-Remaining", "0")
			h.Set("RateLimit-Reset", strconv.Itoa(retryAfter))
			writeAPIError(w, http.StatusTooManyRequests, "too_many_requests", "rate limit exceeded; retry after the Retry-After interval, or use https://docs.digitalocean.com/docs-corpus.jsonl for bulk access")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// take spends one token for key. It reports whether the request is within
// budget, the whole seconds until the next token, and whether this request
// is the first over budget since the client was last allowed.
func (rl *rateLimiter) take(key string) (allowed bool, retryAfter int, startedLimiting bool) {
	s := &rl.shards[maphash.String(rl.seed, key)%limiterShards]
	s.mu.Lock()
	defer s.mu.Unlock()

	now := rl.now()
	b := rl.bucketLocked(s, key, now)
	allowed = b.limiter.AllowN(now, 1)
	if !allowed {
		deficit := 1 - b.limiter.TokensAt(now)
		retryAfter = max(1, int(math.Ceil(deficit/rl.cfg.RPS)))
	}
	startedLimiting = !allowed && !b.limited
	b.limited = !allowed
	return allowed, retryAfter, startedLimiting
}

// bucketLocked returns the bucket for key, creating it if needed. A full
// shard drops idle buckets (at most once per IdleTTL/10, so a flood of new
// clients does not rescan on every request) and otherwise evicts one
// arbitrary bucket; both costs are bounded by the shard size, not the total.
func (rl *rateLimiter) bucketLocked(s *limiterShard, key string, now time.Time) *clientBucket {
	if b, ok := s.clients[key]; ok {
		b.lastSeen = now
		return b
	}
	if len(s.clients) >= rl.perShard {
		if now.Sub(s.lastSweep) >= rl.cfg.IdleTTL/10 {
			s.lastSweep = now
			for k, b := range s.clients {
				if now.Sub(b.lastSeen) > rl.cfg.IdleTTL {
					delete(s.clients, k)
				}
			}
		}
		if len(s.clients) >= rl.perShard {
			for k := range s.clients {
				delete(s.clients, k)
				break
			}
		}
	}
	b := &clientBucket{limiter: rate.NewLimiter(rate.Limit(rl.cfg.RPS), rl.cfg.Burst), lastSeen: now}
	s.clients[key] = b
	return b
}

func (rl *rateLimiter) size() int {
	n := 0
	for i := range rl.shards {
		s := &rl.shards[i]
		s.mu.Lock()
		n += len(s.clients)
		s.mu.Unlock()
	}
	return n
}

// clientKey identifies a client by IP, reporting false when no usable IP is
// available. With a trusted header, the rightmost entry of a list is used:
// it is the one the trusted edge appended. IPv6 addresses are grouped by /64
// because a single host commonly controls a whole /64.
func clientKey(r *http.Request, trustedHeader string) (string, bool) {
	var raw string
	if trustedHeader != "" {
		raw = r.Header.Get(trustedHeader)
		if i := strings.LastIndexByte(raw, ','); i >= 0 {
			raw = raw[i+1:]
		}
		raw = strings.TrimSpace(raw)
	} else {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		raw = host
	}
	ip := net.ParseIP(raw)
	if ip == nil {
		return "", false
	}
	if v4 := ip.To4(); v4 != nil {
		return v4.String(), true
	}
	return ip.Mask(net.CIDRMask(64, 128)).String() + "/64", true
}
