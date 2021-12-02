package server

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/bytebase/bytebase/api"
	"go.uber.org/zap"
)

var (
	// 32 MiB
	cacheSize                  = 1024 * 1024 * 32
	_         api.CacheService = (*CacheService)(nil)
)

// CacheService implements a cache.
type CacheService struct {
	cache *fastcache.Cache

	l *zap.Logger
}

// NewCacheService creates a cache service.
func NewCacheService(logger *zap.Logger) *CacheService {
	return &CacheService{
		cache: fastcache.New(cacheSize),
		l:     logger,
	}
}

// FindCache finds the value in cache.
func (s *CacheService) FindCache(namespace api.CacheNamespace, id int, entry interface{}) (bool, error) {
	buf1 := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	binary.LittleEndian.PutUint64(buf1, uint64(id))

	buf2, has := s.cache.HasGet(nil, append([]byte(namespace), buf1...))
	if has {
		dec := gob.NewDecoder(bytes.NewReader(buf2))
		if err := dec.Decode(entry); err != nil {
			return false, fmt.Errorf("failed to decode entry for cache namespace: %s, error: %w", namespace, err)
		}
		return true, nil
	}

	return false, nil
}

// UpsertCache upserts the value to cache.
func (s *CacheService) UpsertCache(namespace api.CacheNamespace, id int, entry interface{}) error {
	buf1 := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	binary.LittleEndian.PutUint64(buf1, uint64(id))

	var buf2 bytes.Buffer
	enc := gob.NewEncoder(&buf2)
	if err := enc.Encode(entry); err != nil {
		return fmt.Errorf("failed to encode entry for cache namespace: %s, error: %w", namespace, err)
	}
	s.cache.Set(append([]byte(namespace), buf1...), buf2.Bytes())

	return nil
}
