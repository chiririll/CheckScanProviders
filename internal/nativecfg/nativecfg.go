package nativecfg

import (
	"encoding/json"
	"strings"
	"sync"
)

var (
	mu   sync.RWMutex
	vals = map[string]string{}
)

func SetJSON(raw string) {
	raw = strings.TrimSpace(raw)
	next := map[string]string{}
	if raw != "" {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
			for key, value := range parsed {
				if s, ok := value.(string); ok {
					s = strings.TrimSpace(s)
					if s != "" {
						next[key] = s
					}
				}
			}
		}
	}
	mu.Lock()
	vals = next
	mu.Unlock()
}

func Get(key string) string {
	mu.RLock()
	defer mu.RUnlock()
	return vals[key]
}

func Reset() {
	mu.Lock()
	vals = map[string]string{}
	mu.Unlock()
}
