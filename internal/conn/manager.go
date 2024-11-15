package conn

import (
	"encoding/gob"
	"fmt"
	"net"
	"reflect"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/suutaku/sshx/internal/utils"
	"github.com/suutaku/sshx/pkg/impl"
	"github.com/suutaku/sshx/pkg/types"
)

// manage all supported connection implementations
type ConnectionManager struct {
	css []ConnectionService
	stm *StatManager
}

func NewConnectionManager(enabledService []ConnectionService) *ConnectionManager {
	return &ConnectionManager{
		stm: NewStatManager(),
		css: enabledService,
	}
}

func (cm *ConnectionManager) Start() {
	logrus.Debug("Start connection manager")
	for _, v := range cm.css {
		v.SetStateManager(cm.stm)
		v.Start()
		typeName := ""
		if t := reflect.TypeOf(v); t.Kind() == reflect.Ptr {
			typeName = "*" + t.Elem().Name()
		} else {
			typeName = t.Name()
		}
		logrus.Debug("Start ", typeName)
	}
}

func (cm *ConnectionManager) Stop() {
	for _, v := range cm.css {
		v.Stop()
	}
}

func (cm *ConnectionManager) CreateConnection(sender *impl.Sender, sock net.Conn, poolId types.PoolId) error {
	for i := 0; i < len(cm.css); i++ {
		num := i
		if cm.css[num].IsReady() {
			go func(cs ConnectionService, num int) {
				s, c := net.Pipe()
				err := cs.CreateConnection(sender, c, poolId)
				if err != nil {
					logrus.Error(err, num)
					return
				}
				sender.PairId = []byte(poolId.String(CONNECTION_DRECT_OUT))
				err = cs.ResponseTCP(sender, sock)
				if err != nil {
					logrus.Error(err, num)
					return
				}
				utils.Pipe(&sock, &s)
			}(cm.css[num], num)
		}

		// go func(idx int) {
		// 	if cm.css[idx].IsReady() {
		// 		err := cm.css[idx].CreateConnection(sender, sock, poolId)
		// 		if err != nil {
		// 			logrus.Error(err)
		// 			return
		// 		}
		// 	}
		// }(i)
	}
	return nil
}

func (cm *ConnectionManager) DestroyConnection(sender *impl.Sender, conn net.Conn) error {
	err := cm.css[0].DestroyConnection(sender)
	if err != nil {
		return err
	}
	err = cm.css[0].ResponseTCP(sender, conn)
	if err != nil {
		logrus.Error(err)
		return err
	}
	return nil
}

func (cm *ConnectionManager) AttachConnection(sender *impl.Sender, sock net.Conn) error {
	go func() {
		s, c := net.Pipe()
		err := cm.css[0].AttachConnection(sender, c)
		if err != nil {
			logrus.Error(err)
			return
		}
		logrus.Debug("attached ", sender.GetImpl().HostId())
		err = cm.css[0].ResponseTCP(sender, sock)
		if err != nil {
			logrus.Error(err)
			return
		}
		utils.Pipe(&s, &sock)
	}()
	return nil
}

func (cm *ConnectionManager) Status(sender impl.Sender, conn net.Conn) error {
	imp := sender.GetImpl()
	bs := NewBaseConnectionService(imp.HostId())
	imp.SetConn(conn)
	err := bs.ResponseTCP(&sender, conn)
	if err != nil {
		logrus.Error(err)
		return err
	}
	res := []types.Status{}
	err = gob.NewDecoder(conn).Decode(&res)
	if err != nil {
		logrus.Error(err)
		return err
	}
	res = cm.stm.Stat()
	logrus.Debug("responsed ----->", res)
	err = gob.NewEncoder(conn).Encode(res)
	if err != nil {
		logrus.Error(err)
		return err
	}
	logrus.Debug("responsed <-----")
	return nil
}

type StatManager struct {
	stats    map[string]types.Status
	children map[string][]string
	cpPool   map[string]Connection
	running  bool
	lock     sync.Mutex
}

func NewStatManager() *StatManager {

	return &StatManager{
		stats:    make(map[string]types.Status),
		children: make(map[string][]string),
		cpPool:   make(map[string]Connection),
	}
}

func (stm *StatManager) Stop() {
	stm.running = false
}

