package auth

import (
	"errors"
	"github.com/zalando/go-keyring"
)

type systemKeychain struct{}

func (systemKeychain) Get(service, account string) (string, error) {
	v, err := keyring.Get(service, account)
	if errors.Is(err, keyring.ErrNotFound) {
		err = errKeyNotFound
	}
	return v, err
}
func (systemKeychain) Set(service, account, value string) error {
	return keyring.Set(service, account, value)
}
func (systemKeychain) Delete(service, account string) error {
	err := keyring.Delete(service, account)
	if errors.Is(err, keyring.ErrNotFound) {
		return errKeyNotFound
	}
	return err
}
func protectSession(dir, binding string, b []byte) ([]byte, error) {
	return (keychainCipher{systemKeychain{}}).seal(dir, binding, b)
}
func unprotectSession(dir, binding string, b []byte) ([]byte, error) {
	return (keychainCipher{systemKeychain{}}).open(dir, binding, b)
}
func forgetProtection(dir, binding string) error {
	return (keychainCipher{systemKeychain{}}).forget(dir, binding)
}
