//go:build unix

// Package fslock fournit un verrou exclusif inter-processus (advisory) posé sur un fichier ouvert.
// Il sert à sérialiser la section critique « lire la dernière entrée + ajouter » du journal d'audit
// entre plusieurs invocations concurrentes de Kanjō (§17.5).
package fslock

import (
	"os"

	"golang.org/x/sys/unix"
)

// Lock pose un verrou exclusif bloquant sur le fichier et renvoie la fonction de déverrouillage.
// Le verrou est associé à la description de fichier ouverte : il est libéré par unlock() ou à la
// fermeture du descripteur (donc aussi à la mort du processus, sans verrou fantôme).
func Lock(f *os.File) (unlock func() error, err error) {
	fd := int(f.Fd())
	if err := unix.Flock(fd, unix.LOCK_EX); err != nil {
		return nil, err
	}
	return func() error { return unix.Flock(fd, unix.LOCK_UN) }, nil
}
