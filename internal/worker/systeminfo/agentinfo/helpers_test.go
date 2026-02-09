package agentinfo

import "context"

// fakeVersion returns a versionFunc that responds with the given output for
// the expected binary, and ("", false) for anything else.
func fakeVersion(binary, output string) versionFunc {
	return func(_ context.Context, b string) (string, bool) {
		if b == binary {
			return output, true
		}
		return "", false
	}
}

// notFound is a versionFunc that always reports the binary as missing.
func notFound(_ context.Context, _ string) (string, bool) {
	return "", false
}
