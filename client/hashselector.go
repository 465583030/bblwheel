package client

import "github.com/gqf2008/bblwheel"

//NewHashSelector ....
func NewHashSelector() Selector {
	s := &hashSelector{}
	s.instances = map[string]*hashset{}
	return s
}

type hashSelector struct {
	*BaseSelector
}

func (s *hashSelector) Select(name, key string) *bblwheel.Service {
	list := s.FindService(name)
	if len(list) == 0 {
		return nil
	}
	return list[int(bblwheel.Murmur3([]byte(key)))%len(list)]
}
