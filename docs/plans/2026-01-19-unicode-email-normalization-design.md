# Unicode Email Normalization Fix

**Issue**: [#18943](https://github.com/bytebase/bytebase/issues/18943)
**Date**: 2026-01-19

## Problem

Users can create accounts with visually identical but technically distinct email addresses using Unicode homoglyphs. For example:
- `testo@gmail.com` (ASCII 'o')
- `testо@gmail.com` (Cyrillic 'о', U+043E)

This creates security risks: account confusion, privilege mis-assignment, audit ambiguity, and password reset/SSO mismatch.

## Solution

Enforce ASCII-only characters in email addresses for all new account creation and email updates.

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Validation scope | Full email (not just local part) | Simpler, eliminates all homoglyph vectors |
| Migration strategy | Reject new only | Non-breaking, existing accounts unchanged |
| Validation placement | Frontend + Backend | Immediate UX feedback + authoritative security |
| Domain handling | ASCII-only (IDN rejected) | Stricter security, simpler implementation |

## Implementation

### Backend

**File**: `backend/api/v1/user_service.go`

Update `validateEmail()`:

```go
func validateEmail(email string) error {
    if email != strings.ToLower(email) {
        return errors.New("email should be lowercase")
    }

    // Validate entire email contains only ASCII characters
    for _, r := range email {
        if r > 127 {
            return errors.New("email must contain only ASCII characters")
        }
    }

    if _, err := mail.ParseAddress(email); err != nil {
        return err
    }
    return nil
}
```

**File**: `backend/api/directory-sync/webhook.go`

Add ASCII validation after `normalizeEmail()` calls to reject non-ASCII emails from SCIM providers.

### Frontend

**File**: `frontend/src/components/EmailInput.vue`

Add validation helper:

```typescript
const isAsciiEmail = (email: string): boolean => {
  return /^[\x00-\x7F]*$/.test(email);
};
```

**File**: `frontend/src/locales/en-US.json`

```json
{
  "common": {
    "email-ascii-only": "Email addresses with special Unicode characters are not supported. Please use standard characters (a-z, 0-9, and common symbols)."
  }
}
```

Add translations to other locale files as needed.

## Files to Modify

| File | Change |
|------|--------|
| `backend/api/v1/user_service.go` | Add ASCII check in `validateEmail()` |
| `backend/api/directory-sync/webhook.go` | Add ASCII validation after `normalizeEmail()` |
| `frontend/src/components/EmailInput.vue` | Add `isAsciiEmail()` validation |
| `frontend/src/locales/en-US.json` | Add `common.email-ascii-only` message |
| `frontend/src/locales/zh-CN.json` | Add Chinese translation |

## Testing

- Unit test for `validateEmail()` with ASCII and non-ASCII inputs
- Frontend test for `EmailInput.vue` validation
- Manual test: attempt signup with `testо@gmail.com` (Cyrillic о)

## Security Considerations

- Existing accounts with non-ASCII emails remain functional (grandfathered)
- New validation applies only to create/update operations
- Backend is the authoritative validation layer; frontend is for UX only
