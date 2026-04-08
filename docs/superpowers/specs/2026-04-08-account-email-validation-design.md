# Account Email Validation

## Goal

Apply the existing backend email validation rule consistently to system-generated
service account and workload identity emails.

## Scope

- Reject `service_account_id` values that generate malformed emails.
- Reject `workload_identity_id` values that generate malformed emails.
- Reject malformed service account emails supplied through resource names.
- Reject malformed workload identity emails supplied through resource names.

## Approach

- Reuse `common.ValidateEmail(...)` as the canonical syntax check.
- Preserve service-account and workload-identity suffix checks.
- Validate generated emails immediately after `BuildServiceAccountEmail(...)`
  and `BuildWorkloadIdentityEmail(...)`.
- Validate extracted resource-name emails before store lookups in the affected
  handlers.

## Testing

- Add integration coverage for invalid service account creation.
- Add integration coverage for invalid workload identity creation.
- Add integration coverage for malformed service account resource names.
- Add integration coverage for malformed workload identity resource names.
