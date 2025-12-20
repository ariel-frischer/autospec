package worktree

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyDir recursively copies a directory from src to dst.
// It creates the destination directory if it doesn't exist.
func CopyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("accessing source directory: %w", err)
	}

	if !srcInfo.IsDir() {
		return fmt.Errorf("source %q is not a directory", src)
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}

	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walking directory: %w", err)
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("computing relative path: %w", err)
		}

		dstPath := filepath.Join(dst, relPath)

		if d.IsDir() {
			return createDir(path, dstPath)
		}

		return copyFile(path, dstPath)
	})
}

// createDir creates a directory at dstPath with the same permissions as srcPath.
func createDir(srcPath, dstPath string) error {
	info, err := os.Stat(srcPath)
	if err != nil {
		return fmt.Errorf("getting source directory info: %w", err)
	}

	if err := os.MkdirAll(dstPath, info.Mode()); err != nil {
		return fmt.Errorf("creating directory %q: %w", dstPath, err)
	}

	return nil
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source file: %w", err)
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("getting source file info: %w", err)
	}

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("creating destination file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copying file content: %w", err)
	}

	return nil
}

// CopyDirs copies multiple directories from srcRoot to dstRoot.
// Missing source directories are silently skipped.
// Returns a list of directories that were successfully copied.
func CopyDirs(srcRoot, dstRoot string, dirs []string) ([]string, error) {
	var copied []string

	for _, dir := range dirs {
		src := filepath.Join(srcRoot, dir)
		dst := filepath.Join(dstRoot, dir)

		// Skip if source doesn't exist
		if _, err := os.Stat(src); os.IsNotExist(err) {
			continue
		}

		if err := CopyDir(src, dst); err != nil {
			return copied, fmt.Errorf("copying %q: %w", dir, err)
		}

		copied = append(copied, dir)
	}

	return copied, nil
}
