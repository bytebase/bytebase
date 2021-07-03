package api

type CacheNamespace string

const (
	PrincipalCache CacheNamespace = "p"
)

type CacheService interface {
	FindCache(namespace CacheNamespace, id int, entry interface{}) (bool, error)
	UpsertCache(namespace CacheNamespace, id int, entry interface{}) error
}
