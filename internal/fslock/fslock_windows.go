//go:build windows

package fslock

import (
	"os"

	"golang.org/x/sys/windows"
)

// Lock pose un verrou exclusif bloquant sur le premier octet du fichier (LockFileEx) et renvoie la
// fonction de déverrouillage. Windows libère aussi le verrou à la fermeture du handle.
func Lock(f *os.File) (unlock func() error, err error) {
	h := windows.Handle(f.Fd())
	ol := new(windows.Overlapped)
	if err := windows.LockFileEx(h, windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, ol); err != nil {
		return nil, err
	}
	return func() error { return windows.UnlockFileEx(h, 0, 1, 0, ol) }, nil
}
