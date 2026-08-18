import { describe, expect, test } from "vitest";
import { getPasswordErrors } from "@/routes/workspace/profile/UserPasswordSection";
import type { WorkspaceProfileSetting_PasswordRestriction } from "@/types/proto-es/v1/setting_service_pb";
import { generatePassword } from "./generatePassword";

// Every combination of the four boolean rules, since they interact: the
// uppercase rule shadows the letter rule in the validator's if/else ladder.
const FLAGS = [
  "requireNumber",
  "requireLetter",
  "requireUppercaseLetter",
  "requireSpecialCharacter",
] as const;

function combinations(): WorkspaceProfileSetting_PasswordRestriction[] {
  const out: WorkspaceProfileSetting_PasswordRestriction[] = [];
  for (let mask = 0; mask < 1 << FLAGS.length; mask++) {
    const restriction = { minLength: 12 } as Record<string, unknown>;
    FLAGS.forEach((flag, i) => {
      restriction[flag] = (mask & (1 << i)) !== 0;
    });
    out.push(restriction as WorkspaceProfileSetting_PasswordRestriction);
  }
  return out;
}

describe("generatePassword", () => {
  // The generator hands its output to an admin as a working credential. If it
  // can emit a password the server rejects, that failure lands mid-recovery,
  // on the one path that exists because the user is already locked out.
  test("always satisfies the restriction it was given", () => {
    for (const restriction of combinations()) {
      for (let i = 0; i < 50; i++) {
        const password = generatePassword(restriction);
        const errors = getPasswordErrors(password, password, restriction);
        expect(
          errors.hasHint,
          `restriction ${JSON.stringify(restriction)} produced ${password}`
        ).toBe(false);
        expect(password.length).toBeGreaterThanOrEqual(restriction.minLength);
      }
    }
  });

  test("honors a minimum length above its own floor", () => {
    const restriction = {
      minLength: 40,
      requireNumber: true,
      requireUppercaseLetter: true,
      requireSpecialCharacter: true,
    } as WorkspaceProfileSetting_PasswordRestriction;
    expect(generatePassword(restriction)).toHaveLength(40);
  });

  test("still produces a usable password with no restriction configured", () => {
    const password = generatePassword(undefined);
    expect(password.length).toBeGreaterThanOrEqual(16);
    expect(getPasswordErrors(password, password, undefined).hasHint).toBe(
      false
    );
  });

  test("does not repeat itself", () => {
    const seen = new Set(
      Array.from({ length: 100 }, () => generatePassword(undefined))
    );
    expect(seen.size).toBe(100);
  });

  test("omits characters that are misread when relayed by hand", () => {
    // The admin reads this off a screen and types or dictates it, so the
    // l/1/I and O/0 families are excluded on purpose.
    const password = Array.from({ length: 50 }, () =>
      generatePassword(undefined)
    ).join("");
    expect(password).not.toMatch(/[lIO01]/);
  });
});
