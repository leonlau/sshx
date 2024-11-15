package main

import (
	"errors"
	"os"

	"github.com/sirupsen/logrus"
)

const (
	defaultHomePath = "/etc/sshx"
)

func getRootPath() string {
	rootStr := os.Getenv("SSHX_HOME")
	if rootStr == "" {
		rootStr = defaultHomePath
	}
	if _, err := os.Stat(rootStr); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(rootStr, 0766)
		if err != nil {
			logrus.Error(err)
		}
	}
	return rootStr
}
