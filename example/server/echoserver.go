package main

import (
	"flag"

	"log"

	"github.com/gqf2008/bblwheel"
	"github.com/gqf2008/bblwheel/client"
	"github.com/gqf2008/bblwheel/rpc"
)

var (
	cfile string
)

//编译时设定
//-ldflags "-X main.Version=`date -u +%Y%m%d.%H%M%S` -X main.Author=gqf -X main.Mail=gao.qingfeng@gmail.com"
var (
	//Version ....
	Version string
	//Mail ....
	Mail string
	//Author ....
	Author string
)

func init() {
	flag.StringVar(&cfile, "f", "service.json", "service define file")
}

/**
{
  "ID": "001",
  "Address": "127.0.0.1:7654",
  "DataCenter": "aliyun-huadong1",
  "Node": "cloud-test-001",
  "Weight": 100,
  "Name": "echoserver",
  "Single": false,
  "Status": 1,
  "Endpoints":["127.0.0.1:23790"]
}
**/
func main() {
	flag.Parse()
	provider, err := client.NewServiceProviderFromFile(cfile)
	if err != nil {
		log.Fatalln(err)
	}
	provider.Stats = &bblwheel.Stats{}
	server := rpc.NewFuncServer()

	//stats.
	server.HandleCallFunc("/echo", func(req *rpc.Request, resp *rpc.Response) error {
		resp.Content = req.Content
		provider.Stats.Count++
		//time.Sleep(time.Duration(rand.Int()) * time.Millisecond)
		return nil
	})
	provider.Tags = []string{"Author: " + Author, "Mail: " + Mail, "Version: " + Version}
	provider.Single = false
	config := provider.LookupConfig([]string{provider.Key()})[provider.Key()]
	laddr, has := config.Get("listen.addr")
	if !has {
		laddr = "0.0.0.0:7654"
		config.Put("listen.addr", laddr)

	}
	provider.UpdateConfig(config)
	server.Serve(laddr)
	provider.Online()
	go provider.Register()
	server.Join()
}
