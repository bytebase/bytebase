package store

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"sync"

	"github.com/pkg/errors"
)

// cacheNamespace is the type of a cache.
type cacheNamespace string

const (
	// principalCacheNamespace is the cache type of principals.
	principalCacheNamespace cacheNamespace = "p"
	// environmentCacheNamespace is the cache type of environments.
	environmentCacheNamespace cacheNamespace = "e"
	// projectCacheNamespace is the cache type of projects.
	projectCacheNamespace cacheNamespace = "r"
	// projectMemberCacheNamespace is the cache type of project members.
	projectMemberCacheNamespace cacheNamespace = "pm"
	// instanceCacheNamespace is the cache type of instances.
	instanceCacheNamespace cacheNamespace = "i"
	// databaseCacheNamespace is the cache type of databases.
	databaseCacheNamespace cacheNamespace = "d"
	// memberCacheNamespace is the cache type of members.
	memberCacheNamespace cacheNamespace = "m"
	// pipelineCacheNamespace is the cache type of pipelines.
	pipelineCacheNamespace cacheNamespace = "pl"
	// issueCacheNamespace is the cache type of issues.
	issueCacheNamespace cacheNamespace = "is"
	// databaseLabelCacheNamespace is the cache type of database labels.
	databaseLabelCacheNamespace cacheNamespace = "dl"
	// dataSourceCacheNamespace is the cache type of data sources.
	dataSourceCacheNamespace cacheNamespace = "ds"
	// tierPolicyCacheNamespace is the cache type of tier policy.
	tierPolicyCacheNamespace cacheNamespace = "pot"
	// approvalPolicyCacheNamespace is the cache type of approval policy.
	approvalPolicyCacheNamespace cacheNamespace = "app"
)

// CacheService implements a cache.
type CacheService struct {
	sync.Mutex
	cache map[string][]byte
}

// newCacheService creates a cache service.
func newCacheService() *CacheService {
	return &CacheService{
		cache: make(map[string][]byte),
	}
}

// FindCache finds the value in cache.
func (s *CacheService) FindCache(namespace cacheNamespace, id int, entry interface{}) (bool, error) {
	key := generateKey(namespace, id)

	s.Lock()
	defer s.Unlock()
	value, exists := s.cache[key]
	if exists {
		dec := gob.NewDecoder(bytes.NewReader(value))
		if err := dec.Decode(entry); err != nil {
			return false, errors.Wrapf(err, "failed to decode entry for cache namespace: %s", namespace)
		}
		return true, nil
	}
	return false, nil
}

// UpsertCache upserts the value to cache.
func (s *CacheService) UpsertCache(namespace cacheNamespace, id int, entry interface{}) error {
	key := generateKey(namespace, id)

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(entry); err != nil {
		return errors.Wrapf(err, "failed to encode entry for cache namespace: %s", namespace)
	}

	s.Lock()
	defer s.Unlock()
	s.cache[key] = buf.Bytes()

	return nil
}

// DeleteCache deletes the key from cache.
func (s *CacheService) DeleteCache(namespace cacheNamespace, id int) {
	key := generateKey(namespace, id)

	s.Lock()
	defer s.Unlock()
	delete(s.cache, key)
}

func generateKey(namespace cacheNamespace, id int) string {
	return fmt.Sprintf("%s%d", namespace, id)
}
