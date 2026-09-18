// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package content

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/mobiletoly/goldr"
)

const (
	metadataFilename = "page.json"
	htmlFilename     = "body.html"
	markdownFilename = "body.md"

	maxMetadataBytes = 16 * 1024
	maxBodyBytes     = 512 * 1024
)

type entryFiles struct {
	metadata fs.DirEntry
	html     fs.DirEntry
	markdown fs.DirEntry
}

type loadedEntry struct {
	source   string
	metadata goldr.PageMetadata
	body     []byte
	markdown bool
}

func checkDirectory(files fs.FS, directory string, root bool) error {
	entries, err := fs.ReadDir(files, directory)
	if err != nil {
		return contentError(displayPath(directory), "read directory", err)
	}

	pageFiles := recognizedEntryFiles(entries)
	if root && pageFiles.recognized() {
		return contentError(".", "recognized page files are not allowed at the filesystem root", nil)
	}
	for _, entry := range entries {
		entryPath := joinPath(directory, entry.Name())
		if err := validateDirectoryEntry(entryPath, entry); err != nil {
			return err
		}
		if entry.IsDir() && !validSegment(entry.Name()) {
			return contentError(entryPath, "directory name must use lowercase letters, digits, and single hyphens", nil)
		}
	}

	if !root && pageFiles.recognized() {
		entry, err := loadEntry(files, directory, pageFiles)
		if err != nil {
			return err
		}
		if _, err := renderEntryBody(entry); err != nil {
			return err
		}
	}
	for _, entry := range entries {
		if entry.IsDir() {
			if err := checkDirectory(files, joinPath(directory, entry.Name()), false); err != nil {
				return err
			}
		}
	}
	return nil
}

func loadRequestedEntry(files fs.FS, entryPath string) (loadedEntry, bool, error) {
	directory := "."
	for segment := range strings.SplitSeq(entryPath, "/") {
		entries, err := fs.ReadDir(files, directory)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return loadedEntry{}, false, nil
			}
			return loadedEntry{}, false, contentError(displayPath(directory), "read directory", err)
		}
		entry, ok := findDirectoryEntry(entries, segment)
		if !ok {
			return loadedEntry{}, false, nil
		}
		segmentPath := joinPath(directory, segment)
		isDirectory, err := validateTraversedDirectory(segmentPath, entry)
		if err != nil {
			return loadedEntry{}, false, err
		}
		if !isDirectory {
			return loadedEntry{}, false, nil
		}
		directory = segmentPath
	}

	entries, err := fs.ReadDir(files, directory)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return loadedEntry{}, false, nil
		}
		return loadedEntry{}, false, contentError(directory, "read directory", err)
	}
	pageFiles := recognizedEntryFiles(entries)
	if !pageFiles.recognized() {
		return loadedEntry{}, false, nil
	}
	entry, err := loadEntry(files, directory, pageFiles)
	if err != nil {
		return loadedEntry{}, false, err
	}
	return entry, true, nil
}

func recognizedEntryFiles(entries []fs.DirEntry) entryFiles {
	var found entryFiles
	for _, entry := range entries {
		switch entry.Name() {
		case metadataFilename:
			found.metadata = entry
		case htmlFilename:
			found.html = entry
		case markdownFilename:
			found.markdown = entry
		}
	}
	return found
}

func (files entryFiles) recognized() bool {
	return files.metadata != nil || files.html != nil || files.markdown != nil
}

func loadEntry(files fs.FS, directory string, entry entryFiles) (loadedEntry, error) {
	if entry.metadata == nil {
		return loadedEntry{}, contentError(directory, "recognized entry is missing page.json", nil)
	}
	if (entry.html == nil) == (entry.markdown == nil) {
		return loadedEntry{}, contentError(directory, "recognized entry must contain exactly one of body.html or body.md", nil)
	}

	metadataPath := joinPath(directory, metadataFilename)
	metadataBytes, err := readRecognizedFile(files, metadataPath, entry.metadata, maxMetadataBytes)
	if err != nil {
		return loadedEntry{}, err
	}
	metadata, err := parseMetadata(metadataPath, metadataBytes)
	if err != nil {
		return loadedEntry{}, err
	}

	bodyEntry := entry.html
	bodyName := htmlFilename
	markdown := false
	if entry.markdown != nil {
		bodyEntry = entry.markdown
		bodyName = markdownFilename
		markdown = true
	}
	bodyPath := joinPath(directory, bodyName)
	body, err := readRecognizedFile(files, bodyPath, bodyEntry, maxBodyBytes)
	if err != nil {
		return loadedEntry{}, err
	}
	if !utf8.Valid(body) {
		return loadedEntry{}, contentError(bodyPath, "body must be valid UTF-8", nil)
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return loadedEntry{}, contentError(bodyPath, "body must not be empty or whitespace-only", nil)
	}

	return loadedEntry{
		source:   bodyPath,
		metadata: metadata,
		body:     body,
		markdown: markdown,
	}, nil
}

