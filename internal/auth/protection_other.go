//go:build !darwin

package auth

func protectSession(dir, binding string, b []byte) ([]byte, error)   { return seal(b) }
func unprotectSession(dir, binding string, b []byte) ([]byte, error) { return unseal(b) }
func forgetProtection(dir, binding string) error                     { return nil }
