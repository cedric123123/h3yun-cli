package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/cedric123123/h3yun-cli/internal/localstate"
)

func StoreDir(override string) (string, error) {
	if override != "" {
		return filepath.Abs(override)
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("cannot locate user config directory")
	}
	return filepath.Join(base, "h3yun", "auth"), nil
}

func checkDir(dir string, create bool) error {
	if create {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("cannot create session directory")
		}
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("session directory must not be a symlink")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("session directory must have mode 0700")
	}
	return nil
}

func sessionPath(dir, binding string) (string, error) {
	if len(binding) != 64 {
		return "", fmt.Errorf("invalid session binding")
	}
	for _, r := range binding {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return "", fmt.Errorf("invalid session binding")
		}
	}
	return filepath.Join(dir, binding+".session"), nil
}

func Save(dir string, s Session) error {
	path, err := sessionPath(dir, s.Binding)
	if err != nil {
		return err
	}
	if err := checkDir(dir, true); err != nil {
		return err
	}
	unlock, err := localstate.Lock(path)
	if err != nil {
		return err
	}
	defer unlock()
	b, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("cannot encode session")
	}
	b, err = protectSession(dir, s.Binding, b)
	if err != nil {
		return err
	}
	return localstate.Replace(path, b)
}

func Load(dir, binding string) (Session, error) {
	path, err := sessionPath(dir, binding)
	if err != nil {
		return Session{}, err
	}
	if err := checkDir(dir, false); err != nil {
		return Session{}, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return Session{}, err
	}
	if !info.Mode().IsRegular() || (runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0) || info.Size() > 2<<20 {
		return Session{}, fmt.Errorf("invalid session file or permissions")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return Session{}, fmt.Errorf("cannot read session")
	}
	b, err = unprotectSession(dir, binding, b)
	if err != nil {
		return Session{}, err
	}
	var s Session
	if json.Unmarshal(b, &s) != nil || s.Binding != binding || s.AccessToken == "" {
		return Session{}, fmt.Errorf("session binding mismatch or invalid session")
	}
	if !s.ExpiresAt.After(time.Now().Add(30 * time.Second)) {
		return Session{}, fmt.Errorf("session expired; run auth login")
	}
	return s, nil
}

func Logout(dir, binding string) error {
	path, err := sessionPath(dir, binding)
	if err != nil {
		return err
	}
	if err := checkDir(dir, false); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	unlock, err := localstate.Lock(path)
	if err != nil {
		return err
	}
	defer unlock()
	err = os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return forgetProtection(dir, binding)
}

func StorageBackend() string {
	switch runtime.GOOS {
	case "darwin":
		return "macos_keychain_aes256gcm"
	case "windows":
		return "windows_dpapi"
	default:
		return "owner_only_file_unencrypted"
	}
}
