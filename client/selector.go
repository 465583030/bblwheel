package client

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/gqf2008/bblwheel"
)

//Selector ....
type Selector interface {
	Select(name, key string) *bblwheel.Service
	AddService(*bblwheel.Service) error
	RemoveService(id, name string)
}

//MaxServiceInstance ....
const MaxServiceInstance = 1024

//BaseSelector ....
type BaseSelector struct {
	instances map[string]*hashset
	lock      sync.RWMutex
}

//AddService ....
func (s *BaseSelector) AddService(srv *bblwheel.Service) error {
	s.lock.Lock()
	inss, has := s.instances[srv.Name]
	if !has {
		inss = newHashSet()
	}
	if inss.len() >= MaxServiceInstance {
		s.lock.Unlock()
		return fmt.Errorf("service instance over %v", MaxServiceInstance)
	}
	inss.add(srv)
	s.lock.Unlock()
	return nil
}

//RemoveService ....
func (s *BaseSelector) RemoveService(id, name string) {
	s.lock.Lock()
	inss, has := s.instances[name]
	if has {
		inss.remove(name + "/" + id)
	}
	s.lock.Unlock()
}

//FindService ....
func (s *BaseSelector) FindService(name string) []*bblwheel.Service {
	list := []*bblwheel.Service{}
	if l, has := s.instances[name]; has {
		list = l.values()
	}
	return list
}

type hashset struct {
	m map[string]*bblwheel.Service
}

func newHashSet() *hashset {
	return &hashset{m: make(map[string]*bblwheel.Service)}
}

func (set *hashset) add(e *bblwheel.Service) (b bool) {
	key := e.Name + "/" + e.ID
	if set.m[key] == nil {
		set.m[key] = e
		return true
	}
	return false
}

func (set *hashset) remove(key string) {
	delete(set.m, key)
}

func (set *hashset) clear() {
	set.m = make(map[string]*bblwheel.Service)
}
func (set *hashset) contains(key string) bool {
	return set.m[key] != nil
}

func (set *hashset) len() int {
	return len(set.m)
}

func (set *hashset) values() []*bblwheel.Service {
	list := []*bblwheel.Service{}
	for _, v := range set.m {
		list = append(list, v)
	}
	return list
}

func (set *hashset) String() string {
	var buf bytes.Buffer

	buf.WriteString("set{")

	first := true

	for _, v := range set.m {
		if first {
			first = false
		} else {
			buf.WriteString(" ")
		}

		buf.WriteString(fmt.Sprintf("%v", v))
	}

	buf.WriteString("}")

	return buf.String()
}
