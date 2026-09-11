import { create } from "@bufbuild/protobuf";
import type { Duration } from "@bufbuild/protobuf/wkt";
import { DurationSchema } from "@bufbuild/protobuf/wkt";
import {
  ChevronDown,
  ChevronRight,
  ExternalLink,
  Info,
  Trash2,
} from "lucide-react";
import {
  useCallback,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { EngineIcon } from "@/components/EngineIcon";
import { EnvironmentSelect } from "@/components/EnvironmentSelect";
import { FeatureBadge } from "@/components/FeatureBadge";
import { LabelListEditor } from "@/components/LabelListEditor";
import { LearnMoreLink } from "@/components/LearnMoreLink";
import { ResourceIdField } from "@/components/ResourceIdField";
import { RouterLink } from "@/components/RouterLink";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  FormControlGroup,
  FormControlRow,
  FormLabel,
  FormSection,
  ResponsiveFormLayout,
} from "@/components/ui/form";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { SegmentedControl } from "@/components/ui/segmented-control";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import {
  environmentNamePrefix,
  instanceNamePrefix,
} from "@/stores/modules/v1/common";
import {
  isValidEnvironmentName,
  UNKNOWN_ID,
  UNKNOWN_INSTANCE_NAME,
  type ValidatedMessage,
} from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  DataSource_AddressSchema,
  DataSource_AuthenticationType,
  DataSource_RedisType,
  DataSourceType,
  type SyncDatabases as SyncDatabasesMessage,
  SyncDatabasesSchema,
} from "@/types/proto-es/v1/instance_service_pb";
import {
  PlanFeature,
  PlanType,
} from "@/types/proto-es/v1/subscription_service_pb";
import {
  engineNameV1,
  extractInstanceResourceName,
  onlyAllowNumber,
  RE_GCP_INSTANCE_ID,
  RE_GCP_PROJECT_ID,
  supportedEngineV1List,
  urlfy,
} from "@/utils";
import type { EditDataSource } from "./common";
import {
  MongoDBConnectionStringSchemaList,
  RedisConnectionType,
  SnowflakeExtraLinkPlaceHolder,
} from "./constants";
import { DataSourceForm, RedisSentinelFields } from "./DataSourceForm";
import { DataSourceSection } from "./DataSourceSection";
import { useInstanceFormContext } from "./InstanceFormContext";
import { hasInfoContent, type InfoSection } from "./info-content";
import {
  ValidationField as FormField,
  ValidationInput as Input,
  ValidationProvider,
} from "./ValidationField";

// --- Inline sub-components ---

function GCPEndpointInput({
  endpoint,
  port,
  placeholder,
  example,
  onUpdate,
  allowEdit,
}: Readonly<{
  endpoint: string;
  port: string;
  placeholder: string;
  example: string;
  onUpdate: (update: { host?: string; port?: string }) => void;
  allowEdit: boolean;
}>) {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState(false);
  const visible = expanded || !!endpoint || !!port;

  if (!visible) {
    if (!allowEdit) return null;
    return (
      <div className="col-span-2">
        <Button
          type="button"
          appearance="link"
          size="xs"
          className="h-auto p-0 text-sm"
          onClick={() => setExpanded(true)}
        >
          {t("instance.gcp-endpoint-toggle")}
        </Button>
      </div>
    );
  }

  return (
    <>
      <FormField title={<>{t("instance.endpoint")}</>}>
        <Input
          value={endpoint}
          placeholder={placeholder}
          className="w-full"
          disabled={!allowEdit}
          onChange={(e) => onUpdate({ host: e.target.value.trim() })}
        />
      </FormField>
      <FormField title={<>{t("instance.port")}</>}>
        <Input
          value={port}
          placeholder="443"
          className="w-full"
          disabled={!allowEdit}
          onChange={(e) => {
            if (e.target.value && !onlyAllowNumber(e.target.value)) return;
            onUpdate({ port: e.target.value.trim() });
          }}
        />
      </FormField>
      <p className="col-span-2 text-xs leading-4 text-control-light">
        {t("instance.gcp-endpoint-tip", { example })}
      </p>
    </>
  );
}

function SpannerHostInput({
  projectId,
  instanceId,
  endpoint,
  port,
  onUpdate,
  allowEdit,
}: {
  projectId: string;
  instanceId: string;
  endpoint: string;
  port: string;
  onUpdate: (update: {
    projectId?: string;
    instanceId?: string;
    host?: string;
    port?: string;
  }) => void;
  allowEdit: boolean;
}) {
  const { t } = useTranslation();
  const [dirty, setDirty] = useState(false);

  const isValidProjectId = RE_GCP_PROJECT_ID.test(projectId);
  const isValidInstanceId = RE_GCP_INSTANCE_ID.test(instanceId);

  return (
    <div className="grid grid-cols-2 gap-x-2 gap-y-1">
      <FormField
        validationField="projectId"
        title={
          <>
            {t("instance.project-id")}
            <span className="text-error"> *</span>
          </>
        }
      >
        <Input
          aria-label={t("instance.project-id")}
          value={projectId}
          required
          placeholder="projectId"
          className={`w-full ${dirty && !isValidProjectId ? "border-error" : ""}`}
          disabled={!allowEdit}
          onChange={(e) => {
            setDirty(true);
            onUpdate({ projectId: e.target.value.trim() });
          }}
        />
      </FormField>
      <FormField
        validationField="instanceId"
        title={
          <>
            {t("instance.instance-id")}
            <span className="text-error"> *</span>
          </>
        }
      >
        <Input
          aria-label={t("instance.instance-id")}
          value={instanceId}
          required
          placeholder="instanceId"
          className={`w-full ${dirty && !isValidInstanceId ? "border-error" : ""}`}
          disabled={!allowEdit}
          onChange={(e) => {
            setDirty(true);
            onUpdate({ instanceId: e.target.value.trim() });
          }}
        />
      </FormField>
      <p className="col-span-2 text-xs leading-4 text-control-light">
        {t("instance.find-gcp-project-id-and-instance-id")}{" "}
        <a
          href="https://docs.bytebase.com/get-started/connect/gcp?source=console"
          target="_blank"
          rel="noopener noreferrer"
          className="normal-link inline-flex items-center"
        >
          {t("common.detailed-guide")}
          <ExternalLink className="size-4 ml-1" />
        </a>
      </p>
      <GCPEndpointInput
        endpoint={endpoint}
        port={port}
        placeholder="spanner.googleapis.com"
        example="spanner-nonprod.p.googleapis.com"
        onUpdate={onUpdate}
        allowEdit={allowEdit}
      />
    </div>
  );
}

