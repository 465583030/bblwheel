package bblwheel

import (
	"flag"
	"fmt"
	"io"
	"net"
	"runtime"
	"time"

	"google.golang.org/grpc"

	"golang.org/x/net/context"

	grpclog "log"

	"github.com/looplab/fsm"
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
	grpclog.SetFlags(grpclog.Lshortfile | grpclog.LstdFlags)
}

//StartWheel ....
func StartWheel() error {
	return startEtcd(func() error {
		cli = etcdClient([]string{AdvertiseClientAddr}, "", "")
		startAuthWatcher()
		startConfigWatcher()
		startServiceWatcher()
		wheel, err := newWheel()
		if err != nil {
			return err
		}
		confmgt.observer = wheel.onConfigChanged
		srvmgt.observer = wheel
		aumgt.observer = wheel
		list := srvmgt.findAllService()
		for _, srv := range list {
			wheel.instances[srv.key()] = &serviceInstance{srv: srv, lastActiveTime: time.Now().Unix(), wheel: wheel}
		}
		grpclog.Println("StartWheel.Instances", wheel.instances)
		return wheel.serve()
	})

}

func newWheel() (*Wheel, error) {
	wheel := &Wheel{
		instances: map[string]*serviceInstance{},
		events:    make(chan *event, 1024),
		fn:        map[string]func(*event){},
	}
	wheel.fn = map[string]func(*event){
		//"onRegister":      wheel.doRegister,
		"onExit":          wheel.doExit,
		"onEvent":         wheel.doEvent,
		"onGrant":         wheel.doGrant,
		"onCancel":        wheel.doCancel,
		"onUpdate":        wheel.doUpdate,
		"onDelete":        wheel.doDelete,
		"onConfigChanged": wheel.doConfigChanged,
	}
	return wheel, nil
}

//Wheel ....
type Wheel struct {
	instances map[string]*serviceInstance
	events    chan *event
	fn        map[string]func(*event)
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
	var srv *Service
	defer func() {
		if srv != nil {
			s.events <- newEvent("onExit", &struct {
				ch  BblWheel_EventsServer
				srv *Service
			}{ch, srv})
		}
		grpclog.Println("Events channel", ch, "exist")
	}()
	for {
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
		srv = ev.Service
		if srv == nil {
			err = fmt.Errorf("Error Event.Service is nil")
			grpclog.Println(err)
			return err
		}
		e := newEvent("onEvent", &struct {
			ch  BblWheel_EventsServer
			srv *Service
		}{ch, srv})
		s.events <- e
		if err := <-e.ctx.err; err != nil {
			grpclog.Println("onEvent.Error", err)
			return err
		}
	}
}
func (s *Wheel) doExit(ev *event) {
	grpclog.Println("doExit", ev)
	obj := ev.ctx.obj.(*struct {
		ch  BblWheel_EventsServer
		srv *Service
	})
	ch := obj.ch
	srv := obj.srv
	if ins, has := s.instances[srv.key()]; has && ins.ch != nil && ins.ch == ch {
		ins.ch = nil
	}
}
func (s *Wheel) doEvent(ev *event) {
	grpclog.Println("doEvent", ev)
	obj := ev.ctx.obj.(*struct {
		ch  BblWheel_EventsServer
		srv *Service
	})
	ch := obj.ch
	srv := obj.srv
	ins, has := s.instances[srv.key()]
	//grpclog.Println("Instances", s.instances)
	if !has {
		ins := serviceInstance{srv: srv, lastActiveTime: time.Now().Unix(), wheel: s, ch: ch}
		err := srvmgt.register(srv)
		if err != nil {
			ev.ctx.err <- err
			return
		}
		s.instances[srv.key()] = &ins
		ev.ctx.err <- nil
		return
	}
	if ins.ch == nil {
		ins.ch = ch
	}
	if ins.ch != ch {
		err := fmt.Errorf("Error Event.Service %s channel exist", srv.key())
		ev.ctx.err <- err
		return
	}
	ins.srv = srv
	ins.lastActiveTime = time.Now().Unix()
	if err := srvmgt.update(srv); err != nil {
		grpclog.Println("srvmgt.update", srv, err)
	}
	ev.ctx.err <- nil
}

func (s *Wheel) onGrant(from string, to string) {
	grpclog.Println("onGrant", from, to)
	s.events <- newEvent("onGrant", &struct {
		from string
		to   string
	}{from, to})
}
func (s *Wheel) doGrant(ev *event) {
	grpclog.Println("doGrant", ev)
	///v1/bblwheel/service/grant/serviceA/testService1 1
	obj := ev.ctx.obj.(*struct {
		from string
		to   string
	})
	from := obj.from
	to := obj.to

	srvs := srvmgt.findServiceList([]string{from})
	for _, ins := range s.instances {
		for _, dep := range ins.srv.DependentServices {
			if dep == from && ins.srv.Name == to {
				grpclog.Println("onGrant", dep, ins.srv.Name)
				go func(ins *serviceInstance) {
					for _, srv := range srvs {
						grpclog.Println("onGrant", ins.srv.Name, ins.srv.ID)
						ins.notify(&Event{Type: Event_DISCOVERY, Service: srv})
					}
				}(ins)
			}
		}
	}
}
func (s *Wheel) onCancel(from string, to string) {
	grpclog.Println("onCancel", from, to)
	s.events <- newEvent("onCancel", &struct {
		from string
		to   string
	}{from, to})
}

