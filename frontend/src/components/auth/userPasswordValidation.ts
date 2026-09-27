import type { WorkspaceProfileSetting_PasswordRestriction } from "@/types/proto-es/v1/setting_service_pb";

export type PasswordCheckItem = {
  readonly key: string;
  readonly matched: boolean;
};

export type PasswordValidation = {
  readonly hint: boolean;
  readonly mismatch: boolean;
  readonly checks: readonly PasswordCheckItem[];
};

export function computePasswordValidation(
  password: string,
  passwordConfirm: string,
  restriction?: WorkspaceProfileSetting_PasswordRestriction
): PasswordValidation {
  const minLength = restriction?.minLength ?? 8;
  const checks: PasswordCheckItem[] = [
    { key: "min-length", matched: password.length >= minLength },
  ];

  if (restriction?.requireNumber) {
    checks.push({
      key: "require-number",
      matched: /[0-9]+/.test(password),
    });
  }
  if (restriction?.requireUppercaseLetter) {
    checks.push({
      key: "require-uppercase-letter",
      matched: /[A-Z]+/.test(password),
    });
  } else if (restriction?.requireLetter) {
    checks.push({
      key: "require-letter",
      matched: /[a-zA-Z]+/.test(password),
    });
  }
  if (restriction?.requireSpecialCharacter) {
    checks.push({
      key: "require-special-character",
      matched: /[!@#$%^&*()_+\-=[\]{};':"\\|,.<>/?]+/.test(password),
    });
  }

  return {
    hint: password.length > 0 && checks.some((c) => !c.matched),
    mismatch: password.length > 0 && password !== passwordConfirm,
    checks,
  };
}

export function passwordCheckMinLength(minLength: number): number {
  return minLength;
}
