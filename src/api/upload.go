package api

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
)

func parseMultipartWithinLimit(w http.ResponseWriter, r *http.Request) error {
	limit := bootstrap.GetConfig().MaxUploadSize
	if limit <= 0 {
		limit = 20 * 1024 * 1024
	}
	// Multipart boundaries and headers need a small allowance in addition to the
	// configured file limit. The copied file is checked against the exact limit.
	r.Body = http.MaxBytesReader(w, r.Body, limit+(1<<20))
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return fmt.Errorf("upload exceeds the configured limit of %d bytes", limit)
		}
		return fmt.Errorf("invalid multipart upload: %w", err)
	}
	return nil
}

func storeUploadedSave(header *multipart.FileHeader) error {
	name, err := factorio.NormalizeSaveName(header.Filename)
	if err != nil {
		return err
	}
	config := bootstrap.GetConfig()
	target, err := factorio.ResolveDataPath(config.FactorioSavesDir, name)
	if err != nil {
		return err
	}
	if _, err = os.Stat(target); err == nil {
		return fmt.Errorf("save %q already exists", name)
	} else if !os.IsNotExist(err) {
		return err
	}

	source, err := header.Open()
	if err != nil {
		return err
	}
	defer source.Close()

	temporary, err := os.CreateTemp(config.FactorioSavesDir, ".fsm-upload-*.zip")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)

	limit := config.MaxUploadSize
	if limit <= 0 {
		limit = 20 * 1024 * 1024
	}
	written, copyErr := io.Copy(temporary, io.LimitReader(source, limit+1))
	closeErr := temporary.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if written > limit {
		return fmt.Errorf("save exceeds the configured limit of %d bytes", limit)
	}

	archive, err := zip.OpenReader(temporaryName)
	if err != nil {
		return fmt.Errorf("uploaded save is not a valid ZIP archive: %w", err)
	}
	if len(archive.File) == 0 {
		archive.Close()
		return fmt.Errorf("uploaded save ZIP is empty")
	}
	if err = archive.Close(); err != nil {
		return err
	}
	if err = os.Rename(temporaryName, target); err != nil {
		return err
	}
	return nil
}

func uploadedFiles(r *http.Request, field string) []*multipart.FileHeader {
	if r.MultipartForm == nil {
		return nil
	}
	return r.MultipartForm.File[field]
}
