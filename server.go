package bblwheel

import (
	"flag"
	"fmt"
	"net"

	"golang.org/x/net/context"

	"strings"

	"log"

	v3 "github.com/coreos/etcd/clientv3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

var (
	//RPCListenAddr ....
	RPCListenAddr = "0.0.0.0:7654"
)

func init() {
	flag.StringVar(&RPCListenAddr, "rpc.address", RPCListenAddr, "rpc listen address")
}

//HandleCall ....
type HandleCall func(*Request, *Response) error

//HandleMessage ....
type HandleMessage func(*Message) (*Message, error)

//ListenAndServe ....
func ListenAndServe() error {
	srv, err := NewServer()
	if err != nil {
		return err
	}
	return srv.Serve()
}

//NewServer ....
func NewServer() (*Server, error) {
	bbl := &Server{
		routerA: map[string]func(*Request, *Response) error{},
		routerB: map[string]func(*Message) (*Message, error){},
	}

	return bbl, nil
}

func (r *Request) newResponse() *Response {
	return &Response{
		ClientID:   r.ClintID,
		ID:         r.ID,
		Status:     200,
		StatusText: "OK",
	}
}

//Server ....
type Server struct {
	routerA map[string]func(*Request, *Response) error
	routerB map[string]func(*Message) (*Message, error)
}

//Serve ....
func (s *Server) Serve() error {
	if !flag.Parsed() {
		flag.Parse()
	}
	lis, err := net.Listen("tcp", RPCListenAddr)
	if err != nil {
		grpclog.Fatalf("failed to listen: %v", err)
	}
	return startEtcd(func() {
		client, err := v3.New(v3.Config{
			Endpoints:   strings.Split(ListenClientAddr, ","),
			DialTimeout: OperateTimeout,
		})
		if err != nil {
			panic(err)
		}
		cli = client
		var opts []grpc.ServerOption
		server := grpc.NewServer(opts...)
		RegisterRpcServer(server, s)
		log.Println("bblwheel server listen at", RPCListenAddr)
		err = server.Serve(lis)
		if err != nil {
			panic(err)
		}
	})
}

//Call ....
func (s *Server) Call(ctx context.Context, req *Request) (*Response, error) {
	resp := req.newResponse()
	defer func() {
		if err := recover(); err != nil {
			resp.Status = 500
			resp.StatusText = "500"
			grpclog.Println(err)
		}
	}()
	if f, found := s.routerA[req.Path]; found {
		if err := f(req, resp); err != nil {
			return nil, err
		}
	}
	return resp, fmt.Errorf("Path: %s, 处理函数不存在", req.Path)
}

//Channel ....
func (s *Server) Channel(ch Rpc_ChannelServer) error {
	defer func() {
		if err := recover(); err != nil {
			grpclog.Println(err)
		}
	}()
	for {
		msg, err := ch.Recv()
		if err != nil {
			return err
		}
		if f, found := s.routerB[msg.Path]; found {
			if m, err := f(msg); err == nil && m != nil {
				return ch.Send(m)
			}
		}
	}
}

//HandleCallFunc ....
func (s *Server) HandleCallFunc(path string, h HandleCall) {
	s.routerA[path] = h
}

//HandleMessageFunc ....
func (s *Server) HandleMessageFunc(path string, h HandleMessage) {
	s.routerB[path] = h
}