function BigQueryHostInput({
  projectId,
  endpoint,
  port,
  onUpdate,
  allowEdit,
}: {
  projectId: string;
  endpoint: string;
  port: string;
  onUpdate: (update: {
    projectId?: string;
    host?: string;
    port?: string;
  }) => void;
  allowEdit: boolean;
}) {
  const { t } = useTranslation();
  const [dirty, setDirty] = useState(false);

  const isValidProjectId = RE_GCP_PROJECT_ID.test(projectId);

  return (
    <div className="grid grid-cols-2 gap-x-2 gap-y-1">
      <FormField
        validationField="projectId"
        title={
          <>
            {t("instance.project-id")}
            <span className="text-error"> *</span>
          </>
        }
      >
        <Input
          aria-label={t("instance.project-id")}
          value={projectId}
          required
          placeholder="projectId"
          className={`w-full ${dirty && !isValidProjectId ? "border-error" : ""}`}
          disabled={!allowEdit}
          onChange={(e) => {
            setDirty(true);
            onUpdate({ projectId: e.target.value.trim() });
          }}
        />
      </FormField>
      <p className="col-span-2 text-xs leading-4 text-control-light">
        {t("instance.find-gcp-project-id")}{" "}
        <a
          href="https://docs.bytebase.com/get-started/connect/gcp?source=console"
          target="_blank"
          rel="noopener noreferrer"
          className="normal-link inline-flex items-center"
        >
          {t("common.detailed-guide")}
          <ExternalLink className="size-4 ml-1" />
        </a>
      </p>
      <GCPEndpointInput
        endpoint={endpoint}
        port={port}
        placeholder="bigquery.googleapis.com"
        example="bigquery-nonprod.p.googleapis.com"
        onUpdate={onUpdate}
        allowEdit={allowEdit}
      />
    </div>
  );
}

function InstanceEngineRadioGrid({
  engine,
  engineList,
  onEngineChange,
  isEngineBeta,
}: {
  engine: Engine;
  engineList: Engine[];
  onEngineChange: (engine: Engine) => void;
  isEngineBeta: (engine: Engine) => boolean;
}) {
  return (
    <div className="w-full grid grid-cols-2 sm:grid-cols-[repeat(auto-fit,minmax(170px,1fr))] gap-2">
      {engineList.map((eng) => (
        <Button
          key={eng}
          type="button"
          appearance="outline"
          size="lg"
          className={cn(
            "h-auto justify-start rounded-sm px-3 py-2 text-left",
            eng === engine
              ? "border-accent bg-accent/5 ring-1 ring-accent"
              : "hover:border-accent/50 hover:bg-control-bg"
          )}
          onClick={() => onEngineChange(eng)}
        >
          <EngineIcon engine={eng} className="size-5" />
          <span className="truncate">{engineNameV1(eng)}</span>
          {isEngineBeta(eng) && (
            <span className="ml-auto shrink-0 rounded-full bg-accent/10 px-2 py-0.5 text-xs text-accent">
              Beta
            </span>
          )}
        </Button>
      ))}
    </div>
  );
}

const MIN_SCAN_MINUTES = 30;

function ScanIntervalInput({
  scanInterval,
  allowEdit,
  onScanIntervalChange,
}: {
  scanInterval: Duration | undefined;
  allowEdit: boolean;
  onScanIntervalChange: (interval: Duration | undefined) => void;
}) {
  const { t } = useTranslation();
  const ctx = useInstanceFormContext();
  const { instance: _instance, hideAdvancedFeatures } = ctx;

  const extractState = (
    duration: Duration | undefined
  ): { mode: "DEFAULT" | "CUSTOM"; minutes: number | undefined } => {
    if (!duration || Number(duration.seconds) === 0) {
      return { mode: "DEFAULT", minutes: undefined };
    }
    return {
      mode: "CUSTOM",
      minutes: Math.floor(Number(duration.seconds) / 60),
    };
  };

  const [mode, setMode] = useState<"DEFAULT" | "CUSTOM">(
    () => extractState(scanInterval).mode
  );
  const [minutes, setMinutes] = useState<number | undefined>(
    () => extractState(scanInterval).minutes
  );
  const [isValid, setIsValid] = useState(true);

  useEffect(() => {
    const s = extractState(scanInterval);
    setMode(s.mode);
    setMinutes(s.minutes);
    setIsValid(true);
  }, [scanInterval]);

  const handleModeChange = (targetMode: "DEFAULT" | "CUSTOM") => {
    if (targetMode === mode) return;
    setMode(targetMode);
    if (targetMode === "DEFAULT") {
      onScanIntervalChange(create(DurationSchema, { seconds: BigInt(0) }));
    } else {
      setMinutes(24 * 60);
      onScanIntervalChange(
        create(DurationSchema, { seconds: BigInt(24 * 60 * 60) })
      );
    }
  };

  const handleMinuteChange = (value: string) => {
    const num = parseInt(value, 10);
    if (Number.isNaN(num)) {
      setMinutes(undefined);
      setIsValid(false);
      return;
    }
    setMinutes(num);
    if (num < MIN_SCAN_MINUTES) {
      setIsValid(false);
      return;
    }
    setIsValid(true);
    onScanIntervalChange(create(DurationSchema, { seconds: BigInt(num * 60) }));
  };

  if (hideAdvancedFeatures) return null;

  return (
    <FormField
      title={
        <span className="flex items-center gap-x-2">
          {t("instance.scan-interval.self")}
          <FeatureBadge
            feature={PlanFeature.FEATURE_CUSTOM_INSTANCE_SYNC_TIME}
          />
        </span>
      }
      description={t("instance.scan-interval.description")}
    >
      <RadioGroup
        className="gap-x-6"
        value={mode}
        onValueChange={(value) => handleModeChange(value as typeof mode)}
      >
        <RadioGroupItem value="DEFAULT" disabled={!allowEdit}>
          {t("instance.scan-interval.default-never")}
        </RadioGroupItem>
        <RadioGroupItem
          value="CUSTOM"
          disabled={!allowEdit}
          contentClassName="flex items-center gap-x-2"
        >
          <span>{t("common.custom")}</span>
          <Input
            type="number"
            value={minutes ?? ""}
            className={`w-16 ${!isValid ? "border-error" : ""}`}
            placeholder={`>= ${MIN_SCAN_MINUTES}`}
            disabled={mode !== "CUSTOM"}
            onChange={(e) => handleMinuteChange(e.target.value)}
          />
          {!isValid ? (
            <span className="text-error text-sm">
              {t("instance.scan-interval.min-value", {
                value: MIN_SCAN_MINUTES,
              })}
            </span>
          ) : (
            <span className="text-sm">{t("common.minutes")}</span>
          )}
        </RadioGroupItem>
      </RadioGroup>
    </FormField>
  );
}

