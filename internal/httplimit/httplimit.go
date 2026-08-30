package httplimit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chiririll/CheckScanProviders/internal/nativelog"
	"github.com/chiririll/CheckScanProviders/pkg/eq"
)

const (
	ExtensionKey    = "checkscan.rate_limited"
	maxCooldown     = time.Hour
	defaultCooldown = 10 * time.Minute
)

var (
	ErrLimited   = errors.New("rate_limited")
	ErrThrottled = errors.New("throttled")
)

type Window struct {
	Limit  int
	Period time.Duration
}

type Policy struct {
	Windows         []Window
	DefaultCooldown time.Duration
}

type gate struct {
	mu      sync.Mutex
	until   time.Time
	backoff time.Duration
	hits    []time.Time
}

var (
	gates           sync.Map
	policyOverrides sync.Map
)

func OverridePolicy(host string, p Policy) {
	policyOverrides.Store(normalize(host), p)
}

func PolicyFor(host string) Policy {
	host = normalize(host)
	if v, ok := policyOverrides.Load(host); ok {
		return v.(Policy)
	}
	switch host {
	case "proverkacheka.com":
		return Policy{
			Windows: []Window{
				{Limit: 5, Period: time.Minute},
				{Limit: 10, Period: 10 * time.Minute},
				{Limit: 15, Period: time.Hour},
			},
			DefaultCooldown: 15 * time.Minute,
		}
	default:
		return Policy{
			Windows: []Window{
				{Limit: 5, Period: time.Minute},
				{Limit: 15, Period: 10 * time.Minute},
				{Limit: 30, Period: time.Hour},
			},
			DefaultCooldown: defaultCooldown,
		}
	}
}

func Allow(host string) bool {
	g := get(host)
	g.mu.Lock()
	defer g.mu.Unlock()
	return !time.Now().Before(g.until)
}

func Acquire(ctx context.Context, host string, wait bool) error {
	if ctx == nil {
		ctx = context.Background()
	}
	policy := PolicyFor(host)
	for {
		g := get(host)
		g.mu.Lock()
		now := time.Now()
		if now.Before(g.until) {
			remain := g.until.Sub(now)
			g.mu.Unlock()
			nativelog.Warn("%s httplimit limited host=%s remain=%s", nativelog.Call(ctx), host, remain.Round(time.Second))
			return ErrLimited
		}
		g.prune(now, policy)
		if delay := windowDelay(g.hits, now, policy.Windows); delay > 0 {
			g.mu.Unlock()
			if !wait {
				nativelog.Info("%s httplimit throttle host=%s delay=%s wait=false", nativelog.Call(ctx), host, delay.Round(time.Millisecond))
				return ErrThrottled
			}
			nativelog.Info("%s httplimit wait host=%s delay=%s", nativelog.Call(ctx), host, delay.Round(time.Millisecond))
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				nativelog.Warn("%s httplimit wait canceled host=%s: %v", nativelog.Call(ctx), host, ctx.Err())
				return ctx.Err()
			case <-timer.C:
			}
			continue
		}
		hits := len(g.hits) + 1
		g.hits = append(g.hits, now)
		g.mu.Unlock()
		nativelog.Info("%s httplimit acquire host=%s hits=%d", nativelog.Call(ctx), host, hits)
		save()
		return nil
	}
}

func Note(host string, status int, header http.Header) {
	if !limitedStatus(status) {
		return
	}
	wait := retryAfter(header)
	policy := PolicyFor(host)
	g := get(host)
	g.mu.Lock()
	if wait <= 0 {
		if g.backoff == 0 {
			g.backoff = policy.DefaultCooldown
			if g.backoff <= 0 {
				g.backoff = defaultCooldown
			}
		} else {
			g.backoff *= 2
			if g.backoff > maxCooldown {
				g.backoff = maxCooldown
			}
		}
		wait = g.backoff
	}
	until := time.Now().Add(wait)
	if until.After(g.until) {
		g.until = until
	}
	g.mu.Unlock()
	nativelog.Warn("httplimit cooldown host=%s status=%d wait=%s until=%s",
		host, status, wait.Round(time.Second), until.Format(time.RFC3339))
	save()
}

func NoteError(host string, err error) {
	if err == nil || errors.Is(err, ErrLimited) || errors.Is(err, ErrThrottled) {
		return
	}
	var status int
	if _, scanErr := fmt.Sscanf(err.Error(), "http_%d", &status); scanErr == nil {
		Note(host, status, nil)
	}
}

func IsLimit(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrLimited) {
		return true
	}
	var status int
	if _, scanErr := fmt.Sscanf(err.Error(), "http_%d", &status); scanErr == nil {
		return limitedStatus(status)
	}
	return false
}

func Mark(receipt *eq.Receipt) {
	if receipt == nil {
		return
	}
	if receipt.Extensions == nil {
		receipt.Extensions = map[string]any{}
	}
	receipt.Extensions[ExtensionKey] = true
}

func Reset(host string) {
	gates.Delete(normalize(host))
	save()
}

func ResetAll() {
	gates.Range(func(key, _ any) bool {
		gates.Delete(key)
		return true
	})
	policyOverrides.Range(func(key, _ any) bool {
		policyOverrides.Delete(key)
		return true
	})
	clearPersist()
}

func get(host string) *gate {
	load()
	host = normalize(host)
	if actual, ok := gates.Load(host); ok {
		return actual.(*gate)
	}
	fresh := &gate{}
	actual, _ := gates.LoadOrStore(host, fresh)
	return actual.(*gate)
}

func (g *gate) prune(now time.Time, policy Policy) {
	cut := now.Add(-longestPeriod(policy.Windows))
	i := 0
	for i < len(g.hits) && g.hits[i].Before(cut) {
		i++
	}
	if i > 0 {
		g.hits = append([]time.Time(nil), g.hits[i:]...)
	}
}

func windowDelay(hits []time.Time, now time.Time, windows []Window) time.Duration {
	var wait time.Duration
	for _, w := range windows {
		if w.Limit <= 0 || w.Period <= 0 {
			continue
		}
		from := now.Add(-w.Period)
		n := 0
		var oldest time.Time
		for _, hit := range hits {
			if hit.Before(from) {
				continue
			}
			n++
			if oldest.IsZero() || hit.Before(oldest) {
				oldest = hit
			}
		}
		if n >= w.Limit {
			delay := oldest.Add(w.Period).Sub(now)
			if delay > wait {
				wait = delay
			}
		}
	}
	return wait
}

func longestPeriod(windows []Window) time.Duration {
	var max time.Duration
	for _, w := range windows {
		if w.Period > max {
			max = w.Period
		}
	}
	return max
}

func normalize(host string) string {
	return strings.ToLower(strings.TrimSpace(host))
}

func limitedStatus(status int) bool {
	return status == http.StatusTooManyRequests ||
		status == http.StatusServiceUnavailable ||
		status == http.StatusForbidden ||
		status == http.StatusBadGateway
}

func retryAfter(header http.Header) time.Duration {
	if header == nil {
		return 0
	}
	raw := strings.TrimSpace(header.Get("Retry-After"))
	if raw == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(raw); err == nil {
		wait := time.Until(when)
		if wait > 0 {
			return wait
		}
	}
	return 0
}
