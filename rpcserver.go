package bblwheel

import (
	"flag"
	"fmt"
	"net"

	"golang.org/x/net/context"

	"log"

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

var defrpc = NewRPCServer()

//ListenAndServe ....
func ListenAndServe() error {
	return defrpc.Serve()
}

//NewRPCServer ....
func NewRPCServer() *Server {
	bbl := &Server{
		routerA: map[string]func(*Request, *Response) error{},
		routerB: map[string]func(*Message) (*Message, error){},
	}

	return bbl
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
	var opts []grpc.ServerOption
	server := grpc.NewServer(opts...)
	RegisterRpcServer(server, s)
	log.Println("bblwheel rpc server listen at", RPCListenAddr)
	return server.Serve(lis)
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
	return resp, fmt.Errorf("Path: %s, function not found", req.Path)
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

//HandleCallFunc ....
func HandleCallFunc(path string, h HandleCall) {
	defrpc.routerA[path] = h
}

//HandleMessageFunc ....
func HandleMessageFunc(path string, h HandleMessage) {
	defrpc.routerB[path] = h
}
