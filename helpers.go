package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
)

func getCSP() (nonce, csp string, err error) {
	b := make([]byte, 18)
	n, err := rand.Read(b)
	if err != nil {
		return "", "", err
	}
	if n != len(b) {
		return "", "", fmt.Errorf("Not enough random bytes available")
	}
	nonce = base64.StdEncoding.EncodeToString(b)

	var tpl bytes.Buffer
	err = globalCSPTemplate.Execute(&tpl, []string{nonce, globalConfig.ReportURI})
	if err != nil {
		return "", "", err
	}

	return nonce, tpl.String(), nil
}

type zipInfo struct {
	FileName string
	Size     string
}

// getZipList returns a list of all .zip files in the globalConfig.ZipsDir directory that start with the prefix domain
func getZipList(domain string) (list []zipInfo, err error) {
	files, err := ioutil.ReadDir(globalConfig.ZipsDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		// Only list files that has .zip extention and start with the specified domain name.
		if filepath.Ext(file.Name()) == ".zip" && strings.HasPrefix(file.Name(), domain) {
			list = append(list, zipInfo{FileName: file.Name(), Size: readableSize(file.Size())})
		}
	}
	return
}

func fileNameFromURL(path string) string {
	i := strings.LastIndex(path, "/")
	if i > 0 {
		path = path[i+1:]
	}
	i = strings.IndexByte(path, '?')
	if i > 0 {
		path = path[:i]
	}
	if len(path) > 128 {
		path = path[:127]
	}

	file := make([]byte, 0, len(path))
	for _, v := range path {
		switch {
		case v >= 0x30 && v <= 0x39, // 0-9
			v >= 0x41 && v <= 0x5a, // A-Z
			v >= 0x61 && v <= 0x7a, // a-z
			v == 0x2d,              // "-"
			v == 0x2e,              // "."
			v == 0x5f:              // "_"
			file = append(file, byte(v))
		}
	}
	return string(file)
}

func readableSize(b int64) string {
	switch {
	case b > 1024*1024*1024:
		return fmt.Sprintf("%.1f Gb", float64(b)/(1024*1024*1024))
	case b > 1024*1024:
		return fmt.Sprintf("%.1f Mb", float64(b)/(1024*1024))
	case b > 1024:
		return fmt.Sprintf("%.1f kb", float64(b)/1024)
	default:
		return fmt.Sprintf("%d bytes", b)
	}
}
