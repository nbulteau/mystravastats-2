package safefile

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestWriteFileReplacesAtomicallyAndBacksUpPreviousContent(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "settings.json")

	if err := WriteFile(path, []byte("first"), 0600); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if _, err := os.Stat(path + ".bak"); !os.IsNotExist(err) {
		t.Fatalf("first write should not create a backup, stat error: %v", err)
	}
	if err := WriteFile(path, []byte("second"), 0600); err != nil {
		t.Fatalf("second write: %v", err)
	}

	assertFileContent(t, path, "second")
	assertFileContent(t, path+".bak", "first")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat result: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("mode = %o, want 600", got)
	}
	matches, err := filepath.Glob(filepath.Join(directory, ".settings.json.tmp-*"))
	if err != nil {
		t.Fatalf("glob temporary files: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files left behind: %v", matches)
	}
}

func TestWithLockSerializesOperationsForTheSamePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "counter.json")
	const operationCount = 50
	var waitGroup sync.WaitGroup
	for index := 0; index < operationCount; index++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			if err := WithLock(path, func() error {
				current := 0
				if data, err := os.ReadFile(path); err == nil {
					if _, err := fmt.Sscanf(string(data), "%d", &current); err != nil {
						return err
					}
				} else if !os.IsNotExist(err) {
					return err
				}
				return WriteFile(path, []byte(fmt.Sprint(current+1)), 0600)
			}); err != nil {
				t.Errorf("locked operation: %v", err)
			}
		}()
	}
	waitGroup.Wait()
	assertFileContent(t, path, fmt.Sprint(operationCount))
}

func assertFileContent(t *testing.T, path string, expected string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if got := string(data); got != expected {
		t.Fatalf("content of %s = %q, want %q", path, got, expected)
	}
}
