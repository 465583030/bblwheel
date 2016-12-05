package client

import (
	"sync"

	"time"

	"github.com/gqf2008/bblwheel"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

//OnDiscoveryFunc ....
type OnDiscoveryFunc func(*bblwheel.Service)

//OnConfigUpdatedFunc ....
type OnConfigUpdatedFunc func(string, string)

//OnControlFunc ....
type OnControlFunc func(string)

//OnExecFunc ....
type OnExecFunc func(string)

//ServiceInstance ....
type ServiceInstance struct {
	*bblwheel.Service
	lock            sync.RWMutex
	LastActiveTime  int64
	OnDiscovery     OnDiscoveryFunc
	OnConfigUpdated OnConfigUpdatedFunc
	OnControl       OnControlFunc
	OnExec          OnExecFunc
	endpoints       []string
	conn            *grpc.ClientConn
}

//NewServiceInstance ....
func NewServiceInstance(endpoints []string) *ServiceInstance {
	ins := &ServiceInstance{LastActiveTime: time.Now().Unix(), endpoints: endpoints}
	ins.reconnect()
	return ins
}

func (s *ServiceInstance) reconnect() {
	s.lock.Lock()
	defer s.lock.Unlock()
	for {
		if s.conn != nil {
			s.conn.Close()
		}
		conn, err := grpc.Dial(s.endpoints[0], grpc.WithTimeout(30*time.Second))
		if err != nil {
			grpclog.Println("grpc.Dial", err)
			time.Sleep(3 * time.Second)
			continue
		}
		s.conn = conn
	}
}

//LookupService ....
func (s *ServiceInstance) LookupService(deps []string) []*bblwheel.Service {
	cli := bblwheel.NewBblWheelClient(s.conn)
	res, err := cli.LookupService(context.Background(), &bblwheel.LookupServiceReq{DependentServices: deps})
	if err != nil {
		grpclog.Println(err)
		s.reconnect()
		return s.LookupService(deps)
	}
	return res.Services
}

//LookupConfig ....
func (s *ServiceInstance) LookupConfig(deps []string) map[string]*bblwheel.Config {
	cli := bblwheel.NewBblWheelClient(s.conn)
	res, err := cli.LookupConfig(context.Background(), &bblwheel.LookupConfigReq{DependentConfigs: deps})
	if err != nil {
		grpclog.Println(err)
		s.reconnect()
		return s.LookupConfig(deps)
	}
	return res.Configs
}

//UpdateConfig ....
func (s *ServiceInstance) UpdateConfig(conf *bblwheel.Config) {
	cli := bblwheel.NewBblWheelClient(s.conn)
	_, err := cli.UpdateConfig(context.Background(), &bblwheel.UpdateConfigReq{ServiceID: s.ID, ServiceName: s.Name, Config: conf})
	if err != nil {
		grpclog.Println(err)
		s.reconnect()
		s.UpdateConfig(conf)
	}
}

//Register ....
func (s *ServiceInstance) Register() *bblwheel.RegisterResult {
	cli := bblwheel.NewBblWheelClient(s.conn)
	res, err := cli.Register(context.Background(), s.Service)
	if err != nil {
		grpclog.Println(err)
		s.reconnect()
		return s.Register()
	}
	go s.keepAlive()
	return res
}

//Online ....
func (s *ServiceInstance) Online() {
	s.Status = bblwheel.Service_ONLINE
}

//Unregister ....
func (s *ServiceInstance) Unregister() {
	cli := bblwheel.NewBblWheelClient(s.conn)
	_, err := cli.Unregister(context.Background(), s.Service)
	if err != nil {
		grpclog.Println(err)
	}
}

func (s *ServiceInstance) keepAlive() {

	for {
		cli := bblwheel.NewBblWheelClient(s.conn)
		ch, err := cli.Events(context.Background())
		if err != nil {
			grpclog.Println(err)
			s.reconnect()
			continue
		}
		ticker := time.NewTicker((bblwheel.DefaultTTL - 5) * time.Second)
		for {
			ev, err := ch.Recv()
			if err != nil {
				grpclog.Println(err)
				ticker.Stop()
				break
			}
			switch ev.Type {
			case bblwheel.Event_DISCOVERY:
				if s.OnDiscovery != nil && ev.Service != nil {
					s.OnDiscovery(ev.Service)
				}
			case bblwheel.Event_CONFIGUPDATE:
				if s.OnConfigUpdated != nil && ev.Item != nil {
					s.OnConfigUpdated(ev.Item.Key, ev.Item.Value)
				}
			case bblwheel.Event_CONTROL:
				if s.OnControl != nil {
					s.OnControl(ev.Command)
				}
			case bblwheel.Event_EXEC:
				if s.OnExec != nil {
					s.OnExec(ev.Command)
				}
			}
		}

		go func(s *ServiceInstance, ch bblwheel.BblWheel_EventsClient) {
			grpclog.Println("ServiceInstance", s.Name+"/"+s.ID, "ticker running")
			defer grpclog.Println("ServiceInstance", s.Name+"/"+s.ID, "ticker stopped")
			kv := bblwheel.Event{Type: bblwheel.Event_KEEPALIVE}
			for _ = range ticker.C {
				s.lock.RLock()
				kv.Service = s.Service
				s.lock.RUnlock()
				if err := ch.Send(&kv); err != nil {
					grpclog.Println(err)
					break
				}
			}
		}(s, ch)
	}

}
