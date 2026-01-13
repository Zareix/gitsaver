package tarball

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func ExtractTarGz(tarGzPath, destPath string) error {
	file, err := os.Open(tarGzPath)
	if err != nil {
		return fmt.Errorf("Failed to open tar.gz file: %w", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("Failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("Failed to read tar header: %w", err)
		}

		// The tarball from GitHub contains a parent directory,
		// we want to extract the contents of that directory.
		// Example: github-user-repo-name-deadbeef/
		parts := strings.Split(header.Name, "/")
		if len(parts) > 1 {
			header.Name = strings.Join(parts[1:], "/")
		} else {
			continue
		}

		target := filepath.Join(destPath, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("Failed to create directory: %w", err)
			}
		case tar.TypeReg:
			if header.Name == "" {
				continue
			}
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("Failed to create directory for file: %w", err)
			}

			outFile, err := os.Create(target)
			if err != nil {
				return fmt.Errorf("Failed to create file: %w", err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("Failed to copy file content: %w", err)
			}
			outFile.Close()
		default:
			log.Printf("Unsupported tar header type: %c for %s\n", header.Typeflag, header.Name)
		}
	}
	log.Printf("Successfully extracted %s to %s", tarGzPath, destPath)

	if err := os.Remove(tarGzPath); err != nil {
		return fmt.Errorf("Failed to remove tar.gz file: %w", err)
	}

	return nil
}
