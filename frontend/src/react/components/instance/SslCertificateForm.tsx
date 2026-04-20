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
  getLocalTlsCaSource,
  getLocalTlsClientCertSource,
  LOCAL_TLS_CA_SOURCE_FILE_PATH,
  LOCAL_TLS_CA_SOURCE_INLINE_PEM,
  LOCAL_TLS_CA_SOURCE_SYSTEM_TRUST,
  LOCAL_TLS_CLIENT_CERT_SOURCE_FILE_PATH,
  LOCAL_TLS_CLIENT_CERT_SOURCE_INLINE_PEM,
  LOCAL_TLS_CLIENT_CERT_SOURCE_NONE,
  type LocalTlsCaSource,
  type LocalTlsClientCertSource,
} from "./tls";

interface SslCertificateFormProps {
  useSsl?: boolean;
  onUseSslChange?: (val: boolean) => void;
  caSource?: LocalTlsCaSource;
  onCaSourceChange?: (val: LocalTlsCaSource) => void;
  clientCertSource?: LocalTlsClientCertSource;
  onClientCertSourceChange?: (val: LocalTlsClientCertSource) => void;
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

function CaSourceSelector({
  value,
  onChange,
  disabled = false,
}: {
  value: LocalTlsCaSource;
  onChange: (value: LocalTlsCaSource) => void;
  disabled?: boolean;
}) {
  const { t } = useTranslation();
  const options: { value: LocalTlsCaSource; label: string }[] = [
    {
      value: LOCAL_TLS_CA_SOURCE_SYSTEM_TRUST,
      label: t("data-source.ssl.ca-source.system-trust"),
    },
    {
      value: LOCAL_TLS_CA_SOURCE_INLINE_PEM,
      label: t("data-source.ssl.ca-source.inline-pem"),
    },
    {
      value: LOCAL_TLS_CA_SOURCE_FILE_PATH,
      label: t("data-source.ssl.ca-source.file-path"),
    },
  ];

  return (
    <RadioGroup
      value={value}
      onValueChange={(next) => onChange(next as LocalTlsCaSource)}
      aria-label={t("data-source.ssl.ca-source.self")}
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

function ClientCertSourceSelector({
  value,
  onChange,
  disabled = false,
}: {
  value: LocalTlsClientCertSource;
  onChange: (value: LocalTlsClientCertSource) => void;
  disabled?: boolean;
}) {
  const { t } = useTranslation();
  const options: { value: LocalTlsClientCertSource; label: string }[] = [
    {
      value: LOCAL_TLS_CLIENT_CERT_SOURCE_NONE,
      label: t("data-source.ssl.client-cert-source.none"),
    },
    {
      value: LOCAL_TLS_CLIENT_CERT_SOURCE_INLINE_PEM,
      label: t("data-source.ssl.client-cert-source.inline-pem"),
    },
    {
      value: LOCAL_TLS_CLIENT_CERT_SOURCE_FILE_PATH,
      label: t("data-source.ssl.client-cert-source.file-path"),
    },
  ];

  return (
    <RadioGroup
      value={value}
      onValueChange={(next) => onChange(next as LocalTlsClientCertSource)}
      aria-label={t("data-source.ssl.client-cert-source.self")}
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
  useSsl,
  onUseSslChange,
  caSource,
  onCaSourceChange,
  clientCertSource,
  onClientCertSourceChange,
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
  const resolvedCaPlaceholder = t("data-source.ssl.ca-placeholder");
  const resolvedCertPlaceholder = t("data-source.ssl.client-cert-placeholder");
  const resolvedKeyPlaceholder = t("data-source.ssl.client-key-placeholder");
  const resolvedUseSsl = useSsl ?? true;
  const showUseSslSwitch = useSsl !== undefined && !!onUseSslChange;
  const showCaSourceUi = caSource !== undefined && !!onCaSourceChange;
  const showClientCertSourceUi =
    clientCertSource !== undefined && !!onClientCertSourceChange;
  const showPerGroupSourceUi = showCaSourceUi || showClientCertSourceUi;

  const showKeyAndCertFields =
    showKeyAndCert || ![Engine.MSSQL].includes(engineType);

  const inferredCaSource = getLocalTlsCaSource({
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
  const inferredClientCertSource = getLocalTlsClientCertSource({
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
  const resolvedCaSource = showCaSourceUi
    ? caSource!
    : inferredCaSource === LOCAL_TLS_CA_SOURCE_SYSTEM_TRUST
      ? LOCAL_TLS_CA_SOURCE_INLINE_PEM
      : inferredCaSource;
  const resolvedClientCertSource = showClientCertSourceUi
    ? clientCertSource!
    : inferredClientCertSource === LOCAL_TLS_CLIENT_CERT_SOURCE_NONE
      ? LOCAL_TLS_CLIENT_CERT_SOURCE_INLINE_PEM
      : inferredClientCertSource;

  const renderCaMaterial = () => {
    if (resolvedCaSource === LOCAL_TLS_CA_SOURCE_SYSTEM_TRUST) {
      return <p className="mt-1 text-xs textinfolabel">{resolvedCaHint}</p>;
    }

    if (resolvedCaSource === LOCAL_TLS_CA_SOURCE_FILE_PATH) {
      return (
        <div className="flex flex-col gap-y-1">
          <label className="textlabel block">{resolvedCaPathLabel}</label>
          <Input
            value={caPath}
            onChange={(e) => onCaPathChange?.(e.target.value)}
            disabled={disabled}
            placeholder={resolvedCaPathLabel}
          />
        </div>
      );
    }

    return (
      <div className="flex flex-col gap-y-1">
        <label className="textlabel block">{resolvedCaLabel}</label>
        <DroppableTextarea
          value={ca}
          onChange={(val) => onCaChange?.(val)}
          disabled={disabled}
          placeholder={resolvedCaPlaceholder}
        />
        <p className="text-xs textinfolabel">{resolvedCaHint}</p>
      </div>
    );
  };

  const renderClientCertMaterial = () => {
    if (
      !showKeyAndCertFields ||
      resolvedClientCertSource === LOCAL_TLS_CLIENT_CERT_SOURCE_NONE
    ) {
      return null;
    }

    if (resolvedClientCertSource === LOCAL_TLS_CLIENT_CERT_SOURCE_FILE_PATH) {
      return (
        <div className="flex flex-col gap-y-2">
          <div className="flex flex-col gap-y-1">
            <label className="textlabel block">{resolvedCertPathLabel}</label>
            <Input
              value={certPath}
              onChange={(e) => onCertPathChange?.(e.target.value)}
              disabled={disabled}
              placeholder={resolvedCertPathLabel}
            />
          </div>
          <div className="flex flex-col gap-y-1">
            <label className="textlabel block">{resolvedKeyPathLabel}</label>
            <Input
              value={keyPath}
              onChange={(e) => onKeyPathChange?.(e.target.value)}
              disabled={disabled}
              placeholder={resolvedKeyPathLabel}
            />
          </div>
        </div>
      );
    }

    return (
      <div className="flex flex-col gap-y-2">
        <div className="flex flex-col gap-y-1">
          <label className="textlabel block">{resolvedKeyLabel}</label>
          <DroppableTextarea
            value={sslKey}
            onChange={(val) => onKeyChange?.(val)}
            disabled={disabled}
            placeholder={resolvedKeyPlaceholder}
          />
        </div>
        <div className="flex flex-col gap-y-1">
          <label className="textlabel block">{resolvedCertLabel}</label>
          <DroppableTextarea
            value={cert}
            onChange={(val) => onCertChange?.(val)}
            disabled={disabled}
            placeholder={resolvedCertPlaceholder}
          />
        </div>
      </div>
    );
  };

  const renderLegacyMaterial = () => {
    if (resolvedCaSource === LOCAL_TLS_CA_SOURCE_FILE_PATH) {
      return (
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
              <label className="textlabel block">{resolvedCertPathLabel}</label>
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
              <label className="textlabel block">{resolvedKeyPathLabel}</label>
              <Input
                value={keyPath}
                onChange={(e) => onKeyPathChange?.(e.target.value)}
                disabled={disabled}
                placeholder={resolvedKeyPathLabel}
              />
            </div>
          )}
        </div>
      );
    }

    return (
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
            placeholder={resolvedCaPlaceholder}
          />
          <p className="mt-1 text-xs textinfolabel">{resolvedCaHint}</p>
        </TabsPanel>
        {showKeyAndCertFields && (
          <TabsPanel value="KEY" className="pt-1">
            <DroppableTextarea
              value={sslKey}
              onChange={(val) => onKeyChange?.(val)}
              disabled={disabled}
              placeholder={resolvedKeyPlaceholder}
            />
          </TabsPanel>
        )}
        {showKeyAndCertFields && (
          <TabsPanel value="CERT" className="pt-1">
            <DroppableTextarea
              value={cert}
              onChange={(val) => onCertChange?.(val)}
              disabled={disabled}
              placeholder={resolvedCertPlaceholder}
            />
          </TabsPanel>
        )}
      </Tabs>
    );
  };

  return (
    <div className="mt-2 flex flex-col gap-y-2">
      {showUseSslSwitch && (
        <div className="flex flex-row items-center gap-x-1">
          <Switch
            checked={resolvedUseSsl}
            onCheckedChange={(val) => onUseSslChange?.(val)}
            disabled={disabled}
          />
          <label className="textlabel block">
            {t("data-source.ssl-connection")}
          </label>
        </div>
      )}

      {resolvedUseSsl && (
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

          {!showPerGroupSourceUi ? (
            renderLegacyMaterial()
          ) : (
            <>
              <div className="flex flex-col gap-y-2">
                {showCaSourceUi && (
                  <div className="flex flex-col gap-y-1">
                    <label className="textlabel block">
                      {t("data-source.ssl.ca-source.self")}
                    </label>
                    <CaSourceSelector
                      value={resolvedCaSource}
                      onChange={onCaSourceChange}
                      disabled={disabled}
                    />
                  </div>
                )}
                {renderCaMaterial()}
              </div>

              {showKeyAndCertFields && (
                <div className="flex flex-col gap-y-2">
                  {showClientCertSourceUi && (
                    <div className="flex flex-col gap-y-1">
                      <label className="textlabel block">
                        {t("data-source.ssl.client-cert-source.self")}
                      </label>
                      <ClientCertSourceSelector
                        value={resolvedClientCertSource}
                        onChange={onClientCertSourceChange}
                        disabled={disabled}
                      />
                    </div>
                  )}
                  {renderClientCertMaterial()}
                </div>
              )}
            </>
          )}
        </>
      )}
    </div>
  );
}
