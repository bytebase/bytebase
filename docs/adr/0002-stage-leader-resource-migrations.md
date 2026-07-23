# Stage leader-resource migrations across two releases

Changing a responsibility from a global Leader Type to a distinct concrete-resource Leader Type changes which replicas may work concurrently, so mixed behavior during an HA rolling upgrade could violate mutual exclusion. The accepted path therefore separates compatibility from activation.

The compatibility release adds a non-null `resource` column with default `global` and a composite uniqueness constraint on Leader Type and resource. It retains the existing type-only uniqueness constraint and continues global behavior only. It must understand and safely honor the later protocol, but must not enable any concrete-resource Leader Types.

Only the subsequent activation release removes the type-only uniqueness constraint. After that removal, it may enable concrete-resource Leader Types. This favors safe mixed-version operation and rollback over activating the new behavior in the release that first introduces it.

## Consequences

- Each Leader Type retains one resource kind; concrete-resource behavior uses a distinct type rather than broadening a global type in place.
- The compatibility release retains both uniqueness constraints and performs global work only.
- The activation release removes type-only uniqueness before enabling concrete-resource Leader Types, and still needs a defined drain and rollback sequence.
