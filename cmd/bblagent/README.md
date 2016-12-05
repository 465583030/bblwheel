#服务治理中心代理程序
可以通过使用代理程序实现apache、nginx、haproxy等服务注册、发现、配置变更、服务启动、重启、停止等功能
###编译&启动
	
	go build -o bblagent github.com/gqf2008/bblwheel/cmd/bblagent
	./bblagent -agentdir=/var/lib/bblwheel/bblagent
	在/var/lib/bblwheel/bblagent添加需要注册的服务