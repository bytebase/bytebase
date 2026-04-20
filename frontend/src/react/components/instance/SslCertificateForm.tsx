import { Info } from "lucide-react";
import { type DragEvent, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { Input } from "@/react/components/ui/input";
import { RadioGroup, RadioGroupItem } from "@/react/components/ui/radio-group";
import { Switch } from "@/react/components/ui/switch";
import {
  Tabs,
  TabsList,
  TabsPanel,
  TabsTrigger,
} from "@/react/components/ui/tabs";
import { Tooltip } from "@/react/components/ui/tooltip";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  getLocalTlsSource,
  LOCAL_TLS_SOURCE_DISABLED,
  LOCAL_TLS_SOURCE_FILE_PATH,
  type LocalTlsSource,
} from "./tls";

interface SslCertificateFormProps {
  source?: LocalTlsSource;
  onSourceChange?: (val: LocalTlsSource) => void;
  useSsl?: boolean;
  onUseSslChange?: (val: boolean) => void;
  ca?: string;
  onCaChange?: (val: string) => void;
  caPath?: string;
  onCaPathChange?: (val: string) => void;
  cert?: string;
  onCertChange?: (val: string) => void;
  certPath?: string;
  onCertPathChange?: (val: string) => void;
  sslKey?: string;
  onKeyChange?: (val: string) => void;
  keyPath?: string;
  onKeyPathChange?: (val: string) => void;
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
      className="w-full h-24 whitespace-pre-wrap resize-none rounded-xs border border-control-border bg-background px-3 py-2 text-sm focus:outline-hidden focus:border-accent disabled:cursor-not-allowed disabled:opacity-50"
    />
  );
}

function TlsSourceSelector({
  value,
  onChange,
  disabled = false,
}: {
  value: LocalTlsSource;
  onChange: (value: LocalTlsSource) => void;
  disabled?: boolean;
}) {
  const { t } = useTranslation();
  const options: { value: LocalTlsSource; label: string }[] = [
    { value: "DISABLED", label: t("data-source.ssl.source.disabled") },
    { value: "INLINE_PEM", label: t("data-source.ssl.source.inline-pem") },
    { value: "FILE_PATH", label: t("data-source.ssl.source.file-path") },
  ];

  return (
    <RadioGroup
      value={value}
      onValueChange={(next) => onChange(next as LocalTlsSource)}
      aria-label={t("data-source.ssl.source.self")}
      className="mt-2 gap-x-4"
    >
      {options.map((option) => (
        <RadioGroupItem
          key={option.value}
          value={option.value}
          disabled={disabled}
        >
          {option.label}
        </RadioGroupItem>
      ))}
    </RadioGroup>
  );
}

export function SslCertificateForm({
  source,
  onSourceChange,
  ca = "",
  onCaChange,
  caPath = "",
  onCaPathChange,
  cert = "",
  onCertChange,
  certPath = "",
  onCertPathChange,
  sslKey = "",
  onKeyChange,
  keyPath = "",
  onKeyPathChange,
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
  const resolvedCaPathLabel = t("data-source.ssl.ca-path");
  const resolvedCertPathLabel = t("data-source.ssl.client-cert-path");
  const resolvedKeyPathLabel = t("data-source.ssl.client-key-path");
  const resolvedCaHint = t("data-source.ssl.ca-empty-uses-system-trust");
  const showLocalSourceUi = source !== undefined && !!onSourceChange;

  const showKeyAndCertFields =
    showKeyAndCert || ![Engine.MSSQL].includes(engineType);
  const localSource = showLocalSourceUi
    ? source!
    : getLocalTlsSource({
        useSsl: true,
        sslCa: ca,
        sslCert: cert,
        sslKey,
        sslCaPath: caPath,
        sslCertPath: certPath,
        sslKeyPath: keyPath,
        hasSslCaPath: false,
        hasSslCertPath: false,
        hasSslKeyPath: false,
      });

  return (
    <div className="mt-2 flex flex-col gap-y-1">
      {showLocalSourceUi && (
        <div className="flex flex-col gap-y-1">
          <label className="textlabel block">
            {t("data-source.ssl.source.self")}
          </label>
          <TlsSourceSelector
            value={localSource}
            onChange={onSourceChange}
            disabled={disabled}
          />
        </div>
      )}

      {showLocalSourceUi && localSource === LOCAL_TLS_SOURCE_DISABLED ? null : (
        <>
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
                  <Info className="size-4 text-warning" />
                </Tooltip>
              )}
            </div>
          )}

          {localSource === LOCAL_TLS_SOURCE_FILE_PATH ? (
            <div className="flex flex-col gap-y-2">
              <div className="flex flex-col gap-y-1">
                <label className="textlabel block">{resolvedCaPathLabel}</label>
                <Input
                  value={caPath}
                  onChange={(e) => onCaPathChange?.(e.target.value)}
                  disabled={disabled}
                  placeholder={resolvedCaPathLabel}
                />
              </div>
              {showKeyAndCertFields && (
                <div className="flex flex-col gap-y-1">
                  <label className="textlabel block">
                    {resolvedCertPathLabel}
                  </label>
                  <Input
                    value={certPath}
                    onChange={(e) => onCertPathChange?.(e.target.value)}
                    disabled={disabled}
                    placeholder={resolvedCertPathLabel}
                  />
                </div>
              )}
              {showKeyAndCertFields && (
                <div className="flex flex-col gap-y-1">
                  <label className="textlabel block">
                    {resolvedKeyPathLabel}
                  </label>
                  <Input
                    value={keyPath}
                    onChange={(e) => onKeyPathChange?.(e.target.value)}
                    disabled={disabled}
                    placeholder={resolvedKeyPathLabel}
                  />
                </div>
              )}
            </div>
          ) : (
            <Tabs defaultValue="CA">
              <TabsList>
                <TabsTrigger value="CA">{resolvedCaLabel}</TabsTrigger>
                {showKeyAndCertFields && (
                  <TabsTrigger value="KEY">{resolvedKeyLabel}</TabsTrigger>
                )}
                {showKeyAndCertFields && (
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
                <p className="mt-1 text-xs textinfolabel">{resolvedCaHint}</p>
              </TabsPanel>
              {showKeyAndCertFields && (
                <TabsPanel value="KEY" className="pt-1">
                  <DroppableTextarea
                    value={sslKey}
                    onChange={(val) => onKeyChange?.(val)}
                    disabled={disabled}
                    placeholder="Input or drag and drop YOUR_CLIENT_KEY"
                  />
                </TabsPanel>
              )}
              {showKeyAndCertFields && (
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
          )}
        </>
      )}
    </div>
  );
}
