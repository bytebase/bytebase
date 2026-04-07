import { Info } from "lucide-react";
import { type DragEvent, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { Switch } from "@/react/components/ui/switch";
import {
  Tabs,
  TabsList,
  TabsPanel,
  TabsTrigger,
} from "@/react/components/ui/tabs";
import { Tooltip } from "@/react/components/ui/tooltip";
import { Engine } from "@/types/proto-es/v1/common_pb";

interface SslCertificateFormProps {
  useSsl?: boolean;
  onUseSslChange?: (val: boolean) => void;
  ca?: string;
  onCaChange?: (val: string) => void;
  cert?: string;
  onCertChange?: (val: string) => void;
  sslKey?: string;
  onKeyChange?: (val: string) => void;
  disabled?: boolean;
  showVerify?: boolean;
  showKeyAndCert?: boolean;
  verifyLabel?: string;
  caLabel?: string;
  certLabel?: string;
  keyLabel?: string;
  showTooltip?: boolean;
  verify?: boolean;
  onVerifyChange?: (val: boolean) => void;
  engineType?: Engine;
}

function DroppableTextarea({
  value,
  onChange,
  disabled,
  placeholder,
}: {
  value: string;
  onChange: (val: string) => void;
  disabled?: boolean;
  placeholder: string;
}) {
  const handleDrop = useCallback(
    (e: DragEvent<HTMLTextAreaElement>) => {
      e.preventDefault();
      const file = e.dataTransfer.files[0];
      if (!file) return;
      const reader = new FileReader();
      reader.onload = () => {
        if (typeof reader.result === "string") {
          onChange(reader.result);
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
    <textarea
      value={value}
      onChange={(e) => onChange(e.target.value)}
      onDrop={handleDrop}
      onDragOver={handleDragOver}
      disabled={disabled}
      placeholder={placeholder}
      className="w-full h-24 whitespace-pre-wrap resize-none rounded border border-control-border bg-white px-3 py-2 text-sm focus:outline-hidden focus:ring-2 focus:ring-accent disabled:cursor-not-allowed disabled:opacity-50"
    />
  );
}

export function SslCertificateForm({
  ca = "",
  onCaChange,
  cert = "",
  onCertChange,
  sslKey = "",
  onKeyChange,
  disabled = false,
  showVerify = true,
  showKeyAndCert = false,
  verifyLabel,
  caLabel,
  certLabel,
  keyLabel,
  showTooltip = true,
  verify = false,
  onVerifyChange,
  engineType = Engine.ENGINE_UNSPECIFIED,
}: SslCertificateFormProps) {
  const { t } = useTranslation();

  const resolvedVerifyLabel =
    verifyLabel ?? t("data-source.ssl.verify-certificate");
  const resolvedCaLabel = caLabel ?? t("data-source.ssl.ca-cert");
  const resolvedCertLabel = certLabel ?? t("data-source.ssl.client-cert");
  const resolvedKeyLabel = keyLabel ?? t("data-source.ssl.client-key");

  const hasSSLKeyField = showKeyAndCert || ![Engine.MSSQL].includes(engineType);
  const hasSSLCertField =
    showKeyAndCert || ![Engine.MSSQL].includes(engineType);

  return (
    <div className="mt-2 flex flex-col gap-y-1">
      {showVerify && (
        <div className="flex flex-row items-center gap-x-1">
          <Switch
            checked={verify}
            onCheckedChange={(val) => onVerifyChange?.(val)}
            disabled={disabled}
          />
          <label className="textlabel block">{resolvedVerifyLabel}</label>
          {showTooltip && (
            <Tooltip
              content={t("data-source.ssl.verify-certificate-tooltip")}
              side="right"
            >
              <Info className="w-4 h-4 text-yellow-600" />
            </Tooltip>
          )}
        </div>
      )}

      <Tabs defaultValue="CA">
        <TabsList>
          <TabsTrigger value="CA">{resolvedCaLabel}</TabsTrigger>
          {hasSSLKeyField && (
            <TabsTrigger value="KEY">{resolvedKeyLabel}</TabsTrigger>
          )}
          {hasSSLCertField && (
            <TabsTrigger value="CERT">{resolvedCertLabel}</TabsTrigger>
          )}
        </TabsList>
        <TabsPanel value="CA" className="pt-1">
          <DroppableTextarea
            value={ca}
            onChange={(val) => onCaChange?.(val)}
            disabled={disabled}
            placeholder="Input or drag and drop YOUR_CA_CERTIFICATE"
          />
        </TabsPanel>
        {hasSSLKeyField && (
          <TabsPanel value="KEY" className="pt-1">
            <DroppableTextarea
              value={sslKey}
              onChange={(val) => onKeyChange?.(val)}
              disabled={disabled}
              placeholder="Input or drag and drop YOUR_CLIENT_KEY"
            />
          </TabsPanel>
        )}
        {hasSSLCertField && (
          <TabsPanel value="CERT" className="pt-1">
            <DroppableTextarea
              value={cert}
              onChange={(val) => onCertChange?.(val)}
              disabled={disabled}
              placeholder="Input or drag and drop YOUR_CLIENT_CERT"
            />
          </TabsPanel>
        )}
      </Tabs>
    </div>
  );
}
