package bblwheel

import "github.com/looplab/fsm"

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

func loadAndWatchService() {

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
	if instances.exist(id, name) {
		instances.remove(id, name)
	}
	DelKv(registerKey(name, id))
}

type serviceInstance struct {
	srv            *Service
	lastActiveTime int64
	ch             BblWheel_KeepAliveServer
	done           chan struct{}
	fsm            *fsm.FSM
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