func validateDirectoryEntry(source string, entry fs.DirEntry) error {
	if entry.Type()&fs.ModeSymlink != 0 {
		return contentError(source, "symbolic links are not supported", nil)
	}
	info, err := entry.Info()
	if err != nil {
		return contentError(source, "read file information", err)
	}
	if !info.IsDir() && !info.Mode().IsRegular() {
		return contentError(source, "special files are not supported", nil)
	}
	return nil
}

func validateTraversedDirectory(source string, entry fs.DirEntry) (bool, error) {
	if entry.Type()&fs.ModeSymlink != 0 {
		return false, contentError(source, "symbolic link directory is not supported", nil)
	}
	info, err := entry.Info()
	if err != nil {
		return false, contentError(source, "read directory information", err)
	}
	if info.IsDir() {
		return true, nil
	}
	if info.Mode().IsRegular() {
		return false, nil
	}
	return false, contentError(source, "content path segment is not a directory", nil)
}

func readRecognizedFile(files fs.FS, source string, entry fs.DirEntry, limit int64) ([]byte, error) {
	if entry.Type()&fs.ModeSymlink != 0 {
		return nil, contentError(source, "symbolic link file is not supported", nil)
	}
	entryInfo, err := entry.Info()
	if err != nil {
		return nil, contentError(source, "read file information", err)
	}
	if !entryInfo.Mode().IsRegular() {
		return nil, contentError(source, "recognized page file must be regular", nil)
	}

	file, err := files.Open(source)
	if err != nil {
		return nil, contentError(source, "open file", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, contentError(source, "stat opened file", err)
	}
	if !info.Mode().IsRegular() {
		return nil, contentError(source, "opened page file must be regular", nil)
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, contentError(source, "read file", err)
	}
	if int64(len(data)) > limit {
		return nil, contentError(source, fmt.Sprintf("file exceeds %d byte limit", limit), nil)
	}
	return data, nil
}

func parseMetadata(source string, data []byte) (goldr.PageMetadata, error) {
	if !utf8.Valid(data) {
		return goldr.PageMetadata{}, contentError(source, "metadata must be valid UTF-8", nil)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	first, err := decoder.Token()
	if err != nil {
		return goldr.PageMetadata{}, contentError(source, "decode metadata object", err)
	}
	if delimiter, ok := first.(json.Delim); !ok || delimiter != '{' {
		return goldr.PageMetadata{}, contentError(source, "metadata must be one JSON object", nil)
	}

	seen := make(map[string]struct{}, 2)
	var metadata goldr.PageMetadata
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return goldr.PageMetadata{}, contentError(source, "decode metadata key", err)
		}
		key, ok := keyToken.(string)
		if !ok {
			return goldr.PageMetadata{}, contentError(source, "metadata key must be a string", nil)
		}
		if _, ok := seen[key]; ok {
			return goldr.PageMetadata{}, contentError(source, "duplicate metadata field "+key, nil)
		}
		seen[key] = struct{}{}

		valueToken, err := decoder.Token()
		if err != nil {
			return goldr.PageMetadata{}, contentError(source, "decode metadata field "+key, err)
		}
		value, ok := valueToken.(string)
		if !ok {
			return goldr.PageMetadata{}, contentError(source, "metadata field "+key+" must be a string", nil)
		}
		switch key {
		case "title":
			metadata.Title = value
		case "description":
			metadata.Description = value
		default:
			return goldr.PageMetadata{}, contentError(source, "unknown metadata field "+key, nil)
		}
	}
	if _, err := decoder.Token(); err != nil {
		return goldr.PageMetadata{}, contentError(source, "close metadata object", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return goldr.PageMetadata{}, contentError(source, "metadata contains trailing JSON value", nil)
		}
		return goldr.PageMetadata{}, contentError(source, "decode metadata trailing data", err)
	}
	if strings.TrimSpace(metadata.Title) == "" {
		return goldr.PageMetadata{}, contentError(source, "metadata title must be a nonblank string", nil)
	}
	return metadata, nil
}

func findDirectoryEntry(entries []fs.DirEntry, name string) (fs.DirEntry, bool) {
	for _, entry := range entries {
		if entry.Name() == name {
			return entry, true
		}
	}
	return nil, false
}

func joinPath(directory, name string) string {
	if directory == "." {
		return name
	}
	return path.Join(directory, name)
}

func displayPath(directory string) string {
	if directory == "" {
		return "."
	}
	return directory
}
