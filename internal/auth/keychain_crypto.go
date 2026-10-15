package auth

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
)

var errKeyNotFound = errors.New("key not found")

const keychainService = "h3yun-cli.oauth.v1"
const sealedPrefix = "H3YUN-KC1\n"

type keyStore interface {
	Get(service, account string) (string, error)
	Set(service, account, value string) error
	Delete(service, account string) error
}

// Only a small random encryption key goes into Keychain, not an arbitrarily
// large OAuth token. Callers serialize Save/Logout with the same session lock.
type keychainCipher struct{ keys keyStore }

func keyAccount(dir, binding string) (string, error) {
	if _, err := sessionPath(dir, binding); err != nil {
		return "", err
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("invalid session directory")
	}
	// Resolve directory aliases so two paths to the same cache use one key.
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	sum := sha256.Sum256([]byte(abs + "\x00" + binding))
	return hex.EncodeToString(sum[:]), nil
}

func (k keychainCipher) aead(account string, create bool) (cipher.AEAD, error) {
	value, err := k.keys.Get(keychainService, account)
	if errors.Is(err, errKeyNotFound) && create {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, fmt.Errorf("cannot generate session key")
		}
		value = base64.StdEncoding.EncodeToString(key)
		if err = k.keys.Set(keychainService, account, value); err != nil {
			return nil, fmt.Errorf("Keychain write failed; no plaintext fallback")
		}
	} else if err != nil {
		return nil, fmt.Errorf("Keychain unavailable or key missing; no plaintext fallback")
	}
	key, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("invalid Keychain key; refusing to replace it")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize session encryption")
	}
	return cipher.NewGCM(block)
}

func (k keychainCipher) seal(dir, binding string, data []byte) ([]byte, error) {
	account, err := keyAccount(dir, binding)
	if err != nil {
		return nil, err
	}
	aead, err := k.aead(account, true)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("cannot generate session nonce")
	}
	out := append([]byte(sealedPrefix), nonce...)
	return aead.Seal(out, nonce, data, []byte(account)), nil
}

func (k keychainCipher) open(dir, binding string, data []byte) ([]byte, error) {
	if !bytes.HasPrefix(data, []byte(sealedPrefix)) {
		return nil, fmt.Errorf("legacy plaintext or invalid session; run auth login to replace it with Keychain protection")
	}
	account, err := keyAccount(dir, binding)
	if err != nil {
		return nil, err
	}
	aead, err := k.aead(account, false)
	if err != nil {
		return nil, err
	}
	data = data[len(sealedPrefix):]
	if len(data) < aead.NonceSize()+aead.Overhead() {
		return nil, fmt.Errorf("invalid encrypted session")
	}
	plain, err := aead.Open(nil, data[:aead.NonceSize()], data[aead.NonceSize():], []byte(account))
	if err != nil {
		return nil, fmt.Errorf("session integrity verification failed")
	}
	return plain, nil
}

func (k keychainCipher) forget(dir, binding string) error {
	account, err := keyAccount(dir, binding)
	if err != nil {
		return err
	}
	if err := k.keys.Delete(keychainService, account); err != nil && !errors.Is(err, errKeyNotFound) {
		return fmt.Errorf("cannot remove selected Keychain key")
	}
	return nil
}
