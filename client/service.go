package client

import (
	"io"
	"io/ioutil"
	grpclog "log"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gqf2008/bblwheel"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"

	"fmt"

	"encoding/json"

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
	haconn          *haconn
}

//NewServiceProvider ....
func NewServiceProvider() *ServiceProvider {
	return &ServiceProvider{Service: &bblwheel.Service{}, LastActiveTime: time.Now().Unix()}
}

//NewServiceProviderFromFile ....
func NewServiceProviderFromFile(file string) (*ServiceProvider, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	sp := &ServiceProvider{Service: &bblwheel.Service{}, LastActiveTime: time.Now().Unix()}
	err = json.Unmarshal(b, sp)
	if err != nil {
		return nil, err
	}
	if len(sp.Endpoints) == 0 {
		return nil, fmt.Errorf("endpoints is required")
	}
	if strings.TrimSpace(sp.ID) == "" || strings.TrimSpace(sp.Name) == "" || strings.TrimSpace(sp.DataCenter) == "" || strings.TrimSpace(sp.Node) == "" {
		return nil, fmt.Errorf("service id,name,dc,node is required")
	}
	return sp, nil
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
	if s.close == nil {
		s.close = make(chan struct{})
	}
	if s.haconn == nil {
		s.haconn = newHaConn(s.Endpoints)
	}
	for {
		if s.conn != nil {
			s.conn.Close()
		}
		s.conn = s.haconn.Get()
		return
	}
}

//LookupService ....
func (s *ServiceProvider) LookupService(deps []string) []*bblwheel.Service {
	if s.conn == nil {
		s.reconnect()
	}
	cli := bblwheel.NewBblWheelClient(s.conn)
	res, err := cli.LookupService(context.Background(), &bblwheel.LookupServiceReq{ServiceID: s.ID, ServiceName: s.Name, DependentServices: deps})
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
	if err != nil {
		grpclog.Println("ServiceInstance.LookupConfig", err)
		s.reconnect()
		return s.LookupConfig(deps)
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
				if kv.Service.Stats == nil {
					kv.Service.Stats = &bblwheel.Stats{}
				}
				s.stat(kv.Service.Stats)
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

var _startTime = time.Now()

func (s *ServiceProvider) stat(stats *bblwheel.Stats) {
	v, _ := mem.VirtualMemory()
	stats.LastActiveTime = time.Now().Unix()
	stats.UsedMem = int64(v.Total - v.Free)
	stats.FreeMem = int64(v.Free)
	stats.ServiceID = s.ID
	stats.ServiceName = s.Name
	stats.UpTime = int64(time.Now().Sub(_startTime)) / int64(time.Second)
	stats.Threads = int64(runtime.NumGoroutine())
	stats.Other = map[string]string{}
	i, _ := cpu.Counts(true)
	stats.Other["cpu_num"] = fmt.Sprintf("%v", i)
	f, _ := cpu.Percent(0, false)
	stats.Other["used_cpu"] = fmt.Sprintf("%f%%", f[0])
	avg, _ := load.Avg()
	stats.Other["load1"] = fmt.Sprintf("%f%%", avg.Load1)
	stats.Other["load5"] = fmt.Sprintf("%f%%", avg.Load5)
	stats.Other["load15"] = fmt.Sprintf("%f%%", avg.Load15)
}

//Key ....
func (s *ServiceProvider) Key() string {
	return fmt.Sprintf("%s/%s", s.Name, s.ID)
}
