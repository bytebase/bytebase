package api

// CacheNamespace is the type of a cache.
type CacheNamespace string

const (
	// PrincipalCache is the cache type of principals.
	PrincipalCache CacheNamespace = "p"
	// EnvironmentCache is the cache type of environments.
	EnvironmentCache CacheNamespace = "e"
	// ProjectCache is the cache type of projects.
	ProjectCache CacheNamespace = "r"
	// InstanceCache is the cache type of instances.
	InstanceCache CacheNamespace = "i"
	// DatabaseCache is the cache type of databases.
	DatabaseCache CacheNamespace = "d"
	// MemberCache is the cache type of members.
	MemberCache CacheNamespace = "m"
	// PipelineCache is the cache type of pipelines.
	PipelineCache CacheNamespace = "pl"
	// IssueCache is the cache type of issues.
	IssueCache CacheNamespace = "is"
	// DatabaseLabelCache is the cache type of database labels.
	DatabaseLabelCache CacheNamespace = "dl"
	// DataSourceCache is the cache type of data sources.
	DataSourceCache CacheNamespace = "ds"
)

// CacheService is the service for caches.
type CacheService interface {
	FindCache(namespace CacheNamespace, id int, entry interface{}) (bool, error)
	UpsertCache(namespace CacheNamespace, id int, entry interface{}) error
	DeleteCache(namespace CacheNamespace, id int)
}