func (s *Wheel) doCancel(ev *event) {
	grpclog.Println("doConfigChanged", ev)
	obj := ev.ctx.obj.(*struct {
		from string
		to   string
	})
	from := obj.from
	to := obj.to

	srvs := srvmgt.findServiceList([]string{from})
	for _, ins := range s.instances {
		for _, dep := range ins.srv.DependentServices {
			if dep == from && ins.srv.Name == to {
				grpclog.Println("onGrant", dep, ins.srv.Name)
				go func(ins *serviceInstance) {
					for _, srv := range srvs {
						grpclog.Println("onGrant", ins.srv.Name, ins.srv.ID)
						ins.notify(&Event{Type: Event_DISCOVERY, Service: &Service{ID: srv.ID, Name: srv.Name, Status: Service_UNAUTHORIZE}})
					}
				}(ins)
			}
		}
	}
}

func (s *Wheel) onConfigChanged(key string, item *ConfigEntry) {
	grpclog.Println("onConfigChanged", key, item)
	s.events <- newEvent("onConfigChanged", &struct {
		key  string
		item *ConfigEntry
	}{key, item})
}
func (s *Wheel) doConfigChanged(ev *event) {
	grpclog.Println("doConfigChanged", ev)
	obj := ev.ctx.obj.(*struct {
		key  string
		item *ConfigEntry
	})
	key := obj.key
	item := obj.item
	grpclog.Println("onConfigChanged", key, item)
	for _, ins := range s.instances {
		for _, n := range ins.srv.DependentConfigs {
			if n == key {
				go ins.notify(&Event{Type: Event_CONFIGUPDATE, Item: item})
			}
		}
	}
}
func (s *Wheel) doUpdate(ev *event) {
	grpclog.Println("doUpdate", ev)
	srv := ev.ctx.obj.(*Service)
	if ins, has := s.instances[srv.key()]; has {
		ins.srv = srv
		ins.lastActiveTime = time.Now().Unix()
	} else {
		ins = &serviceInstance{srv: srv, lastActiveTime: time.Now().Unix(), wheel: s}
		s.instances[srv.key()] = ins
	}
	for _, ins := range s.instances {
		for _, n := range ins.srv.DependentServices {
			if n == srv.Name && aumgt.has(srv.Name, ins.srv.Name) {
				go ins.notify(&Event{Type: Event_DISCOVERY, Service: srv})
			}
		}
	}
}
func (s *Wheel) onUpdate(srv *Service) {
	grpclog.Println("onUpdate", srv.key(), Service_Status_name[int32(srv.Status)])
	s.events <- newEvent("onUpdate", srv)

}
func (s *Wheel) onDelete(name, id string) {
	grpclog.Println("onDelete", name+"/"+id)
	s.events <- newEvent("onDelete", &struct {
		name, id string
	}{name, id})
}
func (s *Wheel) doDelete(ev *event) {
	grpclog.Println("doDelete", ev)
	obj := ev.ctx.obj.(*struct {
		name, id string
	})
	name := obj.name
	id := obj.id
	for _, ins := range s.instances {
		for _, n := range ins.srv.DependentServices {
			if n == name {
				go ins.notify(&Event{Type: Event_DISCOVERY, Service: &Service{ID: id, Name: name, Status: Service_OFFLINE}})
			}
		}
	}
	delete(s.instances, name+"/"+id)
}

func (s *Wheel) onKeepAlive() {
	grpclog.Printf("NumGoroutine %d NumCPU %d\n", runtime.NumGoroutine(), runtime.NumCPU())
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
	grpclog.Println("bblwheel server listen at", ListenAddr)
	go s.dowork()
	err = server.Serve(lis)
	if err != nil {
		close(s.events)
	}
	return err
}

func (s *Wheel) dowork() {
	grpclog.Println("Wheel.dowork running")
	defer grpclog.Println("Wheel.dowork exit")
	for ev := range s.events {
		if f, has := s.fn[ev.name]; has {
			f(ev)
		} else {
			grpclog.Println(ev.name, "func not found")
		}
	}
}

type serviceInstance struct {
	srv            *Service
	lastActiveTime int64
	ch             BblWheel_EventsServer
	fsm            *fsm.FSM
	wheel          *Wheel
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

func newEvent(name string, obj interface{}) *event {
	return &event{name: name, ctx: newCtx(obj)}
}

type event struct {
	name string
	ctx  *ctx
}

func newCtx(o interface{}) *ctx {
	return &ctx{o, make(chan error, 1)}
}

type ctx struct {
	obj interface{}
	err chan error
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
