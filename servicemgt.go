package bblwheel

import (
	"strings"
	"sync"

	"encoding/json"

	"fmt"

	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"google.golang.org/grpc/grpclog"
)

//DefaultTTL ....
const DefaultTTL = 30

var srvmgt *servicemgt

func startServiceWatcher() {
	srvmgt = &servicemgt{}
	go srvmgt.watch()
	go srvmgt.keepalive()
}

type serviceListener interface {
	onUpdate(*Service)
	onDelete(string, string)
	onKeepAlive()
}
type servicemgt struct {
	once     Once
	lock     sync.RWMutex
	observer serviceListener
}

func (s *servicemgt) setObserver(obs serviceListener) {
	s.observer = obs
}

func (s *servicemgt) register(srv *Service) error {
	if srv.Name == "" {
		return fmt.Errorf("Service.Name is required")
	}
	if srv.ID == "" {
		return fmt.Errorf("Service.ID is required")
	}
	resp, err := GetWithPrfix(registerKey(srv.Name))
	if err != nil {
		return err
	}
	if resp.Count == 0 {
		b, err := json.MarshalIndent(srv, "", "  ")
		if err != nil {
			return err
		}
		return PutKvWithTTL(registerKey(srv.Name, srv.ID), string(b), DefaultTTL)
	}
	var other = Service{}
	err = json.Unmarshal(resp.Kvs[0].Value, &other)
	if err != nil {
		return err
	}
	if other.Single {
		return fmt.Errorf("service %v is single", srv.Name)
	}
	b, err := json.MarshalIndent(srv, "", "  ")
	if err != nil {
		return err
	}
	return PutKvWithTTL(registerKey(srv.Name, srv.ID), string(b), DefaultTTL)
}

func (s *servicemgt) unregister(id, name string) error {
	if err := DelKv(registerKey(name, id)); err != nil {
		return err
	}
	return DelKv(statKey(name, id))
}

func (s *servicemgt) update(srv *Service) error {
	if srv.Name == "" {
		return fmt.Errorf("Service.Name is required")
	}
	if srv.ID == "" {
		return fmt.Errorf("Service.ID is required")
	}
	resp, err := GetKv(registerKey(srv.Name))
	if err != nil {
		return err
	}
	if resp.Count == 0 {
		return fmt.Errorf("Service %s/%s not found", srv.Name, srv.ID)
	}
	var o = Service{}
	err = json.Unmarshal(resp.Kvs[0].Value, &o)
	if err != nil {
		return err
	}
	if o.ID != srv.ID {
		return fmt.Errorf("error Service.ID %s <> %s", o.ID, srv.ID)
	}
	if o.Name != srv.Name {
		return fmt.Errorf("error Service.Name %s <> %s", o.Name, srv.Name)
	}
	if o.Status == Service_ONLINE {
		b, err := json.MarshalIndent(srv, "", "  ")
		if err != nil {
			return err
		}
		return PutKvWithTTL(registerKey(srv.Name, srv.ID), string(b), DefaultTTL)
	}
	srv.Status = o.Status
	b, err := json.MarshalIndent(srv, "", "  ")
	if err != nil {
		return err
	}
	return PutKvWithTTL(registerKey(srv.Name, srv.ID), string(b), DefaultTTL)
}
func (s *servicemgt) findService(name, id string) (*Service, error) {
	resp, err := GetKv(registerKey(name, id))
	if err != nil {
		return nil, err
	}
	if resp.Count == 0 {
		return nil, nil
	}
	var other = Service{}
	err = json.Unmarshal(resp.Kvs[0].Value, &other)
	if err != nil {
		return nil, err
	}
	if other.ID != id {
		return nil, fmt.Errorf("error Service.ID %s <> %s", other.ID, id)
	}
	if other.Name != name {
		return nil, fmt.Errorf("error Service.Name %s <> %s", other.Name, name)
	}
	return &other, nil
}

func (s *servicemgt) findServiceList(names []string) ([]*Service, error) {
	var list = []*Service{}
	for _, name := range names {
		resp, err := GetWithPrfix(registerKey(name))
		if err != nil {
			return list, err
		}
		for _, kv := range resp.Kvs {
			o := Service{}
			err = json.Unmarshal(kv.Value, &o)
			if err != nil {
				return list, err
			}
			if o.Name != name {
				return list, fmt.Errorf("error Service.Name %s <> %s", o.Name, name)
			}
			list = append(list, &o)
		}
	}
	return list, nil
}
func (s *servicemgt) keepalive() {
	for {
		grpclog.Println("keepalive")
		ch, err := KeepAlive(DefaultTTL)
		if err != nil {
			panic(err)
		}
		for _ = range ch {
			//grpclog.Printf("lease %016x keepalived with TTL(%d)\n", resp.ID, resp.TTL)
			if s.observer != nil {
				s.observer.onKeepAlive()
			}
		}
		grpclog.Println("keepalive", "lease expired or revoked.")
		time.Sleep(1 * time.Second)
	}

}
func (s *servicemgt) watch() {
	s.once.Do(func() {
		grpclog.Println("watch register")
		var err error
		err = WaitPrefixEvents(ServiceRegisterPrefix+"/",
			[]mvccpb.Event_EventType{mvccpb.PUT, mvccpb.DELETE},
			func(event *clientv3.Event) {
				key := strings.SplitN(strings.TrimPrefix(string(event.Kv.Key), ServiceRegisterPrefix+"/"), "/", 2)
				///v1/bblwheel/service/register/mysql/001 add
				///v1/bblwheel/service/register/mysql/002 add
				///v1/bblwheel/service/register/srvA/001 aaa
				///v1/bblwheel/service/register/srvA/002 aaa
				if len(key) != 2 || key[0] == "" || key[1] == "" {
					grpclog.Println("invalid service info", string(event.Kv.Key), string(event.Kv.Value))
					return
				}
				if event.Type == mvccpb.PUT {
					if s.observer != nil {
						var other = Service{}
						err = json.Unmarshal(event.Kv.Value, &other)
						if err != nil {
							grpclog.Println(err)
							grpclog.Println(string(event.Kv.Key), string(event.Kv.Value))
							return
						}
						s.observer.onUpdate(&other)
					}
				} else if event.Type == mvccpb.DELETE {
					if s.observer != nil {
						s.observer.onDelete(key[0], key[1])
					}
				}
			})
		if err != nil {
			grpclog.Fatalln("watch register", err)
		}
	})
}
