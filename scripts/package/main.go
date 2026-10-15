// Package release archives using only the Go standard library. Unix archives
// preserve executable permission bits even when generated on Windows.
package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"h3yun-cli/pkg/h3yun"
)

func main() {
	rootFlag := flag.String("root", ".", "project root")
	outFlag := flag.String("out", "..", "release directory outside project")
	flag.Parse()
	root, err := filepath.Abs(*rootFlag)
	check(err)
	out, err := filepath.Abs(*outFlag)
	check(err)
	if root == out || strings.HasPrefix(out, root+string(os.PathSeparator)) {
		check(fmt.Errorf("release directory must be outside project"))
	}
	check(os.MkdirAll(out, 0755))
	files := []string{}
	rootFiles := map[string]bool{"README.md": true, "AGENT.md": true, "AUTH.md": true, "MACOS.md": true, "VALIDATION.md": true, "API-CONTRACT.md": true, "THIRD_PARTY_NOTICES.md": true, "go.mod": true, "go.sum": true, ".gitignore": true, "config.example.json": true, "profiles.example.json": true, "operations.example.json": true}
	dirs := map[string]bool{"cmd": true, "internal": true, "pkg": true, "scripts": true, "skills": true, "examples": true, "dist": true, "licenses": true}
	check(filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if d.IsDir() {
			if !dirs[parts[0]] || strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if len(parts) == 1 && !rootFiles[rel] {
			return nil
		}
		if strings.Contains(rel, ".local.") || strings.HasPrefix(d.Name(), ".env") {
			return nil
		}
		files = append(files, rel)
		return nil
	}))
	check(makeZip(root, filepath.Join(out, "h3yun-cli-v"+h3yun.Version+".zip"), files))
	for _, arch := range []string{"arm64", "amd64"} {
		check(makeTar(root, filepath.Join(out, "h3yun-v"+h3yun.Version+"-darwin-"+arch+".tar.gz"), "dist/h3yun-darwin-"+arch))
	}
	fmt.Println("Created source/multiplatform ZIP and macOS arm64/amd64 tar.gz packages")
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func executable(name string) bool {
	return strings.HasPrefix(filepath.ToSlash(name), "dist/") || strings.HasSuffix(name, ".sh")
}
func makeZip(root, out string, files []string) (err error) {
	f, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	z := zip.NewWriter(f)
	for _, name := range files {
		info, e := os.Stat(filepath.Join(root, name))
		if e != nil {
			return e
		}
		h, e := zip.FileInfoHeader(info)
		if e != nil {
			return e
		}
		h.Name = "h3yun-cli/" + filepath.ToSlash(name)
		h.Method = zip.Deflate
		mode := fs.FileMode(0644)
		if executable(name) {
			mode = 0755
		}
		h.SetMode(mode)
		w, e := z.CreateHeader(h)
		if e != nil {
			return e
		}
		if e = copyFile(w, filepath.Join(root, name)); e != nil {
			return e
		}
	}
	if err = z.Close(); err != nil {
		return err
	}
	return f.Sync()
}
func makeTar(root, out, binary string) (err error) {
	f, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	g := gzip.NewWriter(f)
	t := tar.NewWriter(g)
	names := []string{binary, "README.md", "MACOS.md", "AUTH.md", "AGENT.md", "VALIDATION.md", "API-CONTRACT.md", "THIRD_PARTY_NOTICES.md", "config.example.json", "profiles.example.json", "operations.example.json", "examples/record.json", "examples/records.json", "examples/filter.json", "skills/h3yun-shared/SKILL.md", "skills/h3yun-records/SKILL.md", "skills/h3yun-files/SKILL.md", "scripts/smoke.sh"}
	licenseFiles, err := filepath.Glob(filepath.Join(root, "licenses", "*.txt"))
	if err != nil {
		return err
	}
	if len(licenseFiles) == 0 {
		return fmt.Errorf("third-party license files are missing")
	}
	for _, license := range licenseFiles {
		names = append(names, filepath.Join("licenses", filepath.Base(license)))
	}
	for _, name := range names {
		info, e := os.Stat(filepath.Join(root, name))
		if e != nil {
			return e
		}
		h, e := tar.FileInfoHeader(info, "")
		if e != nil {
			return e
		}
		h.Name = filepath.ToSlash(name)
		h.Mode = 0644
		if executable(name) {
			h.Mode = 0755
		}
		if name == binary {
			h.Name = "h3yun"
		}
		if e = t.WriteHeader(h); e != nil {
			return e
		}
		if e = copyFile(t, filepath.Join(root, name)); e != nil {
			return e
		}
	}
	if err = t.Close(); err != nil {
		return err
	}
	if err = g.Close(); err != nil {
		return err
	}
	return f.Sync()
}
func copyFile(out io.Writer, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(out, f)
	return err
}
