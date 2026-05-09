import { type DragEvent, useCallback, useEffect, useId, useState } from "react";
import { useTranslation } from "react-i18next";
import { Input } from "@/react/components/ui/input";
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
  instance: _instance,
  disabled = false,
  onChange,
}: SshConnectionFormProps) {
  const { t } = useTranslation();
  const radioGroupId = useId();
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
    <>
      {/* SSH type radio buttons */}
      <div className="flex flex-row items-center gap-x-4 mt-2">
        {SSH_TYPES.map((type) => {
          const id = `${radioGroupId}-${type}`;
          return (
            <label
              key={type}
              htmlFor={id}
              className="flex items-center gap-x-1.5 cursor-pointer text-sm text-main"
            >
              <input
                id={id}
                type="radio"
                name={radioGroupId}
                value={type}
                checked={sshType === type}
                disabled={disabled}
                onChange={() => handleSelectType(type)}
                className="accent-accent"
              />
              {getSshTypeLabel(type)}
            </label>
          );
        })}
      </div>

      {sshType !== "NONE" && (
        <>
          {/* Host and Port */}
          <div className="sm:col-span-1 sm:col-start-1 mt-4 grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-4">
            <div className="sm:col-span-3 sm:col-start-1">
              <label htmlFor="sshHost" className="textlabel block">
                {t("data-source.ssh.host")}
              </label>
              <Input
                id="sshHost"
                className="mt-2 w-full"
                value={value.sshHost}
                disabled={disabled}
                onChange={(e) => onChange({ sshHost: e.target.value })}
              />
            </div>
            <div className="sm:col-span-1">
              <label htmlFor="sshPort" className="textlabel block">
                {t("data-source.ssh.port")}
              </label>
              <Input
                id="sshPort"
                className="mt-2 w-full"
                value={value.sshPort}
                disabled={disabled}
                onChange={(e) => {
                  const val = e.target.value;
                  if (val === "" || /^\d+$/.test(val)) {
                    onChange({ sshPort: val });
                  }
                }}
              />
            </div>
          </div>

          {/* User and Password */}
          <div className="mt-2 grid grid-cols-1 gap-y-2 gap-x-4 border-none sm:grid-cols-3">
            <div className="mt-2 sm:col-span-3 sm:col-start-1">
              <label htmlFor="sshUser" className="textlabel block">
                {t("data-source.ssh.user")}
              </label>
              <Input
                id="sshUser"
                className="mt-2 w-full"
                value={value.sshUser}
                disabled={disabled}
                onChange={(e) => onChange({ sshUser: e.target.value })}
              />
            </div>
            <div className="mt-2 sm:col-span-3 sm:col-start-1">
              <label htmlFor="sshPassword" className="textlabel block">
                {t("data-source.ssh.password")}
              </label>
              <Input
                id="sshPassword"
                className="mt-2 w-full"
                placeholder={t("instance.password-write-only")}
                value={value.sshPassword}
                disabled={disabled}
                onChange={(e) => onChange({ sshPassword: e.target.value })}
              />
            </div>
          </div>

          {/* Private Key textarea with drag-and-drop */}
          <div className="mt-4 sm:col-span-3 sm:col-start-1">
            <div className="mt-2 sm:col-span-1 sm:col-start-1 flex flex-col">
              <label htmlFor="sshPrivateKey" className="textlabel block">
                {t("data-source.ssh.ssh-key")} ({t("common.optional")})
              </label>
              <textarea
                id="sshPrivateKey"
                className="w-full h-24 mt-2 whitespace-pre-wrap rounded-xs border border-control-border bg-transparent px-3 py-2 text-sm text-main transition-colors placeholder:text-control-placeholder focus:outline-hidden focus:ring-2 focus:ring-accent focus:border-accent disabled:cursor-not-allowed disabled:bg-control-bg disabled:opacity-50 resize-none"
                value={value.sshPrivateKey}
                disabled={disabled}
                placeholder={t("common.sensitive-placeholder")}
                onDrop={handleDrop}
                onDragOver={handleDragOver}
                onChange={(e) => onChange({ sshPrivateKey: e.target.value })}
              />
            </div>
          </div>
        </>
      )}
    </>
  );
}
