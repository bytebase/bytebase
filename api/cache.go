package api

type CacheNamespace string

const (
	PrincipalCache   CacheNamespace = "p"
	EnvironmentCache CacheNamespace = "e"
	ProjectCache     CacheNamespace = "r"
)

type CacheService interface {
	FindCache(namespace CacheNamespace, id int, entry interface{}) (bool, error)
	UpsertCache(namespace CacheNamespace, id int, entry interface{}) error
}
