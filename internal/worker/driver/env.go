package driver

import "os"

// BuildEnv merges the current process environment with extra key-value pairs.
// Extra vars override existing entries with the same key.
func BuildEnv(extra map[string]string) []string {
	base := os.Environ()
	if len(extra) == 0 {
		return base
	}

	// Build a set of keys to override so we can skip them in the base.
	overrides := make(map[string]struct{}, len(extra))
	for k := range extra {
		overrides[k] = struct{}{}
	}

	env := make([]string, 0, len(base)+len(extra))
	for _, entry := range base {
		key, _, _ := cutEnv(entry)
		if _, ok := overrides[key]; ok {
			continue // will be replaced by extra
		}
		env = append(env, entry)
	}
	for k, v := range extra {
		env = append(env, k+"="+v)
	}
	return env
}

// cutEnv splits an environment variable entry "KEY=VALUE" into key and value.
func cutEnv(entry string) (key, value string, ok bool) {
	for i := 0; i < len(entry); i++ {
		if entry[i] == '=' {
			return entry[:i], entry[i+1:], true
		}
	}
	return entry, "", false
}
