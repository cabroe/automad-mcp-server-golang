package docs

import "context"

// NewSeededService creates a Service with a pre-populated cache entry for
// the given URL and page, bypassing HTTP fetching. This is intended for use
// in tests to exercise service and server logic without network access.
//
// It is safe to call NewSeededService with url="" and page=nil, in which case
// a regular empty service is returned.
func NewSeededService(url string, page *Page) *Service {
	svc := &Service{
		lifecycle: context.Background(),
		fetcher:   NewFetcher(),
		cache:     NewCache(DefaultCacheTTL),
	}
	if url != "" && page != nil {
		svc.cache.Set(url, page)
	}
	return svc
}
