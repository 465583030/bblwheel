package bblwheel

import (
	"sync"

	"strings"

	"strconv"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"google.golang.org/grpc/grpclog"
)

const (
	//ServiceRegisterPrefix ....
	ServiceRegisterPrefix = "/v1/bblwheel/service/register"
	//ServiceConfigPrefix ....
	ServiceConfigPrefix = "/v1/bblwheel/service/config"
	//ServiceStatPrefix ....
	ServiceStatPrefix = "/v1/bblwheel/service/stat"
	//ServiceGrantPrefix ....
	ServiceGrantPrefix = "/v1/bblwheel/service/grant"
)

var instances = multiinstance{entries: map[string]inslist{}}
var grantService *authService

//var grantTable = map[string][]string{}

func startServiceManagement() {
	loadAllService()
	loadAndWatchGrantTable()
}

func loadAllService() {

}

func loadAndWatchGrantTable() {
	grantService = &authService{table: map[string][]string{}}
	go grantService.watch()
}

func registerService(srv *Service) (*serviceInstance, error) {
	var ins *serviceInstance
	// if instances.e; has {

	// } else {
	// 	ins = &serviceInstance{srv: srv, lastActiveTime: time.Now().Unix(), done: make(chan struct{}, 1)}
	// 	instances[srv.Name+"/"+srv.ID] = []*serviceInstance{ins}
	// 	ins.start()
	// }
	return ins, nil
}

func unregisterService(id, name string) {

}

type serviceInstance struct {
	srv            *Service
	lastActiveTime int64
	ch             BblWheel_KeepAliveServer
	done           chan struct{}
}

func (ins *serviceInstance) dependentService() []*Service {
	return nil
}

func (ins *serviceInstance) dependentConfig() map[string]*Config {
	return nil
}

func (ins *serviceInstance) putConfig(conf *Config) {

}

func (ins *serviceInstance) watch(ch BblWheel_KeepAliveServer) {

}

func (ins *serviceInstance) reset() {
	ins.stop()
	ins.ch.Context().Done()
	ins.start()
}

func (ins *serviceInstance) start() {
	for {

	}
}

func (ins *serviceInstance) stop() {
	ins.done <- struct{}{}
}

type inslist []*serviceInstance

func (l inslist) exist(id string) bool {
	for _, ins := range l {
		if ins.srv.ID == id {
			return true
		}
	}
	return false
}

func (l inslist) get(id string) *serviceInstance {
	for _, ins := range l {
		if ins.srv.ID == id {
			return ins
		}
	}
	return nil
}

func (l inslist) add(ins *serviceInstance) {
	l = append(l, ins)
}

func (l inslist) remove(id string) {
	var j = 0
	for i, ins := range l {
		if ins.srv.ID == id {
			j = i
			break
		}
	}
	l = append(l[:j], l[j+1:]...)
}

type multiinstance struct {
	entries map[string]inslist
}

func (m *multiinstance) get(id, name string) *serviceInstance {
	if insl, has := m.entries[name]; has {
		return insl.get(id)
	}
	return nil
}

func (m *multiinstance) getList(name string) inslist {
	if insl, has := m.entries[name]; has {
		return insl
	}
	return nil
}

func (m *multiinstance) exist(id, name string) bool {
	if insl, has := m.entries[name]; has {
		return insl.exist(id)
	}
	return false
}

func (m *multiinstance) put(ins *serviceInstance) {
	if insl, has := m.entries[ins.srv.Name]; has {
		insl.add(ins)
	} else {
		m.entries[ins.srv.Name] = inslist{ins}
	}
}

func (m *multiinstance) remove(id, name string) {
	if insl, has := m.entries[name]; has {
		insl.remove(id)
	}
}

type authService struct {
	table map[string][]string
	lock  sync.RWMutex
	once  Once
}

func (t *authService) has(from, to string) bool {
	t.lock.RLock()
	if tt, has := t.table[from]; has {
		for _, v := range tt {
			if v == to {
				t.lock.RUnlock()
				return true
			}
		}
	}
	t.lock.RUnlock()
	return false
}

func (t *authService) cancel(from, to string) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if tt, has := t.table[from]; has {
		var j = 0
		for i, v := range tt {
			if v == to {
				j = i
				break
			}
		}
		tt = append(tt[:j], tt[:j+1]...)
	}
}

func (t *authService) add(from, to string) {
	if t.has(from, to) {
		return
	}
	t.lock.Lock()
	defer t.lock.Unlock()
	if tt, has := t.table[from]; has {
		tt = append(tt, to)
	} else {
		tt = []string{to}
		t.table[from] = tt
	}
}
func (t *authService) watch() {
	t.once.Do(func() {
		t.table = map[string][]string{}
		resp, err := GetWithPrfix(ServiceGrantPrefix)
		if err != nil {
			grpclog.Fatalln(err)
			return
		}
		for _, kv := range resp.Kvs {
			name := strings.SplitN(strings.TrimPrefix(string(kv.Key), ServiceGrantPrefix+"/"), "/", 2)
			if len(name) != 2 || name[0] == "" || name[1] == "" {
				grpclog.Println("invalid grant info")
				continue
			}
			val, _ := strconv.ParseBool(string(kv.Value))
			if val {
				t.add(name[0], name[1])
			}
		}
		err = WaitPrefixEvents(ServiceGrantPrefix,
			resp.Header.Revision,
			[]mvccpb.Event_EventType{mvccpb.PUT, mvccpb.DELETE},
			func(event *clientv3.Event) {
				name := strings.SplitN(strings.TrimPrefix(string(event.Kv.Key), ServiceGrantPrefix+"/"), "/", 2)
				if len(name) != 2 || name[0] == "" || name[1] == "" {
					grpclog.Println("invalid grant info")
					return
				}
				val, _ := strconv.ParseBool(string(event.Kv.Value))
				if event.Type == mvccpb.PUT {
					if val {
						t.add(name[0], name[1])
					} else {
						t.cancel(name[0], name[1])
					}
				} else if event.Type == mvccpb.DELETE {
					t.cancel(name[0], name[1])
				}
			})
		if err != nil {
			grpclog.Fatalln(err)
		}
	})
}

func registerKey(suffix ...string) string {
	return joinKey(ServiceRegisterPrefix, suffix...)
}
func configKey(suffix ...string) string {
	return joinKey(ServiceConfigPrefix, suffix...)
}
func statKey(suffix ...string) string {
	return joinKey(ServiceStatPrefix, suffix...)
}

func joinKey(prefix string, suffix ...string) string {
	key := prefix
	for _, s := range suffix {
		if "/" == s {
			key = key + s
		} else {
			key = "/" + s
		}
	}
	return key
}
