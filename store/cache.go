package store

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/pkg/errors"
)

var (
	// 32 MiB.
	cacheSize = 1024 * 1024 * 32
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
)

// CacheService implements a cache.
type CacheService struct {
	cache *fastcache.Cache
}

// newCacheService creates a cache service.
func newCacheService() *CacheService {
	return &CacheService{
		cache: fastcache.New(cacheSize),
	}
}

// FindCache finds the value in cache.
func (s *CacheService) FindCache(namespace cacheNamespace, id int, entry interface{}) (bool, error) {
	buf1 := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	binary.LittleEndian.PutUint64(buf1, uint64(id))

	buf2, has := s.cache.HasGet(nil, append([]byte(namespace), buf1...))
	if has {
		dec := gob.NewDecoder(bytes.NewReader(buf2))
		if err := dec.Decode(entry); err != nil {
			return false, errors.Wrapf(err, "failed to decode entry for cache namespace: %s", namespace)
		}
		return true, nil
	}

	return false, nil
}

// UpsertCache upserts the value to cache.
func (s *CacheService) UpsertCache(namespace cacheNamespace, id int, entry interface{}) error {
	buf1 := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	binary.LittleEndian.PutUint64(buf1, uint64(id))

	var buf2 bytes.Buffer
	enc := gob.NewEncoder(&buf2)
	if err := enc.Encode(entry); err != nil {
		return errors.Wrapf(err, "failed to encode entry for cache namespace: %s", namespace)
	}
	s.cache.Set(append([]byte(namespace), buf1...), buf2.Bytes())

	return nil
}

// DeleteCache deletes the key from cache.
func (s *CacheService) DeleteCache(namespace cacheNamespace, id int) {
	buf1 := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	binary.LittleEndian.PutUint64(buf1, uint64(id))
	s.cache.Del(append([]byte(namespace), buf1...))
}
