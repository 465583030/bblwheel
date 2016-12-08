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
	flag.StringVar(&bblwheel.ListenPeerAddr, "peer.address", bblwheel.ListenPeerAddr, "peer listen address")
	flag.StringVar(&bblwheel.ListenClientAddr, "client.address", bblwheel.ListenClientAddr, "client listen address")

	flag.StringVar(&bblwheel.AdvertisePeerAddr, "advertise.peer.address", AdvertisePeerAddr, "advertise peer listen address")
	flag.StringVar(&bblwheel.AdvertiseClientAddr, "advertise.client.address", AdvertiseClientAddr, "advertise client listen address")
}
func main() {
	flag.Parse()
	bblwheel.StartWheel()
}
