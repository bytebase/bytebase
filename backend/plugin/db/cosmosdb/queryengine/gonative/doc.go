// Package gonative implements queryengine.QueryEngine entirely in Go
// (no cgo, no native plugins, no external toolchain).
//
// The CosmosDB driver engages it per query: the gateway runs what it can on
// its own, and only a cross-partition query it refuses is replayed through
// this engine. SupportedFeatures lists the operators the engine implements;
// a plan that needs anything else fails with ErrUnsupportedPlanFeature, and
// the caller then surfaces the gateway's original error.
package gonative
