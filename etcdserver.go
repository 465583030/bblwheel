package bblwheel

import (
	"log"
	"time"

	"net/url"
	"strings"

	"path"

	"github.com/coreos/etcd/embed"
)

var (
	//ListenPeerAddr ....
	ListenPeerAddr = "http://0.0.0.0:2380"
	//ListenClientAddr ....
	ListenClientAddr = "http://0.0.0.0:2379"
	//AdvertisePeerAddr ....
	AdvertisePeerAddr = "http://127.0.0.1:2380"
	//AdvertiseClientAddr ....
	AdvertiseClientAddr = "http://127.0.0.1:2379"
	//InitialCluster ....
	InitialCluster = ""
	//ClusterToken ....
	ClusterToken = "wheel-cluster-001"
	//ClusterState ....
	ClusterState = "new"
	//EtcdName ....
	EtcdName = "wheel"
)

const (
	//EtcdDirectory ....
	EtcdDirectory = "wheel.etcd"
)

func parseUrls(rawurls string) ([]url.URL, error) {
	rurls := []url.URL{}
	for _, u := range strings.Split(rawurls, ",") {
		urll, err := url.Parse(u)
		if err != nil {
			return nil, err
		}
		rurls = append(rurls, *urll)
	}
	return rurls, nil
}
func startEtcd(ready func() error) error {
	cfg := embed.NewConfig()
	cfg.Name = EtcdName
	lpurls, err := parseUrls(ListenPeerAddr)
	if err != nil {
		return err
	}
	cfg.LPUrls = lpurls
	lcurls, err := parseUrls(ListenClientAddr)
	if err != nil {
		return err
	}
	cfg.LCUrls = lcurls
	apurls, err := parseUrls(AdvertisePeerAddr)
	if err != nil {
		return err
	}
	cfg.APUrls = apurls
	acurls, err := parseUrls(AdvertiseClientAddr)
	if err != nil {
		return err
	}
	cfg.ACUrls = acurls
	//etcd01=http://192.168.12.37:2380,etcd02=http://192.168.12.178:2380,etcd03=http://192.168.12.179:2380
	if InitialCluster == "" {
		cfg.InitialCluster = cfg.InitialClusterFromName(cfg.Name)
	} else {
		cfg.InitialCluster = InitialCluster
		cfg.ClusterState = ClusterState
		cfg.InitialClusterToken = ClusterToken
	}
	log.Println("InitialCluster==========", cfg.InitialCluster)

	cfg.AutoCompactionRetention = 32
	cfg.Dir = path.Join(WorkDir, EtcdDirectory)
	cfg.PeerAutoTLS = false
	cfg.ClientAutoTLS = false
	e, err := embed.StartEtcd(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer e.Close()
	select {
	case <-e.Server.ReadyNotify():
		go ready()
	case <-time.After(60 * time.Second):
		e.Server.Stop()
		log.Println("server took too long to start!")
	}
	return <-e.Err()
}

func parserInitialCluster(name, raw string) string {
	var ret = ""
	for _, u := range strings.Split(raw, ",") {
		ret = ret + "," + name + "=" + u
	}
	return ret[1:]
}