const MAX_VISIBLE_DATABASES = 100;

export function SyncDatabases({
  isCreating: isCreatingProp,
  showLabel,
  allowEdit,
  projectName,
  onOpenInfoPanel,
  syncDatabases,
  disabledReason,
  onSyncDatabasesChange,
}: {
  isCreating: boolean;
  showLabel: boolean;
  allowEdit: boolean;
  projectName?: string;
  onOpenInfoPanel?: (section: InfoSection) => void;
  syncDatabases?: SyncDatabasesMessage;
  disabledReason?: string;
  onSyncDatabasesChange: (databases: string[], syncAll: boolean) => void;
}) {
  const { t } = useTranslation();
  const ctx = useInstanceFormContext();
  const { hideAdvancedFeatures, instance, pendingCreateInstance } = ctx;

  const disabledReasonId = useId();
  const [syncAll, setSyncAll] = useState(syncDatabases === undefined);
  const [selectedSet, setSelectedSet] = useState<Set<string>>(
    () => new Set(syncDatabases?.databases ?? [])
  );
  const [databaseList, setDatabaseList] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(false);
  const [searchText, setSearchText] = useState("");
  const [inputDatabase, setInputDatabase] = useState("");
  const [visibleDatabaseCount, setVisibleDatabaseCount] = useState(
    MAX_VISIBLE_DATABASES
  );
  const pendingScrollDatabaseRef = useRef<string | null>(null);
  const firstNewDatabaseRef = useRef<HTMLLabelElement | null>(null);

  // Notify parent only when selection actually changes.
  const onSyncDatabasesChangeRef = useRef(onSyncDatabasesChange);
  onSyncDatabasesChangeRef.current = onSyncDatabasesChange;
  const prevNotifiedRef = useRef<string | null>(null);

  useEffect(() => {
    const key = [
      syncAll ? "all" : "selected",
      ...[...selectedSet].sort((a, b) => a.localeCompare(b)),
    ].join("\0");
    if (key === prevNotifiedRef.current) return;
    prevNotifiedRef.current = key;
    onSyncDatabasesChangeRef.current(syncAll ? [] : [...selectedSet], syncAll);
  }, [syncAll, selectedSet]);

  const databaseListInstance = isCreatingProp
    ? pendingCreateInstance
    : instance;

  useEffect(() => {
    if (syncAll) return;
    let cancelled = false;
    const fetchDatabases = async () => {
      const inst = databaseListInstance;
      if (!inst) return;
      setLoading(true);
      try {
        const resp = await useAppStore
          .getState()
          .listInstanceDatabases(inst.name, isCreatingProp ? inst : undefined);
        if (!cancelled) {
          setDatabaseList(new Set([...resp.databases, ...selectedSet]));
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    };
    fetchDatabases();
    return () => {
      cancelled = true;
    };
  }, [syncAll, isCreatingProp, databaseListInstance]);

  useEffect(() => {
    setVisibleDatabaseCount(MAX_VISIBLE_DATABASES);
  }, [databaseList, searchText]);

  useEffect(() => {
    if (!pendingScrollDatabaseRef.current) return;
    firstNewDatabaseRef.current?.scrollIntoView({ block: "nearest" });
    pendingScrollDatabaseRef.current = null;
    firstNewDatabaseRef.current = null;
  }, [visibleDatabaseCount]);

  if (hideAdvancedFeatures) return null;

  const lowerSearch = searchText.toLowerCase();
  const filteredDatabases = lowerSearch
    ? [...databaseList].filter((db) => db.toLowerCase().includes(lowerSearch))
    : [...databaseList];
  const visibleDatabases = filteredDatabases.slice(0, visibleDatabaseCount);
  const hasMore = filteredDatabases.length > visibleDatabaseCount;
  const hasProjectContext = !!projectName && isCreatingProp;

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.nativeEvent.isComposing) return;
    const trimmed = inputDatabase.trim();
    if (!trimmed) return;
    if (e.key === "Enter") {
      setDatabaseList((prev) => new Set([...prev, trimmed]));
      setSelectedSet((prev) => new Set([...prev, trimmed]));
      setInputDatabase("");
    }
  };

  const toggleDatabase = (db: string) => {
    setSelectedSet((prev) => {
      const next = new Set(prev);
      if (next.has(db)) {
        next.delete(db);
      } else {
        next.add(db);
      }
      return next;
    });
  };

  const loadMoreDatabases = () => {
    pendingScrollDatabaseRef.current =
      filteredDatabases[visibleDatabaseCount] ?? null;
    setVisibleDatabaseCount((count) => count + MAX_VISIBLE_DATABASES);
  };

  return (
    <FormField
      title={
        showLabel ? (
          <span className="flex items-center gap-x-1">
            {t("instance.sync-databases.self")}
            {onOpenInfoPanel && (
              <Button
                type="button"
                appearance="link"
                size="xs"
                className="w-6 shrink-0 p-0"
                aria-label={t("instance.sync-databases.self")}
                onClick={() => onOpenInfoPanel("sync-databases")}
              >
                <Info className="size-3.5" />
              </Button>
            )}
          </span>
        ) : undefined
      }
      description={
        showLabel ? t("instance.sync-databases.description") : undefined
      }
    >
      <div className="flex flex-col gap-y-2">
        <SegmentedControl
          value={syncAll ? "all" : "selected"}
          onValueChange={(value) => setSyncAll(value === "all")}
          options={[
            { value: "all", label: t("instance.sync-databases.all-databases") },
            {
              value: "selected",
              label: t("instance.sync-databases.selected-databases"),
            },
          ]}
          disabled={!allowEdit}
          ariaLabel={
            hasProjectContext
              ? t("instance.sync-databases.project-sync-all")
              : t("instance.sync-databases.self")
          }
          aria-describedby={disabledReason ? disabledReasonId : undefined}
          size="sm"
        />
        {disabledReason && (
          <p
            id={disabledReasonId}
            className="text-xs leading-4 text-control-light"
          >
            {disabledReason}
          </p>
        )}
        {!syncAll && (
          <div>
            {loading ? (
              <div className="opacity-60 text-sm text-control-light">
                {t("common.loading")}...
              </div>
            ) : (
              <div className="pl-4 flex flex-col gap-y-2">
                <Input
                  value={searchText}
                  className="w-full"
                  placeholder={t("instance.sync-databases.search-database")}
                  onChange={(e) => setSearchText(e.target.value)}
                />
                <div className="max-h-[250px] overflow-y-auto flex flex-col gap-y-1">
                  {visibleDatabases.map((db) => (
                    <label
                      key={db}
                      ref={(element) => {
                        if (db === pendingScrollDatabaseRef.current) {
                          firstNewDatabaseRef.current = element;
                        }
                      }}
                      className="flex items-center gap-x-2 cursor-pointer text-sm"
                    >
                      <Checkbox
                        checked={selectedSet.has(db)}
                        disabled={!allowEdit}
                        onCheckedChange={() => toggleDatabase(db)}
                      />
                      <span>{db}</span>
                    </label>
                  ))}
                </div>
                {hasMore && (
                  <Button
                    type="button"
                    appearance="secondary"
                    size="sm"
                    className="self-start"
                    onClick={loadMoreDatabases}
                  >
                    {t("common.load-more")}
                  </Button>
                )}
                <Input
                  value={inputDatabase}
                  className="w-full"
                  placeholder={t("instance.sync-databases.add-database")}
                  disabled={!allowEdit}
                  onChange={(e) => setInputDatabase(e.target.value)}
                  onKeyDown={handleKeyDown}
                />
              </div>
            )}
          </div>
        )}
      </div>
    </FormField>
  );
}