func (stm *StatManager) addChild(parent, child string) {
	if stm.children[parent] == nil {
		stm.children[parent] = make([]string, 0)
	}
	stm.children[parent] = append(stm.children[parent], child)
}

func (stm *StatManager) getChildren(parent string) []string {
	return stm.children[parent]
}

func (stm *StatManager) removeParent(parent string) {
	delete(stm.children, parent)
}

func (stm *StatManager) putStat(stat types.Status) {
	if stat.PairId == "" {
		logrus.Warn("empty paird id for status: ", stat)
		return
	}
	if stm.stats[stat.PairId].PairId == stat.PairId {
		return
	}
	stm.stats[stat.PairId] = stat
	logrus.Debug("put status ", stat.PairId)
}

func (stm *StatManager) getStat() []types.Status {
	ret := make([]types.Status, 0)

	for _, v := range stm.stats {
		ret = append(ret, []types.Status{v}...)
	}
	return ret
}

func (stm *StatManager) removeStat(pid string) {
	delete(stm.stats, pid)
	logrus.Debug("remove status for ", pid)
}

func PoolIdFromInt(id int64) string {
	return fmt.Sprintf("conn_%d", id)
}

func (stm *StatManager) Stat() []types.Status {
	return stm.getStat()
}

func (stm *StatManager) RemovePair(id CleanRequest) {
	stm.lock.Lock()
	defer stm.lock.Unlock()
	children := stm.getChildren(id.Key)
	logrus.Debug("ready to clear children ", children)
	// close children
	for _, v := range children {
		if stm.cpPool[v] != nil && stm.cpPool[v].Name() == id.ConnectionName {
			stm.cpPool[v].Close()
			delete(stm.cpPool, v)
			stm.removeStat(id.Key)
		}

	}
	// close parent
	if stm.cpPool[id.Key] != nil && stm.cpPool[id.Key].Name() == id.ConnectionName {
		stm.cpPool[id.Key].Close()
		delete(stm.cpPool, id.Key)
		stm.removeStat(id.Key)
		stm.removeParent(id.Key)
	}
}

func (stm *StatManager) doAddPair(pair Connection) error {
	stm.cpPool[pair.PoolId().String(pair.Direction())] = pair
	logrus.Debugf("add pair %s %s successfully\n", pair.PoolId().String(pair.Direction()), pair.Name())
	stat := types.Status{
		PairId:    pair.PoolId().String(pair.Direction()),
		TargetId:  pair.TargetId(),
		ImplType:  pair.GetImpl().Code(),
		StartTime: time.Now(),
	}

	if pair.GetImpl().ParentId() != "" {
		logrus.Debug("add child ", pair.PoolId().String(pair.Direction()), " to ", pair.GetImpl().ParentId())
		stat.ParentPairId = pair.GetImpl().ParentId()
		stm.addChild(pair.GetImpl().ParentId(), pair.PoolId().String(pair.Direction()))
	}
	stm.putStat(stat)
	logrus.Debug("put pair on stat ", impl.GetImplName(pair.GetImpl().Code()), " with pair id ", stat.PairId)
	return nil

}

func (stm *StatManager) AddPair(pair Connection) error {

	if pair == nil {
		return fmt.Errorf("pair was empty")
	}

	oldPair := stm.cpPool[pair.PoolId().String(pair.Direction())]

	if oldPair != nil {
		if oldPair.IsReady() {
			pair.Close()
			return fmt.Errorf("pair already exist, drop %s", pair.Name())
		}
		// old pair not ready
		if pair.IsReady() {
			//replace old pair
			return stm.doAddPair(pair)
		}
		for !oldPair.IsReady() && !pair.IsReady() {
			logrus.Debugf("watting %s\n", pair.Name())
			time.Sleep(500 * time.Millisecond)
		}
		if oldPair.IsReady() {
			return fmt.Errorf("pair already exist, drop %s", pair.Name())
		}
		if pair.IsReady() {
			return fmt.Errorf("replace pair from %s to %s ", oldPair.Name(), pair.Name())
		}
	} else {
		return stm.doAddPair(pair)
	}
	return nil
}

func (stm *StatManager) GetPair(id string) Connection {
	return stm.cpPool[id]
}
