//go:build !windows && !darwin

package auth

// Linux retains the documented owner-only file backend. There is no automatic
// fallback from macOS Keychain to this backend.
func seal(b []byte) ([]byte, error)   { return b, nil }
func unseal(b []byte) ([]byte, error) { return b, nil }
