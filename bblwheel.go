package bblwheel

import (
	"flag"
	"fmt"
	"io"
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
	return &Wheel{instances: map[string]*serviceInstance{}}, nil
}

//Wheel ....
type Wheel struct {
	instances map[string]*serviceInstance
	lock      sync.RWMutex
}

//LookupConfig ....
func (s *Wheel) LookupConfig(_ context.Context, req *LookupConfigReq) (*LookupConfigResp, error) {
	return &LookupConfigResp{Configs: confmgt.get(req.DependentConfigs)}, nil
}

//LookupService ....
func (s *Wheel) LookupService(_ context.Context, req *LookupServiceReq) (*LookupServiceResp, error) {
	resp := &LookupServiceResp{Services: []*Service{}}
	if len(req.DependentServices) > 0 {
		var dep = []string{}
		for _, name := range req.DependentServices {
			if aumgt.has(name, req.ServiceName) {
				dep = append(dep, name)
			}
		}
		resp.Services = srvmgt.findServiceList(dep)
	}
	return resp, nil
}

//Register ....
func (s *Wheel) Register(_ context.Context, srv *Service) (*RegisterResult, error) {
	res := &RegisterResult{Desc: "SUCCESS"}
	err := srvmgt.register(srv)
	if err != nil {
		res.Desc = err.Error()
		grpclog.Println("Register.register", err)
		return res, nil
	}
	if len(srv.DependentServices) > 0 {
		var dep = []string{}
		for _, name := range srv.DependentServices {
			grpclog.Println("DependentServices", name, srv.Name)
			if aumgt.has(name, srv.Name) {
				dep = append(dep, name)
			}
		}
		grpclog.Println("findServiceList", dep)
		res.Service = srvmgt.findServiceList(dep)
	}
	if len(srv.DependentConfigs) > 0 {
		res.Configs = confmgt.get(srv.DependentConfigs)
	}

	ins := serviceInstance{srv: srv, lastActiveTime: time.Now().Unix(), wheel: s}
	s.lock.Lock()
	s.instances[srv.key()] = &ins
	s.lock.Unlock()
	return res, nil
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

//UpdateConfig ....
func (s *Wheel) UpdateConfig(_ context.Context, req *UpdateConfigReq) (*Void, error) {
	if err := confmgt.put(req.ServiceName, req.ServiceID, req.Config); err != nil {
		grpclog.Println(err)
	}
	return &Void{}, nil
}

//Events ....
func (s *Wheel) Events(ch BblWheel_EventsServer) error {
	grpclog.Println("Wheel.Events channel", ch)
	defer grpclog.Println("Events channel", ch, "exist")
	ev, err := ch.Recv()
	if err == io.EOF {
		grpclog.Println("Wheel.Events", err)
		return nil
	}
	if err != nil {
		grpclog.Println("Wheel.Events", err)
		return err
	}
	if ev.Type != Event_KEEPALIVE {
		err = fmt.Errorf("Error Event.Type %s", Event_EventType_name[int32(ev.Type)])
		grpclog.Println(err)
		return err
	}
	srv := ev.Service
	if srv == nil {
		err = fmt.Errorf("Error Event.Service is nil")
		grpclog.Println(err)
		return err
	}
	s.lock.Lock()
	ins, has := s.instances[srv.key()]
	if !has {
		s.lock.Unlock()
		err = fmt.Errorf("Error Event.Service %s not registered", srv.key())
		grpclog.Println(err)
		return err
	}
	if ins.ch != nil {
		s.lock.Unlock()
		err = fmt.Errorf("Error Event.Service %s channel exist", srv.key())
		grpclog.Println(err)
		return err
	}
	ins.ch = ch
	s.lock.Unlock()
	return ins.serve()
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
	///v1/bblwheel/service/grant/serviceA/testService1 1
	grpclog.Println("onGrant", from, to)
	srvs := srvmgt.findServiceList([]string{from})
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, ins := range s.instances {
		for _, n := range ins.srv.DependentServices {
			if n == from && ins.srv.Name == to {
				go func() {
					for _, srv := range srvs {
						ins.notify(&Event{Type: Event_DISCOVERY, Service: srv})
					}
				}()
			}
		}
	}
}
func (s *Wheel) onCancel(from string, to string) {
	grpclog.Println("onCancel", from, to)
	s.lock.RLock()
	defer s.lock.RUnlock()
	srvs := srvmgt.findServiceList([]string{from})
	for _, ins := range s.instances {
		for _, n := range ins.srv.DependentServices {
			if n == from && ins.srv.Name == to {
				go func() {
					for _, srv := range srvs {
						ins.notify(&Event{Type: Event_DISCOVERY, Service: &Service{ID: srv.ID, Name: srv.Name, Status: Service_UNAUTHORIZE}})
					}
				}()
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
					go oins.notify(&Event{Type: Event_DISCOVERY, Service: srv})
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
					go oins.notify(&Event{Type: Event_DISCOVERY, Service: &Service{ID: id, Name: name, Status: Service_OFFLINE}})
				}
			}
		}
	}
	delete(s.instances, name+"/"+id)
}

func (s *Wheel) onKeepAlive() {
	grpclog.Printf("NumGoroutine %d NumCPU %d\n", runtime.NumGoroutine(), runtime.NumCPU())
}

type serviceInstance struct {
	srv            *Service
	lastActiveTime int64
	ch             BblWheel_EventsServer
	fsm            *fsm.FSM
	wheel          *Wheel
}

func (ins *serviceInstance) serve() error {
	for {
		ev, err := ins.ch.Recv()
		if err == io.EOF {
			grpclog.Println("Wheel.Events", err)
			return nil
		}
		if err != nil {
			grpclog.Println("Wheel.Events", err)
			return err
		}

		if ev.Type != Event_KEEPALIVE {
			err = fmt.Errorf("Error Event.Type %s", Event_EventType_name[int32(ev.Type)])
			grpclog.Println(err)
			return err
		}
		if ev.Service == nil {
			err = fmt.Errorf("Error Event.Service is nil")
			grpclog.Println(err)
			return err
		}
		ins.srv = ev.Service
		ins.lastActiveTime = time.Now().Unix()
		err = srvmgt.update(ev.Service)
		if err != nil {
			//grpclog.Println("srvmgt.update", err)
			grpclog.Println("srvmgt.update", ev.Service, err)
		}
	}
}

func (ins *serviceInstance) notify(ev *Event) {
	if ins.ch == nil {
		return
	}
	grpclog.Println("notify", ins.srv.key(), ev)
	if err := ins.ch.Send(ev); err != nil {
		grpclog.Println("serviceInstance.notify", err)
		if err := srvmgt.unregister(ins.srv.ID, ins.srv.Name); err != nil {
			grpclog.Println("srvmgt.unregister", err)
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
			key = key + "/" + s
		}
	}
	return key
}
