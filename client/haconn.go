package client

import (
	"google.golang.org/grpc"
)

type connwrapper struct {
	*grpc.ClientConn
	ha *haconn
}

func (c *connwrapper) Close() {

}

type haconn struct {
	endpoints map[string]bool
	clients   []*grpc.ClientConn
}

func (c *haconn) Close() {

}
