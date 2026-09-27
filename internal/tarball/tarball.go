package tarball

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func ExtractTarGz(tarGzPath, destPath string) error {
	file, err := os.Open(tarGzPath)
	if err != nil {
		return fmt.Errorf("failed to open tar.gz file: %w", err)
	}
	defer func(file *os.File) {
		if err := file.Close(); err != nil {
			slog.Error("Failed to close file", "path", tarGzPath, "error", err)
		}
	}(file)

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func(gzipReader *gzip.Reader) {
		if err := gzipReader.Close(); err != nil {
			slog.Error("Failed to close gzip reader", "error", err)
		}
	}(gzipReader)

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		parts := strings.Split(header.Name, "/")
		if len(parts) <= 1 {
			continue
		}
		header.Name = strings.Join(parts[1:], "/")

		if header.Name == "" {
			continue
		}

		target := filepath.Join(destPath, header.Name)
		if !isWithinDir(destPath, target) {
			return fmt.Errorf("illegal path in tarball: %q", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("failed to create directory for file: %w", err)
			}

			outFile, err := os.Create(target)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				_ = outFile.Close()
				return fmt.Errorf("failed to copy file content: %w", err)
			}
			if err := outFile.Close(); err != nil {
				return fmt.Errorf("failed to close file: %w", err)
			}
		default:
			slog.Debug("Skipping unsupported tar entry", "type", string(header.Typeflag), "name", header.Name)
		}
	}

	slog.Info("Extracted tarball", "path", tarGzPath, "dest", destPath)

	if err := os.Remove(tarGzPath); err != nil {
		return fmt.Errorf("failed to remove tar.gz file: %w", err)
	}

	return nil
}

func isWithinDir(dir, target string) bool {
	rel, err := filepath.Rel(dir, target)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..")
}
