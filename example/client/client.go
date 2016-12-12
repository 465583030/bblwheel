package main

import (
	"context"
	"flag"
	"log"
	"sync"
	"time"

	"github.com/gqf2008/bblwheel/client"
	"github.com/gqf2008/bblwheel/rpc"
)

var endpoint string
var concurrency int
var count int
var t int
var b int

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

/**
service define
{
  "ID": "000",
  "DataCenter": "aliyun-huadong1",
  "Node": "cloud-test-001",
  "Name": "echoclient",
  "Status": 1,
  "Endpoints":["127.0.0.1:23790"],
  "DependentServices":["echoserver"]
}
**/
func init() {
	flag.StringVar(&cfile, "f", "service.json", "service define file")
	flag.IntVar(&concurrency, "c", 1, "并发数")
	flag.IntVar(&count, "count", 1000, "每个并发请求数")
	flag.IntVar(&t, "t", 60, "测试时间，单位秒")
	flag.IntVar(&b, "b", 1024, "块大小")
}
func main() {
	flag.Parse()
	provider, err := client.NewServiceProviderFromFile(cfile)
	if err != nil {
		log.Fatalln(err)
	}
	provider.Tags = []string{"Author: " + Author, "Mail: " + Mail, "Version: " + Version}
	srvs := provider.LookupService(provider.DependentServices)
	if len(srvs) == 0 {

		log.Fatalln(provider.DependentServices, "offline or not authorize")
	}
	cli, err := rpc.NewClient(srvs[0].Address)
	if err != nil {
		log.Fatalln(err)
	}
	var wg sync.WaitGroup
	bt := time.Now()
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go doRequest(&wg, cli, count)
	}
	go provider.Register()
	wg.Wait()
	log.Println("concurrency", concurrency, "count", count, "cost", "total request", concurrency*count, time.Now().Unix()-bt.Unix(), "s")
}

func doRequest(wg *sync.WaitGroup, cli *rpc.FuncClient, c int) {
	defer wg.Done()
	block := make([]byte, b)
	for i := 0; i < c; i++ {
		_, err := cli.Call(context.Background(), &rpc.Request{ID: int64(i), ClientID: "aaaa", Path: "/echo", Content: block})
		if err != nil {
			log.Fatalln(err)
		}
	}
}
