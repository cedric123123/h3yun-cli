package localstate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLockAndReplace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state")
	unlock, err := Lock(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Lock(path); err == nil {
		t.Fatal("concurrent writer accepted")
	}
	if err := Replace(path, []byte("before")); err != nil {
		t.Fatal(err)
	}
	if err := Replace(path, []byte("after")); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil || string(b) != "after" {
		t.Fatal(err)
	}
	unlock()
	unlock, err = Lock(path)
	if err != nil {
		t.Fatal(err)
	}
	unlock()
}
