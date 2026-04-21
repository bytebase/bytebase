import {
  CircleAlert,
  CircleCheck,
  CircleHelp,
  Eye,
  EyeOff,
} from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Input } from "@/react/components/ui/input";
import { Tooltip } from "@/react/components/ui/tooltip";
import type { WorkspaceProfileSetting_PasswordRestriction } from "@/types/proto-es/v1/setting_service_pb";
import {
  computePasswordValidation,
  type PasswordValidation,
} from "./userPasswordValidation";

type Props = {
  readonly password: string;
  readonly passwordConfirm: string;
  readonly onPasswordChange: (value: string) => void;
  readonly onPasswordConfirmChange: (value: string) => void;
  readonly passwordRestriction?: WorkspaceProfileSetting_PasswordRestriction;
  readonly disabled?: boolean;
};

const CHECK_KEY_TO_I18N: Record<string, string> = {
  "min-length": "settings.general.workspace.password-restriction.min-length",
  "require-number":
    "settings.general.workspace.password-restriction.require-number",
  "require-letter":
    "settings.general.workspace.password-restriction.require-letter",
  "require-uppercase-letter":
    "settings.general.workspace.password-restriction.require-uppercase-letter",
  "require-special-character":
    "settings.general.workspace.password-restriction.require-special-character",
};

export function UserPasswordFields(props: Props) {
  const { t } = useTranslation();
  const [showPassword, setShowPassword] = useState(false);
  const validation: PasswordValidation = computePasswordValidation(
    props.password,
    props.passwordConfirm,
    props.passwordRestriction
  );

  const minLength = props.passwordRestriction?.minLength ?? 8;

  const restrictionList = (
    <ul className="list-disc pl-4">
      {validation.checks.map((check) => (
        <li key={check.key} className="flex gap-x-1 items-center">
          {check.matched ? (
            <CircleCheck className="w-4 text-success" />
          ) : (
            <CircleAlert className="w-4 text-error" />
          )}
          <span>
            {t(CHECK_KEY_TO_I18N[check.key], {
              min: minLength,
            })}
          </span>
        </li>
      ))}
    </ul>
  );

  return (
    <div className="flex flex-col gap-y-6">
      <div>
        <label className="block text-sm font-medium leading-5 text-control">
          {t("settings.profile.password")}
          <span className="text-error ml-0.5">*</span>
        </label>
        <span
          className={`flex items-center gap-x-1 textinfolabel text-sm! ${
            validation.hint ? "text-error!" : ""
          }`}
        >
          {t("settings.profile.password-hint")}
          <Tooltip content={restrictionList}>
            <CircleHelp className="w-4" />
          </Tooltip>
        </span>
        <div className="w-full flex flex-col gap-y-1">
          <div className="mt-1 relative flex flex-row items-center">
            <Input
              type={showPassword ? "text" : "password"}
              value={props.password}
              placeholder={t("common.sensitive-placeholder")}
              autoComplete="new-password"
              disabled={props.disabled}
              aria-invalid={validation.hint ? "true" : undefined}
              onChange={(e) => props.onPasswordChange(e.target.value)}
              className={validation.hint ? "border-error" : ""}
            />
            <button
              type="button"
              className="hover:cursor-pointer absolute right-3"
              onClick={() => setShowPassword((v) => !v)}
              aria-label="Toggle password visibility"
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
      <div>
        <label className="block text-sm font-medium leading-5 text-control">
          {t("settings.profile.password-confirm")}
          <span className="text-error ml-0.5">*</span>
        </label>
        <div className="w-full mt-1 flex flex-col justify-start items-start">
          <div className="w-full relative flex flex-row items-center">
            <Input
              type={showPassword ? "text" : "password"}
              value={props.passwordConfirm}
              placeholder={t("settings.profile.password-confirm-placeholder")}
              autoComplete="new-password"
              disabled={props.disabled}
              aria-invalid={validation.mismatch ? "true" : undefined}
              onChange={(e) => props.onPasswordConfirmChange(e.target.value)}
              className={validation.mismatch ? "border-error" : ""}
            />
            <button
              type="button"
              className="hover:cursor-pointer absolute right-3"
              onClick={() => setShowPassword((v) => !v)}
              aria-label="Toggle password visibility"
            >
              {showPassword ? (
                <Eye className="w-4 h-4" />
              ) : (
                <EyeOff className="w-4 h-4" />
              )}
            </button>
          </div>
          {validation.mismatch && (
            <span className="text-error text-sm mt-1 pl-1">
              {t("settings.profile.password-mismatch")}
            </span>
          )}
        </div>
      </div>
    </div>
  );
}
