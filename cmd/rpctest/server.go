package main

import (
	"log"

	"github.com/gqf2008/bblwheel/rpc"
)

func main() {
	rpc.HandleCallFunc("/echo", func(req *rpc.Request, resp *rpc.Response) error {
		req.Content = []byte("echoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoechoecho")
		return nil
	})
	log.Println(rpc.ListenAndServe())
}
