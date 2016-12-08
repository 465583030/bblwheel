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

###集群&启动
./bblwheeld \
	#设置客户端监听地址1
	-client.listen.addr=http://10.27.123.131:2379,http://127.0.0.1:2379 \
	#设置peer监听地址
	-peer.listen.addr=http://10.27.123.131:2380 \
	#设置客户端可以被外部访问的地址
	-client.advertise.addr=http://10.27.123.131:2379 \
	#设置peer可以被外部访问的地址
	-peer.advertise.addr=http://10.27.123.131:2380 \
	#设置集群地址，格式：$CLUSTER_NAME=$PEER_ADVERTISE_ADDR；多个地址用“,”隔开，注意：$CLUSTER_NAME不能相同
	-cluster.addr=wheel0=http://10.27.123.131:2380,wheel1=http://10.27.123.161:2380 \
	#设置本节点在集群中唯一的名字
	-cluster.name=wheel0 \
	#设置集群状态，new | existing
	-cluster.state=new \
	#设置集群之间访问令牌，一个集群令牌必须一致
	-cluster.token=wheel-cluster-001 \
	#设置工作目录
	-workdir=/home/gqf/build

###客户端
- [java客户端和例子](https://github.com/gqf2008/bblwheel-java)
- [go客户端和例子](https://github.com/gqf2008/bblwheel/tree/master/client)
- [bblagent](https://github.com/gqf2008/bblwheel/tree/master/cmd/bblagent)
