package bblwheel

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
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

func (s *servicemgt) register(srv *Service) error {
	//log.Println("servicemgt.register", srv)
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
	if other.Single && other.Name == srv.Name {
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
	//log.Println("servicemgt.update", srv)
	if srv.Name == "" {
		return fmt.Errorf("Service.Name is required")
	}
	if srv.ID == "" {
		return fmt.Errorf("Service.ID is required")
	}
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

func (s *servicemgt) findServiceList(names []string) []*Service {
	var list = []*Service{}
	for _, name := range names {
		resp, err := GetWithPrfix(registerKey(name))
		if err != nil {
			log.Println(err)
			break
		}
		for _, kv := range resp.Kvs {
			o := Service{}
			err = json.Unmarshal(kv.Value, &o)
			if err != nil {
				log.Println(err)
				continue
			}
			if o.Name != name {
				log.Println(fmt.Errorf("error Service.Name %s <> %s", o.Name, name))
				continue
			}
			list = append(list, &o)
		}
	}
	return list
}
func (s *servicemgt) findAllService() []*Service {
	var list = []*Service{}
	resp, err := GetWithPrfix(registerKey("/"))
	if err != nil {
		log.Println(err)
		return list
	}
	if resp.Count != int64(len(resp.Kvs)) {
		panic(fmt.Sprintf("resp.Count %d resp.Kvs %d resp.More %v\n", resp.Count, len(resp.Kvs), resp.More))
	}

	for _, kv := range resp.Kvs {
		o := Service{}
		err = json.Unmarshal(kv.Value, &o)
		if err != nil {
			log.Println(err)
			continue
		}
		list = append(list, &o)
	}
	return list
}
func (s *servicemgt) keepalive() {
	for {
		log.Println("keepalive")
		ch, err := KeepAlive(DefaultTTL)
		if err != nil {
			log.Println(err)
		}
		for _ = range ch {
			if s.observer != nil {
				s.observer.onKeepAlive()
			}
		}
		log.Println("keepalive", "lease expired or revoked.")
		time.Sleep(1 * time.Second)
	}

}
func (s *servicemgt) watch() {
	s.once.Do(func() {
		log.Println("watch register")
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
					log.Println("invalid service info", string(event.Kv.Key), string(event.Kv.Value))
					return
				}
				if event.Type == mvccpb.PUT {
					if s.observer != nil {
						var other = Service{}
						err = json.Unmarshal(event.Kv.Value, &other)
						if err != nil {
							log.Println(err)
							log.Println(string(event.Kv.Key), string(event.Kv.Value))
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
			log.Fatalln("watch register", err)
		}
	})
}
