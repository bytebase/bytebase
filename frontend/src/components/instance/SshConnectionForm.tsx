import { type ReactNode, useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { SecretInput } from "@/components/SecretInput";
import {
  FormControlRow,
  FormField,
  FormLabel,
  ResponsiveFormLayout,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";

const SSH_TYPES = ["NONE", "TUNNEL+PK"] as const;
type SshType = (typeof SSH_TYPES)[number];

interface SshValue {
  sshHost: string;
  sshPort: string;
  sshUser: string;
  sshPassword: string;
  sshPrivateKey: string;
}

interface SshConnectionFormProps {
  title?: ReactNode;
  value: SshValue;
  instance?: Instance;
  disabled?: boolean;
  isCreating?: boolean;
  secretValues?: { sshPassword?: string; sshPrivateKey?: string };
  onChange: (value: Partial<SshValue>) => void;
}

function guessSshType(value: Partial<SshValue>): SshType {
  if (
    value.sshHost ||
    value.sshPort ||
    value.sshUser ||
    value.sshPassword ||
    value.sshPrivateKey
  ) {
    return "TUNNEL+PK";
  }
  return "NONE";
}

export function SshConnectionForm({
  value,
  title,
  instance,
  isCreating = instance === undefined,
  secretValues,
  disabled = false,
  onChange,
}: SshConnectionFormProps) {
  const { t } = useTranslation();
  const [sshType, setSshType] = useState<SshType>(() => guessSshType(value));

  // Sync type from props when value changes externally.
  useEffect(() => {
    setSshType(guessSshType(value));
  }, [value]);

  const handleSelectType = useCallback(
    (type: SshType) => {
      setSshType(type);
      if (type === "NONE") {
        onChange({
          sshHost: "",
          sshPort: "",
          sshUser: "",
          sshPassword: "",
          sshPrivateKey: "",
        });
      }
    },
    [onChange]
  );

  return (
    <div className="flex flex-col gap-4">
      <FormField title={title ?? t("data-source.ssh-connection")}>
        <div className="flex flex-col gap-4">
          <FormControlRow className="w-fit">
            <Switch
              checked={sshType === "TUNNEL+PK"}
              onCheckedChange={(checked) =>
                handleSelectType(checked ? "TUNNEL+PK" : "NONE")
              }
              aria-label={t("data-source.ssh-connection")}
              disabled={disabled}
            />
            <span className="text-sm">{t("data-source.ssh-connection")}</span>
          </FormControlRow>
          {sshType !== "NONE" && (
            <ResponsiveFormLayout>
              <fieldset className="flex flex-col gap-4 rounded-xs border border-control-border px-3 py-2">
                <legend className="px-1 textlabel">
                  {t("data-source.ssh-connection")}
                </legend>
                <FormField>
                  <FormLabel htmlFor="sshHost">
                    {t("data-source.ssh.host")}
                  </FormLabel>
                  <div className="flex flex-wrap items-center gap-2">
                    <Input
                      id="sshHost"
                      className="min-w-0 flex-1"
                      value={value.sshHost}
                      disabled={disabled}
                      onChange={(e) => onChange({ sshHost: e.target.value })}
                    />
                    <FormLabel htmlFor="sshPort">
                      {t("data-source.ssh.port")}
                    </FormLabel>
                    <Input
                      id="sshPort"
                      className="w-20 shrink-0"
                      value={value.sshPort}
                      disabled={disabled}
                      onChange={(e) => {
                        const val = e.target.value;
                        if (val === "" || /^\d+$/.test(val))
                          onChange({ sshPort: val });
                      }}
                    />
                  </div>
                </FormField>
                <FormField>
                  <FormLabel htmlFor="sshUser">
                    {t("data-source.ssh.user")}
                  </FormLabel>
                  <Input
                    id="sshUser"
                    value={value.sshUser}
                    disabled={disabled}
                    onChange={(e) => onChange({ sshUser: e.target.value })}
                  />
                </FormField>
                <FormField>
                  <FormLabel htmlFor="sshPassword">
                    {t("data-source.ssh.password")}
                  </FormLabel>
                  <SecretInput
                    id="sshPassword"
                    resetKey={instance?.name}
                    aria-label={t("data-source.ssh.password")}
                    value={
                      isCreating ? value.sshPassword : secretValues?.sshPassword
                    }
                    isCreating={isCreating}
                    disabled={disabled}
                    onValueChange={(password) =>
                      onChange({ sshPassword: password })
                    }
                  />
                </FormField>
                <FormField>
                  <FormLabel htmlFor="sshPrivateKey">
                    {t("data-source.ssh.ssh-key")} ({t("common.optional")})
                  </FormLabel>
                  <SecretInput
                    id="sshPrivateKey"
                    resetKey={instance?.name}
                    aria-label={t("data-source.ssh.ssh-key")}
                    multiline
                    value={
                      isCreating
                        ? value.sshPrivateKey
                        : secretValues?.sshPrivateKey
                    }
                    isCreating={isCreating}
                    disabled={disabled}
                    onValueChange={(key) => onChange({ sshPrivateKey: key })}
                  />
                </FormField>
              </fieldset>
            </ResponsiveFormLayout>
          )}
        </div>
      </FormField>
    </div>
  );
}
