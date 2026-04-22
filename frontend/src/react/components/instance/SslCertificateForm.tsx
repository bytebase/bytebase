import { Info } from "lucide-react";
import { type DragEvent, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { Badge } from "@/react/components/ui/badge";
import { Input } from "@/react/components/ui/input";
import {
  SegmentedControl,
  type SegmentedControlOption,
} from "@/react/components/ui/segmented-control";
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
  isLocalTlsClientIdentitySupported,
  LOCAL_TLS_CA_SOURCE_FILE_PATH,
  LOCAL_TLS_CA_SOURCE_INLINE_PEM,
  LOCAL_TLS_CA_SOURCE_SYSTEM_TRUST,
  LOCAL_TLS_CLIENT_CERT_SOURCE_FILE_PATH,
  LOCAL_TLS_CLIENT_CERT_SOURCE_INLINE_PEM,
  LOCAL_TLS_CLIENT_CERT_SOURCE_NONE,
  LOCAL_TLS_POSTURE_DISABLED,
  LOCAL_TLS_POSTURE_MUTUAL_TLS,
  LOCAL_TLS_POSTURE_TLS,
  type LocalTlsCaSource,
  type LocalTlsClientCertSource,
  type LocalTlsPosture,
} from "./tls";

interface SslCertificateFormProps {
  useSsl?: boolean;
  onUseSslChange?: (val: boolean) => void;
  caSource?: LocalTlsCaSource;
  onCaSourceChange?: (val: LocalTlsCaSource) => void;
  clientCertSource?: LocalTlsClientCertSource;
  onClientCertSourceChange?: (val: LocalTlsClientCertSource) => void;
  posture?: LocalTlsPosture;
  onPostureChange?: (val: LocalTlsPosture) => void;
  isSaaSMode?: boolean;
  ca?: string;
  hasCa?: boolean;
  onCaChange?: (val: string) => void;
  caPath?: string;
  hasCaPath?: boolean;
  onCaPathChange?: (val: string) => void;
  cert?: string;
  hasCert?: boolean;
  onCertChange?: (val: string) => void;
  certPath?: string;
  hasCertPath?: boolean;
  onCertPathChange?: (val: string) => void;
  sslKey?: string;
  hasKey?: boolean;
  onKeyChange?: (val: string) => void;
  keyPath?: string;
  hasKeyPath?: boolean;
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
  isSaaSMode = false,
}: {
  value: LocalTlsCaSource;
  onChange: (value: LocalTlsCaSource) => void;
  disabled?: boolean;
  isSaaSMode?: boolean;
}) {
  const { t } = useTranslation();
  const options: SegmentedControlOption<LocalTlsCaSource>[] = [
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
      disabled: isSaaSMode,
      tooltip: isSaaSMode
        ? t("data-source.ssl.ca-source.file-path-unavailable-saas")
        : undefined,
    },
  ];

  return (
    <SegmentedControl
      value={value}
      onValueChange={onChange}
      ariaLabel={t("data-source.ssl.ca-source.self")}
      options={options}
      disabled={disabled}
      className="mt-2"
    />
  );
}

function ClientCertSourceSelector({
  value,
  onChange,
  disabled = false,
  isSaaSMode = false,
  allowNone = false,
}: {
  value: LocalTlsClientCertSource;
  onChange: (value: LocalTlsClientCertSource) => void;
  disabled?: boolean;
  isSaaSMode?: boolean;
  allowNone?: boolean;
}) {
  const { t } = useTranslation();
  const options: SegmentedControlOption<LocalTlsClientCertSource>[] = [
    ...(allowNone
      ? [
          {
            value: LOCAL_TLS_CLIENT_CERT_SOURCE_NONE,
            label: t("data-source.ssl.client-cert-source.none"),
          },
        ]
      : []),
    {
      value: LOCAL_TLS_CLIENT_CERT_SOURCE_INLINE_PEM,
      label: t("data-source.ssl.client-cert-source.inline-pem"),
    },
    {
      value: LOCAL_TLS_CLIENT_CERT_SOURCE_FILE_PATH,
      label: t("data-source.ssl.client-cert-source.file-path"),
      disabled: isSaaSMode,
      tooltip: isSaaSMode
        ? t("data-source.ssl.client-cert-source.file-path-unavailable-saas")
        : undefined,
    },
  ];

  return (
    <SegmentedControl
      value={value}
      onValueChange={(next) => onChange(next)}
      ariaLabel={t("data-source.ssl.client-cert-source.self")}
      options={options}
      disabled={disabled}
      className="mt-2"
    />
  );
}

