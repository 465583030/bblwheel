package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"path/filepath"

	"github.com/gqf2008/bblwheel/client"
)

var endpoints = "127.0.0.1:23790"
var workdir = "/var/bblwheel/bblagent"

func init() {
	flag.StringVar(&endpoints, "endpoints", endpoints, "bblwheel server address")
	flag.StringVar(&workdir, "workdir", workdir, "service file directory")
}
func main() {
	flag.Parse()
	filepath.Walk(workdir, listfunc)
}

func listfunc(path string, f os.FileInfo, err error) error {
	var strRet string
	strRet, _ = os.Getwd()
	//ostype := os.Getenv("GOOS") // windows, linux

	if ostype == "windows" {
		strRet += "\\"
	} else if ostype == "linux" {
		strRet += "/"
	}

	if f == nil {
		return err
	}
	if f.IsDir() {
		return nil
	}

	strRet += path //+ "\r\n"

	//用strings.HasSuffix(src, suffix)//判断src中是否包含 suffix结尾
	ok := strings.HasSuffix(strRet, ".go")
	if ok {

		listfile = append(listfile, strRet) //将目录push到listfile []string中
	}
	//fmt.Println(ostype) // print ostype
	fmt.Println(strRet) //list the file

	return nil
}

func work(ins *client.ServiceInstance) {

}
