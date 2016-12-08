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

//ServiceProvider ....
type ServiceProvider struct {
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

//NewServiceProvider ....
func NewServiceProvider() *ServiceProvider {
	return &ServiceProvider{LastActiveTime: time.Now().Unix()}
}

//Disconnect ....
func (s *ServiceProvider) disconnect() {
	if s.conn != nil {
		s.conn.Close()
	}
}

func (s *ServiceProvider) reconnect() {
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
func (s *ServiceProvider) LookupService(deps []string) []*bblwheel.Service {
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
func (s *ServiceProvider) LookupConfig(deps []string) map[string]*bblwheel.Config {
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
func (s *ServiceProvider) UpdateConfig(conf *bblwheel.Config) {
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

//Online ....
func (s *ServiceProvider) Online() {
	s.lock.Lock()
	s.Status = bblwheel.Service_ONLINE
	s.lock.Unlock()
}

//Update ....
func (s *ServiceProvider) Update(srv *ServiceProvider) {
	s.lock.Lock()
	s.Service = srv.Service
	s.lock.Unlock()
}

//Unregister ....
func (s *ServiceProvider) Unregister() {
	if s.close != nil {
		close(s.close)
	}
	s.disconnect()
}

//Register ....
func (s *ServiceProvider) Register() {
	defer grpclog.Println("ServiceProvider.KeepAlive", s.Name+"/"+s.ID, "exit")
	if s.conn == nil {
		s.reconnect()
	}
	if s.close == nil {
		s.close = make(chan struct{})
	}
	kv := bblwheel.Event{Type: bblwheel.Event_KEEPALIVE}
	for {
		select {
		case <-s.close:
			return
		default:
		}
		cli := bblwheel.NewBblWheelClient(s.conn)
		ch, err := cli.Events(context.Background())
		if err != nil {
			grpclog.Println("ServiceProvider.keepAlive", err)
			s.reconnect()
			continue
		}
		s.lock.RLock()
		kv.Service = s.Service
		s.lock.RUnlock()
		err = ch.Send(&kv)
		if err != nil {
			grpclog.Println(err)
			time.Sleep(3 * time.Second)
			s.reconnect()
			continue
		}
		ticker := time.NewTicker((bblwheel.DefaultTTL - 10) * time.Second)
		go func(s *ServiceProvider, ch bblwheel.BblWheel_EventsClient) {
			grpclog.Println("ServiceProvider", s.Name+"/"+s.ID, "ticker running")
			defer grpclog.Println("ServiceProvider", s.Name+"/"+s.ID, "ticker stopped")

			for t := range ticker.C {
				grpclog.Println("ServiceProvider", s.Name+"/"+s.ID, "ticker", t)
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
				ticker.Stop()
				grpclog.Println(err)
				time.Sleep(3 * time.Second)
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
