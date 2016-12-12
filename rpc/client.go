package rpc

import (
	"log"
	"sync"
	"time"

	"context"

	"google.golang.org/grpc"
)

//NewClient ....
func NewClient(endpoint string) (*FuncClient, error) {
	c := &FuncClient{
		endpoint: endpoint,
		opts:     []grpc.DialOption{grpc.WithInsecure(), grpc.WithBackoffMaxDelay(30 * time.Second)},
	}
	err := c.connect()
	if err != nil {
		return nil, err
	}
	return c, nil
}

//FuncClient 经过测试，单连接和多连接性能差距不大，为了保存实现简单改用单连接方式，不实现连接池
type FuncClient struct {
	endpoint string
	opts     []grpc.DialOption
	lock     sync.RWMutex
	conn     *grpc.ClientConn
}

func (c *FuncClient) connect() error {
	conn, err := grpc.Dial(c.endpoint, c.opts...)
	if err != nil {
		log.Println("grpc.Dial", err)
		return err
	}
	c.conn = conn
	return nil
}

//Close ....
func (c *FuncClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

//Call ....
func (c *FuncClient) Call(ctx context.Context, req *Request) (*Response, error) {
	cli := NewFuncServiceClient(c.conn)
	resp, err := cli.Call(ctx, req)
	if err != nil {
		c.Close()
		return resp, err
	}
	return resp, nil
}

//Channel ....
func (c *FuncClient) Channel(ctx context.Context) (FuncService_ChannelClient, error) {
	ch, err := NewFuncServiceClient(c.conn).Channel(ctx)
	if err != nil {
		c.Close()
		return nil, err
	}
	return ch, nil
}
