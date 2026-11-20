import { create } from "@bufbuild/protobuf";
import type { Duration } from "@bufbuild/protobuf/wkt";
import { isEqual, omit } from "lodash-es";
import {
  AlertCircle,
  ExternalLink,
  Info,
  Link,
  Server,
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
import neonIcon from "@/assets/db/neon.svg";
import { EngineIcon } from "@/components/EngineIcon";
import { EnvironmentSelect } from "@/components/EnvironmentSelect";
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
  FormError,
  FormFieldGroup,
  FormLabel,
  FormSection,
  ResponsiveFormLayout,
  FormField as SharedFormField,
} from "@/components/ui/form";
import { Input as SharedInput } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Tooltip } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { AgentHelpButton } from "@/modules/agent/components/AgentHelpButton";
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
  Vendor,
} from "@/types/proto-es/v1/instance_service_pb";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";
import {
  engineNameV1,
  extractInstanceResourceName,
  onlyAllowNumber,
  RE_GCP_INSTANCE_ID,
  RE_GCP_PROJECT_ID,
  supportedEngineV1List,
  urlfy,
} from "@/utils";
import { normalizeAuthenticationType } from "./authentication";
import { CollapsibleTilePicker, TileRadioItem } from "./CollapsibleTilePicker";
import {
  type DataSourceEditState,
  type EditDataSource,
  extractDataSourceEditState,
} from "./common";
import {
  applyParsedConnectionString,
  type ConnectionStringEngine,
  detectConnectionStringEngine,
  type ParsedConnectionString,
  parseConnectionString,
  supportsConnectionString,
} from "./connection-string";
import {
  MongoDBConnectionStringSchemaList,
  RedisConnectionType,
  SnowflakeExtraLinkPlaceHolder,
} from "./constants";
import { DataSourceForm, RedisSentinelFields } from "./DataSourceForm";
import { DataSourceSection } from "./DataSourceSection";
import { useInstanceFormContext } from "./InstanceFormContext";
import { hasInfoContent, type InfoSection } from "./info-content";
import { isNeonPooledHost } from "./neon-import";
import { ScanIntervalInput } from "./ScanIntervalInput";
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
  const endpointId = useId();
  const portId = useId();
  const [expanded, setExpanded] = useState(false);
  const visible = expanded || !!endpoint || !!port;

  if (!visible) {
    if (!allowEdit) return null;
    return (
      <FormField title={t("instance.endpoint")}>
        <Button
          type="button"
          appearance="link"
          size="xs"
          className="h-auto p-0 text-sm self-start"
          onClick={() => setExpanded(true)}
        >
          {t("instance.gcp-endpoint-toggle")}
        </Button>
      </FormField>
    );
  }

  return (
    <FormField
      title={
        <FormLabel htmlFor={endpointId}>{t("instance.endpoint")}</FormLabel>
      }
    >
      <div className="flex flex-wrap items-center gap-2">
        <Input
          id={endpointId}
          value={endpoint}
          placeholder={placeholder}
          className="min-w-0 flex-1"
          disabled={!allowEdit}
          onChange={(e) => onUpdate({ host: e.target.value.trim() })}
        />
        <FormLabel htmlFor={portId}>{t("instance.port")}</FormLabel>
        <Input
          id={portId}
          aria-label={t("instance.port")}
          value={port}
          placeholder="443"
          className="w-20 shrink-0"
          disabled={!allowEdit}
          onChange={(e) => {
            if (e.target.value && !onlyAllowNumber(e.target.value)) return;
            onUpdate({ port: e.target.value.trim() });
          }}
        />
      </div>
      <p className="text-xs leading-4 text-control-light">
        {t("instance.gcp-endpoint-tip", { example })}
      </p>
    </FormField>
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
    <div className="flex flex-col gap-4">
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
        <p className="text-xs leading-4 text-control-light">
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
      </FormField>
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
    <div className="flex flex-col gap-4">
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
        <p className="text-xs leading-4 text-control-light">
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
      </FormField>
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

function VendorIcon({
  vendor,
  className,
}: {
  vendor: Vendor;
  className?: string;
}) {
  if (vendor === Vendor.NEON) {
    return (
      <img
        src={neonIcon}
        alt=""
        className={cn("shrink-0 object-contain", className)}
      />
    );
  }
  return (
    <Server
      aria-hidden="true"
      className={cn("shrink-0 text-control-light", className)}
    />
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
  const selectedSetRef = useRef(selectedSet);
  selectedSetRef.current = selectedSet;
  const [databaseList, setDatabaseList] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(false);
  const [searchText, setSearchText] = useState("");
  const [inputDatabase, setInputDatabase] = useState("");
  const [visibleDatabaseCount, setVisibleDatabaseCount] = useState(
    MAX_VISIBLE_DATABASES
  );
  const pendingScrollDatabaseRef = useRef<string | null>(null);
  const firstNewDatabaseRef = useRef<HTMLLabelElement | null>(null);
  const databaseListInstance = isCreatingProp
    ? pendingCreateInstance
    : instance;
  const previousDatabaseListInstanceRef =
    useRef<typeof databaseListInstance>(undefined);
  const activeDatabaseListRequestRef = useRef(0);
  const isMountedRef = useRef(false);

  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  // Notify parent only when selection actually changes.
  const onSyncDatabasesChangeRef = useRef(onSyncDatabasesChange);
  onSyncDatabasesChangeRef.current = onSyncDatabasesChange;
  const prevNotifiedRef = useRef<string | null>(null);

  useEffect(() => {
    const selectedDatabases = syncDatabases?.databases ?? [];
    const nextSyncAll = syncDatabases === undefined;

    setSyncAll((current) => (current === nextSyncAll ? current : nextSyncAll));
    setSelectedSet((current) =>
      current.size === selectedDatabases.length &&
      selectedDatabases.every((database) => current.has(database))
        ? current
        : new Set(selectedDatabases)
    );
    setDatabaseList((current) =>
      selectedDatabases.every((database) => current.has(database))
        ? current
        : new Set([...current, ...selectedDatabases])
    );
  }, [syncDatabases]);

  useEffect(() => {
    const key = [
      syncAll ? "all" : "selected",
      ...[...selectedSet].sort((a, b) => a.localeCompare(b)),
    ].join("\0");
    if (key === prevNotifiedRef.current) return;
    prevNotifiedRef.current = key;
    onSyncDatabasesChangeRef.current(syncAll ? [] : [...selectedSet], syncAll);
  }, [syncAll, selectedSet]);

  useEffect(() => {
    if (syncAll) {
      previousDatabaseListInstanceRef.current = undefined;
      activeDatabaseListRequestRef.current += 1;
      setLoading(false);
      return;
    }
    const fetchDatabases = async () => {
      const inst = databaseListInstance;
      if (!inst) {
        activeDatabaseListRequestRef.current += 1;
        setLoading(false);
        return;
      }
      if (
        isEqual(
          omit(previousDatabaseListInstanceRef.current, "syncDatabases"),
          omit(inst, "syncDatabases")
        )
      ) {
        return;
      }
      previousDatabaseListInstanceRef.current = inst;
      const requestId = activeDatabaseListRequestRef.current + 1;
      activeDatabaseListRequestRef.current = requestId;
      setLoading(true);
      try {
        const resp = await useAppStore
          .getState()
          .listInstanceDatabases(inst.name, isCreatingProp ? inst : undefined);
        if (
          isMountedRef.current &&
          requestId === activeDatabaseListRequestRef.current
        ) {
          setDatabaseList(
            new Set([...selectedSetRef.current, ...resp.databases])
          );
        }
      } finally {
        if (
          isMountedRef.current &&
          requestId === activeDatabaseListRequestRef.current
        ) {
          setLoading(false);
        }
      }
    };
    void fetchDatabases();
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
  const showDisabledReason = !!disabledReason && !syncAll;

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
                className="-ml-1 w-6 shrink-0 p-0"
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
        <FormControlRow className="w-fit">
          <Switch
            checked={syncAll}
            onCheckedChange={setSyncAll}
            disabled={!allowEdit || !!disabledReason}
            aria-label={
              hasProjectContext
                ? t("instance.sync-databases.project-sync-all")
                : t("instance.sync-databases.self")
            }
            aria-describedby={showDisabledReason ? disabledReasonId : undefined}
          />
          <span
            className={cn("text-sm", disabledReason && "text-control-light")}
          >
            {syncAll
              ? t("instance.sync-databases.all-databases")
              : t("instance.sync-databases.selected-databases")}
          </span>
        </FormControlRow>
        {showDisabledReason && (
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
              <div className="flex flex-col gap-y-2">
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

const NEON_CONNECTION_STRING_PLACEHOLDER =
  "postgresql://role:password@ep-example.us-east-2.aws.neon.tech/neondb?sslmode=require";

const CONNECTION_STRING_PLACEHOLDERS: Record<ConnectionStringEngine, string> = {
  [Engine.MYSQL]:
    "mysql://user:password@db.example.com:3306/mydb?ssl-mode=REQUIRED",
  [Engine.POSTGRES]:
    "postgresql://user:password@db.example.com:5432/mydb?sslmode=require",
  [Engine.MSSQL]:
    "Server=tcp:db.example.com,1433;Initial Catalog=mydb;User ID=sa;Password=…;Encrypt=True;",
  [Engine.ORACLE]: "tcps://user:password@db.example.com:1522/orclpdb1",
  [Engine.MARIADB]:
    "mariadb://user:password@db.example.com:3306/mydb?ssl-mode=REQUIRED",
  [Engine.TIDB]:
    "mysql://user:password@db.example.com:4000/mydb?ssl-mode=VERIFY_IDENTITY",
  [Engine.OCEANBASE]:
    "mysql://user@tenant#cluster:password@db.example.com:2883/mydb",
  [Engine.STARROCKS]: "mysql://user:password@db.example.com:9030/mydb",
  [Engine.DORIS]: "mysql://user:password@db.example.com:9030/mydb",
  [Engine.COCKROACHDB]:
    "postgresql://user:password@db.example.com:26257/defaultdb?sslmode=verify-full",
  [Engine.REDSHIFT]: "redshift://user:password@cluster.example.com:5439/dev",
  [Engine.MONGODB]: "mongodb+srv://user:password@cluster.example.com/mydb",
  [Engine.REDIS]: "rediss://default:password@db.example.com:6379",
  [Engine.CASSANDRA]:
    "cassandra://user:password@node1.example.com:9042,node2.example.com:9042",
  [Engine.ELASTICSEARCH]: "https://user:password@search.example.com:9200",
  [Engine.CLICKHOUSE]:
    "clickhouses://user:password@db.example.com:9440/default",
  [Engine.SNOWFLAKE]:
    "snowflake://user:password@account.snowflakecomputing.com",
  [Engine.TRINO]:
    "trino://user:password@trino.example.com:8080/catalog?SSL=true",
  [Engine.HIVE]:
    "jdbc:hive2://hive.example.com:10000/default;user=hive;password=…",
  [Engine.DATABRICKS]:
    "jdbc:databricks://dbc.example.com:443/default;httpPath=/sql/1.0/warehouses/…;PWD=…",
};

type ConnectionUrlFillState =
  | "complete"
  | "missing-password"
  | "missing-token"
  | "neon-pooled";
type ConnectionUrlFocusTarget = "password" | "host";

function ConnectionStringImportPopover({
  engine,
  vendor = Vendor.VENDOR_UNSPECIFIED,
  disabled,
  resolvePasswordInput,
  resolveHostInput,
  onParsed,
}: {
  engine: ConnectionStringEngine;
  vendor?: Vendor;
  disabled: boolean;
  resolvePasswordInput: () => HTMLInputElement | null;
  resolveHostInput: () => HTMLInputElement | null;
  onParsed: (
    parsed: ParsedConnectionString,
    fillState: ConnectionUrlFillState
  ) => void;
}) {
  const { t } = useTranslation();
  const inputId = useId();
  const errorId = `${inputId}-error`;
  const [open, setOpen] = useState(false);
  const [value, setValue] = useState("");
  const [checked, setChecked] = useState(false);
  const [pendingFocusTarget, setPendingFocusTarget] =
    useState<ConnectionUrlFocusTarget | null>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const focusOnCloseRef = useRef<ConnectionUrlFocusTarget | null>(null);
  const invalid = checked && value.trim() !== "";
  const isNeon = engine === Engine.POSTGRES && vendor === Vendor.NEON;

  const resolveFocusElement = useCallback(
    (target: ConnectionUrlFocusTarget) =>
      target === "host" ? resolveHostInput() : resolvePasswordInput(),
    [resolveHostInput, resolvePasswordInput]
  );

  useEffect(() => {
    if (open || !pendingFocusTarget) return;
    setPendingFocusTarget(null);
    const targetEl = resolveFocusElement(pendingFocusTarget);
    if (targetEl && !targetEl.disabled) {
      targetEl.focus();
    }
  }, [open, pendingFocusTarget, resolveFocusElement]);

  const helpTextByEngine: Record<ConnectionStringEngine, string> = {
    [Engine.MYSQL]: t("instance.connection-string.help.mysql"),
    [Engine.POSTGRES]: t("instance.connection-string.help.postgres"),
    [Engine.MSSQL]: t("instance.connection-string.help.mssql"),
    [Engine.ORACLE]: t("instance.connection-string.help.oracle"),
    [Engine.MARIADB]: t("instance.connection-string.help.mariadb"),
    [Engine.TIDB]: t("instance.connection-string.help.tidb"),
    [Engine.OCEANBASE]: t("instance.connection-string.help.oceanbase"),
    [Engine.STARROCKS]: t("instance.connection-string.help.starrocks"),
    [Engine.DORIS]: t("instance.connection-string.help.doris"),
    [Engine.COCKROACHDB]: t("instance.connection-string.help.cockroachdb"),
    [Engine.REDSHIFT]: t("instance.connection-string.help.redshift"),
    [Engine.MONGODB]: t("instance.connection-string.help.mongodb"),
    [Engine.REDIS]: t("instance.connection-string.help.redis"),
    [Engine.CASSANDRA]: t("instance.connection-string.help.cassandra"),
    [Engine.ELASTICSEARCH]: t("instance.connection-string.help.elasticsearch"),
    [Engine.CLICKHOUSE]: t("instance.connection-string.help.clickhouse"),
    [Engine.SNOWFLAKE]: t("instance.connection-string.help.snowflake"),
    [Engine.TRINO]: t("instance.connection-string.help.trino"),
    [Engine.HIVE]: t("instance.connection-string.help.hive"),
    [Engine.DATABRICKS]: t("instance.connection-string.help.databricks"),
  };
  const helpText = isNeon
    ? t("instance.connection-string.help.neon")
    : helpTextByEngine[engine];

  const invalidTextByEngine: Record<ConnectionStringEngine, string> = {
    [Engine.MYSQL]: t("instance.connection-string.invalid.mysql"),
    [Engine.POSTGRES]: t("instance.connection-string.invalid.postgres"),
    [Engine.MSSQL]: t("instance.connection-string.invalid.mssql"),
    [Engine.ORACLE]: t("instance.connection-string.invalid.oracle"),
    [Engine.MARIADB]: t("instance.connection-string.invalid.mariadb"),
    [Engine.TIDB]: t("instance.connection-string.invalid.tidb"),
    [Engine.OCEANBASE]: t("instance.connection-string.invalid.oceanbase"),
    [Engine.STARROCKS]: t("instance.connection-string.invalid.starrocks"),
    [Engine.DORIS]: t("instance.connection-string.invalid.doris"),
    [Engine.COCKROACHDB]: t("instance.connection-string.invalid.cockroachdb"),
    [Engine.REDSHIFT]: t("instance.connection-string.invalid.redshift"),
    [Engine.MONGODB]: t("instance.connection-string.invalid.mongodb"),
    [Engine.REDIS]: t("instance.connection-string.invalid.redis"),
    [Engine.CASSANDRA]: t("instance.connection-string.invalid.cassandra"),
    [Engine.ELASTICSEARCH]: t(
      "instance.connection-string.invalid.elasticsearch"
    ),
    [Engine.CLICKHOUSE]: t("instance.connection-string.invalid.clickhouse"),
    [Engine.SNOWFLAKE]: t("instance.connection-string.invalid.snowflake"),
    [Engine.TRINO]: t("instance.connection-string.invalid.trino"),
    [Engine.HIVE]: t("instance.connection-string.invalid.hive"),
    [Engine.DATABRICKS]: t("instance.connection-string.invalid.databricks"),
  };
  const invalidText = isNeon
    ? t("instance.connection-string.invalid.neon")
    : invalidTextByEngine[engine];

  const detectedEngine = invalid
    ? detectConnectionStringEngine(value, engine)
    : undefined;
  const errorMessage = detectedEngine
    ? t("instance.connection-string.invalid.engine-mismatch", {
        detected: engineNameV1(detectedEngine),
        selected: engineNameV1(engine),
      })
    : invalidText;

  const handleOpenChange = (nextOpen: boolean) => {
    setOpen(nextOpen);
    if (!nextOpen) {
      setValue("");
      setChecked(false);
    }
  };

  const tryImport = (text: string): boolean => {
    const parsed = parseConnectionString(engine, text, vendor);
    if (!parsed) return false;
    const fillState: ConnectionUrlFillState =
      isNeon && isNeonPooledHost(parsed.host)
        ? "neon-pooled"
        : engine === Engine.DATABRICKS
          ? parsed.token !== undefined
            ? "complete"
            : "missing-token"
          : parsed.password !== undefined
            ? "complete"
            : "missing-password";
    onParsed(parsed, fillState);
    const focusTarget: ConnectionUrlFocusTarget | null =
      fillState === "neon-pooled"
        ? "host"
        : fillState !== "complete"
          ? "password"
          : null;
    if (focusTarget) {
      focusOnCloseRef.current = focusTarget;
      setPendingFocusTarget(focusTarget);
    }
    handleOpenChange(false);
    return true;
  };

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger
        ref={triggerRef}
        render={
          <Button
            type="button"
            appearance="soft"
            size="sm"
            disabled={disabled}
            className="group relative gap-1.5 pl-2 pr-2.5 transition-[color,background-color,transform] duration-150 ease-out active:scale-[0.97] data-[popup-open]:bg-control-bg-hover data-[popup-open]:text-control-hover after:absolute after:-inset-y-2 after:inset-x-0 motion-reduce:transition-none"
          />
        }
      >
        <Link
          className="size-3.5 shrink-0 text-control-light transition-colors duration-150 ease-out group-hover:text-control group-data-[popup-open]:text-control motion-reduce:transition-none"
          aria-hidden="true"
        />
        <span>{t("instance.connection-string.trigger")}</span>
      </PopoverTrigger>
      <PopoverContent
        align="start"
        sideOffset={6}
        initialFocus={() =>
          typeof window !== "undefined" &&
          window.matchMedia?.("(hover: none) and (pointer: coarse)")?.matches
            ? false
            : inputRef.current
        }
        finalFocus={() => {
          if (focusOnCloseRef.current) {
            const target = focusOnCloseRef.current;
            focusOnCloseRef.current = null;
            const targetEl = resolveFocusElement(target);
            if (targetEl && !targetEl.disabled) {
              return targetEl;
            }
          }
          return triggerRef.current;
        }}
        className="w-[min(28rem,calc(100vw-2rem))] origin-(--transform-origin) p-4 transition-[transform,opacity] duration-150 ease-[cubic-bezier(0.23,1,0.32,1)] data-[ending-style]:scale-95 data-[ending-style]:opacity-0 data-[ending-style]:duration-100 data-[ending-style]:ease-in data-[starting-style]:scale-95 data-[starting-style]:opacity-0 motion-reduce:transition-none"
      >
        <form
          className="flex flex-col gap-3"
          onSubmit={(event) => {
            event.preventDefault();
            event.stopPropagation();
            if (!value.trim() || !tryImport(value)) {
              setChecked(true);
            }
          }}
        >
          <SharedFormField
            layout="vertical"
            title={
              <FormLabel htmlFor={inputId}>
                {t("instance.connection-string.label")}
              </FormLabel>
            }
            description={helpText}
          >
            <SharedInput
              ref={inputRef}
              id={inputId}
              type="text"
              data-agent-sensitive
              data-1p-ignore
              data-lpignore="true"
              value={value}
              disabled={disabled}
              autoComplete="off"
              spellCheck={false}
              aria-invalid={invalid || undefined}
              aria-describedby={invalid ? errorId : undefined}
              placeholder={
                isNeon
                  ? NEON_CONNECTION_STRING_PLACEHOLDER
                  : CONNECTION_STRING_PLACEHOLDERS[engine]
              }
              className="font-mono"
              onChange={(event) => {
                const next = event.target.value;
                setValue(next);
                if (
                  checked &&
                  (!next.trim() || parseConnectionString(engine, next, vendor))
                ) {
                  setChecked(false);
                }
              }}
              onPaste={(event) => {
                const pasted = event.clipboardData.getData("text");
                const target = event.currentTarget;
                const start = target.selectionStart ?? value.length;
                const end = target.selectionEnd ?? value.length;
                const combined = `${value.slice(0, start)}${pasted}${value.slice(end)}`;
                event.preventDefault();
                if (!tryImport(combined)) {
                  setValue(combined);
                  setChecked(combined.trim() !== "");
                }
              }}
            />
            {invalid && (
              <FormError id={errorId} className="flex items-start gap-1.5">
                <AlertCircle
                  className="mt-0.5 size-3.5 shrink-0"
                  aria-hidden="true"
                />
                <span>{errorMessage}</span>
              </FormError>
            )}
          </SharedFormField>
          <div className="flex items-center justify-end gap-2">
            <Button
              type="button"
              appearance="secondary"
              size="sm"
              onClick={() => handleOpenChange(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button
              type="submit"
              size="sm"
              disabled={disabled || !value.trim()}
            >
              {t("common.import")}
            </Button>
          </div>
        </form>
      </PopoverContent>
    </Popover>
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
          <FormControlRow key={index} style={{ alignItems: "flex-end" }}>
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
              className="mb-1"
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
  onAskAgent?: () => void;
  disabled?: boolean;
  engineOnly?: boolean;
  hideEngine?: boolean;
}

// Engines without a host field (or with a non-local default) drop the local
// development placeholder host when the user switches to them.
const clearLocalPlaceholderHost = (ds: EditDataSource) => {
  if (ds.host === "127.0.0.1" || ds.host === "host.docker.internal") {
    ds.host = "";
  }
};

export function InstanceFormBody({
  onOpenInfoPanel,
  onAskAgent,
  disabled = false,
  engineOnly = false,
  hideEngine = false,
}: InstanceFormBodyProps) {
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
    setDataSourceEditState,
    adminDataSource,
    editingDataSource,
    resetDataSource,
    setResourceIdValidated,
    parent,
  } = ctx;
  const connectionDataSource = editingDataSource ?? adminDataSource;
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
    return connectionDataSource.srv === false
      ? MongoDBConnectionStringSchemaList[0]
      : MongoDBConnectionStringSchemaList[1];
  }, [connectionDataSource.srv]);

  const currentRedisConnectionType = useMemo(() => {
    switch (connectionDataSource.redisType) {
      case DataSource_RedisType.STANDALONE:
        return RedisConnectionType[0];
      case DataSource_RedisType.SENTINEL:
        return RedisConnectionType[1];
      case DataSource_RedisType.CLUSTER:
        return RedisConnectionType[2];
      default:
        return RedisConnectionType[0];
    }
  }, [connectionDataSource.redisType]);

  const showAdditionalAddresses = useMemo(() => {
    if (basicInfo.engine === Engine.CASSANDRA) return true;
    if (basicInfo.engine === Engine.MONGODB && !connectionDataSource.srv)
      return true;
    if (
      basicInfo.engine === Engine.REDIS &&
      (connectionDataSource.redisType === DataSource_RedisType.CLUSTER ||
        connectionDataSource.redisType === DataSource_RedisType.SENTINEL)
    )
      return true;
    return false;
  }, [
    basicInfo.engine,
    connectionDataSource.srv,
    connectionDataSource.redisType,
  ]);

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

  const updateConnectionDataSource = useCallback(
    (partial: Partial<EditDataSource>) => {
      setDataSourceEditState((prev) => ({
        ...prev,
        dataSources: prev.dataSources.map((ds) =>
          ds.id === prev.editingDataSourceId ? { ...ds, ...partial } : ds
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
          const updated = {
            ...ds,
            authenticationType: normalizeAuthenticationType(
              engine,
              ds.authenticationType
            ),
          };
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
      setBasicInfo((prev) => ({
        ...prev,
        engine,
        vendor: Vendor.VENDOR_UNSPECIFIED,
      }));
    },
    [resetDataSource, setDataSourceEditState, setBasicInfo]
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
      updateConnectionDataSource({ redisType });
    },
    [updateConnectionDataSource]
  );

  const handleMongodbConnectionStringSchemaChange = useCallback(
    (type: string) => {
      if (type === MongoDBConnectionStringSchemaList[1]) {
        updateConnectionDataSource({
          port: "",
          additionalAddresses: [],
          replicaSet: "",
          directConnection: false,
          srv: true,
        });
      } else {
        updateConnectionDataSource({ srv: false });
      }
    },
    [updateConnectionDataSource]
  );

  const removeDSAdditionalAddress = useCallback(
    (index: number) => {
      setDataSourceEditState((prev) => ({
        ...prev,
        dataSources: prev.dataSources.map((ds) => {
          if (ds.id !== prev.editingDataSourceId) return ds;
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
        if (ds.id !== prev.editingDataSourceId) return ds;
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
  }, [setDataSourceEditState]);

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
        updateConnectionDataSource({ port: value });
      }
    },
    [updateConnectionDataSource]
  );

  const handleAdditionalAddressHostChange = useCallback(
    (index: number, host: string) => {
      setDataSourceEditState((prev) => ({
        ...prev,
        dataSources: prev.dataSources.map((ds) => {
          if (ds.id !== prev.editingDataSourceId) return ds;
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
          if (ds.id !== prev.editingDataSourceId) return ds;
          const newAddresses = [...ds.additionalAddresses];
          newAddresses[index] = { ...newAddresses[index], port };
          return { ...ds, additionalAddresses: newAddresses };
        }),
      }));
    },
    [setDataSourceEditState]
  );

  const isNeon =
    basicInfo.engine === Engine.POSTGRES && basicInfo.vendor === Vendor.NEON;

  const connectionSectionRef = useRef<HTMLElement>(null);

  const resolvePasswordInput = useCallback(
    () =>
      connectionSectionRef.current?.querySelector<HTMLInputElement>(
        '[data-connection-primary-secret] input[type="password"]'
      ) ??
      connectionSectionRef.current?.querySelector<HTMLInputElement>(
        '[data-agent-sensitive] input[type="password"]'
      ) ??
      null,
    []
  );

  const resolveHostInput = useCallback(
    () =>
      connectionSectionRef.current?.querySelector<HTMLInputElement>("#host") ??
      null,
    []
  );

  const fillFromConnectionString = useCallback(
    (parsed: ParsedConnectionString, fillState: ConnectionUrlFillState) => {
      if (!supportsConnectionString(basicInfo.engine)) return;
      const currentEngine = basicInfo.engine;
      const currentVendor = basicInfo.vendor;
      setDataSourceEditState((prev) => ({
        ...prev,
        dataSources: prev.dataSources.map((ds) =>
          ds.id === prev.editingDataSourceId
            ? applyParsedConnectionString(
                ds,
                currentEngine,
                parsed,
                currentVendor
              )
            : ds
        ),
      }));
      const toastTitle =
        fillState === "neon-pooled"
          ? t("instance.connection-string.filled-neon-pooled")
          : fillState === "missing-token"
            ? t("instance.connection-string.filled-missing-token")
            : fillState === "missing-password"
              ? t("instance.connection-string.filled-missing-password")
              : t("instance.connection-string.filled");
      pushNotification({
        module: "bytebase",
        style: fillState === "neon-pooled" ? "WARN" : "SUCCESS",
        title: toastTitle,
      });
    },
    [basicInfo.engine, basicInfo.vendor, setDataSourceEditState, t]
  );

  const showAskAgent = isCreating && !!onAskAgent;

  const vendorDrafts = useRef(new Map<Vendor, DataSourceEditState>());
  const changeVendor = (vendor: Vendor) => {
    if (isCreating) {
      vendorDrafts.current.set(basicInfo.vendor, ctx.dataSourceEditState);
      let draft = vendorDrafts.current.get(vendor);
      if (!draft && vendor === Vendor.NEON) {
        draft = extractDataSourceEditState(undefined);
        draft.dataSources = draft.dataSources.map((ds) => ({
          ...ds,
          host: "",
          port: "5432",
          useSsl: true,
          verifyTlsCertificate: true,
        }));
      }
      if (draft) setDataSourceEditState(draft);
    }
    if (vendor === Vendor.NEON) {
      setIsEngineSelectorCollapsed(true);
    }
    updateBasicInfo({ vendor });
  };

  const engineSelection = (
    <>
      {isCreating && (
        <FormField title={t("database.engine")}>
          <CollapsibleTilePicker
            label={t("database.engine")}
            value={String(basicInfo.engine)}
            summary={
              <>
                <EngineIcon engine={basicInfo.engine} className="size-4" />
                <span className="truncate">
                  {engineNameV1(basicInfo.engine)}
                </span>
              </>
            }
            collapsed={isEngineSelectorCollapsed}
            onCollapsedChange={setIsEngineSelectorCollapsed}
            disabled={disabled}
            onValueChange={(value) =>
              changeInstanceEngine(Number(value) as Engine)
            }
          >
            {supportedEngineV1List().map((eng) => (
              <TileRadioItem
                key={eng}
                value={String(eng)}
                icon={<EngineIcon engine={eng} className="size-5" />}
                badge={
                  isEngineBeta(eng) ? (
                    <span className="rounded-xs border border-transparent bg-accent/10 px-1.5 py-0.5 text-xs text-accent">
                      Beta
                    </span>
                  ) : undefined
                }
              >
                {engineNameV1(eng)}
              </TileRadioItem>
            ))}
          </CollapsibleTilePicker>
        </FormField>
      )}
      {isCreating && basicInfo.engine === Engine.POSTGRES && (
        <FormField title={t("instance.vendor.self")}>
          <CollapsibleTilePicker
            label={t("instance.vendor.self")}
            value={isNeon ? "neon" : "generic"}
            summary={
              <>
                <VendorIcon vendor={basicInfo.vendor} className="size-4" />
                <span className="truncate">
                  {isNeon
                    ? t("instance.vendor.neon")
                    : t("instance.vendor.generic")}
                </span>
              </>
            }
            disabled={!allowEdit || disabled}
            onValueChange={(value) =>
              changeVendor(
                value === "neon" ? Vendor.NEON : Vendor.VENDOR_UNSPECIFIED
              )
            }
          >
            <TileRadioItem
              value="generic"
              icon={
                <VendorIcon
                  vendor={Vendor.VENDOR_UNSPECIFIED}
                  className="size-5"
                />
              }
            >
              {t("instance.vendor.generic")}
            </TileRadioItem>
            <TileRadioItem
              value="neon"
              icon={<VendorIcon vendor={Vendor.NEON} className="size-5" />}
            >
              {t("instance.vendor.neon")}
            </TileRadioItem>
          </CollapsibleTilePicker>
        </FormField>
      )}
    </>
  );
  if (engineOnly) {
    return (
      <FormFieldGroup density="compact" className="w-full max-w-5xl">
        {engineSelection}
      </FormFieldGroup>
    );
  }

  return (
    <ValidationProvider
      className="flex flex-col pb-2"
      errors={{
        ...ctx.getDataSourceErrors(connectionDataSource),
        ...(!basicInfo.title.trim() ? { title: "required" } : {}),
      }}
    >
      <div className="w-full max-w-5xl flex flex-col">
        {/* Basic Info Card */}
        <FormSection layout="stacked" title={t("instance.section.basic-info")}>
          <div className="flex flex-col gap-4">
            {!hideEngine && engineSelection}

            {/* Instance Name */}
            <FormField validationField="title">
              <FormLabel htmlFor="name" className="flex flex-row items-center">
                {t("instance.instance-name")}
                <span className="ml-0.5 text-error">*</span>
              </FormLabel>
              <Input
                id="name"
                value={basicInfo.title}
                required
                className="w-full"
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
              <EnvironmentSelect
                className="w-full"
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
            </FormField>

            {/* Labels */}
            {(allowEdit || labelKVList.length > 0) && (
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
                  <div className="inline-flex items-center gap-x-1">
                    <FormLabel htmlFor="external-link">
                      {basicInfo.engine === Engine.SNOWFLAKE
                        ? t("instance.snowflake-web-console")
                        : t("instance.external-link")}
                    </FormLabel>
                    {basicInfo.engine !== Engine.SNOWFLAKE && (
                      <Tooltip
                        content={t("instance.sentence.console.snowflake")}
                        popupClassName="max-w-96"
                      >
                        <Button
                          type="button"
                          appearance="link"
                          size="xs"
                          className="-ml-1 w-6 p-0"
                          aria-label={`${t("instance.external-link")} ${t(
                            "common.info"
                          )}`}
                        >
                          <Info className="size-3.5" />
                        </Button>
                      </Tooltip>
                    )}
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
                )}
              </FormField>
            )}

            {/* Scan Interval (edit mode only) */}
            {!isCreating && instance && !ctx.hideAdvancedFeatures && (
              <ScanIntervalInput
                scanInterval={basicInfo.syncInterval}
                allowEdit={allowEdit}
                onScanIntervalChange={changeScanInterval}
              />
            )}
          </div>
        </FormSection>

        {/* Connection Card */}
        <FormSection
          ref={connectionSectionRef}
          layout="stacked"
          title={t("instance.section.connection")}
          action={
            (isCreating && supportsConnectionString(basicInfo.engine)) ||
            showAskAgent ? (
              <div className="flex min-w-0 flex-1 flex-wrap items-center gap-2">
                {isCreating && supportsConnectionString(basicInfo.engine) && (
                  <ConnectionStringImportPopover
                    key={`${basicInfo.engine}-${basicInfo.vendor}`}
                    engine={basicInfo.engine}
                    vendor={basicInfo.vendor}
                    disabled={!allowEdit || disabled}
                    resolvePasswordInput={resolvePasswordInput}
                    resolveHostInput={resolveHostInput}
                    onParsed={fillFromConnectionString}
                  />
                )}
                {showAskAgent && (
                  <AgentHelpButton
                    type="button"
                    size="sm"
                    className="ml-auto"
                    onClick={() => onAskAgent()}
                  >
                    {t("instance.ask-agent")}
                  </AgentHelpButton>
                )}
              </div>
            ) : undefined
          }
        >
          <FormFieldGroup density="compact">
            <DataSourceSection
              hideOptions
              hideAuthentication
              onOpenInfoPanel={onOpenInfoPanel}
            >
              <div className="flex flex-col gap-4">
                {isSaaSMode && (
                  <Alert variant="info">
                    <div className="flex flex-wrap items-center gap-x-2">
                      <span>{t("instance.sentence.firewall-info")}</span>
                      <LearnMoreLink href="https://docs.bytebase.com/get-started/cloud#prerequisites" />
                    </div>
                  </Alert>
                )}
                {editingDataSource && (
                  <DataSourceForm
                    authOnly
                    dataSource={connectionDataSource}
                    onDataSourceChange={handleDataSourceChange}
                    onOpenInfoPanel={onOpenInfoPanel}
                  />
                )}
                {/* Host input */}
                {basicInfo.engine === Engine.SPANNER ? (
                  <SpannerHostInput
                    projectId={connectionDataSource.projectId}
                    instanceId={connectionDataSource.instanceId}
                    endpoint={connectionDataSource.host}
                    port={connectionDataSource.port}
                    onUpdate={(update) => updateConnectionDataSource(update)}
                    allowEdit={allowEdit}
                  />
                ) : basicInfo.engine === Engine.BIGQUERY ? (
                  <BigQueryHostInput
                    projectId={connectionDataSource.projectId}
                    endpoint={connectionDataSource.host}
                    port={connectionDataSource.port}
                    onUpdate={(update) => updateConnectionDataSource(update)}
                    allowEdit={allowEdit}
                  />
                ) : (
                  <FormField
                    validationField="host"
                    showErrors={
                      isNeon && isNeonPooledHost(connectionDataSource.host)
                    }
                    title={
                      <span className="flex items-center gap-1">
                        <FormLabel htmlFor="host">
                          {basicInfo.engine === Engine.SNOWFLAKE
                            ? t("instance.account-locator")
                            : basicInfo.engine === Engine.COSMOSDB
                              ? t("instance.endpoint")
                              : connectionDataSource.authenticationType ===
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
                        value={connectionDataSource.host}
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
                        onChange={(e) =>
                          updateConnectionDataSource({ host: e.target.value })
                        }
                      />
                      {basicInfo.engine !== Engine.DATABRICKS &&
                        basicInfo.engine !== Engine.COSMOSDB &&
                        connectionDataSource.authenticationType !==
                          DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM && (
                          <>
                            <FormLabel htmlFor="port">
                              {t("instance.port")}
                            </FormLabel>
                            <Input
                              id="port"
                              value={connectionDataSource.port}
                              className="w-20 shrink-0"
                              placeholder={defaultPort}
                              disabled={!allowEdit || !allowEditPort}
                              onChange={handlePortChange}
                            />
                          </>
                        )}
                    </div>
                    {connectionDataSource.authenticationType ===
                      DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM && (
                      <p className="text-xs text-control-light">
                        {t(
                          "instance.sentence.google-cloud-sql.instance-name-tips",
                          { instance: "{project-id}:{region}:{instance-name}" }
                        )}
                      </p>
                    )}
                    {basicInfo.engine === Engine.SNOWFLAKE && (
                      <div className="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm">
                        <span className="text-control-light">
                          {t("instance.account-locator-description")}
                        </span>
                        <LearnMoreLink href="https://docs.snowflake.com/en/user-guide/admin-account-identifier#using-an-account-locator-as-an-identifier" />
                      </div>
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
                        handleMongodbConnectionStringSchemaChange(
                          value as string
                        )
                      }
                      aria-label={t("data-source.connection-string-schema")}
                    >
                      {MongoDBConnectionStringSchemaList.map((type) => (
                        <RadioGroupItem
                          key={type}
                          value={type}
                          disabled={!allowEdit}
                        >
                          {type}
                        </RadioGroupItem>
                      ))}
                    </RadioGroup>
                    {!connectionDataSource.srv && (
                      <ResponsiveFormLayout
                        layout="inherit"
                        density="compact"
                        className="mt-2"
                      >
                        <fieldset
                          aria-label={t("data-source.connection-string-schema")}
                          className="flex flex-col gap-4 rounded-xs border border-control-border px-3 py-2"
                        >
                          {/* Additional addresses */}
                          {showAdditionalAddresses && (
                            <AdditionalAddressesFields
                              addresses={
                                connectionDataSource.additionalAddresses
                              }
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
                            !connectionDataSource.srv && (
                              <FormField>
                                <FormLabel htmlFor="replicaSet">
                                  {t("data-source.replica-set")}
                                </FormLabel>
                                <Input
                                  value={connectionDataSource.replicaSet}
                                  required
                                  className="w-full"
                                  disabled={!allowEdit}
                                  onChange={(e) =>
                                    updateConnectionDataSource({
                                      replicaSet: e.target.value,
                                    })
                                  }
                                />
                              </FormField>
                            )}
                          {/* MongoDB direct connection */}
                          {basicInfo.engine === Engine.MONGODB &&
                            !connectionDataSource.srv &&
                            connectionDataSource.additionalAddresses.length ===
                              0 && (
                              <FormControlRow className="w-fit">
                                <Switch
                                  id="directConnection"
                                  checked={
                                    connectionDataSource.directConnection
                                  }
                                  disabled={!allowEdit}
                                  onCheckedChange={(checked) =>
                                    updateConnectionDataSource({
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
                    <Select
                      value={currentRedisConnectionType}
                      onValueChange={(value) => {
                        if (value) handleRedisConnectionTypeChange(value);
                      }}
                      disabled={!allowEdit}
                    >
                      <SelectTrigger
                        aria-label={t("data-source.connection-type")}
                        className="w-full sm:w-80"
                      >
                        <SelectValue>{currentRedisConnectionType}</SelectValue>
                      </SelectTrigger>
                      <SelectContent>
                        {RedisConnectionType.map((type) => (
                          <SelectItem key={type} value={type}>
                            {type}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    {showAdditionalAddresses && (
                      <ResponsiveFormLayout
                        layout="inherit"
                        density="compact"
                        className="mt-2"
                      >
                        <fieldset
                          aria-label={currentRedisConnectionType}
                          className="flex flex-col gap-4 rounded-xs border border-control-border px-3 py-2"
                        >
                          <AdditionalAddressesFields
                            addresses={connectionDataSource.additionalAddresses}
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
                      <ResponsiveFormLayout
                        layout="inherit"
                        density="compact"
                        className="mt-2"
                      >
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
                    addresses={connectionDataSource.additionalAddresses}
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
            </DataSourceSection>

            {basicInfo.engine !== Engine.DYNAMODB && editingDataSource && (
              <DataSourceForm
                dataSource={editingDataSource}
                optionsOnly
                onDataSourceChange={handleDataSourceChange}
                onOpenInfoPanel={onOpenInfoPanel}
              />
            )}

            {basicInfo.engine !== Engine.DYNAMODB &&
              connectionDataSource.type === DataSourceType.ADMIN && (
                <SyncDatabases
                  isCreating={isCreating}
                  showLabel
                  allowEdit={
                    isCreating ? allowEdit && !!allowCreate : allowEdit
                  }
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
              )}
          </FormFieldGroup>
        </FormSection>
      </div>
    </ValidationProvider>
  );
}
