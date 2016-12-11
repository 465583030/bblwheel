package rpc

import (
	"log"
	"sync"
	"time"

	"context"

	"github.com/gqf2008/bblwheel"
	uuid "github.com/satori/go.uuid"
	"google.golang.org/grpc"
)

var clients = map[string]Client{}

//RegisterClient ....
func RegisterClient(driver string, c Client) {
	clients[driver] = c
}

//NewClient ....
func NewClient(driver string, endpoints []string) Client {
	if c, has := clients[driver]; has {
		c = c.Clone()
		for _, endpoint := range endpoints {
			c.AddEndpoint(endpoint)
		}
		return c
	}
	return &hashClient{endpoints: endpoints, opts: []grpc.DialOption{grpc.WithInsecure(), grpc.WithBackoffMaxDelay(30 * time.Second)}}
}

//Client ....
type Client interface {
	Clone() Client
	AddEndpoint(endpoint string)
	RemoveEndpoint(endpoint string)
	Call(context.Context, *Request) (*Response, error)
	Send(*Message) error
	Recv() (*Message, error)
}

type hashClient struct {
	endpoints []string
	opts      []grpc.DialOption
	lock      sync.RWMutex
	con       *grpc.ClientConn
	ch        FuncService_ChannelClient
}

func (c *hashClient) Clone() Client {
	return &hashClient{opts: []grpc.DialOption{grpc.WithInsecure(), grpc.WithBackoffMaxDelay(30 * time.Second)}}
}

func (c *hashClient) AddEndpoint(endpoint string) {
	c.lock.RLock()
	c.endpoints = append(c.endpoints, endpoint)
	c.lock.RUnlock()
}
func (c *hashClient) RemoveEndpoint(endpoint string) {
	c.lock.Lock()
	for i, end := range c.endpoints {
		if end == endpoint {
			c.endpoints = append(c.endpoints[:i], c.endpoints[i+1:]...)
			c.lock.Unlock()
			return
		}
	}
	c.lock.Unlock()
}

func (c *hashClient) connect() *grpc.ClientConn {
	for {
		c.lock.RLock()
		conn, err := grpc.Dial(c.endpoints[int(bblwheel.Murmur3(uuid.NewV4().Bytes()))%len(c.endpoints)], c.opts...)
		c.lock.RUnlock()
		if err != nil {
			log.Println("grpc.Dial", err)
			continue
		}
		return conn
	}
}
func (c *hashClient) Call(ctx context.Context, req *Request) (*Response, error) {
	if c.con == nil {
		c.con = c.connect()
	}
	cli := NewFuncServiceClient(c.con)
	return cli.Call(ctx, req)
}
func (c *hashClient) Send(msg *Message) error {
	if c.con == nil {
		c.con = c.connect()
		cli := NewFuncServiceClient(c.con)
		ch, err := cli.Channel(context.Background())
		if err != nil {
			return err
		}
		c.ch = ch
	}
	return c.ch.Send(msg)
}

func (c *hashClient) Recv() (*Message, error) {
	if c.con == nil {
		c.con = c.connect()
		cli := NewFuncServiceClient(c.con)
		ch, err := cli.Channel(context.Background())
		if err != nil {
			return nil, err
		}
		c.ch = ch
	}
	return c.ch.Recv()
}
