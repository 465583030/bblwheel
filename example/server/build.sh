go build -ldflags "-s -w -X main.Version=`date -u +%Y%m%d.%H%M%S` -X main.Author=gqf -X main.Mail=gao.qingfeng@gmail.com" echoserver.go
