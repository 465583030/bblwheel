package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	grpclog "log"

	"github.com/fsnotify/fsnotify"
	"github.com/gqf2008/bblwheel"
	"github.com/gqf2008/bblwheel/client"
)

var endpoints = "127.0.0.1:23790"
var workdir = "/var/lib/bblwheel/bblagent"

var agents = map[string]*agent{}
var watcher *fsnotify.Watcher

func init() {
	flag.StringVar(&endpoints, "endpoints", endpoints, "bblwheel server address")
	flag.StringVar(&workdir, "agentdir", workdir, "service file directory")
	grpclog.SetFlags(grpclog.Lshortfile | grpclog.LstdFlags)
}
func main() {
	flag.Parse()
	w, err := fsnotify.NewWatcher()
	if err != nil {
		grpclog.Fatalln(err)
	}
	watcher = w
	filepath.Walk(workdir, listfunc)
	for {
		select {
		case event := <-watcher.Events:
			log.Println("event:", event)
			if event.Op&fsnotify.Write == fsnotify.Write {
				fi, err := os.Stat(event.Name)
				if err != nil {
					grpclog.Println("agent", event.Name, err)
					break
				}
				if fi.IsDir() || !strings.HasSuffix(event.Name, ".json") {
					break
				}
				if a, has := agents[event.Name]; has {
					a.ch <- "reload"
				} else {
					grpclog.Println("agent", event.Name, "not found")
				}
			} else if event.Op&fsnotify.Create == fsnotify.Create {
				fi, err := os.Stat(event.Name)
				if err != nil {
					grpclog.Println("agent", event.Name, err)
					break
				}
				if fi.IsDir() {
					grpclog.Println("add watch", event.Name, watcher.Add(event.Name))
					break
				}
				if !strings.HasSuffix(event.Name, ".json") {
					break
				}
				if _, has := agents[event.Name]; has {
					grpclog.Println("agent", event.Name, "exist")
				} else if strings.HasSuffix(event.Name, ".json") {
					a, err := newAgent(event.Name)
					if err != nil {
						grpclog.Println(event.Name, err)
						break
					}
					agents[event.Name] = a
					go work(a)
				}
			} else if event.Op&fsnotify.Remove == fsnotify.Remove || event.Op&fsnotify.Rename == fsnotify.Rename {
				if !strings.HasSuffix(event.Name, ".json") {
					break
				}
				if a, has := agents[event.Name]; has {
					a.done <- struct{}{}
					delete(agents, event.Name)
				}
			}
		case err := <-watcher.Errors:
			log.Println("error:", err)
		}
	}
}

func listfunc(path string, f os.FileInfo, err error) error {
	if f == nil {
		return err
	}
	if f.IsDir() {
		grpclog.Println("add watch", path, watcher.Add(path))
		return nil
	}
	if strings.HasSuffix(path, ".json") {
		a, err := newAgent(path)
		if err != nil {
			grpclog.Println(path, err)
			return nil
		}
		agents[path] = a
		go work(a)
	}
	return nil
}

func newAgent(path string) (*agent, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ins := client.ServiceInstance{}
	err = json.Unmarshal(b, &ins)
	if err != nil {
		return nil, err
	}
	a := agent{service: &ins, path: path, ch: make(chan string, 1), done: make(chan struct{})}
	return &a, nil
}

type agent struct {
	service *client.ServiceInstance
	path    string
	ch      chan string
	done    chan struct{}
}

func (a *agent) onDiscovery(srv *bblwheel.Service) {
	grpclog.Println("onDiscovery", srv)
}

func (a *agent) OnConfigUpdated(key, value string) {
	grpclog.Println("OnConfigUpdated", key, "=", value)
}

func (a *agent) onControl(cmd string) {
	grpclog.Println("onControl", cmd)
	switch cmd {
	case "start":
	case "restart":
	case "reload":
	case "stop":
	}
}

func (a *agent) onExec(cmd string) {
	grpclog.Println("onControl", cmd)
}

func work(a *agent) {
	grpclog.Println("agent", a.service.Name+"/"+a.service.ID, "running")
	defer grpclog.Println("agent", a.service.Name+"/"+a.service.ID, "exit")
	a.service.OnDiscovery = a.onDiscovery
	a.service.OnConfigUpdated = a.OnConfigUpdated
	a.service.OnControl = a.onControl
	a.service.OnExec = a.onExec
	a.service.Endpoints = strings.Split(endpoints, ",")
	for {
		res := a.service.Register()
		grpclog.Println("Register", a.service, res)
		if "SUCCESS" == res.Desc {
			break
		}
		time.Sleep(3 * time.Second)
	}
	for {
		select {
		case <-a.done:
			a.service.Unregister()
			return
		case cmd := <-a.ch:
			grpclog.Println(cmd)
		}
	}
}
