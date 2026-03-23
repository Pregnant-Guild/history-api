package assets

import "embed"

//go:embed resources/*
var files embed.FS

func GetFileContent(path string) (string, error) {
	data, err := files.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
