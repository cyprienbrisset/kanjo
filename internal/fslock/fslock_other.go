//go:build !unix && !windows

package fslock

import "os"

// Lock est un repli sans verrouillage inter-processus pour les plateformes dépourvues de flock
// (best-effort). Les six cibles officielles de Kanjō (linux/darwin/windows) utilisent les
// implémentations dédiées ; ce repli garantit seulement la compilation ailleurs.
func Lock(_ *os.File) (unlock func() error, err error) {
	return func() error { return nil }, nil
}
