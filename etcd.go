package bblwheel

import (
	"errors"
	"time"

	v3 "github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/concurrency"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"golang.org/x/net/context"
)

const (
	//OperateTimeout ...
	OperateTimeout = 5 * time.Second
)

//GetString ....
func GetString(key string) (string, error) {
	resp, err := cli.Get(context.TODO(), key, v3.WithLimit(1))
	if err != nil {
		return "", err

	}
	if resp.Count == 0 {
		return "", errors.New(key + " not found")
	}
	return string(resp.Kvs[0].Value), nil
}

//GetStringWithPrfix ....
func GetStringWithPrfix(keyPrefix string) (map[string]string, error) {

	resp, err := GetWithPrfix(keyPrefix)
	if err != nil {
		return nil, err
	}
	if resp.Count == 0 {
		return nil, errors.New(keyPrefix + " not found")
	}
	var vals = map[string]string{}
	for _, kv := range resp.Kvs {
		vals[string(kv.Key)] = string(kv.Value)
	}

	return vals, nil
}

//GetKv ....
func GetKv(key string) (*v3.GetResponse, error) {
	return cli.Get(context.TODO(), key, v3.WithLimit(1))
}

//GetWithPrfix ....
func GetWithPrfix(keyPrefix string) (*v3.GetResponse, error) {
	return cli.Get(context.TODO(), keyPrefix, v3.WithFirstKey()...)
}

//PutKv ....
func PutKv(key, value string) error {
	_, err := cli.Put(context.TODO(), key, value)
	return err
}

//PutKvWithTTL ....
func PutKvWithTTL(key, value string, ttl int64) error {
	lease, err := cli.Grant(context.TODO(), ttl)
	if err != nil {
		return err
	}
	_, err = cli.Put(context.TODO(), key, value, v3.WithLease(lease.ID))
	return err
}

//DelKv ....
func DelKv(key string) error {
	_, err := cli.Delete(context.TODO(), key)
	return err
}

//KeepAlive ....
func KeepAlive(ttl int64) (<-chan *v3.LeaseKeepAliveResponse, error) {
	lease, err := cli.Grant(context.Background(), ttl)
	if err != nil {
		return nil, err
	}
	return cli.KeepAlive(context.Background(), lease.ID)
}

//Locker ....
type Locker struct {
	session *concurrency.Session
	mux     *concurrency.Mutex
}

//NewLocker ....
func NewLocker(lockname string, ttl int) (*Locker, error) {
	s, err := concurrency.NewSession(cli, concurrency.WithTTL(ttl))
	if err != nil {
		return nil, err
	}
	m := concurrency.NewMutex(s, lockname)
	return &Locker{s, m}, nil
}

// //Lock ....
// func Lock(c *v3.Client, lockname string, ttl int) error {
// 	s, err := concurrency.NewSession(c, concurrency.WithTTL(ttl))
// 	if err != nil {
// 		return err
// 	}
// 	m := concurrency.NewMutex(s, lockname)

// 	ctx, cancel := context.WithCancel(context.TODO())

// 	// unlock in case of ordinary shutdown
// 	donec := make(chan struct{})
// 	sigc := make(chan os.Signal, 1)
// 	signal.Notify(sigc, os.Interrupt, os.Kill)
// 	go func() {
// 		<-sigc
// 		cancel()
// 		close(donec)
// 	}()

// 	if err := m.Lock(ctx); err != nil {
// 		return err
// 	}

// 	k, kerr := c.Get(ctx, m.Key())
// 	if kerr != nil {
// 		return kerr
// 	}
// 	if len(k.Kvs) == 0 {
// 		return errors.New("lock lost on init")
// 	}

// 	select {
// 	case <-donec:
// 		return m.Unlock(context.TODO())
// 	case <-s.Done():
// 	}

// 	return errors.New("session expired")
// }

// WaitEvents ....
func WaitEvents(key string, rev int64, evs []mvccpb.Event_EventType, onEvent func(*v3.Event)) error {
	wc := cli.Watch(context.Background(), key, v3.WithRev(rev))
	if wc == nil {
		return errors.New("error no watcher")
	}
	return waitEvents(wc, evs, onEvent)
}

//WaitPrefixEvents ....
func WaitPrefixEvents(prefix string, rev int64, evs []mvccpb.Event_EventType, onEvent func(*v3.Event)) error {
	wc := cli.Watch(context.Background(), prefix, v3.WithPrefix(), v3.WithRev(rev))
	if wc == nil {
		return errors.New("error no watcher")
	}
	return waitEvents(wc, evs, onEvent)
}

func waitEvents(wc v3.WatchChan, evs []mvccpb.Event_EventType, onEvent func(*v3.Event)) error {
	has := func(ev *v3.Event) bool {
		for _, e := range evs {
			if ev.Type == e {
				return true
			}
		}
		return false
	}
	for wresp := range wc {
		for _, ev := range wresp.Events {
			if has(ev) {
				onEvent(ev)
			}
		}
	}
	return nil
}
