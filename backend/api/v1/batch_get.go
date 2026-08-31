package v1

// resolveBatchGet walks the requested names in order and splits them into the
// resources that resolved and the names that did not.
//
// Every BatchGet RPC shares this contract: resources come back in request
// order, a repeated name is served once from its first occurrence, and a name
// the caller cannot be given — the resource is gone, or invisible to them — is
// reported back rather than silently dropped. resolve signals that case with
// matched=false, and the two reasons are deliberately indistinguishable so the
// response cannot be used to probe what exists.
//
// An error from resolve fails the whole batch, so a store outage can never read
// as an absent resource.
func resolveBatchGet[T any](names []string, resolve func(name string) (T, bool, error)) ([]T, []string, error) {
	resources := make([]T, 0, len(names))
	var unmatched []string
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		if seen[name] {
			continue
		}
		seen[name] = true
		resource, matched, err := resolve(name)
		if err != nil {
			return nil, nil, err
		}
		if !matched {
			unmatched = append(unmatched, name)
			continue
		}
		resources = append(resources, resource)
	}
	return resources, unmatched, nil
}
