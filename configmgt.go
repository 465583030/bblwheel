package bblwheel

import (
	"strings"

	"sync"

	grpclog "log"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
)

var confmgt *configmgt

func startConfigWatcher() {
	confmgt = &configmgt{}
	go confmgt.watch()
}

type configmgt struct {
	once Once
	lock sync.RWMutex
	//listeners map[string]func(*ConfigEntry)
	observer func(string, *ConfigEntry)
}

func (c *configmgt) get(keys []string) map[string]*Config {
	var result = map[string]*Config{}
	for _, key := range keys {
		resp, err := GetWithPrfix(configKey(key) + "/")
		if err != nil {
			grpclog.Println(err)
			break
		}
		items := []*ConfigEntry{}
		for _, kv := range resp.Kvs {
			it := strings.SplitN(strings.TrimPrefix(string(kv.Key), configKey(key)+"/"), "/", 2)
			if len(it) != 2 || it[0] == "" {
				grpclog.Println("invalid config info", string(kv.Key), string(kv.Value))
				continue
			}
			items = append(items, &ConfigEntry{Key: it[0], Value: it[1]})
		}
		result[key] = &Config{Items: items}
	}
	return result
}

func (c *configmgt) put(name, id string, conf *Config) error {
	for _, item := range conf.Items {
		err := PutKv(configKey(name, id, item.Key), item.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *configmgt) watch() {
	c.once.Do(func() {
		grpclog.Println("watch config table")
		err := WaitPrefixEvents(ServiceConfigPrefix+"/",
			//resp.Header.Revision,
			[]mvccpb.Event_EventType{mvccpb.PUT, mvccpb.DELETE},
			func(event *clientv3.Event) {
				key := strings.Split(strings.TrimPrefix(string(event.Kv.Key), ServiceGrantPrefix+"/"), "/")
				///v1/bblwheel/service/config/mysql/db1.uri add
				///v1/bblwheel/service/config/mysql/db2.uri add
				///v1/bblwheel/service/config/srv/001/aaa aaa
				if len(key) < 2 || key[0] == "" || key[1] == "" {
					grpclog.Println("invalid config info", string(event.Kv.Key), string(event.Kv.Value))
					return
				}
				var val = ""
				if event.Type == mvccpb.PUT {
					val = string(event.Kv.Value)
				}
				if c.observer != nil {
					c.observer(strings.Join(key[:len(key)-1], "/"), &ConfigEntry{Key: key[len(key)-1], Value: val})
				}

			})
		if err != nil {
			grpclog.Fatalln("watch config table", err)
		}
	})
}
