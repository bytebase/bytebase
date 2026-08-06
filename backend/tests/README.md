# Backend integration tests

Public v1 API tests backed by real PostgreSQL live here. Cross-Project
collision tests use `setupCollidingProjects` plus `snapshotProject` /
`assertProjectUnchanged` from `collision_helper_test.go` (see
`docs/pre-pr-checklist.md` step 3 for the full contract).

## plan_webhook_delivery collision coverage

`plan_webhook_delivery` has no public gRPC read API, so collision coverage reads
the table directly from the metadata DB. Two extra requirements apply to any
test for a writer of this table:

- Delivery rows are claimed asynchronously after rollout completion. Wait for
  the project's delivery row set to stabilize before taking the baseline
  snapshot, and again before the post-operation snapshot.
- `assertProjectUnchanged` intentionally does NOT compare
  `PlanWebhookDeliveries`. The test must assert those rows separately with its
  own stabilized comparison.

`TestCollision_PlanWebhookDeliveryWrite` demonstrates both requirements.
Without them, a raw-row snapshot that only calls `assertProjectUnchanged`
produces false-positive coverage: a cross-project write to this composite-PK
table would go undetected.
