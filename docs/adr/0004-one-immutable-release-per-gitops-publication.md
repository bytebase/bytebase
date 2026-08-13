# One immutable Release per GitOps publication

A Release is a project-scoped, immutable snapshot of the ordered database changes submitted in one publication — the governed change artifact handed from Git to Bytebase. Bytebase retains the exact content through project-scoped sheet references and content hashes, and creation never deduplicates by commit or content, so execution never refetches from Git and every publication stays separately identifiable and auditable.

The canonical publication is one `bytebase-action rollout` invocation selecting files from one repository commit: `bytebase-action check` validates candidate files without persisting anything, while `rollout` (unless given an existing plan) publishes the Release before creating the Plan and Rollout. The API does not restrict publication to that path — any caller may create a Release, and VCS provenance (the commit URL) is optional; a Release without it has unrecorded provenance, not an error.

A Release contains either versioned migrations or a declarative desired-state definition, never both — the change type is a property of the Release, not of individual files. Versioned files are normalized into version order at creation, and neither files nor change type can change afterward; a different manifest requires another Release. Release records what was published, never where or how it deploys: Plan owns deployment targets and intent, Rollout owns execution, and Release itself has only an artifact lifecycle — archiving withdraws it from new use, restoring changes neither its identity nor its contents.

## Considered alternatives

- **Pull request as Release identity.** Rejected: a pull request spans multiple commits, exists before publication, and is not required by every repository; the selected commit states precisely what was published.
- **Deduplicate by commit and file manifest.** Rejected: each publication must remain separately identifiable and auditable, and content-equivalence semantics add complexity without governance value.
- **Commit alone as Release identity.** Rejected: one commit may contain multiple independently selected database-change manifests.
- **Fetch files from Git during execution.** Rejected: execution must not depend on Git availability or history mutability; a force-push must not change what executes.

## Consequences

- A project's release list may contain Releases with identical content and provenance, differing only in resource name and creation time; re-running a workflow yields another Release iteration.
- Versioned execution may skip migrations already applied to a target without changing the Release; deployment status is never an intrinsic Release property, though Release UI may summarize related activity.
- Releases created outside the GitOps flow, or without provenance, are valid. Requiring provenance, deduplicating Releases, or constraining Release-to-Plan cardinality would each be a separate decision.
