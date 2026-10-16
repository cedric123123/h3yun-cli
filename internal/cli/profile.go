package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/cedric123123/h3yun-cli/internal/localstate"
	"github.com/cedric123123/h3yun-cli/pkg/h3yun"
)

func switchProfile(v values, env func(string) string) (any, error) {
	path := v.s("config")
	if path == "" {
		path = env("H3YUN_CONFIG")
	}
	if path == "" {
		return nil, h3yun.Fail("CONFIG", "profile switch requires --config or H3YUN_CONFIG; no file is chosen implicitly", 3)
	}
	if env("H3YUN_PROFILE") != "" {
		return nil, h3yun.Fail("CONFIG", "remove --profile/H3YUN_PROFILE before changing the persisted default", 3)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, h3yun.Fail("CONFIG", "cannot resolve config path", 3)
	}
	// Resolve parent aliases before locking, but do not follow the target itself.
	parent, err := filepath.EvalSymlinks(filepath.Dir(abs))
	if err != nil {
		return nil, h3yun.Fail("CONFIG", "config parent directory is inaccessible", 3)
	}
	path = filepath.Join(parent, filepath.Base(abs))
	var unlock func()
	if !v.b("dry-run") {
		unlock, err = localstate.Lock(path)
		if err != nil {
			return nil, h3yun.Fail("CONFIG_LOCKED", err.Error(), 3)
		}
		defer unlock()
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, h3yun.Fail("CONFIG", "config must be an existing regular file, not a symlink", 3)
	}
	before, err := readFile(path)
	if err != nil {
		return nil, err
	}
	var doc ConfigFile
	if err = strictJSON(before, &doc); err != nil {
		return nil, err
	}
	name := v.s("name")
	if name == "-" {
		name = doc.PreviousProfile
	}
	if _, ok := doc.Profiles[name]; name == "" || !ok {
		return nil, h3yun.Fail("PROFILE_NOT_FOUND", "target or previous profile does not exist", 3)
	}
	old := doc.CurrentProfile
	result := map[string]any{"current_profile": name, "previous_profile": old, "config": path, "changed": false, "dry_run": v.b("dry-run"), "credentials_unchanged": true}
	if name == old {
		result["previous_profile"] = doc.PreviousProfile
		return result, nil
	}
	if v.b("dry-run") {
		result["would_change"] = true
		return result, nil
	}
	doc.CurrentProfile, doc.PreviousProfile = name, old
	encoded, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, h3yun.Fail("CONFIG", "cannot encode configuration", 3)
	}
	// A unique backup preserves the exact original bytes. Do not touch auth files.
	backup, err := os.CreateTemp(parent, ".h3yun-config-backup-*")
	if err != nil {
		return nil, h3yun.Fail("OUTPUT", "cannot back up configuration", 6)
	}
	if err = backup.Chmod(0600); err == nil {
		_, err = backup.Write(before)
	}
	if err == nil {
		err = backup.Sync()
	}
	closeErr := backup.Close()
	if err != nil || closeErr != nil {
		_ = os.Remove(backup.Name())
		return nil, h3yun.Fail("OUTPUT", "cannot write configuration backup", 6)
	}
	latest, err := readFile(path)
	currentInfo, statErr := os.Lstat(path)
	if err != nil || statErr != nil || !os.SameFile(info, currentInfo) || !bytes.Equal(before, latest) {
		return nil, h3yun.Fail("CONFIG_CHANGED", "configuration changed during switch; original backup retained", 3)
	}
	if err := localstate.Replace(path, append(encoded, '\n')); err != nil {
		return nil, h3yun.Fail("OUTPUT", "cannot replace configuration; original backup retained", 6)
	}
	result["changed"], result["backup"] = true, backup.Name()
	return result, nil
}
