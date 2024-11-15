package main

import (
	cli "github.com/jawher/mow.cli"
	"github.com/sirupsen/logrus"
	"github.com/suutaku/sshx/pkg/impl"
	"github.com/suutaku/sshx/pkg/types"
)

func cmdCopyId(cmd *cli.Cmd) {
	cmd.Spec = "ADDR"
	addr := cmd.StringArg("ADDR", "", "remote target address [username]@[host]:[port]")
	cmd.Action = func() {
		if addr == nil || *addr == "" {
			return
		}
		imp := impl.NewSSH(*addr, false, "", false)
		err := imp.Preper()
		if err != nil {
			logrus.Error(err)
			return
		}
		sender := impl.NewSender(imp, types.OPTION_TYPE_UP)
		conn, err := sender.Send()
		if err != nil {
			logrus.Error(err)
			return
		}
		imp.OpenTerminal(conn)
	}
}

func cmdConnect(cmd *cli.Cmd) {
	cmd.Spec = "[ -i ] ADDR"

	ident := cmd.StringOpt("i identification", "", "a private path, default empty for ~/.ssh/id_rsa")

	addr := cmd.StringArg("ADDR", "", "remote target address [username]@[nodeid]")
	cmd.Action = func() {
		if addr == nil || *addr == "" {
			return
		}
		imp := impl.NewSSH(*addr, false, *ident, false)
		err := imp.Preper()
		if err != nil {
			logrus.Error(err)
			return
		}
		sender := impl.NewSender(imp, types.OPTION_TYPE_UP)
		conn, err := sender.Send()
		if err != nil {
			logrus.Error(err)
			return
		}
		imp.OpenTerminal(conn)
	}
}
