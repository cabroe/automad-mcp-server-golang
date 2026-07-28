package starterkit

import "context"

// NewSeededService creates a Service with a pre-populated cache, bypassing
// GitHub API access entirely. This is intended for use in tests to exercise
// service and server logic without network access, mirroring
// docs.NewSeededService.
//
// tree may be nil to leave the tree cache empty. files may be nil/empty to
// leave the file cache empty.
func NewSeededService(tree *Tree, files map[string][]byte) *Service {
	svc := &Service{
		lifecycle: context.Background(),
		client:    NewClient(),
		cache:     NewCache(DefaultCacheTTL),
	}
	if tree != nil {
		svc.cache.SetTree(tree)
	}
	for path, content := range files {
		svc.cache.SetFile(path, content)
	}
	return svc
}
