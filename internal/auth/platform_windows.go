//go:build windows

package auth

import (
	"fmt"
	"os/exec"
	"runtime"
	"syscall"
	"unsafe"
)

type blob struct {
	Size uint32
	Data *byte
}

func crypt(data []byte, decrypt bool) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty session")
	}
	input := blob{uint32(len(data)), &data[0]}
	var output blob
	lib := syscall.NewLazyDLL("crypt32.dll")
	proc := lib.NewProc("CryptProtectData")
	if decrypt {
		proc = lib.NewProc("CryptUnprotectData")
	}
	result, _, _ := proc.Call(uintptr(unsafe.Pointer(&input)), 0, 0, 0, 0, 1, uintptr(unsafe.Pointer(&output)))
	runtime.KeepAlive(data)
	if result == 0 {
		return nil, fmt.Errorf("Windows DPAPI session encryption/decryption failed")
	}
	defer syscall.NewLazyDLL("kernel32.dll").NewProc("LocalFree").Call(uintptr(unsafe.Pointer(output.Data)))
	return append([]byte{}, unsafe.Slice(output.Data, int(output.Size))...), nil
}
func seal(b []byte) ([]byte, error)   { return crypt(b, false) }
func unseal(b []byte) ([]byte, error) { return crypt(b, true) }
func OpenBrowser(url string) error {
	cmd := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", url)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}
