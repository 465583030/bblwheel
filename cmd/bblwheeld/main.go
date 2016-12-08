package main

import (
	"flag"
	"log"

	"github.com/coreos/etcd/pkg/netutil"
	"github.com/gqf2008/bblwheel"
)

func init() {
	flag.StringVar(&bblwheel.ListenAddr, "bblwheel.address", bblwheel.ListenAddr, "rpc listen address")
	flag.StringVar(&bblwheel.WorkDir, "workdir", bblwheel.WorkDir, "work directory")
	ip, err := netutil.GetDefaultHost()
	if err != nil {
		ip = "127.0.0.1"
		log.Println(err)
	}
	AdvertisePeerAddr := "http://" + ip + ":2380"
	AdvertiseClientAddr := "http://" + ip + ":2379"
	flag.StringVar(&bblwheel.ListenPeerAddr, "peer.listen.addr", bblwheel.ListenPeerAddr, "peer listen address")
	flag.StringVar(&bblwheel.ListenClientAddr, "client.listen.addr", bblwheel.ListenClientAddr, "client listen address")

	flag.StringVar(&bblwheel.AdvertisePeerAddr, "peer.advertise.addr", AdvertisePeerAddr, "advertise peer address")
	flag.StringVar(&bblwheel.AdvertiseClientAddr, "client.advertise.addr", AdvertiseClientAddr, "advertise client address")
	flag.StringVar(&bblwheel.InitialCluster, "cluster.addr", bblwheel.InitialCluster, "initial cluster")
	flag.StringVar(&bblwheel.ClusterToken, "cluster.token", bblwheel.ClusterToken, "cluster token")
	flag.StringVar(&bblwheel.EtcdName, "cluster.name", bblwheel.EtcdName, "cluster name")
	flag.StringVar(&bblwheel.ClusterState, "cluster.state", bblwheel.ClusterState, "cluster state")
}
func main() {
	flag.Parse()
	bblwheel.StartWheel()
}
