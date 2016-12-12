package rpc

//BeforeFilters ....
func (s *Server) BeforeFilters(f []Filter) {
	if f == nil || len(f) == 0 {
		return
	}
	s.chain.befchain = f
	s.updateChain()
}

//AddBeforeFilter ....
func (s *Server) AddBeforeFilter(f Filter) {
	s.chain.befchain = append(s.chain.befchain, f)
	s.updateChain()
}

//AfterFilters ....
func (s *Server) AfterFilters(f []Filter) {
	if f == nil || len(f) == 0 {
		return
	}
	s.chain.afterchain = f
	s.updateChain()
}

//AddAfterFilter ....
func (s *Server) AddAfterFilter(f Filter) {
	s.chain.afterchain = append(s.chain.afterchain, f)
	s.updateChain()
}

func (s *Server) updateChain() {
	s.chain.chain = []Filter{}
	s.chain.chain = append(s.chain.chain, s.chain.befchain...)
	s.chain.chain = append(s.chain.chain, s.chain.router)
	s.chain.chain = append(s.chain.chain, s.chain.afterchain...)
}

//Filter ....
type Filter interface {
	DoFilter(*Request, *Response, FilterChain)
}

//FilterChain ....
type FilterChain interface {
	DoFilter(*Request, *Response)
}

type filterChain struct {
	befchain   []Filter
	afterchain []Filter
	chain      []Filter
	idx        int
	router     *routerFilter
}

func (f *filterChain) DoFilter(req *Request, resp *Response) {
	if f.idx == len(f.chain) {
		return
	}
	filter := f.chain[f.idx]
	f.idx++
	filter.DoFilter(req, resp, f)
}
