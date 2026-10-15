package auth

import "github.com/zalando/go-keyring"

// Unit tests must never access or modify a developer's actual Keychain.
func init() { keyring.MockInit() }
