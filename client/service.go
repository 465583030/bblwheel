package client

import (
	"io"
	grpclog "log"
	"sync"
	"time"

	"github.com/gqf2008/bblwheel"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func init() {
	grpclog.SetFlags(grpclog.Lshortfile | grpclog.LstdFlags)
}

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
	Endpoints       []string
	conn            *grpc.ClientConn
	close           chan struct{}
	once            *bblwheel.Once
}

//NewServiceInstance ....
func NewServiceInstance() *ServiceInstance {
	return &ServiceInstance{LastActiveTime: time.Now().Unix()}
}

//Disconnect ....
func (s *ServiceInstance) disconnect() {
	s.once.Do(func() {
		if s.conn != nil {
			s.conn.Close()
		}
		if s.close != nil {
			s.close <- struct{}{}
		}
	})
}

func (s *ServiceInstance) reconnect() {
	s.lock.Lock()
	defer s.lock.Unlock()
	for {
		if s.conn != nil {
			s.conn.Close()
		}
		var opts = []grpc.DialOption{grpc.WithInsecure()}
		conn, err := grpc.Dial(s.Endpoints[0], opts...)
		if err != nil {
			grpclog.Println("grpc.Dial", err)
			time.Sleep(3 * time.Second)
			continue
		}
		s.conn = conn
		return
	}
}

//LookupService ....
func (s *ServiceInstance) LookupService(deps []string) []*bblwheel.Service {
	if s.conn == nil {
		s.reconnect()
	}
	cli := bblwheel.NewBblWheelClient(s.conn)
	res, err := cli.LookupService(context.Background(), &bblwheel.LookupServiceReq{DependentServices: deps})
	if err == io.EOF {
		grpclog.Println("ServiceInstance.LookupService", err)
		s.reconnect()
		return s.LookupService(deps)
	}
	if err != nil {
		grpclog.Println(err)
		return []*bblwheel.Service{}
	}
	return res.Services
}

//LookupConfig ....
func (s *ServiceInstance) LookupConfig(deps []string) map[string]*bblwheel.Config {
	if s.conn == nil {
		s.reconnect()
	}
	cli := bblwheel.NewBblWheelClient(s.conn)
	res, err := cli.LookupConfig(context.Background(), &bblwheel.LookupConfigReq{DependentConfigs: deps})
	if err == io.EOF {
		grpclog.Println("ServiceInstance.LookupConfig", err)
		s.reconnect()
		return s.LookupConfig(deps)
	}
	if err != nil {
		grpclog.Println(err)
		return map[string]*bblwheel.Config{}
	}
	return res.Configs
}

//UpdateConfig ....
func (s *ServiceInstance) UpdateConfig(conf *bblwheel.Config) {
	if s.conn == nil {
		s.reconnect()
	}
	cli := bblwheel.NewBblWheelClient(s.conn)
	_, err := cli.UpdateConfig(context.Background(), &bblwheel.UpdateConfigReq{ServiceID: s.ID, ServiceName: s.Name, Config: conf})
	if err == io.EOF {
		grpclog.Println("ServiceInstance.UpdateConfig", err)
		s.reconnect()
		s.UpdateConfig(conf)
		return
	}
	if err != nil {
		grpclog.Println(err)
	}
}

//Register ....
func (s *ServiceInstance) Register() *bblwheel.RegisterResult {
	if s.conn == nil {
		s.reconnect()
	}
	cli := bblwheel.NewBblWheelClient(s.conn)
	res, err := cli.Register(context.Background(), s.Service)
	if err == io.EOF || err == grpc.ErrClientConnClosing || err == grpc.ErrClientConnTimeout {
		grpclog.Println("ServiceInstance.Register", err)
		s.reconnect()
		return s.Register()
	}
	if err != nil {
		grpclog.Println("ServiceInstance.Register", err)
		res = &bblwheel.RegisterResult{}
		res.Desc = err.Error()
		return res
	}
	s.close = make(chan struct{}, 1)
	s.once = &bblwheel.Once{}
	go s.keepAlive()
	return res
}

//Online ....
func (s *ServiceInstance) Online() {
	s.Status = bblwheel.Service_ONLINE
}

//Unregister ....
func (s *ServiceInstance) Unregister() {
	if s.once == nil {
		return
	}
	s.once.Do(func() {
		cli := bblwheel.NewBblWheelClient(s.conn)
		_, err := cli.Unregister(context.Background(), s.Service)
		if err != nil {
			grpclog.Println(err)
		}
		s.disconnect()
	})
}

func (s *ServiceInstance) keepAlive() {
	kv := bblwheel.Event{Type: bblwheel.Event_KEEPALIVE}
	for {
		select {
		case <-s.close:
			return
		default:
			break
		}
		cli := bblwheel.NewBblWheelClient(s.conn)
		ch, err := cli.Events(context.Background())
		if err != nil {
			grpclog.Println("ServiceInstance.keepAlive", err)
			s.reconnect()
			continue
		}

		s.lock.RLock()
		kv.Service = s.Service
		s.lock.RUnlock()
		err = ch.Send(&kv)
		if err == io.EOF || err == grpc.ErrClientConnClosing || err == grpc.ErrClientConnTimeout {
			grpclog.Println(err)
			s.reconnect()
			continue
		}
		if err != nil {
			grpclog.Println(err)
			time.Sleep(3 * time.Second)
			continue
		}
		ticker := time.NewTicker((bblwheel.DefaultTTL - 10) * time.Second)
		go func(s *ServiceInstance, ch bblwheel.BblWheel_EventsClient) {
			grpclog.Println("ServiceInstance", s.Name+"/"+s.ID, "ticker running")
			defer grpclog.Println("ServiceInstance", s.Name+"/"+s.ID, "ticker stopped")

			for t := range ticker.C {
				grpclog.Println("ServiceInstance", s.Name+"/"+s.ID, "ticker", t)
				s.lock.RLock()
				kv.Service = s.Service
				s.lock.RUnlock()
				if err := ch.Send(&kv); err != nil {
					grpclog.Println(err)
					break
				}
			}
		}(s, ch)
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
	}

}
