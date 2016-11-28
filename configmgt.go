package bblwheel

import "sync"

func loadAndWatchConfig() {

}

type configService struct {
	table map[string]map[string]string
	lock  sync.RWMutex
	once  Once
}

func (c *configService) watch() {

}
