package bblwheel

import (
	"flag"
	"fmt"
	"log"
	"net"
	"runtime"
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/looplab/fsm"
	"google.golang.org/grpc"
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

var (
	//ListenAddr ....
	ListenAddr = "0.0.0.0:23790"
	//WorkDir ....
	WorkDir = "/var/lib/bblwheel"
)

func init() {
	flag.StringVar(&ListenAddr, "bblwheel.address", ListenAddr, "rpc listen address")
	flag.StringVar(&WorkDir, "workdir", WorkDir, "work directory")
}

//StartWheel ....
func StartWheel() error {
	return startEtcd(func() error {
		cli = etcdClient([]string{AdvertiseClientAddr}, "", "")
		startAuthWatcher()
		startConfigWatcher()
		startServiceWatcher()
		srv, err := newWheel()
		if err != nil {
			return err
		}
		confmgt.observer = srv.onConfigChanged
		srvmgt.observer = srv
		aumgt.observer = srv
		return srv.serve()
	})

}

func newWheel() (*Wheel, error) {
	return &Wheel{}, nil
}

//Wheel ....
type Wheel struct {
	instances map[string]*serviceInstance
	lock      sync.RWMutex
}

//Register ....
func (s *Wheel) Register(srv *Service, ch BblWheel_RegisterServer) error {
	srv.Status = Service_INIT
	res := &Event{}
	err := srvmgt.register(srv)
	if err != nil {
		res.Desc = err.Error()
		return ch.Send(res)
	}
	if len(srv.DependentServices) > 0 {
		var dep = []string{}
		for _, name := range srv.DependentServices {
			if aumgt.has(name, srv.Name) {
				dep = append(dep, name)
			}
		}
		deps, err := srvmgt.findServiceList(dep)
		if err != nil {
			res.Desc = err.Error()
			return ch.Send(res)
		}
		res.Service = deps
	}
	if len(srv.DependentConfigs) > 0 {
		res.Configs = confmgt.get(srv.DependentConfigs)
	}

	if err := ch.Send(res); err != nil {
		return err
	}
	ins := serviceInstance{srv: srv, ch: ch, lastActiveTime: time.Now().Unix(), wheel: s}
	s.lock.Lock()
	s.instances[srv.key()] = &ins
	s.lock.Unlock()
	return nil
}

//Unregister ....
func (s *Wheel) Unregister(ctx context.Context, srv *Service) (*Void, error) {
	if err := srvmgt.unregister(srv.ID, srv.Name); err != nil {
		grpclog.Println(err)
	}
	s.lock.Lock()
	delete(s.instances, srv.key())
	s.lock.Unlock()
	return &Void{}, nil
}
func (srv *Service) key() string {
	return fmt.Sprintf("%s/%s", srv.Name, srv.ID)
}

//Online ....
func (s *Wheel) Online(_ context.Context, srv *Service) (*Void, error) {
	srv.Status = Service_ONLINE

	err := srvmgt.register(srv)
	if err != nil {
		return &Void{}, err
	}
	s.lock.Lock()
	if ins, has := s.instances[srv.key()]; has {
		ins.srv = srv
	}
	s.lock.Unlock()
	return &Void{}, nil
}

//UpdateStatus ....
func (s *Wheel) UpdateStatus(ctx context.Context, srv *Service) (*Void, error) {
	if err := srvmgt.update(srv); err != nil {
		grpclog.Println(err)
		return &Void{}, err
	}
	return &Void{}, nil
}

//UpdateConfig ....
func (s *Wheel) UpdateConfig(ctx context.Context, req *UpdateConfigReq) (*Void, error) {
	if err := confmgt.put(req.ServiceName, req.ServiceID, req.Config); err != nil {
		grpclog.Println(err)
	}
	return &Void{}, nil
}

//Serve ....
func (s *Wheel) serve() error {
	if !flag.Parsed() {
		flag.Parse()
	}
	lis, err := net.Listen("tcp", ListenAddr)
	if err != nil {
		grpclog.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	server := grpc.NewServer(opts...)
	RegisterBblWheelServer(server, s)
	log.Println("bblwheel server listen at", ListenAddr)
	return server.Serve(lis)
}

func (s *Wheel) onGrant(from string, to string) {
	///v1/bblwheel/service/grant/a/b 1
	for _, ins := range s.instances {
		for _, n := range ins.srv.DependentServices {
			if n == from && ins.srv.Name == to {
				srvs := []*Service{}
				for _, o := range s.instances {
					if o.srv.Name == from {
						srvs = append(srvs, o.srv)
					}
				}
				go ins.notify(&Event{Type: Event_DISCOVERY, Service: srvs})
			}
		}
	}
}
func (s *Wheel) onCancel(from string, to string) {
	for _, ins := range s.instances {
		for _, n := range ins.srv.DependentServices {
			if n == from && ins.srv.Name == to {
				srvs := []*Service{}
				for _, o := range s.instances {
					if o.srv.Name == from {
						srvs = append(srvs, &Service{ID: o.srv.ID, Name: o.srv.Name, Status: Service_UNAUTHORIZE})
					}
				}
				go ins.notify(&Event{Type: Event_DISCOVERY, Service: srvs})
			}
		}
	}
}

func (s *Wheel) onConfigChanged(key string, item *ConfigEntry) {
	grpclog.Println("onConfigChanged")
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, ins := range s.instances {
		for _, n := range ins.srv.DependentConfigs {
			if n == key {
				if oins, has := s.instances[ins.srv.key()]; has {
					go oins.notify(&Event{Type: Event_CONFIGUPDATE, Item: item})
				}
			}
		}
	}
}

func (s *Wheel) onUpdate(srv *Service) {
	grpclog.Println("onUpdate")
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, ins := range s.instances {
		for _, n := range ins.srv.DependentServices {
			if n == srv.Name {
				if oins, has := s.instances[ins.srv.key()]; has && oins.srv.Status != srv.Status && aumgt.has(srv.Name, n) {
					go oins.notify(&Event{Type: Event_DISCOVERY, Service: []*Service{srv}})
				}
			}
		}
	}
}
func (s *Wheel) onDelete(name, id string) {
	grpclog.Println("onDelete")
	s.lock.Lock()
	defer s.lock.Unlock()
	for _, ins := range s.instances {
		for _, n := range ins.srv.DependentServices {
			if n == name {
				if oins, has := s.instances[ins.srv.key()]; has {
					go oins.notify(&Event{Type: Event_DISCOVERY, Service: []*Service{&Service{ID: id, Name: name, Status: Service_OFFLINE}}})
				}
			}
		}
	}
	s.lock.Lock()
	delete(s.instances, name+"/"+id)
	s.lock.Unlock()
}

var kve = &Event{Type: Event_KEEPALIVE}

func (s *Wheel) onKeepAlive() {
	grpclog.Println("NumGoroutine", runtime.NumGoroutine(), "NumCPU", runtime.NumCPU())
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, ins := range s.instances {
		go func(ins *serviceInstance) {
			ins.notify(kve)
		}(ins)
	}
}

type serviceInstance struct {
	srv            *Service
	lastActiveTime int64
	ch             BblWheel_RegisterServer
	fsm            *fsm.FSM
	wheel          *Wheel
}

func (ins *serviceInstance) notify(ev *Event) {
	if err := ins.ch.Send(kve); err != nil {
		grpclog.Println(err)
		if err := srvmgt.unregister(ins.srv.ID, ins.srv.Name); err != nil {
			grpclog.Println(err)
		}
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
