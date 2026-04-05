import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Input } from "@/react/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/react/components/ui/select";
import { useVueState } from "@/react/hooks/useVueState";
import { useSettingV1Store } from "@/store";

// WHATWG HTML spec email validation (lowercase only).
const emailRegex =
  /^[a-z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)*$/;

function isValidEmail(email: string): boolean {
  return emailRegex.test(email);
}

interface EmailInputProps {
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  domain?: string;
  showDomain?: boolean;
}

export function EmailInput({
  value,
  onChange,
  disabled = false,
  domain: domainProp,
  showDomain = false,
}: EmailInputProps) {
  const { t } = useTranslation();
  const settingV1Store = useSettingV1Store();

  const enforceIdentityDomain = useVueState(
    () => settingV1Store.workspaceProfile.enforceIdentityDomain
  );
  const workspaceDomains = useVueState(
    () => settingV1Store.workspaceProfile.domains
  );

  const enforceDomain = enforceIdentityDomain || showDomain;

  const domainSelectOptions = useMemo(() => {
    if (domainProp) {
      return [{ label: domainProp, value: domainProp }];
    }
    return workspaceDomains
      .filter((d) => d && d.trim() !== "")
      .map((d) => {
        const v = d.trim();
        return { label: v, value: v };
      });
  }, [domainProp, workspaceDomains]);

  const [localPart, setLocalPart] = useState(() => value.split("@")[0] ?? "");
  const [selectedDomain, setSelectedDomain] = useState(
    () => value.split("@")[1] ?? ""
  );
  const [fullValue, setFullValue] = useState(value);

  // Ensure selectedDomain is valid when options change
  useEffect(() => {
    if (domainSelectOptions.length > 0) {
      if (!domainSelectOptions.find((o) => o.value === selectedDomain)) {
        setSelectedDomain(domainSelectOptions[0].value);
      }
    }
  }, [domainSelectOptions, selectedDomain]);

  // Emit the composed email whenever parts change
  const emitEmail = useCallback(
    (lp: string, dom: string, full: string) => {
      if (enforceDomain) {
        onChange(lp ? `${lp}@${dom}` : "");
      } else {
        onChange(full);
      }
    },
    [enforceDomain, onChange]
  );

  const handleLocalPartChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const v = e.target.value;
      setLocalPart(v);
      emitEmail(v, selectedDomain, fullValue);
    },
    [selectedDomain, fullValue, emitEmail]
  );

  const handleDomainChange = useCallback(
    (val: string | null) => {
      if (val === null) return;
      setSelectedDomain(val);
      emitEmail(localPart, val, fullValue);
    },
    [localPart, fullValue, emitEmail]
  );

  const handleFullChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const v = e.target.value;
      setFullValue(v);
      emitEmail(localPart, selectedDomain, v);
    },
    [localPart, selectedDomain, emitEmail]
  );

  const composedEmail = enforceDomain
    ? `${localPart}@${selectedDomain}`
    : fullValue;
  const hasEmailError =
    composedEmail && composedEmail.includes("@")
      ? !isValidEmail(composedEmail)
      : false;

  if (enforceDomain && !disabled) {
    return (
      <div className="flex flex-col">
        <div className="flex items-center gap-0">
          <Input
            value={localPart}
            onChange={handleLocalPartChange}
            disabled={disabled}
            className={
              hasEmailError
                ? "rounded-r-none border-error focus:ring-error"
                : "rounded-r-none"
            }
          />
          <span className="flex items-center px-2 h-9 border-y border-control-border bg-control-bg text-sm text-control-light">
            @
          </span>
          <Select value={selectedDomain} onValueChange={handleDomainChange}>
            <SelectTrigger className="rounded-l-none border-l-0 min-w-28">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {domainSelectOptions.map((opt) => (
                <SelectItem key={opt.value} value={opt.value}>
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        {hasEmailError && (
          <span className="text-error text-sm mt-1">
            {t("common.email-ascii-only")}
          </span>
        )}
      </div>
    );
  }

  return (
    <div className="flex flex-col">
      <Input
        value={fullValue}
        onChange={handleFullChange}
        disabled={disabled}
        className={hasEmailError ? "border-error focus:ring-error" : ""}
      />
      {hasEmailError && (
        <span className="text-error text-sm mt-1">
          {t("common.email-ascii-only")}
        </span>
      )}
    </div>
  );
}
