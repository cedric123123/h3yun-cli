// Package localstate provides fail-fast cooperative locks and staged local writes.
package localstate

import (
	"fmt"
	"os"
	"path/filepath"
)

// Lock never steals an existing lock. After a process crash the caller must
// verify no writer is running before manually removing the exact lock directory.
func Lock(path string) (func(), error) {
	if err := os.Mkdir(path+".lock", 0700); err != nil {
		return nil, fmt.Errorf("local state is locked or its directory is inaccessible")
	}
	return func() { _ = os.Remove(path + ".lock") }, nil
}

func Replace(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".h3yun-state-*")
	if err != nil {
		return fmt.Errorf("cannot stage local state")
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		return fmt.Errorf("cannot write local state")
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return fmt.Errorf("cannot publish local state")
	}
	return nil
}
