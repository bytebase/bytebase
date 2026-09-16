import {
  type DragEvent,
  type ReactNode,
  useCallback,
  useEffect,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import {
  FormField,
  FormLabel,
  ResponsiveFormLayout,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { SegmentedControl } from "@/components/ui/segmented-control";
import { Textarea } from "@/components/ui/textarea";
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
  instance: _instance,
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

  const getSshTypeLabel = (type: SshType): string => {
    if (type === "TUNNEL+PK") {
      return t("data-source.ssh-type.tunnel-and-private-key");
    }
    return t("data-source.ssh-type.none");
  };

  const handleDrop = useCallback(
    (e: DragEvent<HTMLTextAreaElement>) => {
      e.preventDefault();
      const file = e.dataTransfer.files[0];
      if (!file) return;
      const reader = new FileReader();
      reader.onload = () => {
        if (typeof reader.result === "string") {
          onChange({ sshPrivateKey: reader.result });
        }
      };
      reader.readAsText(file);
    },
    [onChange]
  );

  const handleDragOver = useCallback((e: DragEvent<HTMLTextAreaElement>) => {
    e.preventDefault();
  }, []);

  return (
    <div className="flex flex-col gap-4">
      <FormField title={title ?? t("data-source.ssh-connection")}>
        <div className="flex flex-col gap-4">
          <SegmentedControl
            value={sshType}
            onValueChange={(value) => handleSelectType(value as SshType)}
            ariaLabel={t("data-source.ssh-connection")}
            options={SSH_TYPES.map((type) => ({
              value: type,
              label: getSshTypeLabel(type),
            }))}
            size="sm"
            disabled={disabled}
          />
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
                  <Input
                    id="sshPassword"
                    placeholder={t("instance.password-write-only")}
                    value={value.sshPassword}
                    disabled={disabled}
                    onChange={(e) => onChange({ sshPassword: e.target.value })}
                  />
                </FormField>
                <FormField>
                  <FormLabel htmlFor="sshPrivateKey">
                    {t("data-source.ssh.ssh-key")} ({t("common.optional")})
                  </FormLabel>
                  <Textarea
                    id="sshPrivateKey"
                    className="w-full h-24 whitespace-pre-wrap resize-none"
                    value={value.sshPrivateKey}
                    disabled={disabled}
                    placeholder={t("common.sensitive-placeholder")}
                    onDrop={handleDrop}
                    onDragOver={handleDragOver}
                    onChange={(e) =>
                      onChange({ sshPrivateKey: e.target.value })
                    }
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
