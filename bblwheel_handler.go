package bblwheel

import (
	"flag"
	"log"
	"net"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	v3 "github.com/coreos/etcd/clientv3"
	"golang.org/x/net/context"
)

var (
	//ListenAddr ....
	ListenAddr = "0.0.0.0:23790"
	//WorkDir ....
	WorkDir = "/var/lib/bblwheel"

	cli *v3.Client
)

func init() {
	flag.StringVar(&ListenAddr, "bblwheel.address", ListenAddr, "rpc listen address")
	flag.StringVar(&WorkDir, "workdir", WorkDir, "work directory")
}

//StartWheel ....
func StartWheel() error {
	srv, err := newWheel()
	if err != nil {
		return err
	}
	return srv.Serve()
}

func newWheel() (*Wheel, error) {
	return &Wheel{}, nil
}

//Wheel ....
type Wheel struct {
}

//Register ....
func (s *Wheel) Register(ctx context.Context, srv *Service) (*RegisterResult, error) {
	return nil, nil
}

//Unregister ....
func (s *Wheel) Unregister(context.Context, *Service) (*Void, error) {
	return &Void{}, nil
}

//UpdateStatus ....
func (s *Wheel) UpdateStatus(context.Context, *Service) (*Void, error) {
	return &Void{}, nil
}

//UpdateConfig ....
func (s *Wheel) UpdateConfig(context.Context, *UpdateConfigReq) (*Void, error) {
	return &Void{}, nil
}

//KeepAlive ....
func (s *Wheel) KeepAlive(ch BblWheel_KeepAliveServer) error {
	defer func() {
		if err := recover(); err != nil {
			grpclog.Println(err)
		}
	}()
	for {
		event, err := ch.Recv()
		if err != nil {
			return err
		}
		log.Println(event)
		if event.Type == Event_STATISTICS && event.Stat != nil {

		}
	}
}

//Serve ....
func (s *Wheel) Serve() error {
	if !flag.Parsed() {
		flag.Parse()
	}
	lis, err := net.Listen("tcp", ListenAddr)
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
		RegisterBblWheelServer(server, s)
		log.Println("bblwheel server listen at", RPCListenAddr)
		err = server.Serve(lis)
		if err != nil {
			panic(err)
		}
	})
}
