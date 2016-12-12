package rpc

import (
	"flag"
	"log"
	"net"
	"time"

	"sync"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	//RPCListenAddr ....
	RPCListenAddr = "0.0.0.0:7654"
)

//HandleCall ....
type HandleCall func(*Request, *Response)

//HandleMessage ....
type HandleMessage func(*Message) *Message

var defrpc = NewFuncServer()

//ListenAndServe ....
func ListenAndServe() error {
	return defrpc.Serve(RPCListenAddr)
}

//NewFuncServer ....
func NewFuncServer() *Server {
	bbl := &Server{
		routerA: map[string]func(*Request, *Response){},
		routerB: map[string]func(*Message) *Message{},
		chain: &filterChain{
			befchain:   []Filter{},
			afterchain: []Filter{},
			chain:      []Filter{},
			idx:        0,
		},
	}
	bbl.chain.router = &routerFilter{bbl.routerA}
	return bbl
}

func (r *Request) newResponse() *Response {
	return &Response{
		ClientID:   r.ClientID,
		ID:         r.ID,
		Timestamp:  time.Now().Unix(),
		Status:     200,
		StatusText: "OK",
	}
}

//Server ....
type Server struct {
	server  *grpc.Server
	wg      sync.WaitGroup
	routerA map[string]func(*Request, *Response)
	routerB map[string]func(*Message) *Message
	chain   *filterChain
}

//Serve ....
func (s *Server) Serve(addr string) error {
	if !flag.Parsed() {
		flag.Parse()
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	server := grpc.NewServer(opts...)
	RegisterFuncServiceServer(server, s)
	log.Println("gRPC server listen at", addr)
	s.server = server
	s.wg.Add(1)
	go func() {
		log.Println(server.Serve(lis))
		s.wg.Done()
	}()
	return nil
}

//Join ....
func (s *Server) Join() {
	s.wg.Wait()
}

//Stop ....
func (s *Server) Stop() {
	if s.server != nil {
		s.server.GracefulStop()
	}
}

//Call ....
func (s *Server) Call(ctx context.Context, req *Request) (*Response, error) {
	resp := req.newResponse()
	defer func() {
		if err := recover(); err != nil {
			resp.Status = 500
			resp.StatusText = "500"
			log.Println(err)
		}
	}()
	s.chain.DoFilter(req, resp)
	return resp, nil
}

//Channel ....
func (s *Server) Channel(ch FuncService_ChannelServer) error {
	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
		}
	}()
	for {
		msg, err := ch.Recv()
		if err != nil {
			return err
		}
		if f, found := s.routerB[msg.Path]; found {
			return ch.Send(f(msg))
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

type routerFilter struct {
	routerA map[string]func(*Request, *Response)
}

func (f *routerFilter) DoFilter(req *Request, resp *Response, chain FilterChain) {
	if f, found := f.routerA[req.Path]; found {
		f(req, resp)
	} else {
		resp.Status = 404
		resp.StatusText = "Not Found"
	}
	chain.DoFilter(req, resp)
}

//HandleCallFunc ....
func HandleCallFunc(path string, h HandleCall) {
	defrpc.routerA[path] = h
}

//HandleMessageFunc ....
func HandleMessageFunc(path string, h HandleMessage) {
	defrpc.routerB[path] = h
}
