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
		return fmt.Errorf("failed to open tar.gz file: %w", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Printf("Failed to close file: %v", err)
		}
	}(file)

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func(gzipReader *gzip.Reader) {
		err := gzipReader.Close()
		if err != nil {
			log.Printf("Failed to close gzip reader: %v", err)
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
		if len(parts) > 1 {
			header.Name = strings.Join(parts[1:], "/")
		} else {
			continue
		}

		target := filepath.Join(destPath, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			if header.Name == "" {
				continue
			}
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create directory for file: %w", err)
			}

			outFile, err := os.Create(target)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				err := outFile.Close()
				if err != nil {
					return err
				}
				return fmt.Errorf("failed to copy file content: %w", err)
			}
			err = outFile.Close()
			if err != nil {
				return err
			}
		default:
			log.Printf("Unsupported tar header type: %c for %s\n", header.Typeflag, header.Name)
		}
	}
	log.Printf("Successfully extracted %s to %s", tarGzPath, destPath)

	if err := os.Remove(tarGzPath); err != nil {
		return fmt.Errorf("failed to remove tar.gz file: %w", err)
	}

	return nil
}
