package conn

import (
	"encoding/gob"
	"fmt"
	"net"

	"github.com/sirupsen/logrus"
	"github.com/suutaku/sshx/pkg/impl"
	"github.com/suutaku/sshx/pkg/types"
)

type DirectInfo struct {
	Id       int64
	ImplCode int32
	HostId   string
}

type DirectService struct {
	BaseConnectionService
	localTCPPort int32
}

func NewDirectService(id string, localTCPPort int32) *DirectService {
	return &DirectService{
		BaseConnectionService: *NewBaseConnectionService(id),
		localTCPPort:          localTCPPort,
	}
}

func (ds *DirectService) Start() error {
	ds.BaseConnectionService.Start()
	listenner, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", ds.localTCPPort))
	if err != nil {
		logrus.Error(err)
	}

	go func() {
		logrus.Debug("runing status ", ds.running)
		for ds.running {
			sock, err := listenner.Accept()
			if err != nil {
				logrus.Error(err)
				continue
			}
			var info DirectInfo
			err = gob.NewDecoder(sock).Decode(&info)
			if err != nil {
				logrus.Error(err)
				continue
			}
			logrus.Debug("new direct info com ", info)
			imp := impl.GetImpl(info.ImplCode)
			imp.SetHostId(info.HostId)
			poolId := types.NewPoolId(info.Id, imp.Code())
			// server reset direction
			conn := NewDirectConnection(imp, ds.Id(), info.HostId, *poolId, CONNECTION_DRECT_IN, &ds.CleanChan, ds.localTCPPort)
			conn.Conn = sock
			err = conn.Response()
			if err != nil {
				logrus.Error(err)
				continue
			}
			ds.AddPair(conn)
		}
	}()
	return nil
}

func (ds *DirectService) CreateConnection(sender *impl.Sender, sock net.Conn, poolId types.PoolId) error {
	// client reset direction
	err := ds.BaseConnectionService.CreateConnection(sender, sock, poolId)
	if err != nil {
		return err
	}
	iface := sender.GetImpl()
	if iface == nil {
		return fmt.Errorf("unknown impl")

	}

	if !sender.Detach {
		iface.SetConn(sock)
	}
	pair := NewDirectConnection(iface, ds.Id(), iface.HostId(), poolId, CONNECTION_DRECT_OUT, &ds.CleanChan, ds.localTCPPort)
	err = pair.Dial()
	if err != nil {
		return err
	}
	err = ds.AddPair(pair)
	if err != nil {
		return err
	}
	return nil
}

func (ds *DirectService) DestroyConnection(tmp *impl.Sender) error {
	pair := ds.GetPair(string(tmp.PairId))
	if pair == nil {
		return fmt.Errorf("cannot get pair for %s", string(tmp.PairId))
	}
	ds.RemovePair(CleanRequest{string(tmp.PairId), (&DirectConnection{}).Name()})
	return nil
}
