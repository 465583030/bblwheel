# bblwheel
一个基于etcd、grpc的服务治理中心，具备简单服务注册、发现、配置、授权、rpc框架等功能，api还没有稳定，请不要用于生产环境

###编译&运行
[Golang 1.6+](https://golang.org/dl/)
	
	go get -u github.com/gqf2008/bblwheel
	go build -o bblwheeld  github.com/gqf2008/bblwheel/cmd/bblwheeld
	./bblwheeld -h
	Usage of ./bblwheeld:
		-advertise.client.address string
    		advertise client listen address (default "http://127.0.0.1:2379")
    	-advertise.peer.address string
    		advertise peer listen address (default "http://127.0.0.1:2380")
    	-bblwheel.address string
    		rpc listen address (default "0.0.0.0:23790")
    	-client.address string
    		client listen address (default "http://0.0.0.0:2379")
    	-peer.address string
    		peer listen address (default "http://0.0.0.0:2380")
    	-rpc.address string
    		rpc listen address (default "0.0.0.0:7654")
    	-workdir string
    		work directory (default "/var/lib/bblwheel")


###客户端
- [java客户端和例子](https://github.com/gqf2008/bblwheel-java)
- [go客户端和例子](https://github.com/gqf2008/bblwheel/tree/master/client)
- [bblagent](https://github.com/gqf2008/bblwheel/tree/master/cmd/bblagent)
