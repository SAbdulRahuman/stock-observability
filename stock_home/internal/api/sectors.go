package api

import "context"

// indexQuotes returns price, change and the latest-session sparkline for each listed index that has data, in list order.
func (s *Server) indexQuotes(q *querier, symbols []string) []Quote {
	out := []Quote{}
	prices := bySymbol(q.vector(`max by (exchange, symbol) (stock_last_price{instrument_type="INDEX"})`))
	changes := bySymbol(q.vector(`max by (exchange, symbol) (stock_change_percent{instrument_type="INDEX"})`))
	abs := bySymbol(q.vector(absChangeQuery(`instrument_type="INDEX"`)))
	for _, sym := range symbols {
		p, ok := prices[sym]
		if !ok {
			continue
		}
		v := p.Value
		qt := Quote{Exchange: p.Labels["exchange"], Symbol: sym, Price: &v, ExploreURL: symbolExploreURL(p.Labels["exchange"], sym)}
		if c, ok := changes[sym]; ok {
			qt.ChangePercent = &c.Value
		}
		if c, ok := abs[sym]; ok {
			qt.Change = &c.Value
		}
		out = append(out, qt)
	}
	if len(out) > 0 {
		sparks := sessionSparks(q, `, instrument_type="INDEX"`, symbolsOf(out), s.now(), s.cfg.Location)
		for i := range out {
			out[i].Spark = sparks[out[i].Symbol]
		}
	}
	return out
}

// SectorIndices are the configured NSE sectoral indices for the Sector View page.
type SectorIndices struct {
	Indices []Quote `json:"indices"`
}

func (s *Server) sectorIndices(ctx context.Context) (any, error) {
	q := s.querier(ctx)
	out := SectorIndices{Indices: s.indexQuotes(q, s.cfg.SectorIndices)}
	if q.err != nil {
		return nil, q.err
	}
	return out, nil
}
