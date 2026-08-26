package safefile

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

var pathLocks sync.Map

// WithLock serializes read-modify-write operations targeting the same file.
// Readers do not need the lock when writers use WriteFile because replacement is atomic.
func WithLock(path string, operation func() error) error {
	cleanPath := filepath.Clean(path)
	lockValue, _ := pathLocks.LoadOrStore(cleanPath, &sync.Mutex{})
	lock := lockValue.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()
	return operation()
}

// WriteFile replaces path atomically and keeps the previous complete version at path + ".bak".
// The caller must create the parent directory before calling WriteFile.
func WriteFile(path string, data []byte, mode fs.FileMode) error {
	if previous, err := os.ReadFile(path); err == nil {
		if err := writeAtomically(path+".bak", previous, mode); err != nil {
			return fmt.Errorf("backup %s: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read existing %s: %w", path, err)
	}

	if err := writeAtomically(path, data, mode); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}

func writeAtomically(path string, data []byte, mode fs.FileMode) (resultErr error) {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() {
		if temporary != nil {
			if err := temporary.Close(); resultErr == nil && err != nil {
				resultErr = err
			}
		}
		if resultErr != nil {
			_ = os.Remove(temporaryPath)
		}
	}()

	if err := temporary.Chmod(mode); err != nil {
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	temporary = nil
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}

	// Persist the directory entry when supported. The file content is already safe if this fails.
	if directoryHandle, err := os.Open(directory); err == nil {
		_ = directoryHandle.Sync()
		_ = directoryHandle.Close()
	}
	return nil
}
