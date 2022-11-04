package util

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type AutoReload struct {
	root   string
	stop   func() error
	start  func() error
	logger *log.Logger
	*PeriodicJob
}

func NewAutoReload(root string, start, stop func() error, loggerOut io.Writer) (r AutoReload) {
	root = strings.TrimSpace(root)
	if root == "" {
		root = "."
	}

	startTime := time.Now()
	r = AutoReload{
		root:   root,
		start:  start,
		stop:   stop,
		logger: log.New(loggerOut, "AutoReloadService ", log.LstdFlags|log.Lshortfile),
	}

	r.PeriodicJob = NewPeriodicJob(func() {
		if r.didFileChange(r.root, startTime) {
			if err := stop(); err != nil {
				r.logger.Println("[ERROR]", "while stopping:", err)
			}

			startTime = time.Now()
			go func() {
				if err := start(); err != nil {
					r.logger.Println("[ERROR]", "while starting server:", err)
				}
			}()
		}
	}, time.Second)

	return
}

func (r *AutoReload) didFileChange(dir string, since time.Time) (changed bool) {
	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.Contains(path, ".go") || !strings.Contains(path, ".git/") {
			return nil
		}

		if info.ModTime().After(since) {
			changed = true
			return nil
		}

		return nil
	}); err != nil {
		r.logger.Println("[ERROR]", "while checking file changes:", err)
	}

	return changed
}

func (r *AutoReload) SetRoot(root string) {
	r.root = root
}

func (r *AutoReload) Root() string {
	return r.root
}
