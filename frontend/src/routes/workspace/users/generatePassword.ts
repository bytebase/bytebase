import type { WorkspaceProfileSetting_PasswordRestriction } from "@/types/proto-es/v1/setting_service_pb";

const LOWER = "abcdefghijkmnopqrstuvwxyz";
const UPPER = "ABCDEFGHJKLMNPQRSTUVWXYZ";
const DIGIT = "23456789";
const SPECIAL = "!@#$%^&*-_=+";

function pick(pool: string, count: number): string[] {
  const out: string[] = [];
  const bytes = new Uint32Array(count);
  crypto.getRandomValues(bytes);
  for (let i = 0; i < count; i++) {
    out.push(pool[bytes[i] % pool.length]);
  }
  return out;
}

/**
 * Build a password that satisfies the workspace restriction.
 *
 * Used when an admin resets a password for someone who cannot sign in. The
 * character pools omit look-alikes (l/1/I, O/0) because the result is read off
 * a screen and typed by hand, or relayed over a phone call.
 */
export function generatePassword(
  restriction: WorkspaceProfileSetting_PasswordRestriction | undefined
): string {
  const required: string[] = [];
  let pool = LOWER;

  if (restriction?.requireUppercaseLetter) {
    required.push(...pick(UPPER, 1));
    pool += UPPER;
  } else if (restriction?.requireLetter) {
    required.push(...pick(LOWER, 1));
  }
  if (restriction?.requireNumber) {
    required.push(...pick(DIGIT, 1));
    pool += DIGIT;
  }
  if (restriction?.requireSpecialCharacter) {
    required.push(...pick(SPECIAL, 1));
    pool += SPECIAL;
  }

  // Comfortably above the floor: the generated value is never memorized, and a
  // longer default costs the admin nothing since they copy it.
  const target = Math.max(restriction?.minLength ?? 8, 16);
  const chars = [...required, ...pick(pool, target - required.length)];

  // Shuffle so the required characters are not always in the leading
  // positions, which would make the pattern guessable across resets.
  const order = new Uint32Array(chars.length);
  crypto.getRandomValues(order);
  for (let i = chars.length - 1; i > 0; i--) {
    const j = order[i] % (i + 1);
    [chars[i], chars[j]] = [chars[j], chars[i]];
  }
  return chars.join("");
}
