import {
  CircleAlert,
  CircleCheck,
  CircleHelp,
  Eye,
  EyeOff,
} from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Input } from "@/react/components/ui/input";
import { Tooltip } from "@/react/components/ui/tooltip";
import type { WorkspaceProfileSetting_PasswordRestriction } from "@/types/proto-es/v1/setting_service_pb";

interface PasswordCheck {
  text: string;
  matched: boolean;
}

/**
 * Returns a list of password errors. Empty array means valid.
 * Exported for parent components to check validity without a ref.
 */
export function getPasswordErrors(
  password: string,
  passwordConfirm: string,
  restriction?: WorkspaceProfileSetting_PasswordRestriction
): { hasHint: boolean; hasMismatch: boolean } {
  if (!password) {
    return { hasHint: false, hasMismatch: false };
  }

  const minLength = restriction?.minLength ?? 8;
  const checks: boolean[] = [password.length >= minLength];

  if (restriction?.requireNumber) {
    checks.push(/[0-9]+/.test(password));
  }
  if (restriction?.requireUppercaseLetter) {
    checks.push(/[A-Z]+/.test(password));
  } else if (restriction?.requireLetter) {
    checks.push(/[a-zA-Z]+/.test(password));
  }
  if (restriction?.requireSpecialCharacter) {
    checks.push(/[!@#$%^&*()_+\-=[\]{};':"\\|,.<>/?]+/.test(password));
  }

  const hasHint = checks.some((c) => !c);
  const hasMismatch = password !== passwordConfirm;
  return { hasHint, hasMismatch };
}

interface UserPasswordSectionProps {
  password: string;
  passwordConfirm: string;
  onPasswordChange: (value: string) => void;
  onPasswordConfirmChange: (value: string) => void;
  passwordRestriction?: WorkspaceProfileSetting_PasswordRestriction;
  disabled?: boolean;
}

export function UserPasswordSection({
  password,
  passwordConfirm,
  onPasswordChange,
  onPasswordConfirmChange,
  passwordRestriction,
  disabled,
}: UserPasswordSectionProps) {
  const { t } = useTranslation();
  const [showPassword, setShowPassword] = useState(false);

  const passwordChecks = useMemo((): PasswordCheck[] => {
    const minLength = passwordRestriction?.minLength ?? 8;
    const checks: PasswordCheck[] = [
      {
        text: t("settings.general.workspace.password-restriction.min-length", {
          min: minLength,
        }),
        matched: password.length >= minLength,
      },
    ];

    if (passwordRestriction?.requireNumber) {
      checks.push({
        text: t(
          "settings.general.workspace.password-restriction.require-number"
        ),
        matched: /[0-9]+/.test(password),
      });
    }
    if (passwordRestriction?.requireUppercaseLetter) {
      checks.push({
        text: t(
          "settings.general.workspace.password-restriction.require-uppercase-letter"
        ),
        matched: /[A-Z]+/.test(password),
      });
    } else if (passwordRestriction?.requireLetter) {
      checks.push({
        text: t(
          "settings.general.workspace.password-restriction.require-letter"
        ),
        matched: /[a-zA-Z]+/.test(password),
      });
    }
    if (passwordRestriction?.requireSpecialCharacter) {
      checks.push({
        text: t(
          "settings.general.workspace.password-restriction.require-special-character"
        ),
        matched: /[!@#$%^&*()_+\-=[\]{};':"\\|,.<>/?]+/.test(password),
      });
    }

    return checks;
  }, [password, passwordRestriction, t]);

  const passwordHint = password
    ? passwordChecks.some((c) => !c.matched)
    : false;
  const passwordMismatch = password ? password !== passwordConfirm : false;

  const toggleVisibility = () => setShowPassword((v) => !v);

  return (
    <div className="flex flex-col gap-y-6">
      {/* Password field */}
      <div>
        <div>
          <label className="block text-sm font-medium leading-5 text-control">
            {t("settings.profile.password")}{" "}
            <span className="text-error">*</span>
          </label>
          <span
            className={`flex items-center gap-x-1 text-sm ${
              passwordHint ? "text-error" : "text-control-light"
            }`}
          >
            {t("settings.profile.password-hint")}
            <Tooltip
              content={
                <ul className="list-disc">
                  {passwordChecks.map((check, i) => (
                    <li key={i} className="flex gap-x-1 items-center">
                      {check.matched ? (
                        <CircleCheck className="w-4 text-green-400" />
                      ) : (
                        <CircleAlert className="w-4 text-red-400" />
                      )}
                      {check.text}
                    </li>
                  ))}
                </ul>
              }
            >
              <CircleHelp className="w-4" />
            </Tooltip>
          </span>
        </div>
        <div className="w-full flex flex-col gap-y-1">
          <div className="mt-1 relative flex flex-row items-center">
            <Input
              value={password}
              type={showPassword ? "text" : "password"}
              autoComplete="new-password"
              placeholder={t("common.sensitive-placeholder")}
              disabled={disabled}
              className={passwordHint ? "border-error focus:ring-error" : ""}
              onChange={(e) => onPasswordChange(e.target.value)}
            />
            <button
              type="button"
              tabIndex={-1}
              className="hover:cursor-pointer absolute right-3"
              onClick={toggleVisibility}
            >
              {showPassword ? (
                <Eye className="w-4 h-4" />
              ) : (
                <EyeOff className="w-4 h-4" />
              )}
            </button>
          </div>
        </div>
      </div>

      {/* Confirm field */}
      <div>
        <label className="block text-sm font-medium leading-5 text-control">
          {t("settings.profile.password-confirm")}{" "}
          <span className="text-error">*</span>
        </label>
        <div className="w-full mt-1 flex flex-col justify-start items-start">
          <div className="w-full relative flex flex-row items-center">
            <Input
              value={passwordConfirm}
              type={showPassword ? "text" : "password"}
              autoComplete="new-password"
              placeholder={t("settings.profile.password-confirm-placeholder")}
              disabled={disabled}
              className={
                passwordMismatch ? "border-error focus:ring-error" : ""
              }
              onChange={(e) => onPasswordConfirmChange(e.target.value)}
            />
            <button
              type="button"
              tabIndex={-1}
              className="hover:cursor-pointer absolute right-3"
              onClick={toggleVisibility}
            >
              {showPassword ? (
                <Eye className="w-4 h-4" />
              ) : (
                <EyeOff className="w-4 h-4" />
              )}
            </button>
          </div>
          {passwordMismatch && (
            <span className="text-error text-sm mt-1 pl-1">
              {t("settings.profile.password-mismatch")}
            </span>
          )}
        </div>
      </div>
    </div>
  );
}
