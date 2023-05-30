// Package catalog provides API definition for catalog service.
package catalog

// Catalog is the service for catalog.
type Catalog interface {
	GetFinder() *Finder
}
