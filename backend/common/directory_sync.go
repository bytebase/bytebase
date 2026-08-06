// nolint:revive // Package name "common" is the established repository-wide name.
package common

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashDirectorySyncToken returns the hex-encoded SHA-256 of a SCIM directory
// sync token. Only this hash is persisted; the plaintext is shown once at
// rotation and never stored.
//
// Every producer and consumer of the stored value must go through this function.
// There are three implementations of the same hash in the tree — the rotate
// handler, the SCIM authentication path, and the SQL in migration
// 3.22/0003##hash_directory_sync_token.sql — and if any of them disagrees on the
// digest or the encoding, every configured Okta/Entra integration stops
// authenticating. Keeping the two Go call sites on one function reduces that to
// a single SQL-versus-Go agreement, which
// TestHashDirectorySyncTokenMigration pins.
//
// SHA-256 rather than a password KDF is deliberate: the token is 122 bits from
// crypto/rand, not a human-chosen secret, so key stretching buys nothing and
// would add latency to every SCIM request.
func HashDirectorySyncToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
