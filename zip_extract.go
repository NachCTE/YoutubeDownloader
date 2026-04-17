package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"strings"
)

func extractFFmpegFromZip(zipPath, destExe string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("abrir zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		// Look for ffmpeg.exe inside any subfolder
		name := f.Name
		if strings.HasSuffix(strings.ToLower(name), "/ffmpeg.exe") ||
			strings.EqualFold(name, "ffmpeg.exe") {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			out, err := os.Create(destExe)
			if err != nil {
				rc.Close()
				return err
			}
			_, err = io.Copy(out, rc)
			rc.Close()
			out.Close()
			if err != nil {
				return err
			}
			appendLog("✅ ffmpeg.exe extraído correctamente.")
			return nil
		}
	}
	return fmt.Errorf("ffmpeg.exe no encontrado en el zip")
}

