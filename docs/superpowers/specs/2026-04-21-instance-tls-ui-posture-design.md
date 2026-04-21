# Instance TLS UI Posture Design

## Goal

Redesign the instance data source TLS UI as a pure presentation change over the existing TLS fields. The UI should make the connection security posture explicit without adding templates, telemetry, proto fields, API fields, or backend behavior.

This design applies only to instance data source connection security. Vault TLS configuration stays on the existing UI.

## Non-Goals

- No "Start from template" flow.
- No telemetry changes.
- No proto, API, database, or backend behavior changes.
- No persisted TLS mode enum.
- No Vault TLS redesign.
- No raw engine-specific labels such as `sslmode` in the user-facing UI.

## Approaches Considered

### Replace the SSL switch with posture

Use one segmented posture control under a `Connection security` section:

`Disabled | TLS | Mutual TLS`

This is the chosen design. It gives one visible control for the user's primary decision and maps cleanly onto existing fields.

### Keep the SSL switch and add posture below it

This keeps the current `SSL Connection` switch and adds another control after it. It was rejected because it creates two controls for the same state: off versus disabled, and TLS versus mTLS.

### Keep the current source-first UI

This keeps the current CA/client certificate source controls without a posture picker. It was rejected because it makes mTLS an implied side effect of choosing client certificate material instead of a first-class security posture.

## Section Structure

The section title is `Connection security`.

The first control is a segmented posture picker:

`Disabled | TLS | Mutual TLS`

For new instance data sources, the default posture is `Disabled`.

For existing data sources, infer posture from saved fields:

- `useSsl=false` means `Disabled`.
- `useSsl=true` with no client certificate/key means `TLS`.
- `useSsl=true` with client certificate/key means `Mutual TLS`.

## Disabled Posture

When `Disabled` is selected:

- Hide server identity fields.
- Hide client identity fields.
- Clear TLS material in the local form state before save.

This avoids hidden TLS configuration under a visibly disabled posture.

## TLS Posture

When `TLS` is selected, show the `Server identity` group.

The group contains:

- `Verify server certificate` toggle.
- CA certificate source segmented control: `System trust | Paste PEM | File path`.
- CA material input only when the selected source needs input.

When `Verify server certificate` is on and CA source is `System trust`, show helper text:

`Uses the system trust store to verify the server certificate.`

When `Verify server certificate` is off, keep CA source and CA inputs enabled, but show this explanation:

`Verification is disabled. The connection is encrypted, but the server identity is not verified; CA settings are ignored.`

Do not clear CA material merely because verification is turned off. Users may turn verification off temporarily and should not lose the CA configuration unless they switch the entire posture to `Disabled` or change the CA source.

## Mutual TLS Posture

When `Mutual TLS` is selected, show both `Server identity` and `Client identity`.

`Server identity` behaves the same as in the `TLS` posture. `System trust` remains available because mTLS does not require a private CA for the server certificate.

`Client identity` contains:

- Client material source segmented control: `Paste PEM | File path`.
- `Certificate` input.
- `Private key` input.

Do not show a `None` option inside `Client identity`. If users do not want client identity, they should choose the `TLS` posture.

Do not add helper copy saying certificate and private key are required together. The backend already validates the pair. The fields should still be visually grouped in one `Client identity` block.

## Source Labels

Use user-facing labels:

- `Paste PEM`, not `Inline PEM`.
- `File path`, not engine-specific or field-specific source labels.

Use specific input labels:

- `CA certificate path`.
- `Certificate path`.
- `Private key path`.
- `Certificate`.
- `Private key`.

Keep existing drag-and-drop behavior for PEM textareas even though the source option says `Paste PEM`.

## SaaS Mode

Bytebase SaaS mode should disable file-path source options because local Bytebase server paths are unavailable in Bytebase Cloud.

Use the existing frontend SaaS signal: `actuatorStore.isSaaSMode`, read from React via `useVueState`.

When SaaS mode is active:

- Disable `File path` in the CA source control.
- Disable `File path` in the Client identity source control.
- Show a hover tooltip on disabled file-path options:

`File paths are unavailable in Bytebase Cloud.`

If an existing data source already has file-path TLS material, show the file-path source as selected but disabled. The UI must represent saved state truthfully instead of silently switching to `System trust` or `Paste PEM`.

## Engine Support

For engines where the form does not support client certificate/key material, keep the `Mutual TLS` posture option visible but disabled.

Show a hover tooltip on the disabled `Mutual TLS` option:

`Mutual TLS is not available for this engine.`

If an existing saved data source has client certificate/key material, show `Mutual TLS` as selected even if the normal creation path would disable selecting it. Saved state should be represented truthfully.

## State Changes

Switching from `Mutual TLS` to `TLS` clears client certificate/key fields immediately.

Switching from `TLS` or `Mutual TLS` to `Disabled` clears CA material and client certificate/key material immediately.

Switching CA source clears inactive CA fields only. It must not clear client certificate/key fields.

Switching client identity source clears inactive client certificate/key fields only. It must not clear CA fields.

## Testing

Frontend tests should cover:

- New data sources default to `Disabled`.
- Existing data sources infer `Disabled`, `TLS`, and `Mutual TLS` from existing fields.
- Switching `Mutual TLS` to `TLS` clears client certificate/key material.
- Switching to `Disabled` clears TLS material.
- Turning verification off keeps CA controls enabled and keeps CA material.
- SaaS mode disables file-path source options and shows the SaaS tooltip.
- Existing file-path saved state remains visibly selected but disabled in SaaS mode.
- Unsupported engines show disabled `Mutual TLS` with the unsupported-engine tooltip.
- Existing client certificate/key saved state renders as `Mutual TLS` even for engines that normally cannot select it.
- Labels use `Connection security`, `TLS`, `Mutual TLS`, `Server identity`, `Client identity`, `Paste PEM`, `Certificate`, and `Private key`.