function AdditionalAddressesFields({
  addresses,
  defaultPort,
  allowEdit,
  allowEditPort,
  onAdd,
  onRemove,
  onHostChange,
  onPortChange,
}: Readonly<{
  addresses: readonly { host: string; port: string }[];
  defaultPort: string;
  allowEdit: boolean;
  allowEditPort: boolean;
  onAdd: () => void;
  onRemove: (index: number) => void;
  onHostChange: (index: number, value: string) => void;
  onPortChange: (index: number, value: string) => void;
}>) {
  const { t } = useTranslation();
  const id = useId();

  return (
    <FormField title={t("data-source.additional-node-addresses")}>
      <FormControlGroup className="mt-1">
        {addresses.map((addr, index) => (
          <FormControlRow key={index} className="items-end">
            <FormField className="min-w-0 flex-1">
              <FormLabel
                htmlFor={`${id}-${index}-host`}
                className={index === 0 ? "font-normal!" : "sr-only"}
              >
                {t("instance.hostname")}
              </FormLabel>
              <Input
                id={`${id}-${index}-host`}
                value={addr.host}
                required
                className="w-full"
                disabled={!allowEdit}
                onChange={(e) => onHostChange(index, e.target.value)}
              />
            </FormField>
            <FormField className="w-32 shrink-0">
              <FormLabel
                htmlFor={`${id}-${index}-port`}
                className={index === 0 ? "font-normal!" : "sr-only"}
              >
                {t("instance.port")}
              </FormLabel>
              <Input
                id={`${id}-${index}-port`}
                value={addr.port}
                className="w-full"
                placeholder={defaultPort}
                disabled={!allowEdit || !allowEditPort}
                onChange={(e) => onPortChange(index, e.target.value)}
              />
            </FormField>
            <Button
              type="button"
              variant="destructive"
              appearance="secondary"
              size="sm"
              aria-label={t("common.delete")}
              disabled={!allowEdit}
              onClick={() => onRemove(index)}
            >
              <Trash2 className="size-4" />
            </Button>
          </FormControlRow>
        ))}
        <div>
          <Button
            appearance="outline"
            size="sm"
            className="w-12!"
            onClick={(e) => {
              e.preventDefault();
              onAdd();
            }}
          >
            {t("common.add")}
          </Button>
        </div>
      </FormControlGroup>
    </FormField>
  );
}

// --- Main component ---

interface InstanceFormBodyProps {
  onOpenInfoPanel?: (section: InfoSection) => void;
}

// Engines without a host field (or with a non-local default) drop the local
// development placeholder host when the user switches to them.
const clearLocalPlaceholderHost = (ds: EditDataSource) => {
  if (ds.host === "127.0.0.1" || ds.host === "host.docker.internal") {
    ds.host = "";
  }
};

