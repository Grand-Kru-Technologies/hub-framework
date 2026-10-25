package remoteforward

import "time"

// CoerceInt reads whole numbers from YAML-style config values (int, int64, or float64).
func CoerceInt(raw interface{}) (int, bool) {
	switch v := raw.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

// CoercePositiveSeconds reads a client timeout in seconds from config.
func CoercePositiveSeconds(raw interface{}) (time.Duration, bool) {
	n, ok := CoerceInt(raw)
	if !ok || n <= 0 {
		return 0, false
	}
	return time.Duration(n) * time.Second, true
}
