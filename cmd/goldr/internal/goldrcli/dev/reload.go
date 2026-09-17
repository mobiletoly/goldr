// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package dev

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

const devProxyReloadPath = "/_templ/reload/events"

const (
	defaultReloadDebounce      = 100 * time.Millisecond
	defaultReloadRetryDelay    = 100 * time.Millisecond
	defaultReloadNotifyTimeout = 2 * time.Second
)

type reloadPath struct {
	path      string
	directory bool
}

func resolveReloadPaths(rawPaths []string) ([]reloadPath, error) {
	paths := make([]reloadPath, 0, len(rawPaths))
	for _, rawPath := range rawPaths {
		if strings.TrimSpace(rawPath) == "" {
			return nil, errors.New("--reload-path must not be empty")
		}
		absolute, err := filepath.Abs(rawPath)
		if err != nil {
			return nil, fmt.Errorf("resolve --reload-path %q: %w", rawPath, err)
		}
		canonical, err := filepath.EvalSymlinks(absolute)
		if err != nil {
			return nil, fmt.Errorf("resolve --reload-path %q: %w", rawPath, err)
		}
		info, err := os.Stat(canonical)
		if err != nil {
			return nil, fmt.Errorf("inspect --reload-path %q: %w", rawPath, err)
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return nil, fmt.Errorf("--reload-path %q must be a regular file or directory", rawPath)
		}
		if info.Mode().IsRegular() {
			file, err := os.Open(canonical)
			if err != nil {
				return nil, fmt.Errorf("read --reload-path %q: %w", rawPath, err)
			}
			if err := file.Close(); err != nil {
				return nil, fmt.Errorf("read --reload-path %q: %w", rawPath, err)
			}
		}
		paths = append(paths, reloadPath{
			path:      filepath.Clean(canonical),
			directory: info.IsDir(),
		})
	}

	sort.Slice(paths, func(i, j int) bool {
		if len(paths[i].path) == len(paths[j].path) {
			return paths[i].path < paths[j].path
		}
		return len(paths[i].path) < len(paths[j].path)
	})

	resolved := make([]reloadPath, 0, len(paths))
	for _, candidate := range paths {
		contained := false
		for _, existing := range resolved {
			if candidate.path == existing.path || existing.directory && pathWithin(existing.path, candidate.path) {
				contained = true
				break
			}
		}
		if !contained {
			resolved = append(resolved, candidate)
		}
	}
	return resolved, nil
}

func pathWithin(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator))
}

func templControlsReloadPath(appRoot, path string) bool {
	if !pathWithin(appRoot, path) {
		return false
	}
	return regexp.MustCompile(devWatchPattern()).MatchString(path) ||
		regexp.MustCompile(devIgnorePattern()).MatchString(path)
}

type reloadWatcher struct {
	watcher     *fsnotify.Watcher
	paths       []reloadPath
	appRoot     string
	debounce    time.Duration
	watchedDirs map[string]bool
}

func newReloadWatcher(ctx context.Context, paths []reloadPath, appRoot string, debounce time.Duration) (*reloadWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create reload watcher: %w", err)
	}
	reload := &reloadWatcher{
		watcher:     watcher,
		paths:       paths,
		appRoot:     appRoot,
		debounce:    debounce,
		watchedDirs: make(map[string]bool),
	}
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			_ = watcher.Close()
			return nil, err
		}
		if path.directory {
			if err := reload.addTree(ctx, path.path); err != nil {
				_ = watcher.Close()
				return nil, err
			}
			continue
		}
		if err := reload.addDir(filepath.Dir(path.path)); err != nil {
			_ = watcher.Close()
			return nil, err
		}
	}
	return reload, nil
}

func (w *reloadWatcher) Close() error {
	return w.watcher.Close()
}

func (w *reloadWatcher) addTree(ctx context.Context, root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err != nil {
			return fmt.Errorf("walk reload directory %q: %w", root, err)
		}
		if !entry.IsDir() {
			return nil
		}
		return w.addDir(path)
	})
}

