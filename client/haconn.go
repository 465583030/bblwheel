package client

import (
	"log"
	"time"

	"github.com/satori/go.uuid"
	"google.golang.org/grpc"
)

type haconn struct {
	endpoints []string
	opts      []grpc.DialOption
}

func newHaConn(endpoints []string) *haconn {
	return &haconn{endpoints: endpoints, opts: []grpc.DialOption{grpc.WithInsecure(), grpc.WithBackoffMaxDelay(30 * time.Second)}}
}
func (c *haconn) Get() *grpc.ClientConn {
	for {
		conn, err := grpc.Dial(c.endpoints[int(Murmur3(uuid.NewV4().Bytes()))%len(c.endpoints)], c.opts...)
		if err != nil {
			log.Println("grpc.Dial", err)
			continue
		}
		return conn
	}
}
