package main

import (
	"flag"
	"yunji/bblwheel"
)

func main() {
	flag.Parse()
	bblwheel.ListenAndServe()
}