func (w *reloadWatcher) addDir(path string) error {
	if w.watchedDirs[path] {
		return nil
	}
	if err := w.watcher.Add(path); err != nil {
		return fmt.Errorf("watch reload directory %q: %w", path, err)
	}
	w.watchedDirs[path] = true
	return nil
}

func (w *reloadWatcher) Run(ctx context.Context, notify func(context.Context) error, stderr io.Writer) error {
	var timer *time.Timer
	var timerChannel <-chan time.Time
	stopTimer := func() {
		if timer != nil && !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}
	defer stopTimer()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err, ok := <-w.watcher.Errors:
			if !ok {
				if ctx.Err() != nil {
					return nil
				}
				return errors.New("reload watcher error channel closed unexpectedly")
			}
			return fmt.Errorf("reload watcher: %w", err)
		case event, ok := <-w.watcher.Events:
			if !ok {
				if ctx.Err() != nil {
					return nil
				}
				return errors.New("reload watcher event channel closed unexpectedly")
			}
			matched, fatal := w.handleEvent(ctx, event)
			if fatal != nil {
				return fatal
			}
			if !matched || templControlsReloadPath(w.appRoot, event.Name) {
				continue
			}
			stopTimer()
			if timer == nil {
				timer = time.NewTimer(w.debounce)
			} else {
				timer.Reset(w.debounce)
			}
			timerChannel = timer.C
		case <-timerChannel:
			timerChannel = nil
			if err := notify(ctx); err != nil && ctx.Err() == nil {
				fmt.Fprintf(stderr, "goldr dev: browser reload notification failed: %v\n", err)
			}
		}
	}
}

func (w *reloadWatcher) handleEvent(ctx context.Context, event fsnotify.Event) (bool, error) {
	interesting := event.Op & (fsnotify.Create | fsnotify.Write | fsnotify.Remove | fsnotify.Rename)
	if interesting == 0 {
		return false, nil
	}
	path := filepath.Clean(event.Name)
	for _, configured := range w.paths {
		if configured.directory && path == configured.path && event.Op&(fsnotify.Remove|fsnotify.Rename) != 0 {
			return false, fmt.Errorf("configured reload directory %q was removed or renamed", configured.path)
		}
	}

	if event.Op&(fsnotify.Remove|fsnotify.Rename) != 0 && w.watchedDirs[path] {
		w.forgetTree(path)
	}
	if event.Op&fsnotify.Create != 0 {
		info, err := os.Stat(path)
		if err == nil && info.IsDir() {
			for _, configured := range w.paths {
				if configured.directory && pathWithin(configured.path, path) {
					if err := w.addTree(ctx, path); err != nil {
						return false, err
					}
					break
				}
			}
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return false, fmt.Errorf("inspect created reload path %q: %w", path, err)
		}
	}

	for _, configured := range w.paths {
		if configured.directory && pathWithin(configured.path, path) || !configured.directory && path == configured.path {
			return true, nil
		}
	}
	return false, nil
}

func (w *reloadWatcher) forgetTree(root string) {
	for path := range w.watchedDirs {
		if pathWithin(root, path) {
			delete(w.watchedDirs, path)
			_ = w.watcher.Remove(path)
		}
	}
}

func notifyDevProxy(ctx context.Context, client *http.Client, proxyURL string, retryDelay, timeout time.Duration) error {
	notifyCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	endpoint := strings.TrimRight(proxyURL, "/") + devProxyReloadPath
	var lastError error
	for {
		request, err := http.NewRequestWithContext(notifyCtx, http.MethodPost, endpoint, nil)
		if err != nil {
			return fmt.Errorf("create browser reload request: %w", err)
		}
		response, err := client.Do(request)
		if err == nil {
			if closeErr := response.Body.Close(); closeErr != nil {
				lastError = fmt.Errorf("close browser reload response: %w", closeErr)
			} else if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
				return nil
			} else {
				lastError = fmt.Errorf("browser reload proxy returned %s", response.Status)
			}
		} else if notifyCtx.Err() == nil || lastError == nil {
			lastError = err
		}

		timer := time.NewTimer(retryDelay)
		select {
		case <-notifyCtx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("notify browser reload: %w", lastError)
		case <-timer.C:
		}
	}
}