export function SslCertificateForm({
  useSsl,
  onUseSslChange,
  caSource,
  onCaSourceChange,
  clientCertSource,
  onClientCertSourceChange,
  posture,
  onPostureChange,
  isSaaSMode = false,
  ca = "",
  hasCa = false,
  onCaChange,
  caPath = "",
  hasCaPath = false,
  onCaPathChange,
  cert = "",
  hasCert = false,
  onCertChange,
  certPath = "",
  hasCertPath = false,
  onCertPathChange,
  sslKey = "",
  hasKey = false,
  onKeyChange,
  keyPath = "",
  hasKeyPath = false,
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
  const resolvedConfiguredLabel = t("data-source.ssl.configured");
  const resolvedCaHint = t("data-source.ssl.ca-empty-uses-system-trust");
  const resolvedCaPlaceholder = t("data-source.ssl.ca-placeholder");
  const resolvedCertPlaceholder = t("data-source.ssl.client-cert-placeholder");
  const resolvedKeyPlaceholder = t("data-source.ssl.client-key-placeholder");
  const resolvedUseSsl = useSsl ?? true;
  const showUseSslSwitch = useSsl !== undefined && !!onUseSslChange;
  const showCaSourceUi = caSource !== undefined && !!onCaSourceChange;
  const showClientCertSourceUi =
    clientCertSource !== undefined && !!onClientCertSourceChange;
  const showPostureUi =
    posture !== undefined &&
    !!onPostureChange &&
    showCaSourceUi &&
    showClientCertSourceUi;
  const showPerGroupSourceUi = showCaSourceUi || showClientCertSourceUi;

  const hasClientIdentityMaterial = !!(
    cert ||
    sslKey ||
    certPath ||
    keyPath ||
    hasCert ||
    hasKey ||
    hasCertPath ||
    hasKeyPath
  );
  const showKeyAndCertFields =
    showKeyAndCert ||
    ![Engine.MSSQL].includes(engineType) ||
    hasClientIdentityMaterial;

  const inferredCaSource = getLocalTlsCaSource({
    useSsl: true,
    sslCa: ca,
    sslCert: cert,
    sslKey,
    sslCaPath: caPath,
    sslCertPath: certPath,
    sslKeyPath: keyPath,
    sslCaSet: false,
    sslCertSet: false,
    sslKeySet: false,
    sslCaPathSet: false,
    sslCertPathSet: false,
    sslKeyPathSet: false,
  });
  const inferredClientCertSource = getLocalTlsClientCertSource({
    useSsl: true,
    sslCa: ca,
    sslCert: cert,
    sslKey,
    sslCaPath: caPath,
    sslCertPath: certPath,
    sslKeyPath: keyPath,
    sslCaSet: false,
    sslCertSet: false,
    sslKeySet: false,
    sslCaPathSet: false,
    sslCertPathSet: false,
    sslKeyPathSet: false,
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
  const inferredPosture = resolvedUseSsl
    ? resolvedClientCertSource === LOCAL_TLS_CLIENT_CERT_SOURCE_NONE
      ? LOCAL_TLS_POSTURE_TLS
      : LOCAL_TLS_POSTURE_MUTUAL_TLS
    : LOCAL_TLS_POSTURE_DISABLED;
  const requestedPosture =
    showPostureUi && posture !== undefined ? posture : inferredPosture;
  const supportsClientIdentity =
    showKeyAndCertFields && isLocalTlsClientIdentitySupported(engineType);
  const canShowMutualTls = supportsClientIdentity || hasClientIdentityMaterial;
  const canSelectMutualTls = supportsClientIdentity;
  const resolvedPosture =
    requestedPosture === LOCAL_TLS_POSTURE_MUTUAL_TLS && !canShowMutualTls
      ? LOCAL_TLS_POSTURE_TLS
      : requestedPosture;
  const showConfiguredBadge = (hasStoredValue: boolean, visibleValue: string) =>
    hasStoredValue && !visibleValue;
  const renderLabel = (
    label: string,
    hasStoredValue: boolean,
    visibleValue: string
  ) => (
    <div className="flex items-center gap-x-2">
      <label className="textlabel block">{label}</label>
      {showConfiguredBadge(hasStoredValue, visibleValue) && (
        <Badge data-testid="tls-configured-badge" variant="success">
          {resolvedConfiguredLabel}
        </Badge>
      )}
    </div>
  );
  const renderPostureControl = () => {
    const options: SegmentedControlOption<LocalTlsPosture>[] = [
      {
        value: LOCAL_TLS_POSTURE_DISABLED,
        label: t("data-source.ssl.posture.disabled"),
      },
      {
        value: LOCAL_TLS_POSTURE_TLS,
        label: t("data-source.ssl.posture.tls"),
      },
      {
        value: LOCAL_TLS_POSTURE_MUTUAL_TLS,
        label: t("data-source.ssl.posture.mutual-tls"),
        disabled: !canSelectMutualTls,
        tooltip: !canSelectMutualTls
          ? t("data-source.ssl.mutual-tls-unavailable-engine")
          : undefined,
      },
    ];

    return (
      <div className="flex flex-col gap-y-1">
        <SegmentedControl
          value={resolvedPosture}
          onValueChange={(next) => onPostureChange?.(next)}
          ariaLabel={t("data-source.ssl.posture.self")}
          options={options}
          disabled={disabled}
        />
      </div>
    );
  };

  const renderCaMaterial = () => {
    if (resolvedCaSource === LOCAL_TLS_CA_SOURCE_SYSTEM_TRUST) {
      return <p className="mt-1 text-xs textinfolabel">{resolvedCaHint}</p>;
    }

    if (resolvedCaSource === LOCAL_TLS_CA_SOURCE_FILE_PATH) {
      return (
        <div className="flex flex-col gap-y-1">
          {renderLabel(resolvedCaPathLabel, hasCaPath, caPath)}
          <Input
            data-testid="tls-ca-path-input"
            value={caPath}
            onChange={(e) => onCaPathChange?.(e.target.value)}
            disabled={disabled || isSaaSMode}
            placeholder={resolvedCaPathLabel}
          />
        </div>
      );
    }

    return (
      <div className="flex flex-col gap-y-1">
        {renderLabel(resolvedCaLabel, hasCa, ca)}
        <DroppableTextarea
          value={ca}
          onChange={(val) => onCaChange?.(val)}
          disabled={disabled}
          placeholder={resolvedCaPlaceholder}
        />
      </div>
    );
  };

  const renderClientCertMaterial = (
    source: LocalTlsClientCertSource = resolvedClientCertSource
  ) => {
    if (!showKeyAndCertFields || source === LOCAL_TLS_CLIENT_CERT_SOURCE_NONE) {
      return null;
    }

    if (source === LOCAL_TLS_CLIENT_CERT_SOURCE_FILE_PATH) {
      return (
        <div className="flex flex-col gap-y-2">
          <div className="flex flex-col gap-y-1">
            {renderLabel(resolvedCertPathLabel, hasCertPath, certPath)}
            <Input
              data-testid="tls-cert-path-input"
              value={certPath}
              onChange={(e) => onCertPathChange?.(e.target.value)}
              disabled={disabled || isSaaSMode}
              placeholder={resolvedCertPathLabel}
            />
          </div>
          <div className="flex flex-col gap-y-1">
            {renderLabel(resolvedKeyPathLabel, hasKeyPath, keyPath)}
            <Input
              data-testid="tls-key-path-input"
              value={keyPath}
              onChange={(e) => onKeyPathChange?.(e.target.value)}
              disabled={disabled || isSaaSMode}
              placeholder={resolvedKeyPathLabel}
            />
          </div>
        </div>
      );
    }

    return (
      <div className="flex flex-col gap-y-2">
        <div className="flex flex-col gap-y-1">
          {renderLabel(resolvedKeyLabel, hasKey, sslKey)}
          <DroppableTextarea
            value={sslKey}
            onChange={(val) => onKeyChange?.(val)}
            disabled={disabled}
            placeholder={resolvedKeyPlaceholder}
          />
        </div>
        <div className="flex flex-col gap-y-1">
          {renderLabel(resolvedCertLabel, hasCert, cert)}
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
            {renderLabel(resolvedCaPathLabel, hasCaPath, caPath)}
            <Input
              data-testid="tls-ca-path-input"
              value={caPath}
              onChange={(e) => onCaPathChange?.(e.target.value)}
              disabled={disabled || isSaaSMode}
              placeholder={resolvedCaPathLabel}
            />
          </div>
          {showKeyAndCertFields && (
            <div className="flex flex-col gap-y-1">
              {renderLabel(resolvedCertPathLabel, hasCertPath, certPath)}
              <Input
                data-testid="tls-cert-path-input"
                value={certPath}
                onChange={(e) => onCertPathChange?.(e.target.value)}
                disabled={disabled || isSaaSMode}
                placeholder={resolvedCertPathLabel}
              />
            </div>
          )}
          {showKeyAndCertFields && (
            <div className="flex flex-col gap-y-1">
              {renderLabel(resolvedKeyPathLabel, hasKeyPath, keyPath)}
              <Input
                data-testid="tls-key-path-input"
                value={keyPath}
                onChange={(e) => onKeyPathChange?.(e.target.value)}
                disabled={disabled || isSaaSMode}
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
          <TabsTrigger value="CA">
            <span className="inline-flex items-center gap-x-2">
              {resolvedCaLabel}
              {showConfiguredBadge(hasCa, ca) && (
                <Badge data-testid="tls-configured-badge" variant="success">
                  {resolvedConfiguredLabel}
                </Badge>
              )}
            </span>
          </TabsTrigger>
          {showKeyAndCertFields && (
            <TabsTrigger value="KEY">
              <span className="inline-flex items-center gap-x-2">
                {resolvedKeyLabel}
                {showConfiguredBadge(hasKey, sslKey) && (
                  <Badge data-testid="tls-configured-badge" variant="success">
                    {resolvedConfiguredLabel}
                  </Badge>
                )}
              </span>
            </TabsTrigger>
          )}
          {showKeyAndCertFields && (
            <TabsTrigger value="CERT">
              <span className="inline-flex items-center gap-x-2">
                {resolvedCertLabel}
                {showConfiguredBadge(hasCert, cert) && (
                  <Badge data-testid="tls-configured-badge" variant="success">
                    {resolvedConfiguredLabel}
                  </Badge>
                )}
              </span>
            </TabsTrigger>
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

  const renderVerifyControl = () => {
    if (!showVerify) {
      return null;
    }

    return (
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
    );
  };

  const renderPostureMaterial = () => {
    if (resolvedPosture === LOCAL_TLS_POSTURE_DISABLED) {
      return null;
    }

    const clientIdentitySource =
      resolvedClientCertSource === LOCAL_TLS_CLIENT_CERT_SOURCE_NONE
        ? LOCAL_TLS_CLIENT_CERT_SOURCE_INLINE_PEM
        : resolvedClientCertSource;

    return (
      <>
        <fieldset className="flex flex-col gap-y-2 rounded-xs border border-control-border px-3 py-2">
          <legend className="px-1 textlabel">
            {t("data-source.ssl.server-identity")}
          </legend>
          {renderVerifyControl()}
          {!verify && (
            <p className="text-xs textinfolabel">
              {t("data-source.ssl.verification-disabled-description")}
            </p>
          )}
          <div className="flex flex-col gap-y-2">
            {showCaSourceUi && (
              <div className="flex flex-col gap-y-1">
                <label className="textlabel block">
                  {t("data-source.ssl.ca-source.self")}
                </label>
                <CaSourceSelector
                  value={resolvedCaSource}
                  onChange={onCaSourceChange!}
                  disabled={disabled}
                  isSaaSMode={isSaaSMode}
                />
              </div>
            )}
            {renderCaMaterial()}
          </div>
        </fieldset>

        {resolvedPosture === LOCAL_TLS_POSTURE_MUTUAL_TLS && (
          <fieldset className="flex flex-col gap-y-2 rounded-xs border border-control-border px-3 py-2">
            <legend className="px-1 textlabel">
              {t("data-source.ssl.client-identity")}
            </legend>
            <div className="flex flex-col gap-y-2">
              {showClientCertSourceUi && (
                <div className="flex flex-col gap-y-1">
                  <label className="textlabel block">
                    {t("data-source.ssl.client-cert-source.self")}
                  </label>
                  <ClientCertSourceSelector
                    value={clientIdentitySource}
                    onChange={onClientCertSourceChange!}
                    disabled={disabled}
                    isSaaSMode={isSaaSMode}
                  />
                </div>
              )}
              {renderClientCertMaterial(clientIdentitySource)}
            </div>
          </fieldset>
        )}
      </>
    );
  };

  return (
    <div className="mt-2 flex flex-col gap-y-3">
      {showPostureUi && (
        <>
          <div className="flex flex-col gap-y-2">
            <label className="textlabel block">
              {t("data-source.ssl.connection-security")}
            </label>
            {renderPostureControl()}
          </div>
          {renderPostureMaterial()}
        </>
      )}

      {!showPostureUi && showUseSslSwitch && (
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

      {!showPostureUi && resolvedUseSsl && (
        <>
          {renderVerifyControl()}

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
                      onChange={onCaSourceChange!}
                      disabled={disabled}
                      isSaaSMode={isSaaSMode}
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
                        onChange={onClientCertSourceChange!}
                        disabled={disabled}
                        isSaaSMode={isSaaSMode}
                        allowNone
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
