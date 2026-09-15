import { type DragEvent, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { Badge } from "@/components/ui/badge";
import { ResponsiveFormLayout } from "@/components/ui/form";
import {
  SegmentedControl,
  type SegmentedControlOption,
} from "@/components/ui/segmented-control";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsList, TabsPanel, TabsTrigger } from "@/components/ui/tabs";
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
import {
  ValidationField as FormField,
  ValidationInput as Input,
  ValidationTextarea as Textarea,
} from "./ValidationField";

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
  verify?: boolean;
  onVerifyChange?: (val: boolean) => void;
  engineType?: Engine;
}

function DroppableTextarea({
  value,
  onChange,
  disabled,
  placeholder,
  label,
}: {
  value: string;
  onChange: (val: string) => void;
  disabled?: boolean;
  placeholder: string;
  label: string;
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
    <Textarea
      aria-label={label}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      onDrop={handleDrop}
      onDragOver={handleDragOver}
      disabled={disabled}
      placeholder={placeholder}
      className="w-full whitespace-pre-wrap resize-none"
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
      size="sm"
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
      size="sm"
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

  // Configured CA material is still submitted when verification is disabled.
  // Keep its controls reachable so validation cannot block on hidden fields.
  const showCaMaterial = verify || !!(ca || caPath || hasCa || hasCaPath);
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
    <div
      data-slot="form-field-title"
      className="flex items-center gap-x-2 text-sm font-normal leading-5 text-control"
    >
      {label}
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
          size="sm"
        />
      </div>
    );
  };

  const renderCaMaterial = () => {
    if (resolvedCaSource === LOCAL_TLS_CA_SOURCE_SYSTEM_TRUST) {
      return null;
    }

    if (resolvedCaSource === LOCAL_TLS_CA_SOURCE_FILE_PATH) {
      return (
        <FormField
          validationField="sslCaPath"
          title={renderLabel(resolvedCaPathLabel, hasCaPath, caPath)}
        >
          <Input
            data-testid="tls-ca-path-input"
            aria-label={resolvedCaPathLabel}
            value={caPath}
            onChange={(e) => onCaPathChange?.(e.target.value)}
            disabled={disabled || isSaaSMode}
            placeholder={resolvedCaPathLabel}
          />
        </FormField>
      );
    }

    return (
      <FormField
        validationField="sslCa"
        title={renderLabel(resolvedCaLabel, hasCa, ca)}
      >
        <DroppableTextarea
          value={ca}
          onChange={(val) => onCaChange?.(val)}
          disabled={disabled}
          label={resolvedCaLabel}
          placeholder={resolvedCaPlaceholder}
        />
      </FormField>
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
        <div className="flex flex-col gap-4">
          <FormField
            validationField="sslCertPath"
            title={renderLabel(resolvedCertPathLabel, hasCertPath, certPath)}
          >
            <Input
              data-testid="tls-cert-path-input"
              aria-label={resolvedCertPathLabel}
              value={certPath}
              onChange={(e) => onCertPathChange?.(e.target.value)}
              disabled={disabled || isSaaSMode}
              placeholder={resolvedCertPathLabel}
            />
          </FormField>
          <FormField
            validationField="sslKeyPath"
            title={renderLabel(resolvedKeyPathLabel, hasKeyPath, keyPath)}
          >
            <Input
              data-testid="tls-key-path-input"
              aria-label={resolvedKeyPathLabel}
              value={keyPath}
              onChange={(e) => onKeyPathChange?.(e.target.value)}
              disabled={disabled || isSaaSMode}
              placeholder={resolvedKeyPathLabel}
            />
          </FormField>
        </div>
      );
    }

    return (
      <div className="flex flex-col gap-4">
        <FormField
          validationField="sslCert"
          title={renderLabel(resolvedCertLabel, hasCert, cert)}
        >
          <DroppableTextarea
            value={cert}
            onChange={(val) => onCertChange?.(val)}
            disabled={disabled}
            label={resolvedCertLabel}
            placeholder={resolvedCertPlaceholder}
          />
        </FormField>
        <FormField
          validationField="sslKey"
          title={renderLabel(resolvedKeyLabel, hasKey, sslKey)}
        >
          <DroppableTextarea
            value={sslKey}
            onChange={(val) => onKeyChange?.(val)}
            disabled={disabled}
            label={resolvedKeyLabel}
            placeholder={resolvedKeyPlaceholder}
          />
        </FormField>
      </div>
    );
  };

  const renderLegacyMaterial = () => {
    if (resolvedCaSource === LOCAL_TLS_CA_SOURCE_FILE_PATH) {
      return (
        <div className="flex flex-col gap-4">
          <FormField
            validationField="sslCaPath"
            title={renderLabel(resolvedCaPathLabel, hasCaPath, caPath)}
          >
            <Input
              data-testid="tls-ca-path-input"
              aria-label={resolvedCaPathLabel}
              value={caPath}
              onChange={(e) => onCaPathChange?.(e.target.value)}
              disabled={disabled || isSaaSMode}
              placeholder={resolvedCaPathLabel}
            />
          </FormField>
          {showKeyAndCertFields && (
            <FormField
              validationField="sslCertPath"
              title={renderLabel(resolvedCertPathLabel, hasCertPath, certPath)}
            >
              <Input
                data-testid="tls-cert-path-input"
                aria-label={resolvedCertPathLabel}
                value={certPath}
                onChange={(e) => onCertPathChange?.(e.target.value)}
                disabled={disabled || isSaaSMode}
                placeholder={resolvedCertPathLabel}
              />
            </FormField>
          )}
          {showKeyAndCertFields && (
            <FormField
              validationField="sslKeyPath"
              title={renderLabel(resolvedKeyPathLabel, hasKeyPath, keyPath)}
            >
              <Input
                data-testid="tls-key-path-input"
                aria-label={resolvedKeyPathLabel}
                value={keyPath}
                onChange={(e) => onKeyPathChange?.(e.target.value)}
                disabled={disabled || isSaaSMode}
                placeholder={resolvedKeyPathLabel}
              />
            </FormField>
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
            label={resolvedCaLabel}
            placeholder={resolvedCaPlaceholder}
          />
        </TabsPanel>
        {showKeyAndCertFields && (
          <TabsPanel value="KEY" className="pt-1">
            <DroppableTextarea
              value={sslKey}
              onChange={(val) => onKeyChange?.(val)}
              disabled={disabled}
              label={resolvedKeyLabel}
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
              label={resolvedCertLabel}
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
      <FormField
        title={
          <span className="text-sm font-normal leading-5 text-control">
            {resolvedVerifyLabel}
          </span>
        }
      >
        <Switch
          className="self-start"
          aria-label={resolvedVerifyLabel}
          checked={verify}
          onCheckedChange={(val) => onVerifyChange?.(val)}
          disabled={disabled}
        />
        {!verify && (
          <p className="text-xs leading-4 text-control-light">
            {t("data-source.ssl.verification-disabled-description")}
          </p>
        )}
      </FormField>
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
        <fieldset className="flex flex-col gap-4 rounded-xs border border-control-border px-3 py-2">
          <legend className="px-1 textlabel">
            {t("data-source.ssl.server-identity")}
          </legend>
          {renderVerifyControl()}
          {showCaMaterial && (
            <div className="flex flex-col gap-4">
              {showCaSourceUi && (
                <FormField
                  title={
                    <span className="text-sm font-normal leading-5 text-control">
                      {t("data-source.ssl.ca-source.self")}
                    </span>
                  }
                >
                  <CaSourceSelector
                    value={resolvedCaSource}
                    onChange={onCaSourceChange!}
                    disabled={disabled}
                    isSaaSMode={isSaaSMode}
                  />
                </FormField>
              )}
              {renderCaMaterial()}
            </div>
          )}
        </fieldset>

        {resolvedPosture === LOCAL_TLS_POSTURE_MUTUAL_TLS && (
          <fieldset className="flex flex-col gap-4 rounded-xs border border-control-border px-3 py-2">
            <legend className="px-1 textlabel">
              {t("data-source.ssl.client-identity")}
            </legend>
            <div className="flex flex-col gap-4">
              {showClientCertSourceUi && (
                <FormField
                  title={
                    <span className="text-sm font-normal leading-5 text-control">
                      {t("data-source.ssl.client-cert-source.self")}
                    </span>
                  }
                >
                  <ClientCertSourceSelector
                    value={clientIdentitySource}
                    onChange={onClientCertSourceChange!}
                    disabled={disabled}
                    isSaaSMode={isSaaSMode}
                  />
                </FormField>
              )}
              {renderClientCertMaterial(clientIdentitySource)}
            </div>
          </fieldset>
        )}
      </>
    );
  };

  return (
    <ResponsiveFormLayout className="flex flex-col gap-4">
      {showPostureUi && (
        <>
          {renderPostureControl()}
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
          <span className="text-sm font-normal leading-5 text-control">
            {t("data-source.ssl-connection")}
          </span>
        </div>
      )}

      {!showPostureUi && resolvedUseSsl && (
        <>
          {renderVerifyControl()}

          {!showPerGroupSourceUi ? (
            renderLegacyMaterial()
          ) : (
            <>
              {showCaMaterial && (
                <div className="flex flex-col gap-4">
                  {showCaSourceUi && (
                    <FormField
                      title={
                        <span className="text-sm font-normal leading-5 text-control">
                          {t("data-source.ssl.ca-source.self")}
                        </span>
                      }
                    >
                      <CaSourceSelector
                        value={resolvedCaSource}
                        onChange={onCaSourceChange!}
                        disabled={disabled}
                        isSaaSMode={isSaaSMode}
                      />
                    </FormField>
                  )}
                  {renderCaMaterial()}
                </div>
              )}

              {showKeyAndCertFields && (
                <div className="flex flex-col gap-4">
                  {showClientCertSourceUi && (
                    <FormField
                      title={
                        <span className="text-sm font-normal leading-5 text-control">
                          {t("data-source.ssl.client-cert-source.self")}
                        </span>
                      }
                    >
                      <ClientCertSourceSelector
                        value={resolvedClientCertSource}
                        onChange={onClientCertSourceChange!}
                        disabled={disabled}
                        isSaaSMode={isSaaSMode}
                        allowNone
                      />
                    </FormField>
                  )}
                  {renderClientCertMaterial()}
                </div>
              )}
            </>
          )}
        </>
      )}
    </ResponsiveFormLayout>
  );
}
