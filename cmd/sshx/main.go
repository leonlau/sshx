package main

import (
	"fmt"
	"os"
	"path"
	"runtime"

	cli "github.com/jawher/mow.cli"
	"github.com/sirupsen/logrus"
	"github.com/suutaku/sshx/internal/utils"
)

func main() {
	if utils.DebugOn() {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.SetReportCaller(true)
		// 自定义格式
		logrus.SetFormatter(&logrus.TextFormatter{
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				// 只返回文件名和行号
				filename := path.Base(f.File)
				return "", fmt.Sprintf("%s:%d", filename, f.Line)
			},
		})
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}
	app := cli.App("sshx", "a webrtc based ssh remote toolbox")
	app.Command("daemon", "launch a sshx daemon", cmdDaemon)
	app.Command("conn", "connect to remote host", cmdConnect)
	app.Command("scp", "copy files or directory from/to remote host", cmdCopy)
	app.Command("proxy", "start proxy", cmdProxy)
	app.Command("stat", "get status", cmdStatus)
	app.Run(os.Args)

}
