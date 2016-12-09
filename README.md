# bblwheel
一个基于etcd、grpc的服务治理中心，具备简单服务注册、发现、配置、授权、rpc框架等功能

###编译
[Golang 1.6+](https://golang.org/dl/)
	
	go get -u github.com/gqf2008/bblwheel
	go build -o bblwheeld  github.com/gqf2008/bblwheel/cmd/bblwheeld

###启动集群
**按照raft设计原理，故障节点数不能大于49.99%，所以节点数不能少于3个，3个节点可以忍受1个节点故障，节点数建议为奇数**
	
	./bblwheeld \
		#设置服务器对外服务端口
	    -bblwheel.listen.addr=0.0.0.0:23790
		#设置client监听地址
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
