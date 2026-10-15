package auth

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

type fakeKeys struct {
	values         map[string]string
	getErr, setErr error
	sets           int
}

func (f *fakeKeys) Get(service, account string) (string, error) {
	if f.getErr != nil {
		return "", f.getErr
	}
	v, ok := f.values[service+account]
	if !ok {
		return "", errKeyNotFound
	}
	return v, nil
}
func (f *fakeKeys) Set(service, account, value string) error {
	f.sets++
	if f.setErr != nil {
		return f.setErr
	}
	f.values[service+account] = value
	return nil
}
func (f *fakeKeys) Delete(service, account string) error {
	delete(f.values, service+account)
	return nil
}

func TestKeychainCipherIsolationTamperingAndLongTokens(t *testing.T) {
	keys := &fakeKeys{values: map[string]string{}}
	c := keychainCipher{keys}
	dir := t.TempDir()
	binding := strings.Repeat("a", 64)
	plain := []byte(strings.Repeat("large-token-value", 1000))
	sealed, err := c.seal(dir, binding, plain)
	if err != nil || bytes.Contains(sealed, plain) {
		t.Fatal("plaintext leak", err)
	}
	got, err := c.open(dir, binding, sealed)
	if err != nil || !bytes.Equal(got, plain) {
		t.Fatal(err)
	}
	second, err := c.seal(dir, binding, plain)
	if err != nil || bytes.Equal(second, sealed) || keys.sets != 1 {
		t.Fatal("nonce reuse or key replacement", err)
	}
	for _, bad := range [][]byte{[]byte("legacy plaintext secret"), []byte(sealedPrefix), append([]byte(nil), sealed...)} {
		if len(bad) == len(sealed) {
			bad[len(bad)-1] ^= 1
		}
		if _, err := c.open(dir, binding, bad); err == nil {
			t.Fatal("accepted invalid data")
		}
	}
	if _, err := c.open(t.TempDir(), binding, sealed); err == nil {
		t.Fatal("cross directory access")
	}
	if _, err := c.open(dir, strings.Repeat("b", 64), sealed); err == nil {
		t.Fatal("cross profile access")
	}
	if err := c.forget(dir, binding); err != nil {
		t.Fatal(err)
	}
	if _, err := c.open(dir, binding, sealed); err == nil {
		t.Fatal("removed key still available")
	}
}

func TestKeychainFailureDoesNotFallbackOrReplaceKeys(t *testing.T) {
	dir, binding := t.TempDir(), strings.Repeat("a", 64)
	for _, keys := range []*fakeKeys{
		{values: map[string]string{}, getErr: errors.New("denied")},
		{values: map[string]string{}, setErr: errors.New("locked")},
	} {
		c := keychainCipher{keys}
		if b, err := c.seal(dir, binding, []byte("secret")); err == nil || b != nil {
			t.Fatal("plaintext fallback")
		}
		if keys.getErr != nil && keys.sets != 0 {
			t.Fatal("replaced unreadable key")
		}
	}
	account, _ := keyAccount(dir, binding)
	keys := &fakeKeys{values: map[string]string{keychainService + account: "corrupt"}}
	if _, err := (keychainCipher{keys}).seal(dir, binding, []byte("secret")); err == nil || keys.sets != 0 {
		t.Fatal("replaced corrupt key")
	}
}
