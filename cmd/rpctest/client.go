package main

import (
	"log"

	"context"

	"flag"

	"time"

	"sync"

	"github.com/gqf2008/bblwheel/rpc"
	"google.golang.org/grpc"
)

var endpoint string
var concurrency int
var count int
var t int

func init() {
	flag.StringVar(&endpoint, "rpc.addr", "127.0.0.1:7654", "rpc server address")
	flag.IntVar(&concurrency, "c", 1, "并发数")
	flag.IntVar(&count, "count", 1000, "每个并发请求数")
	flag.IntVar(&t, "t", 60, "测试时间，单位秒")
}
func main1() {
	flag.Parse()
	var wg sync.WaitGroup
	bt := time.Now()
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go doRequest(&wg, count)
	}
	wg.Wait()
	log.Println("concurrency", concurrency, "count", count, "cost", "total request", concurrency*count, time.Now().Unix()-bt.Unix(), "s")
}

func doRequest(wg *sync.WaitGroup, c int) {
	defer wg.Done()
	conn, err := grpc.Dial(endpoint, []grpc.DialOption{grpc.WithInsecure(), grpc.WithTimeout(30 * time.Second)}...)
	if err != nil {
		log.Fatalln(err)
	}
	cli := rpc.NewRpcClient(conn)
	for i := 0; i < c; i++ {
		_, err := cli.Call(context.Background(), &rpc.Request{ID: int64(i), ClientID: "aaaa", Path: "/echo"})
		if err != nil {
			log.Fatalln(err)
		}
	}
}
