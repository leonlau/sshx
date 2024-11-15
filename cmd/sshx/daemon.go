package main

import (
	cli "github.com/jawher/mow.cli"
	"github.com/suutaku/sshx/internal/node"
	"github.com/suutaku/sshx/pkg/conf"
)

func cmdDaemon(cmd *cli.Cmd) {
	conf.IsDaemon = true
	cmd.Action = func() {
		n := node.NewNode(conf.GetSSHXHome())
		defer n.Stop()
		n.Start()
	}
}