export function InstanceFormBody({ onOpenInfoPanel }: InstanceFormBodyProps) {
  const { t } = useTranslation();
  const ctx = useInstanceFormContext();
  const {
    instance,
    specs,
    isCreating,
    allowEdit,
    allowCreate,
    environment,
    basicInfo,
    setBasicInfo,
    labelKVList,
    setLabelKVList,
    dataSourceEditState,
    setDataSourceEditState,
    adminDataSource,
    editingDataSource,
    resetDataSource,
    setResourceIdValidated,
    parent,
  } = ctx;
  const {
    isEngineBeta,
    defaultPort,
    instanceLink,
    allowEditPort,
    allowUsingEmptyPassword,
  } = specs;

  const hasUnifiedInstanceLicense = useAppStore((s) =>
    s.hasUnifiedInstanceLicense()
  );
  const instanceLicenseCount = useAppStore((s) => s.instanceLicenseCount());
  const activatedInstanceCount = useAppStore((s) => s.activatedInstanceCount());
  const currentPlan = useAppStore((s) => s.currentPlan());
  const isSaaSMode = useAppStore((s) => s.isSaaSMode());

  const [isEngineSelectorCollapsed, setIsEngineSelectorCollapsed] =
    useState(false);
  const [showLabels, setShowLabels] = useState(false);

  // --- Computed values ---

  const availableLicenseCount = useMemo(
    () => Math.max(0, instanceLicenseCount - activatedInstanceCount),
    [instanceLicenseCount, activatedInstanceCount]
  );

  const availableLicenseCountText = useMemo((): string => {
    if (instanceLicenseCount === Number.MAX_VALUE) {
      return t("common.unlimited");
    }
    return `${availableLicenseCount}`;
  }, [instanceLicenseCount, availableLicenseCount, t]);

  const resourceId = useMemo(() => {
    const id = extractInstanceResourceName(basicInfo.name);
    if (id === String(UNKNOWN_ID)) return "";
    return id;
  }, [basicInfo.name]);

  const setResourceId = useCallback(
    (id: string) => {
      setBasicInfo((prev) => ({
        ...prev,
        name: parent ? `${parent}/instances/${id}` : `instances/${id}`,
      }));
    },
    [parent, setBasicInfo]
  );

  // Duplicate-instance check used by the shared ResourceIdField's `validate`
  // callback. Tries to fetch the instance by ID; a successful fetch means the
  // ID is taken. A NotFound (or any error) means the ID is available.
  const validateInstanceId = useCallback(
    async (id: string): Promise<ValidatedMessage[]> => {
      if (!isCreating || !id) return [];
      try {
        const existing = await useAppStore
          .getState()
          .getOrFetchInstanceByName(
            parent ? `${parent}/instances/${id}` : `${instanceNamePrefix}${id}`,
            true /* silent */
          );
        if (existing.name !== UNKNOWN_INSTANCE_NAME) {
          return [
            {
              type: "error",
              message: t("resource-id.validation.duplicated", {
                resource: t("common.instance"),
              }),
            },
          ];
        }
      } catch {
        // NotFound = available.
      }
      return [];
    },
    [isCreating, parent, t]
  );

  const currentMongoDBConnectionSchema = useMemo(() => {
    return adminDataSource.srv === false
      ? MongoDBConnectionStringSchemaList[0]
      : MongoDBConnectionStringSchemaList[1];
  }, [adminDataSource.srv]);

  const currentRedisConnectionType = useMemo(() => {
    switch (adminDataSource.redisType) {
      case DataSource_RedisType.STANDALONE:
        return RedisConnectionType[0];
      case DataSource_RedisType.SENTINEL:
        return RedisConnectionType[1];
      case DataSource_RedisType.CLUSTER:
        return RedisConnectionType[2];
      default:
        return RedisConnectionType[0];
    }
  }, [adminDataSource.redisType]);

  const showAdditionalAddresses = useMemo(() => {
    if (basicInfo.engine === Engine.CASSANDRA) return true;
    if (basicInfo.engine === Engine.MONGODB && !adminDataSource.srv)
      return true;
    if (
      basicInfo.engine === Engine.REDIS &&
      (adminDataSource.redisType === DataSource_RedisType.CLUSTER ||
        adminDataSource.redisType === DataSource_RedisType.SENTINEL)
    )
      return true;
    return false;
  }, [basicInfo.engine, adminDataSource.srv, adminDataSource.redisType]);

  const hasHostInfo = useMemo(
    () => hasInfoContent(basicInfo.engine, "host"),
    [basicInfo.engine]
  );

  // --- Handlers ---

  const updateBasicInfo = useCallback(
    (partial: Partial<typeof basicInfo>) => {
      setBasicInfo((prev) => ({ ...prev, ...partial }));
    },
    [setBasicInfo]
  );

  const updateAdminDS = useCallback(
    (partial: Partial<EditDataSource>) => {
      setDataSourceEditState((prev) => ({
        ...prev,
        dataSources: prev.dataSources.map((ds) =>
          ds.type === DataSourceType.ADMIN ? { ...ds, ...partial } : ds
        ),
      }));
    },
    [setDataSourceEditState]
  );

  const changeInstanceEngine = useCallback(
    (engine: Engine) => {
      resetDataSource();
      // After resetDataSource, we need to adjust the host based on the new engine.
      // Use a direct state update instead.
      setDataSourceEditState((prev) => {
        const dataSources = prev.dataSources.map((ds) => {
          if (ds.type !== DataSourceType.ADMIN) return ds;
          const updated = { ...ds };
          switch (engine) {
            case Engine.SNOWFLAKE: {
              clearLocalPlaceholderHost(updated);
              break;
            }
            case Engine.DYNAMODB: {
              updated.authenticationType =
                DataSource_AuthenticationType.AWS_RDS_IAM;
              clearLocalPlaceholderHost(updated);
              break;
            }
            case Engine.SPANNER:
            case Engine.BIGQUERY: {
              updated.authenticationType =
                DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM;
              clearLocalPlaceholderHost(updated);
              break;
            }
            case Engine.COSMOSDB: {
              updated.authenticationType =
                DataSource_AuthenticationType.AZURE_IAM;
              break;
            }
          }
          return updated;
        });
        return { ...prev, dataSources };
      });
      setBasicInfo((prev) => ({ ...prev, engine }));
    },
    [resetDataSource, setDataSourceEditState, setBasicInfo]
  );

  const handleSelectInstanceEngine = useCallback(
    (engine: Engine) => {
      changeInstanceEngine(engine);
      setIsEngineSelectorCollapsed(true);
    },
    [changeInstanceEngine]
  );

  const handleSelectEnvironment = useCallback(
    (name: string | undefined) => {
      setBasicInfo((prev) => ({ ...prev, environment: name }));
    },
    [setBasicInfo]
  );

  const handleChangeSyncDatabases = useCallback(
    (databases: string[], syncAll: boolean) => {
      setBasicInfo((prev) => ({
        ...prev,
        syncDatabases: syncAll
          ? undefined
          : create(SyncDatabasesSchema, { databases }),
      }));
    },
    [setBasicInfo]
  );

  const changeScanInterval = useCallback(
    (duration: Duration | undefined) => {
      setBasicInfo((prev) => ({ ...prev, syncInterval: duration }));
    },
    [setBasicInfo]
  );

  const handleRedisConnectionTypeChange = useCallback(
    (type: string) => {
      let redisType = DataSource_RedisType.STANDALONE;
      switch (type) {
        case RedisConnectionType[1]:
          redisType = DataSource_RedisType.SENTINEL;
          break;
        case RedisConnectionType[2]:
          redisType = DataSource_RedisType.CLUSTER;
          break;
      }
      updateAdminDS({ redisType });
    },
    [updateAdminDS]
  );

  const handleMongodbConnectionStringSchemaChange = useCallback(
    (type: string) => {
      if (type === MongoDBConnectionStringSchemaList[1]) {
        updateAdminDS({
          port: "",
          additionalAddresses: [],
          replicaSet: "",
          directConnection: false,
          srv: true,
        });
      } else {
        updateAdminDS({ srv: false });
      }
    },
    [updateAdminDS]
  );

  const removeDSAdditionalAddress = useCallback(
    (index: number) => {
      setDataSourceEditState((prev) => ({
        ...prev,
        dataSources: prev.dataSources.map((ds) => {
          if (ds.type !== DataSourceType.ADMIN) return ds;
          const newAddresses = [...ds.additionalAddresses];
          newAddresses.splice(index, 1);
          return {
            ...ds,
            additionalAddresses: newAddresses,
            directConnection:
              newAddresses.length === 0 ? false : ds.directConnection,
          };
        }),
      }));
    },
    [setDataSourceEditState]
  );

  const addDSAdditionalAddress = useCallback(() => {
    setDataSourceEditState((prev) => ({
      ...prev,
      dataSources: prev.dataSources.map((ds) => {
        if (ds.id !== dataSourceEditState.editingDataSourceId) return ds;
        const newAddresses = [
          ...ds.additionalAddresses,
          create(DataSource_AddressSchema, { host: "", port: "" }),
        ];
        return {
          ...ds,
          additionalAddresses: newAddresses,
          directConnection:
            newAddresses.length !== 0 ? false : ds.directConnection,
        };
      }),
    }));
  }, [setDataSourceEditState, dataSourceEditState.editingDataSourceId]);

  const changeInstanceActivation = useCallback(
    async (on: boolean) => {
      updateBasicInfo({ activation: on });
      if (instance) {
        const instancePatch = { ...instance, activation: on };
        const updated = await useAppStore
          .getState()
          .updateInstance(instancePatch, ["activation"]);
        useAppStore.getState().updateDatabaseInstance(updated);
        await useAppStore.getState().fetchServerInfo();
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.updated"),
        });
      }
    },
    [instance, updateBasicInfo, t]
  );

  const handleDataSourceChange = useCallback(
    (updated: EditDataSource) => {
      setDataSourceEditState((prev) => ({
        ...prev,
        dataSources: prev.dataSources.map((ds) =>
          ds.id === updated.id ? updated : ds
        ),
      }));
    },
    [setDataSourceEditState]
  );

  const openInfoPanel = useCallback(
    (section: InfoSection) => {
      if (!hasInfoContent(basicInfo.engine, section)) return;
      onOpenInfoPanel?.(section);
    },
    [basicInfo.engine, onOpenInfoPanel]
  );

  // Port-only numeric filter
  const handlePortChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const value = e.target.value;
      if (value === "" || /^\d+$/.test(value)) {
        updateAdminDS({ port: value });
      }
    },
    [updateAdminDS]
  );

  const handleAdditionalAddressHostChange = useCallback(
    (index: number, host: string) => {
      setDataSourceEditState((prev) => ({
        ...prev,
        dataSources: prev.dataSources.map((ds) => {
          if (ds.type !== DataSourceType.ADMIN) return ds;
          const newAddresses = [...ds.additionalAddresses];
          newAddresses[index] = { ...newAddresses[index], host };
          return { ...ds, additionalAddresses: newAddresses };
        }),
      }));
    },
    [setDataSourceEditState]
  );

  const handleAdditionalAddressPortChange = useCallback(
    (index: number, port: string) => {
      if (port !== "" && !/^\d+$/.test(port)) return;
      setDataSourceEditState((prev) => ({
        ...prev,
        dataSources: prev.dataSources.map((ds) => {
          if (ds.type !== DataSourceType.ADMIN) return ds;
          const newAddresses = [...ds.additionalAddresses];
          newAddresses[index] = { ...newAddresses[index], port };
          return { ...ds, additionalAddresses: newAddresses };
        }),
      }));
    },
    [setDataSourceEditState]
  );

  return (
    <ValidationProvider
      className="flex flex-col pb-2"
      errors={{
        ...ctx.getDataSourceErrors(adminDataSource),
        ...(!basicInfo.title.trim() ? { title: "required" } : {}),
      }}
    >
      <div className="w-full max-w-5xl flex flex-col">
        {/* Basic Info Card */}
        <FormSection layout="stacked" title={t("instance.section.basic-info")}>
          <div className="flex flex-col gap-4">
            {isCreating && (
              <FormField title={t("database.engine")}>
                <Button
                  appearance="outline"
                  className="w-full justify-between"
                  aria-expanded={!isEngineSelectorCollapsed}
                  onClick={() => setIsEngineSelectorCollapsed((prev) => !prev)}
                >
                  <span className="flex items-center gap-2">
                    <EngineIcon engine={basicInfo.engine} className="size-4" />
                    {engineNameV1(basicInfo.engine)}
                  </span>
                  {isEngineSelectorCollapsed ? (
                    <ChevronRight />
                  ) : (
                    <ChevronDown />
                  )}
                </Button>
                <div
                  hidden={isEngineSelectorCollapsed}
                  inert={isEngineSelectorCollapsed ? true : undefined}
                >
                  <InstanceEngineRadioGrid
                    engine={basicInfo.engine}
                    engineList={supportedEngineV1List()}
                    onEngineChange={handleSelectInstanceEngine}
                    isEngineBeta={isEngineBeta}
                  />
                </div>
              </FormField>
            )}

            {/* Instance Name */}
            <FormField validationField="title">
              <FormLabel htmlFor="name" className="flex flex-row items-center">
                {t("instance.instance-name")}
                <span className="ml-0.5 text-error">*</span>
                {instance && (
                  <div className="ml-2 flex items-center">
                    <EngineIcon engine={instance.engine} className="size-4" />
                    <span className="ml-1">{instance.engineVersion}</span>
                  </div>
                )}
              </FormLabel>
              <Input
                id="name"
                value={basicInfo.title}
                required
                className="w-full max-w-[40rem]"
                disabled={!allowEdit}
                maxLength={200}
                onChange={(e) => updateBasicInfo({ title: e.target.value })}
              />
              <ResourceIdField
                suffix
                value={resourceId}
                resourceName={t("common.instance")}
                resourceTitle={basicInfo.title}
                readonly={!isCreating}
                validate={validateInstanceId}
                onChange={setResourceId}
                onValidationChange={setResourceIdValidated}
              />
            </FormField>

            {/* Activation toggle */}
            {currentPlan !== PlanType.FREE &&
              !hasUnifiedInstanceLicense &&
              allowEdit && (
                <FormField
                  title={t("subscription.instance-assignment.assign-license")}
                >
                  <FormControlRow className="w-fit">
                    <Switch
                      aria-label={t(
                        "subscription.instance-assignment.assign-license"
                      )}
                      checked={basicInfo.activation}
                      disabled={
                        !basicInfo.activation && availableLicenseCount === 0
                      }
                      onCheckedChange={changeInstanceActivation}
                    />
                    <RouterLink
                      to="/setting/subscription"
                      className="accent-link"
                    >
                      {t("subscription.instance-assignment.n-license-remain", {
                        n: availableLicenseCountText,
                      })}
                    </RouterLink>
                  </FormControlRow>
                </FormField>
              )}

            {/* Environment */}
            <FormField title={t("common.environment")}>
              <div className="flex items-center gap-4">
                <EnvironmentSelect
                  portal
                  className="w-full max-w-[40rem]"
                  value={
                    isValidEnvironmentName(
                      `${environmentNamePrefix}${environment.id}`
                    )
                      ? `${environmentNamePrefix}${environment.id}`
                      : ""
                  }
                  disabled={!allowEdit}
                  onChange={(value) =>
                    handleSelectEnvironment(value || undefined)
                  }
                />
                {!showLabels && labelKVList.length === 0 && allowEdit && (
                  <Button
                    size="sm"
                    appearance="link"
                    onClick={() => setShowLabels(true)}
                  >
                    {t("instance.add-labels")}
                  </Button>
                )}
              </div>
            </FormField>

            {/* Labels */}
            {(showLabels || labelKVList.length > 0) && (
              <FormField title={t("common.labels")}>
                <LabelListEditor
                  kvList={labelKVList}
                  onChange={setLabelKVList}
                  readonly={!allowEdit}
                  showErrors
                  onErrorsChange={ctx.setLabelErrors}
                />
              </FormField>
            )}

            {/* External link (edit mode only) */}
            {!isCreating && (
              <FormField
                title={
                  <div className="inline-flex items-center">
                    <FormLabel htmlFor="external-link">
                      {basicInfo.engine === Engine.SNOWFLAKE
                        ? t("instance.snowflake-web-console")
                        : t("instance.external-link")}
                    </FormLabel>
                    {(basicInfo.externalLink ?? "").trim().length > 0 && (
                      <Button
                        type="button"
                        appearance="secondary"
                        size="xs"
                        className="ml-1 w-6 p-0"
                        aria-label={t("instance.external-link")}
                        onClick={(e) => {
                          e.preventDefault();
                          window.open(
                            urlfy(basicInfo.externalLink ?? ""),
                            "_blank"
                          );
                        }}
                      >
                        <ExternalLink className="size-4" />
                      </Button>
                    )}
                  </div>
                }
              >
                {basicInfo.engine === Engine.SNOWFLAKE ? (
                  <Input
                    id="external-link"
                    required
                    className="w-full"
                    disabled
                    value={instanceLink}
                  />
                ) : (
                  <>
                    <p className="text-xs leading-4 text-control-light">
                      {t("instance.sentence.console.snowflake")}
                    </p>
                    <Input
                      id="external-link"
                      value={basicInfo.externalLink ?? ""}
                      required
                      className="w-full"
                      disabled={!allowEdit}
                      placeholder={SnowflakeExtraLinkPlaceHolder}
                      onChange={(e) =>
                        updateBasicInfo({ externalLink: e.target.value })
                      }
                    />
                  </>
                )}
              </FormField>
            )}

            {/* Scan Interval (edit mode only) */}
            {!isCreating && instance && (
              <ScanIntervalInput
                scanInterval={basicInfo.syncInterval}
                allowEdit={allowEdit}
                onScanIntervalChange={changeScanInterval}
              />
            )}
          </div>
        </FormSection>

        {/* Connection Card */}
        <FormSection layout="stacked" title={t("instance.section.connection")}>
          <div className="flex flex-col gap-4">
            {isSaaSMode && (
              <Alert variant="info">
                <a
                  href="https://docs.bytebase.com/get-started/cloud#prerequisites"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="normal-link"
                >
                  {t("instance.sentence.firewall-info")}
                </a>
              </Alert>
            )}
            {editingDataSource?.id === adminDataSource.id && (
              <DataSourceForm
                authOnly
                dataSource={adminDataSource}
                onDataSourceChange={handleDataSourceChange}
                onOpenInfoPanel={onOpenInfoPanel}
              />
            )}
            {/* Host input */}
            {basicInfo.engine === Engine.SPANNER ? (
              <SpannerHostInput
                projectId={adminDataSource.projectId}
                instanceId={adminDataSource.instanceId}
                endpoint={adminDataSource.host}
                port={adminDataSource.port}
                onUpdate={(update) => updateAdminDS(update)}
                allowEdit={allowEdit}
              />
            ) : basicInfo.engine === Engine.BIGQUERY ? (
              <BigQueryHostInput
                projectId={adminDataSource.projectId}
                endpoint={adminDataSource.host}
                port={adminDataSource.port}
                onUpdate={(update) => updateAdminDS(update)}
                allowEdit={allowEdit}
              />
            ) : (
              <FormField
                validationField="host"
                title={
                  <span className="flex items-center gap-1">
                    <FormLabel htmlFor="host">
                      {basicInfo.engine === Engine.SNOWFLAKE
                        ? t("instance.account-locator")
                        : basicInfo.engine === Engine.COSMOSDB
                          ? t("instance.endpoint")
                          : adminDataSource.authenticationType ===
                              DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM
                            ? t(
                                "instance.sentence.google-cloud-sql.instance-name"
                              )
                            : t("instance.hostname")}
                      {basicInfo.engine !== Engine.DYNAMODB && (
                        <span className="text-error"> *</span>
                      )}
                    </FormLabel>
                    {onOpenInfoPanel && hasHostInfo && (
                      <Button
                        appearance="link"
                        size="xs"
                        className="h-auto shrink-0 p-0"
                        aria-label={t("instance.hostname")}
                        onClick={() => openInfoPanel("host")}
                      >
                        <Info className="size-3.5" />
                      </Button>
                    )}
                  </span>
                }
              >
                <div className="flex items-center gap-2">
                  <Input
                    id="host"
                    value={adminDataSource.host}
                    required={basicInfo.engine !== Engine.DYNAMODB}
                    placeholder={
                      basicInfo.engine === Engine.SNOWFLAKE
                        ? t("instance.your-snowflake-account-locator")
                        : isSaaSMode
                          ? t("instance.sentence.host.saas")
                          : t("instance.sentence.host.none-snowflake")
                    }
                    className="min-w-0 flex-1"
                    disabled={!allowEdit}
                    onChange={(e) => updateAdminDS({ host: e.target.value })}
                  />
                  {basicInfo.engine !== Engine.DATABRICKS &&
                    basicInfo.engine !== Engine.COSMOSDB &&
                    adminDataSource.authenticationType !==
                      DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM && (
                      <>
                        <FormLabel htmlFor="port">
                          {t("instance.port")}
                        </FormLabel>
                        <Input
                          id="port"
                          value={adminDataSource.port}
                          className="w-20 shrink-0"
                          placeholder={defaultPort}
                          disabled={!allowEdit || !allowEditPort}
                          onChange={handlePortChange}
                        />
                      </>
                    )}
                </div>
                {adminDataSource.authenticationType ===
                  DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM && (
                  <p className="text-xs text-control-light">
                    {t(
                      "instance.sentence.google-cloud-sql.instance-name-tips",
                      { instance: "{project-id}:{region}:{instance-name}" }
                    )}
                  </p>
                )}
                {basicInfo.engine === Engine.SNOWFLAKE && (
                  <LearnMoreLink href="https://docs.snowflake.com/en/user-guide/admin-account-identifier#using-an-account-locator-as-an-identifier" />
                )}
              </FormField>
            )}

            {/* MongoDB connection string schema */}
            {basicInfo.engine === Engine.MONGODB && (
              <FormField>
                <FormLabel htmlFor="connectionStringSchema">
                  {t("data-source.connection-string-schema")}
                </FormLabel>
                <RadioGroup
                  className="gap-x-4"
                  value={currentMongoDBConnectionSchema}
                  onValueChange={(value) =>
                    handleMongodbConnectionStringSchemaChange(value as string)
                  }
                >
                  {MongoDBConnectionStringSchemaList.map((type) => (
                    <RadioGroupItem key={type} value={type}>
                      {type}
                    </RadioGroupItem>
                  ))}
                </RadioGroup>
                {!adminDataSource.srv && (
                  <ResponsiveFormLayout className="mt-2">
                    <fieldset
                      aria-label={t("data-source.connection-string-schema")}
                      className="flex flex-col gap-4 rounded-xs border border-control-border px-3 py-2"
                    >
                      {/* Additional addresses */}
                      {showAdditionalAddresses && (
                        <AdditionalAddressesFields
                          addresses={adminDataSource.additionalAddresses}
                          defaultPort={defaultPort}
                          allowEdit={allowEdit}
                          allowEditPort={allowEditPort}
                          onAdd={addDSAdditionalAddress}
                          onRemove={removeDSAdditionalAddress}
                          onHostChange={handleAdditionalAddressHostChange}
                          onPortChange={handleAdditionalAddressPortChange}
                        />
                      )}
                      {/* MongoDB replica set */}
                      {basicInfo.engine === Engine.MONGODB &&
                        !adminDataSource.srv && (
                          <FormField>
                            <FormLabel htmlFor="replicaSet">
                              {t("data-source.replica-set")}
                            </FormLabel>
                            <Input
                              value={adminDataSource.replicaSet}
                              required
                              className="w-full"
                              disabled={!allowEdit}
                              onChange={(e) =>
                                updateAdminDS({ replicaSet: e.target.value })
                              }
                            />
                          </FormField>
                        )}
                      {/* MongoDB direct connection */}
                      {basicInfo.engine === Engine.MONGODB &&
                        !adminDataSource.srv &&
                        adminDataSource.additionalAddresses.length === 0 && (
                          <FormControlRow className="w-fit">
                            <Checkbox
                              id="directConnection"
                              checked={adminDataSource.directConnection}
                              disabled={!allowEdit}
                              onCheckedChange={(checked) =>
                                updateAdminDS({
                                  directConnection: checked,
                                })
                              }
                            />
                            <FormLabel
                              htmlFor="directConnection"
                              className="font-normal!"
                            >
                              {t("data-source.direct-connection")}
                            </FormLabel>
                          </FormControlRow>
                        )}{" "}
                    </fieldset>
                  </ResponsiveFormLayout>
                )}
              </FormField>
            )}

            {/* Redis connection type */}
            {basicInfo.engine === Engine.REDIS && (
              <FormField>
                <FormLabel htmlFor="connectionStringSchema">
                  {t("data-source.connection-type")}
                </FormLabel>
                <RadioGroup
                  className="gap-x-4"
                  value={currentRedisConnectionType}
                  onValueChange={(value) =>
                    handleRedisConnectionTypeChange(value as string)
                  }
                >
                  {RedisConnectionType.map((type) => (
                    <RadioGroupItem key={type} value={type}>
                      {type}
                    </RadioGroupItem>
                  ))}
                </RadioGroup>
                {showAdditionalAddresses && (
                  <ResponsiveFormLayout className="mt-2">
                    <fieldset
                      aria-label={currentRedisConnectionType}
                      className="flex flex-col gap-4 rounded-xs border border-control-border px-3 py-2"
                    >
                      <AdditionalAddressesFields
                        addresses={adminDataSource.additionalAddresses}
                        defaultPort={defaultPort}
                        allowEdit={allowEdit}
                        allowEditPort={allowEditPort}
                        onAdd={addDSAdditionalAddress}
                        onRemove={removeDSAdditionalAddress}
                        onHostChange={handleAdditionalAddressHostChange}
                        onPortChange={handleAdditionalAddressPortChange}
                      />
                    </fieldset>
                  </ResponsiveFormLayout>
                )}
                {editingDataSource?.redisType ===
                  DataSource_RedisType.SENTINEL && (
                  <ResponsiveFormLayout className="mt-2">
                    <RedisSentinelFields
                      dataSource={editingDataSource}
                      isCreating={isCreating}
                      allowEdit={allowEdit}
                      allowUsingEmptyPassword={allowUsingEmptyPassword}
                      onDataSourceChange={handleDataSourceChange}
                    />
                  </ResponsiveFormLayout>
                )}
              </FormField>
            )}

            {/* Additional addresses */}
            {basicInfo.engine === Engine.CASSANDRA && (
              <AdditionalAddressesFields
                addresses={adminDataSource.additionalAddresses}
                defaultPort={defaultPort}
                allowEdit={allowEdit}
                allowEditPort={allowEditPort}
                onAdd={addDSAdditionalAddress}
                onRemove={removeDSAdditionalAddress}
                onHostChange={handleAdditionalAddressHostChange}
                onPortChange={handleAdditionalAddressPortChange}
              />
            )}
          </div>

          {/* Credentials (auth method, username, password) */}
          <DataSourceSection
            hideOptions
            hideAdminAuthentication
            onOpenInfoPanel={onOpenInfoPanel}
          />

          {basicInfo.engine !== Engine.DYNAMODB && editingDataSource && (
            <div className="mt-4">
              <div>
                <DataSourceForm
                  dataSource={editingDataSource}
                  optionsOnly
                  onDataSourceChange={handleDataSourceChange}
                  onOpenInfoPanel={onOpenInfoPanel}
                />
              </div>
            </div>
          )}

          {basicInfo.engine !== Engine.DYNAMODB && (
            <div className="mt-4">
              <SyncDatabases
                isCreating={isCreating}
                showLabel
                allowEdit={isCreating ? allowEdit && !!allowCreate : allowEdit}
                disabledReason={
                  isCreating && allowEdit && !allowCreate
                    ? !adminDataSource.host &&
                      ![
                        Engine.SPANNER,
                        Engine.BIGQUERY,
                        Engine.DYNAMODB,
                      ].includes(basicInfo.engine)
                      ? t("instance.sync-databases.hostname-required")
                      : t("instance.sync-databases.required-fields")
                    : undefined
                }
                projectName={parent}
                onOpenInfoPanel={onOpenInfoPanel}
                syncDatabases={basicInfo.syncDatabases}
                onSyncDatabasesChange={handleChangeSyncDatabases}
              />
            </div>
          )}
        </FormSection>
      </div>
    </ValidationProvider>
  );
}
