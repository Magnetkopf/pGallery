package utils

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
)

func ModifyPictureExtension(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("can not access: %s", path)
	}
	_, format, err := image.DecodeConfig(file)
	file.Close()
	if err != nil {
		return fmt.Errorf("unknown picture type: %s, err: %v", path, err)
	}

	ext := strings.ToLower(filepath.Ext(path))

	targetExt := "." + format
	if format == "jpeg" {
		targetExt = ".jpg"
	}

	if ext != targetExt {
		finalPath := strings.TrimSuffix(path, ext) + targetExt
		err = os.Rename(path, finalPath)
		if err != nil {
			return fmt.Errorf("failed to rename %s->%s: %v", path, finalPath, err)
		}
	}

	return nil
}
