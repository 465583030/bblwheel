package bblwheel

//Put ....
func (c *Config) Put(key, val string) {
	c.Items = append(c.Items, &ConfigEntry{key, val})
}

//PutAll ....
func (c *Config) PutAll(all map[string]string) {
	for k, v := range all {
		c.Items = append(c.Items, &ConfigEntry{k, v})
	}
}

//Get ....
func (c *Config) Get(key string) (string, bool) {
	for _, item := range c.Items {
		if item.Key == key {
			return item.Value, true
		}
	}
	return "", false
}

//Has ....
func (c *Config) Has(key string) bool {
	for _, item := range c.Items {
		if item.Key == key {
			return true
		}
	}
	return false
}

//Remove ....
func (c *Config) Remove(key string) string {
	for i, item := range c.Items {
		if item.Key == key {
			c.Items = append(c.Items[:i], c.Items[i+1:]...)
			return item.Value
		}
	}
	return ""
}

//RemoveAll ....
func (c *Config) RemoveAll() {
	c.Items = []*ConfigEntry{}
}
