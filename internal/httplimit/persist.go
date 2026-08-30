package httplimit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/chiririll/CheckScanProviders/internal/nativelog"
)

type persistFile struct {
	Gates map[string]persistGate `json:"gates"`
}

type persistGate struct {
	Until   time.Time     `json:"until"`
	Backoff time.Duration `json:"backoff"`
	Hits    []time.Time   `json:"hits"`
}

var (
	persistMu   sync.Mutex
	persistOn   bool
	persistPath string
	loadedOnce  sync.Once
)

func EnablePersist() {
	dir := os.Getenv("TMPDIR")
	if dir == "" {
		dir = os.TempDir()
	}
	persistMu.Lock()
	persistOn = true
	persistPath = filepath.Join(dir, "checkscan-httplimit.json")
	loadedOnce = sync.Once{}
	persistMu.Unlock()
	nativelog.Info("httplimit persist enabled path=%s", persistPath)
	load()
}

func save() {
	persistMu.Lock()
	defer persistMu.Unlock()
	if !persistOn || persistPath == "" {
		return
	}
	out := persistFile{Gates: map[string]persistGate{}}
	gates.Range(func(key, value any) bool {
		host, _ := key.(string)
		g, _ := value.(*gate)
		if host == "" || g == nil {
			return true
		}
		g.mu.Lock()
		out.Gates[host] = persistGate{Until: g.until, Backoff: g.backoff, Hits: append([]time.Time(nil), g.hits...)}
		g.mu.Unlock()
		return true
	})
	raw, err := json.Marshal(out)
	if err != nil {
		return
	}
	_ = os.WriteFile(persistPath, raw, 0o600)
}

func load() {
	loadedOnce.Do(func() {
		persistMu.Lock()
		path := persistPath
		on := persistOn
		persistMu.Unlock()
		if !on || path == "" {
			return
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			if !os.IsNotExist(err) {
				nativelog.Warn("httplimit persist load %s: %v", path, err)
			}
			return
		}
		var in persistFile
		if json.Unmarshal(raw, &in) != nil {
			nativelog.Warn("httplimit persist load invalid %s", path)
			return
		}
		for host, snap := range in.Gates {
			gates.Store(normalize(host), &gate{until: snap.Until, backoff: snap.Backoff, hits: snap.Hits})
		}
		nativelog.Info("httplimit persist loaded path=%s gates=%d", path, len(in.Gates))
	})
}

func clearPersist() {
	loadedOnce = sync.Once{}
	persistMu.Lock()
	path := persistPath
	persistMu.Unlock()
	if path != "" {
		_ = os.Remove(path)
	}
}
