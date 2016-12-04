package bblwheel

import (
	"strconv"
	"strings"
	"sync"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"google.golang.org/grpc/grpclog"
)

var aumgt *authmgt

type authListener interface {
	onGrant(string, string)
	onCancel(string, string)
}

func startAuthWatcher() {
	aumgt = &authmgt{table: map[string][]string{}}
	go aumgt.watch()
}

type authmgt struct {
	table    map[string][]string
	lock     sync.RWMutex
	once     Once
	observer authListener
}

func (t *authmgt) has(from, to string) bool {
	t.lock.RLock()
	if tt, has := t.table[from]; has {
		for _, v := range tt {
			if v == to {
				t.lock.RUnlock()
				return true
			}
		}
	}

	t.lock.RUnlock()
	return false
}

func (t *authmgt) cancel(from, to string) {
	t.lock.Lock()
	if tt, has := t.table[from]; has {
		var j = 0
		for i, v := range tt {
			if v == to {
				j = i
				break
			}
		}
		tt = append(tt[:j], tt[:j+1]...)
	}
	t.lock.Unlock()
	t.observer.onCancel(from, to)
}

func (t *authmgt) add(from, to string) {
	t.lock.Lock()
	if tt, has := t.table[from]; has {
		tt = append(tt, to)
	} else {
		tt = []string{to}
		t.table[from] = tt
	}
	t.lock.Unlock()
	t.observer.onGrant(from, to)
}
func (t *authmgt) watch() {
	t.once.Do(func() {
		grpclog.Println("load grant table")
		t.table = map[string][]string{}
		resp, err := GetWithPrfix(ServiceGrantPrefix)
		if err != nil {
			grpclog.Fatalln(err)
			return
		}
		for _, kv := range resp.Kvs {
			name := strings.SplitN(strings.TrimPrefix(string(kv.Key), ServiceGrantPrefix+"/"), "/", 2)
			///v1/bblwheel/service/grant/srvA/srvB 1
			///v1/bblwheel/service/grant/srvA/srvC 1
			///v1/bblwheel/service/grant/srvA/srvD 1
			if len(name) != 2 || name[0] == "" || name[1] == "" {
				grpclog.Println("invalid grant info")
				continue
			}
			val, _ := strconv.ParseBool(string(kv.Value))
			if val {
				t.add(name[0], name[1])
			}
		}
		grpclog.Println("load grant table", "ok")
		grpclog.Println("watch grant table")
		err = WaitPrefixEvents(ServiceGrantPrefix+"/",
			//resp.Header.Revision,
			[]mvccpb.Event_EventType{mvccpb.PUT, mvccpb.DELETE},
			func(event *clientv3.Event) {
				name := strings.SplitN(strings.TrimPrefix(string(event.Kv.Key), ServiceGrantPrefix+"/"), "/", 2)
				if len(name) != 2 || name[0] == "" || name[1] == "" {
					grpclog.Println("invalid grant info")
					return
				}
				val, _ := strconv.ParseBool(string(event.Kv.Value))
				if event.Type == mvccpb.PUT {
					if val {
						t.add(name[0], name[1])
					} else {
						t.cancel(name[0], name[1])
					}
				} else if event.Type == mvccpb.DELETE {
					t.cancel(name[0], name[1])
				}
			})
		if err != nil {
			grpclog.Fatalln("watch grant table", err)
		}
	})
}
