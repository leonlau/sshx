package node

import (
	"encoding/gob"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/suutaku/sshx/pkg/conf"
	"github.com/suutaku/sshx/pkg/impl"
	"github.com/suutaku/sshx/pkg/types"
)

func (node *Node) ServeTCP() {

	os.Remove(conf.SockFile)
	// 创建目录（如果不存在）
	if err := os.MkdirAll(filepath.Dir(conf.SockFile), 0755); err != nil {
		logrus.Errorf("failed to create socket directory: %w", err)
		panic(err)
	}

	listenner, err := net.Listen("unix", conf.SockFile)
	if err != nil {
		logrus.Error(err)
		panic(err)
	}

	// 设置socket文件权限
	if err := os.Chmod(conf.SockFile, 0666); err != nil {
		logrus.Errorf("chmod socket file error: %s", err)
		panic(err)
	}

	// listenner, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", node.confManager.Conf.LocalTCPPort))
	// if err != nil {
	// 	logrus.Error(err)
	// 	panic(err)
	// }

	defer listenner.Close()
	for node.running {
		sock, err := listenner.Accept()
		if err != nil {
			logrus.Error(err)
			continue
		}
		tmp := impl.Sender{}
		err = gob.NewDecoder(sock).Decode(&tmp)
		if err != nil {
			logrus.Debug("read not ok", err)
			sock.Close()
			continue
		}
		switch tmp.GetOptionCode() {
		case types.OPTION_TYPE_UP:
			logrus.Debug("up option")
			impl := tmp.GetImpl()
			if impl == nil {
				logrus.Error("unkwon implementation")
				continue
			}
			poolId := types.NewPoolId(time.Now().UnixNano(), impl.Code())
			err := node.connMgr.CreateConnection(&tmp, sock, *poolId)
			if err != nil {
				sock.Close()
				logrus.Error(err)
			}

		case types.OPTION_TYPE_DOWN:
			logrus.Debug("down option ", string(tmp.PairId))
			err := node.connMgr.DestroyConnection(&tmp, sock)
			if err != nil {
				logrus.Error(err)
			}

		case types.OPTION_TYPE_STAT:
			logrus.Debug("stat option")
			err := node.connMgr.Status(tmp, sock)
			if err != nil {
				sock.Close()
				logrus.Error(err)
			}
		case types.OPTION_TYPE_ATTACH:
			logrus.Debug("attach option")
			err := node.connMgr.AttachConnection(&tmp, sock)
			if err != nil {
				logrus.Error(err)
			}
		}
	}
}
