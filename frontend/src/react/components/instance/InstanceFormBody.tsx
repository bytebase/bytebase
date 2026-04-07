import { create } from "@bufbuild/protobuf";
import type { Duration } from "@bufbuild/protobuf/wkt";
import { DurationSchema } from "@bufbuild/protobuf/wkt";
import {
  ChevronDown,
  ChevronRight,
  ExternalLink,
  Plus,
  Trash2,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { EnvironmentSelect } from "@/react/components/EnvironmentSelect";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import {
  pushNotification,
  useActuatorV1Store,
  useDatabaseV1Store,
  useInstanceV1Store,
  useSubscriptionV1Store,
} from "@/store";
import {
  environmentNamePrefix,
  instanceNamePrefix,
} from "@/store/modules/v1/common";
import { isValidEnvironmentName, UNKNOWN_ID } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  DataSource_AddressSchema,
  DataSource_AuthenticationType,
  DataSource_RedisType,
  DataSourceType,
} from "@/types/proto-es/v1/instance_service_pb";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";
import {
  engineNameV1,
  extractInstanceResourceName,
  isDev,
  isValidSpannerHost,
  supportedEngineV1List,
  urlfy,
} from "@/utils";
import type { EditDataSource } from "./common";
import {
  EngineIconPath,
  MongoDBConnectionStringSchemaList,
  RedisConnectionType,
  SnowflakeExtraLinkPlaceHolder,
} from "./constants";
import { DataSourceForm } from "./DataSourceForm";
import { DataSourceSection } from "./DataSourceSection";
import { useInstanceFormContext } from "./InstanceFormContext";
import { hasInfoContent, type InfoSection } from "./info-content";

// --- Inline sub-components ---

function SpannerHostInput({
  host,
  onHostChange,
  allowEdit,
}: {
  host: string;
  onHostChange: (host: string) => void;
  allowEdit: boolean;
}) {
  const { t } = useTranslation();
  const RE =
    /^projects\/(?<PROJECT_ID>(?:[a-z]|[-.:]|[0-9])*)\/instances\/(?<INSTANCE_ID>(?:[a-z]|[-]|[0-9])*)$/;
  const RE_PROJECT_ID = /^(?:[a-z]|[-.:]|[0-9])+$/;
  const RE_INSTANCE_ID = /^(?:[a-z]|[-]|[0-9])+$/;

  const parseProjectId = (h: string) => h.match(RE)?.groups?.PROJECT_ID ?? "";
  const parseInstanceId = (h: string) => h.match(RE)?.groups?.INSTANCE_ID ?? "";

  const [projectId, setProjectId] = useState(() => parseProjectId(host));
  const [instanceId, setInstanceId] = useState(() => parseInstanceId(host));
  const [dirty, setDirty] = useState(false);

  useEffect(() => {
    if (!host) return;
    setProjectId(parseProjectId(host));
    setInstanceId(parseInstanceId(host));
  }, [host]);

  const update = useCallback(
    (pId: string, iId: string) => {
      setDirty(true);
      if (!RE_PROJECT_ID.test(pId) || !RE_INSTANCE_ID.test(iId)) {
        onHostChange("");
        return;
      }
      onHostChange(`projects/${pId}/instances/${iId}`);
    },
    [onHostChange]
  );

  const isValidProjectId = RE_PROJECT_ID.test(projectId);
  const isValidInstanceId = RE_INSTANCE_ID.test(instanceId);

  return (
    <div className="grid grid-cols-2 gap-x-2 gap-y-1">
      <div>
        <label className="textlabel">
          {t("instance.project-id")}
          <span style={{ color: "red" }}> *</span>
        </label>
        <Input
          value={projectId}
          required
          placeholder="projectId"
          className={`mt-1 w-full ${dirty && !isValidProjectId ? "border-error" : ""}`}
          disabled={!allowEdit}
          onChange={(e) => {
            const v = e.target.value;
            setProjectId(v);
            update(v, instanceId);
          }}
        />
      </div>
      <div>
        <label className="textlabel">
          {t("instance.instance-id")}
          <span style={{ color: "red" }}> *</span>
        </label>
        <Input
          value={instanceId}
          required
          placeholder="instanceId"
          className={`mt-1 w-full ${dirty && !isValidInstanceId ? "border-error" : ""}`}
          disabled={!allowEdit}
          onChange={(e) => {
            const v = e.target.value;
            setInstanceId(v);
            update(projectId, v);
          }}
        />
      </div>
      <p className="col-span-2 textinfolabel">
        {t("instance.find-gcp-project-id-and-instance-id")}{" "}
        <a
          href="https://docs.bytebase.com/get-started/connect/gcp?source=console"
          target="_blank"
          rel="noopener noreferrer"
          className="normal-link inline-flex items-center"
        >
          {t("common.detailed-guide")}
          <ExternalLink className="w-4 h-4 ml-1" />
        </a>
      </p>
    </div>
  );
}

function BigQueryHostInput({
  host,
  onHostChange,
  allowEdit,
}: {
  host: string;
  onHostChange: (host: string) => void;
  allowEdit: boolean;
}) {
  const { t } = useTranslation();
  const RE_PROJECT_ID = /^(?:[a-z]|[-.:]|[0-9])+$/;
  const [projectId, setProjectId] = useState(() => host || "");
  const [dirty, setDirty] = useState(false);

  useEffect(() => {
    if (!host) return;
    setProjectId(host);
  }, [host]);

  const isValidProjectId = RE_PROJECT_ID.test(projectId);

  return (
    <div className="grid grid-cols-2 gap-x-2 gap-y-1">
      <div>
        <label className="textlabel">
          {t("instance.project-id")}
          <span style={{ color: "red" }}> *</span>
        </label>
        <Input
          value={projectId}
          required
          placeholder="projectId"
          className={`mt-1 w-full ${dirty && !isValidProjectId ? "border-error" : ""}`}
          disabled={!allowEdit}
          onChange={(e) => {
            const v = e.target.value;
            setProjectId(v);
            setDirty(true);
            if (!RE_PROJECT_ID.test(v)) {
              onHostChange("");
            } else {
              onHostChange(v);
            }
          }}
        />
      </div>
      <p className="col-span-2 textinfolabel">
        {t("instance.find-gcp-project-id")}{" "}
        <a
          href="https://docs.bytebase.com/get-started/connect/gcp?source=console"
          target="_blank"
          rel="noopener noreferrer"
          className="normal-link inline-flex items-center"
        >
          {t("common.detailed-guide")}
          <ExternalLink className="w-4 h-4 ml-1" />
        </a>
      </p>
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
        <button
          key={eng}
          type="button"
          className={`flex items-center gap-x-2 rounded-sm border px-3 py-2 text-sm text-left transition-colors ${
            eng === engine
              ? "border-accent bg-accent/5 ring-1 ring-accent"
              : "border-control-border hover:border-accent/50 hover:bg-gray-50"
          }`}
          onClick={() => onEngineChange(eng)}
        >
          {EngineIconPath[eng] && (
            <img
              src={EngineIconPath[eng]}
              alt=""
              className="w-5 h-5 shrink-0"
            />
          )}
          <span className="truncate">{engineNameV1(eng)}</span>
          {isEngineBeta(eng) && (
            <span className="ml-auto shrink-0 rounded-full bg-blue-100 px-2 py-0.5 text-xs text-blue-600">
              Beta
            </span>
          )}
        </button>
      ))}
    </div>
  );
}

// Matches Vue ResourceIdField component behavior exactly:
// - Auto-generates resource ID from title with escape + optional random suffix on duplicate
// - Shows inline "Instance ID: xxx  It cannot be changed later. Edit" by default
// - Only shows input field when "Edit" is clicked (manualEdit mode)
// - Validates: empty, min/max length, pattern, duplicate via fetchResource

const RESOURCE_ID_CHARS = "abcdefghijklmnopqrstuvwxyz1234567890-";
const RESOURCE_ID_PATTERN = /^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$/;

function randomString(len: number): string {
  const chars = "abcdefghijklmnopqrstuvwxyz0123456789";
  let result = "";
  for (let i = 0; i < len; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return result;
}

function escapeResourceTitle(str: string): string {
  return str
    .toLowerCase()
    .split("")
    .map((char) => {
      if (char === " ") return "-";
      if (char.match(/\s/)) return "";
      if (RESOURCE_ID_CHARS.includes(char)) return char;
      // Map non-ASCII to a deterministic letter
      const alpha = "abcdefghijklmnopqrstuvwxyz";
      return alpha.charAt(char.charCodeAt(0) % alpha.length);
    })
    .join("")
    .toLowerCase();
}

interface ValidatedMessage {
  type: "error" | "warning";
  message: string;
}

function ResourceIdField({
  value,
  onChange,
  onValidationChange,
  resourceTitle,
  readonly,
  fetchResource,
}: {
  value: string;
  onChange: (id: string) => void;
  onValidationChange?: (valid: boolean) => void;
  resourceTitle: string;
  readonly: boolean;
  fetchResource?: (id: string) => Promise<unknown>;
}) {
  const { t } = useTranslation();
  const [resourceId, setResourceId] = useState(value);
  const [manualEdit, setManualEdit] = useState(false);
  const [validatedMessages, setValidatedMessages] = useState<
    ValidatedMessage[]
  >([]);
  const initializedRef = useRef(false);

  const resourceName = t("dynamic.resource.instance");

  const validate = useCallback(
    async (id: string): Promise<ValidatedMessage[]> => {
      const msgs: ValidatedMessage[] = [];
      if (id === "") {
        msgs.push({
          type: "error",
          message: t("resource-id.validation.empty", {
            resource: resourceName,
          }),
        });
      } else if (id.length > 64) {
        msgs.push({
          type: "error",
          message: t("resource-id.validation.overflow", {
            resource: resourceName,
          }),
        });
      } else if (!RESOURCE_ID_PATTERN.test(id)) {
        msgs.push({
          type: "error",
          message: t("resource-id.validation.pattern", {
            resource: resourceName,
          }),
        });
      }

      if (msgs.length === 0 && fetchResource && id && !readonly) {
        try {
          const resource = await fetchResource(id);
          if (resource) {
            msgs.push({
              type: "error",
              message: t("resource-id.validation.duplicated", {
                resource: resourceName,
              }),
            });
          }
        } catch {
          // NotFound = available, which is good
        }
      }
      return msgs;
    },
    [fetchResource, readonly, resourceName, t]
  );

  const handleResourceIdChange = useCallback(
    async (newValue: string) => {
      setResourceId(newValue);
      onChange(newValue);
      const msgs = await validate(newValue);
      setValidatedMessages(msgs);
      onValidationChange?.(msgs.length === 0 && newValue !== "");
      return msgs;
    },
    [onChange, validate, onValidationChange]
  );

  // Auto-generate from title (mirrors Vue watcher on resourceTitle)
  useEffect(() => {
    if (readonly || manualEdit) return;

    const escapedTitle = escapeResourceTitle(resourceTitle);
    const name = escapedTitle || "";
    if (!name) return;

    (async () => {
      const msgs = await handleResourceIdChange(name);

      // On first init, if duplicate, append random suffix
      if (!initializedRef.current) {
        if (msgs.length > 0) {
          await handleResourceIdChange(
            name + "-" + randomString(4).toLowerCase()
          );
        }
      }
      initializedRef.current = true;
    })();
  }, [resourceTitle]);

  if (readonly) {
    if (!resourceId) return null;
    return (
      <div className="sm:col-span-3 sm:col-start-1 -mt-4">
        <div className="mt-4 textinfolabel text-sm flex items-center gap-x-1">
          {t("resource-id.self", { resource: resourceName })}:{" "}
          <span className="text-gray-600 font-medium">{resourceId}</span>
        </div>
      </div>
    );
  }

  if (!manualEdit) {
    return (
      <div className="sm:col-span-3 sm:col-start-1 -mt-4">
        <div className="mt-4 textinfolabel text-sm flex items-start flex-wrap gap-x-1">
          <div className="flex items-center gap-x-1">
            {t("resource-id.self", { resource: resourceName })}:
            {resourceId ? (
              <span className="text-gray-600 font-medium mr-1">
                {resourceId}
              </span>
            ) : (
              <span className="text-control-placeholder italic">
                &lt;EMPTY&gt;
              </span>
            )}
          </div>
          <div>
            <span>{t("resource-id.cannot-be-changed-later")}</span>
            <span
              className="text-accent font-medium cursor-pointer hover:opacity-80 ml-1"
              onClick={() => setManualEdit(true)}
            >
              {t("common.edit")}
            </span>
          </div>
        </div>
        {validatedMessages.length > 0 && (
          <ul className="w-full my-2 flex flex-col gap-y-2 list-disc list-outside pl-4">
            {validatedMessages.map((msg) => (
              <li
                key={msg.message}
                className={`text-xs break-words ${
                  msg.type === "warning" ? "text-yellow-600" : "text-red-600"
                }`}
              >
                {msg.message}
              </li>
            ))}
          </ul>
        )}
      </div>
    );
  }

  return (
    <div className="sm:col-span-3 sm:col-start-1 -mt-4">
      <div className="mt-4">
        <label className="textlabel flex items-center">
          {t("resource-id.self", { resource: resourceName })}
          <span className="ml-0.5 text-error">*</span>
        </label>
        <div className="textinfolabel mb-2 mt-1">
          {t("resource-id.description", { resource: resourceName })}
        </div>
        <Input
          value={resourceId}
          className={`w-full max-w-[40rem] ${
            validatedMessages.some((m) => m.type === "error")
              ? "border-error"
              : ""
          }`}
          placeholder={t("resource-id.self", { resource: resourceName })}
          onChange={(e) => {
            handleResourceIdChange(e.target.value);
          }}
        />
      </div>
      {validatedMessages.length > 0 && (
        <ul className="w-full my-2 flex flex-col gap-y-2 list-disc list-outside pl-4">
          {validatedMessages.map((msg) => (
            <li
              key={msg.message}
              className={`text-xs break-words ${
                msg.type === "warning" ? "text-yellow-600" : "text-red-600"
              }`}
            >
              {msg.message}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

// Matches Vue LabelListEditor + LabelEditorRow exactly
const MAX_LABEL_VALUE_LENGTH = 256;

function LabelListEditor({
  kvList,
  onChange,
  readonly,
  showErrors = true,
  onErrorsChange,
}: {
  kvList: { key: string; value: string }[];
  onChange: (list: { key: string; value: string }[]) => void;
  readonly: boolean;
  showErrors?: boolean;
  onErrorsChange?: (errors: string[]) => void;
}) {
  const { t } = useTranslation();

  // Compute validation errors for each kv pair
  const errorList = useMemo(() => {
    return kvList.map((kv) => {
      const errors = { key: [] as string[], value: [] as string[] };
      if (!kv.key) {
        errors.key.push(t("label.error.key-necessary"));
      } else if (kvList.filter((k) => k.key === kv.key).length > 1) {
        errors.key.push(t("label.error.key-duplicated"));
      }
      if (!kv.value) {
        errors.value.push(t("label.error.value-necessary"));
      } else if (kv.value.length > MAX_LABEL_VALUE_LENGTH) {
        errors.value.push(
          t("label.error.max-value-length-exceeded", {
            length: MAX_LABEL_VALUE_LENGTH,
          })
        );
      }
      return errors;
    });
  }, [kvList, t]);

  const flatErrors = useMemo(() => {
    return errorList.flatMap((e) => [...e.key, ...e.value]);
  }, [errorList]);

  const hasErrors = flatErrors.length > 0;

  useEffect(() => {
    onErrorsChange?.(flatErrors);
  }, [flatErrors, onErrorsChange]);

  const allowAddLabel = !hasErrors;

  const updateKey = (index: number, key: string) => {
    onChange(kvList.map((kv, i) => (i === index ? { ...kv, key } : kv)));
  };
  const updateValue = (index: number, value: string) => {
    onChange(kvList.map((kv, i) => (i === index ? { ...kv, value } : kv)));
  };
  const handleRemove = (index: number) => {
    onChange(kvList.filter((_, i) => i !== index));
  };
  const handleAdd = () => {
    onChange([...kvList, { key: "", value: "" }]);
  };

  return (
    <div className="flex flex-col gap-y-2">
      <div className="flex flex-wrap gap-x-2 gap-y-2">
        {kvList.map((kv, index) => {
          const errors = showErrors
            ? (errorList[index] ?? { key: [], value: [] })
            : { key: [], value: [] };
          const combinedErrors = [...errors.key, ...errors.value];
          return (
            <div key={index} className="flex flex-col gap-y-1">
              <div className="text-sm flex gap-x-2">
                <div className="flex flex-col">
                  <span className="text-xs font-medium mb-1">
                    Key {index + 1}
                  </span>
                  {readonly ? (
                    <span className="leading-[34px]">{kv.key}</span>
                  ) : (
                    <Input
                      value={kv.key}
                      placeholder={t("setting.label.key-placeholder")}
                      className={errors.key.length > 0 ? "border-error" : ""}
                      onChange={(e) => updateKey(index, e.target.value)}
                    />
                  )}
                </div>
                <div className="flex flex-col">
                  <span className="text-xs font-medium mb-1">
                    Value {index + 1}
                  </span>
                  <div className="flex items-center gap-x-2">
                    {readonly ? (
                      <span className="leading-[34px]">
                        {kv.value || (
                          <span className="text-control-placeholder">
                            {t("label.empty-label-value")}
                          </span>
                        )}
                      </span>
                    ) : (
                      <Input
                        value={kv.value}
                        placeholder={t("setting.label.value-placeholder")}
                        className={
                          errors.value.length > 0 ? "border-error" : ""
                        }
                        onChange={(e) => updateValue(index, e.target.value)}
                      />
                    )}
                    <button
                      type="button"
                      className={`ml-1 ${readonly ? "invisible" : "visible"} text-control-light hover:text-error`}
                      onClick={() => handleRemove(index)}
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                </div>
              </div>
              {combinedErrors.length > 0 && (
                <ul className="text-xs text-error list-disc list-outside pl-4">
                  {combinedErrors.map((err) => (
                    <li key={err}>{err}</li>
                  ))}
                </ul>
              )}
            </div>
          );
        })}
      </div>
      <div>
        <Button
          variant="outline"
          size="sm"
          disabled={readonly || !allowAddLabel}
          onClick={handleAdd}
        >
          <Plus className="w-4 h-4 mr-1" />
          {t("label.add-label")}
        </Button>
      </div>
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
    <div className="sm:col-span-4 sm:col-start-1 flex flex-col gap-y-2">
      <div className="flex items-center gap-x-2">
        <label className="textlabel">{t("instance.scan-interval.self")}</label>
      </div>
      <div className="textinfolabel">
        {t("instance.scan-interval.description")}
      </div>
      <div className="flex items-center gap-x-6">
        <label className="flex items-center gap-x-2 cursor-pointer">
          <input
            type="radio"
            checked={mode === "DEFAULT"}
            disabled={!allowEdit}
            onChange={() => handleModeChange("DEFAULT")}
          />
          {t("instance.scan-interval.default-never")}
        </label>
        <label className="flex items-center gap-x-2 cursor-pointer">
          <input
            type="radio"
            checked={mode === "CUSTOM"}
            disabled={!allowEdit}
            onChange={() => handleModeChange("CUSTOM")}
          />
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
        </label>
      </div>
    </div>
  );
}

function SyncDatabases({
  isCreating: isCreatingProp,
  showLabel,
  allowEdit,
  syncDatabases,
  onSyncDatabasesChange,
}: {
  isCreating: boolean;
  showLabel: boolean;
  allowEdit: boolean;
  syncDatabases: string[];
  onSyncDatabasesChange: (databases: string[]) => void;
}) {
  const { t } = useTranslation();
  const ctx = useInstanceFormContext();
  const { hideAdvancedFeatures, instance } = ctx;
  const instanceStore = useInstanceV1Store();

  const [syncAll, setSyncAll] = useState(syncDatabases.length === 0);
  const [selectedDatabases, setSelectedDatabases] = useState<string[]>([
    ...syncDatabases,
  ]);
  const [databaseList, setDatabaseList] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(false);
  const [searchText, setSearchText] = useState("");
  const [inputDatabase, setInputDatabase] = useState("");

  useEffect(() => {
    if (syncAll) {
      onSyncDatabasesChange([]);
    } else {
      onSyncDatabasesChange(selectedDatabases);
    }
  }, [syncAll, selectedDatabases]);

  useEffect(() => {
    if (syncAll) return;
    const fetchDatabases = async () => {
      const inst = isCreatingProp ? ctx.pendingCreateInstance : instance;
      if (!inst) return;
      setLoading(true);
      try {
        const resp = await instanceStore.listInstanceDatabases(
          inst.name,
          isCreatingProp ? inst : undefined
        );
        setDatabaseList(new Set([...resp.databases, ...selectedDatabases]));
      } finally {
        setLoading(false);
      }
    };
    fetchDatabases();
  }, [syncAll]);

  if (hideAdvancedFeatures) return null;

  const filteredDatabases = [...databaseList].filter((db) =>
    db.toLowerCase().includes(searchText.toLowerCase())
  );

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.nativeEvent.isComposing) return;
    const trimmed = inputDatabase.trim();
    if (!trimmed) return;
    if (e.key === "Enter") {
      setDatabaseList((prev) => new Set([...prev, trimmed]));
      setSelectedDatabases((prev) => [...prev, trimmed]);
      setInputDatabase("");
    }
  };

  const toggleDatabase = (db: string) => {
    setSelectedDatabases((prev) =>
      prev.includes(db) ? prev.filter((d) => d !== db) : [...prev, db]
    );
  };

  return (
    <div className="sm:col-span-4 sm:col-start-1 flex flex-col gap-y-2">
      {showLabel && (
        <div className="flex items-center gap-x-2">
          <label className="textlabel">
            {t("instance.sync-databases.self")}
          </label>
        </div>
      )}
      <div className="textinfolabel">
        {t("instance.sync-databases.description")}
      </div>
      <div className="flex flex-col gap-y-2">
        <label className="flex items-center gap-x-2 cursor-pointer">
          <input
            type="checkbox"
            checked={syncAll}
            disabled={!allowEdit}
            onChange={(e) => setSyncAll(e.target.checked)}
          />
          {t("instance.sync-databases.sync-all")}
        </label>
        {!syncAll && (
          <div>
            {loading ? (
              <div className="opacity-60 text-sm text-control-light">
                {t("common.loading")}...
              </div>
            ) : (
              <div className="border rounded-xs p-2 flex flex-col gap-y-2">
                <Input
                  value={searchText}
                  className="w-full"
                  placeholder={t("instance.sync-databases.search-database")}
                  onChange={(e) => setSearchText(e.target.value)}
                />
                <div className="max-h-[250px] overflow-y-auto flex flex-col gap-y-1">
                  {filteredDatabases.map((db) => (
                    <label
                      key={db}
                      className="flex items-center gap-x-2 cursor-pointer text-sm"
                    >
                      <input
                        type="checkbox"
                        checked={selectedDatabases.includes(db)}
                        disabled={!allowEdit}
                        onChange={() => toggleDatabase(db)}
                      />
                      <span>{db}</span>
                    </label>
                  ))}
                </div>
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
    </div>
  );
}

// --- Main component ---

interface InstanceFormBodyProps {
  onOpenInfoPanel?: (section: InfoSection) => void;
}

export function InstanceFormBody({ onOpenInfoPanel }: InstanceFormBodyProps) {
  const { t } = useTranslation();
  const ctx = useInstanceFormContext();
  const {
    instance,
    state,
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
    checkDataSource,
    testConnection,
    resetDataSource,
    showConnectionOptionsEvent,
    emitShowConnectionOptions,
    setResourceIdValidated,
  } = ctx;
  const { isEngineBeta, defaultPort, instanceLink, allowEditPort } = specs;

  const instanceV1Store = useInstanceV1Store();
  const actuatorStore = useActuatorV1Store();
  const subscriptionStore = useSubscriptionV1Store();

  const [isEngineSelectorCollapsed, setIsEngineSelectorCollapsed] =
    useState(false);
  const [isConnectionOptionsCollapsed, setIsConnectionOptionsCollapsed] =
    useState(true);

  // Auto-expand connection options when configured
  const showConnectionOptionsCard =
    basicInfo.engine !== Engine.DYNAMODB && !!editingDataSource;

  const hasConfiguredConnectionOptions = useMemo(() => {
    const ds = editingDataSource;
    if (!ds) return false;
    const hasExtraParameters =
      Object.keys(ds.extraConnectionParameters ?? {}).length > 0;
    const hasSslConfig = !!(ds.useSsl || ds.sslCa || ds.sslCert || ds.sslKey);
    const hasSshConfig = !!(
      ds.sshHost ||
      ds.sshPort ||
      ds.sshUser ||
      ds.sshPassword ||
      ds.sshPrivateKey
    );
    return hasExtraParameters || hasSslConfig || hasSshConfig;
  }, [editingDataSource]);

  // Collapse state management based on visibility and configuration
  const prevShowRef = useRef(showConnectionOptionsCard);
  const prevConfiguredRef = useRef(hasConfiguredConnectionOptions);
  useEffect(() => {
    if (!showConnectionOptionsCard) {
      prevShowRef.current = false;
      return;
    }
    const becameVisible = !prevShowRef.current;
    if (becameVisible) {
      setIsConnectionOptionsCollapsed(
        isCreating ? true : !hasConfiguredConnectionOptions
      );
      prevShowRef.current = true;
      prevConfiguredRef.current = hasConfiguredConnectionOptions;
      return;
    }
    if (
      !isCreating &&
      hasConfiguredConnectionOptions &&
      !prevConfiguredRef.current
    ) {
      setIsConnectionOptionsCollapsed(false);
    }
    prevShowRef.current = showConnectionOptionsCard;
    prevConfiguredRef.current = hasConfiguredConnectionOptions;
  }, [showConnectionOptionsCard, isCreating, hasConfiguredConnectionOptions]);

  // Listen for show-connection-options event
  const prevEventRef = useRef(showConnectionOptionsEvent);
  useEffect(() => {
    if (showConnectionOptionsEvent !== prevEventRef.current) {
      prevEventRef.current = showConnectionOptionsEvent;
      if (showConnectionOptionsCard) {
        setIsConnectionOptionsCollapsed(false);
      }
    }
  }, [showConnectionOptionsEvent, showConnectionOptionsCard]);

  // --- Computed values ---

  const availableLicenseCount = useMemo(
    () =>
      Math.max(
        0,
        subscriptionStore.instanceLicenseCount -
          actuatorStore.activatedInstanceCount
      ),
    [
      subscriptionStore.instanceLicenseCount,
      actuatorStore.activatedInstanceCount,
    ]
  );

  const availableLicenseCountText = useMemo((): string => {
    if (subscriptionStore.instanceLicenseCount === Number.MAX_VALUE) {
      return t("common.unlimited");
    }
    return `${availableLicenseCount}`;
  }, [subscriptionStore.instanceLicenseCount, availableLicenseCount, t]);

  const resourceId = useMemo(() => {
    const id = extractInstanceResourceName(basicInfo.name);
    if (id === String(UNKNOWN_ID)) return "";
    return id;
  }, [basicInfo.name]);

  const setResourceId = useCallback(
    (id: string) => {
      setBasicInfo((prev) => ({ ...prev, name: `instances/${id}` }));
    },
    [setBasicInfo]
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

  const allowTestConnection = useMemo(() => {
    if (!allowEdit || state.isRequesting || state.isTestingConnection) {
      return false;
    }
    const ds = editingDataSource;
    if (!ds) return false;
    if (basicInfo.engine === Engine.SPANNER) {
      return isValidSpannerHost(ds.host);
    }
    if (basicInfo.engine === Engine.BIGQUERY) {
      return ds.host !== "";
    }
    if (basicInfo.engine !== Engine.DYNAMODB && ds.host === "") {
      return false;
    }
    return checkDataSource([ds]);
  }, [allowEdit, state, editingDataSource, basicInfo.engine, checkDataSource]);

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
            case Engine.SNOWFLAKE:
            case Engine.DYNAMODB: {
              if (
                updated.host === "127.0.0.1" ||
                updated.host === "host.docker.internal"
              ) {
                updated.host = "";
              }
              break;
            }
            case Engine.SPANNER:
            case Engine.BIGQUERY: {
              updated.authenticationType =
                DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM;
              if (
                updated.host === "127.0.0.1" ||
                updated.host === "host.docker.internal"
              ) {
                updated.host = "";
              }
              break;
            }
            case Engine.COSMOSDB: {
              updated.authenticationType =
                DataSource_AuthenticationType.AZURE_IAM;
              break;
            }
            default: {
              if (!updated.host) {
                updated.host = isDev() ? "127.0.0.1" : "host.docker.internal";
              }
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
    (databases: string[]) => {
      setBasicInfo((prev) => ({ ...prev, syncDatabases: [...databases] }));
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
        const updated = await instanceV1Store.updateInstance(instancePatch, [
          "activation",
        ]);
        useDatabaseV1Store().updateDatabaseInstance(updated);
        await actuatorStore.fetchServerInfo(
          actuatorStore.workspaceResourceName
        );
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.updated"),
        });
      }
    },
    [instance, instanceV1Store, actuatorStore, updateBasicInfo, t]
  );

  const testConnectionForCurrentEditingDS = useCallback(async () => {
    const ds = editingDataSource;
    if (!ds) return;
    const result = await testConnection(ds, false);
    if (!result.success && hasConfiguredConnectionOptions) {
      emitShowConnectionOptions();
    }
  }, [
    editingDataSource,
    testConnection,
    hasConfiguredConnectionOptions,
    emitShowConnectionOptions,
  ]);

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
    <div className="flex flex-col gap-y-6 pb-2">
      <div className="w-full flex flex-col gap-y-6">
        {/* Engine Selector (create only) */}
        {isCreating && (
          <div className="rounded-lg border border-block-border bg-white">
            <button
              type="button"
              className="w-full flex items-center justify-between gap-x-3 px-4 py-3 text-left transition-colors hover:bg-gray-50"
              onClick={() => setIsEngineSelectorCollapsed((prev) => !prev)}
            >
              <div className="min-w-0">
                <p className="text-[11px] font-medium uppercase tracking-[0.14em] text-control-light">
                  {t("database.engine")}
                </p>
                <div className="mt-1 flex items-center gap-x-1.5">
                  {EngineIconPath[basicInfo.engine] && (
                    <img
                      src={EngineIconPath[basicInfo.engine]}
                      alt=""
                      className="w-4 h-4"
                    />
                  )}
                  <span className="text-sm font-medium text-main">
                    {engineNameV1(basicInfo.engine)}
                  </span>
                  {isEngineBeta(basicInfo.engine) && (
                    <span className="rounded-full bg-blue-100 px-2 py-0.5 text-xs text-blue-600">
                      Beta
                    </span>
                  )}
                </div>
              </div>
              <div className="shrink-0 text-control-light">
                {!isEngineSelectorCollapsed ? (
                  <ChevronDown className="w-4 h-4" />
                ) : (
                  <ChevronRight className="w-4 h-4" />
                )}
              </div>
            </button>

            {!isEngineSelectorCollapsed && (
              <div className="border-t border-block-border px-4 py-4">
                <InstanceEngineRadioGrid
                  engine={basicInfo.engine}
                  engineList={supportedEngineV1List()}
                  onEngineChange={handleSelectInstanceEngine}
                  isEngineBeta={isEngineBeta}
                />
              </div>
            )}
          </div>
        )}

        {/* Basic Info Card */}
        <div className="border border-block-border rounded-lg p-5">
          <h3 className="text-base font-medium text-main">
            {t("instance.section.basic-info")}
          </h3>

          <div className="mt-3 grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-4">
            {/* Instance Name */}
            <div className="sm:col-span-2 sm:col-start-1">
              <label
                htmlFor="name"
                className="textlabel flex flex-row items-center"
              >
                {t("instance.instance-name")}
                <span className="ml-0.5 text-error">*</span>
                {instance && (
                  <div className="ml-2 flex items-center">
                    {EngineIconPath[instance.engine] && (
                      <img
                        src={EngineIconPath[instance.engine]}
                        alt=""
                        className="w-4 h-4"
                      />
                    )}
                    <span className="ml-1">{instance.engineVersion}</span>
                  </div>
                )}
              </label>
              <Input
                value={basicInfo.title}
                required
                className="mt-1 w-full max-w-[40rem]"
                disabled={!allowEdit}
                maxLength={200}
                onChange={(e) => updateBasicInfo({ title: e.target.value })}
              />
            </div>

            {/* Activation toggle */}
            {subscriptionStore.currentPlan !== PlanType.FREE && allowEdit && (
              <div className="sm:col-span-2 ml-0 sm:ml-3">
                <label htmlFor="activation" className="textlabel block">
                  {t("subscription.instance-assignment.assign-license")} (
                  <a href="/setting/subscription" className="accent-link">
                    {t("subscription.instance-assignment.n-license-remain", {
                      n: availableLicenseCountText,
                    })}
                  </a>
                  )
                </label>
                <div className="h-8.5 flex flex-row items-center mt-1">
                  <label className="relative inline-flex items-center cursor-pointer">
                    <input
                      type="checkbox"
                      className="sr-only peer"
                      checked={basicInfo.activation}
                      disabled={
                        !basicInfo.activation && availableLicenseCount === 0
                      }
                      onChange={(e) =>
                        changeInstanceActivation(e.target.checked)
                      }
                    />
                    <div className="w-9 h-5 bg-gray-300 peer-focus:outline-none rounded-full peer peer-checked:bg-accent transition-colors after:content-[''] after:absolute after:top-0.5 after:left-0.5 after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:after:translate-x-4" />
                  </label>
                </div>
              </div>
            )}

            {/* Resource ID */}
            <div className="sm:col-span-3 sm:col-start-1 -mt-4">
              <ResourceIdField
                value={resourceId}
                onChange={setResourceId}
                onValidationChange={setResourceIdValidated}
                resourceTitle={basicInfo.title}
                readonly={!isCreating}
                fetchResource={(id) =>
                  instanceV1Store.getOrFetchInstanceByName(
                    `${instanceNamePrefix}${id}`,
                    true /* silent */
                  )
                }
              />
            </div>

            {/* Environment */}
            <div className="sm:col-span-2 sm:col-start-1">
              <label htmlFor="environment" className="textlabel">
                {t("common.environment")}
              </label>
              <EnvironmentSelect
                className="mt-1 w-full max-w-[40rem]"
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
            </div>

            {/* Labels */}
            <div className="sm:col-span-3 sm:col-start-1">
              <label htmlFor="labels" className="textlabel">
                {t("common.labels")}
              </label>
              <div className="mt-1">
                <LabelListEditor
                  kvList={labelKVList}
                  onChange={setLabelKVList}
                  readonly={!allowEdit}
                  showErrors
                  onErrorsChange={ctx.setLabelErrors}
                />
              </div>
            </div>

            {/* External link (edit mode only) */}
            {!isCreating && (
              <div className="sm:col-span-3 sm:col-start-1">
                <label
                  htmlFor="external-link"
                  className="textlabel inline-flex"
                >
                  <span>
                    {basicInfo.engine === Engine.SNOWFLAKE
                      ? t("instance.snowflake-web-console")
                      : t("instance.external-link")}
                  </span>
                  {(basicInfo.externalLink ?? "").trim().length > 0 && (
                    <button
                      className="ml-1 btn-icon"
                      onClick={(e) => {
                        e.preventDefault();
                        window.open(
                          urlfy(basicInfo.externalLink ?? ""),
                          "_blank"
                        );
                      }}
                    >
                      <ExternalLink className="w-4 h-4" />
                    </button>
                  )}
                </label>
                {basicInfo.engine === Engine.SNOWFLAKE ? (
                  <Input
                    required
                    className="mt-1 w-full"
                    disabled
                    value={instanceLink}
                  />
                ) : (
                  <>
                    <div className="mt-1 textinfolabel">
                      {t("instance.sentence.console.snowflake")}
                    </div>
                    <Input
                      value={basicInfo.externalLink ?? ""}
                      required
                      className="textfield mt-1 w-full"
                      disabled={!allowEdit}
                      placeholder={SnowflakeExtraLinkPlaceHolder}
                      onChange={(e) =>
                        updateBasicInfo({ externalLink: e.target.value })
                      }
                    />
                  </>
                )}
              </div>
            )}

            {/* Scan Interval (edit mode only) */}
            {!isCreating && instance && (
              <ScanIntervalInput
                scanInterval={basicInfo.syncInterval}
                allowEdit={allowEdit}
                onScanIntervalChange={changeScanInterval}
              />
            )}

            {/* Sync Databases (edit mode) */}
            {!isCreating && (
              <SyncDatabases
                isCreating={false}
                showLabel
                allowEdit={allowEdit}
                syncDatabases={basicInfo.syncDatabases}
                onSyncDatabasesChange={handleChangeSyncDatabases}
              />
            )}
          </div>
        </div>

        {/* Connection Card */}
        <div className="border border-block-border rounded-lg p-5">
          <h3 className="text-base font-medium text-main">
            {t("instance.section.connection")}
          </h3>

          <div className="mt-3 grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-4">
            {/* Host input */}
            <div className="sm:col-span-3 sm:col-start-1">
              {basicInfo.engine === Engine.SPANNER ? (
                <SpannerHostInput
                  host={adminDataSource.host}
                  onHostChange={(host) => updateAdminDS({ host })}
                  allowEdit={allowEdit}
                />
              ) : basicInfo.engine === Engine.BIGQUERY ? (
                <BigQueryHostInput
                  host={adminDataSource.host}
                  onHostChange={(host) => updateAdminDS({ host })}
                  allowEdit={allowEdit}
                />
              ) : (
                <>
                  <label htmlFor="host" className="textlabel block">
                    {basicInfo.engine === Engine.SNOWFLAKE ? (
                      <>
                        {t("instance.account-locator")}
                        <span className="mr-2 text-error"> *</span>
                        <a
                          href="https://docs.snowflake.com/en/user-guide/admin-account-identifier#using-an-account-locator-as-an-identifier"
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-sm normal-link"
                        >
                          {t("common.learn-more")}
                        </a>
                      </>
                    ) : basicInfo.engine === Engine.COSMOSDB ? (
                      <>
                        {t("instance.endpoint")}
                        <span className="text-error"> *</span>
                      </>
                    ) : adminDataSource.authenticationType ===
                      DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM ? (
                      <div>
                        <span>
                          {t(
                            "instance.sentence.google-cloud-sql.instance-name"
                          )}
                          <span className="text-error"> *</span>
                        </span>
                        <div className="textinfolabel mb-1">
                          {t(
                            "instance.sentence.google-cloud-sql.instance-name-tips"
                          ).replace(
                            "{instance}",
                            "{project-id}:{region}:{instance-name}"
                          )}
                        </div>
                      </div>
                    ) : (
                      <>
                        {t("instance.host-or-socket")}
                        {basicInfo.engine !== Engine.DYNAMODB && (
                          <span className="text-error"> *</span>
                        )}
                        {isCreating && onOpenInfoPanel && hasHostInfo && (
                          <button
                            type="button"
                            className="ml-1 text-accent hover:text-accent-hover text-sm"
                            onClick={() => openInfoPanel("host")}
                          >
                            ?
                          </button>
                        )}
                      </>
                    )}
                  </label>
                  <Input
                    value={adminDataSource.host}
                    required
                    placeholder={
                      basicInfo.engine === Engine.SNOWFLAKE
                        ? t("instance.your-snowflake-account-locator")
                        : t("instance.sentence.host.none-snowflake")
                    }
                    className="mt-1 w-full"
                    disabled={!allowEdit}
                    onChange={(e) => updateAdminDS({ host: e.target.value })}
                  />
                  {basicInfo.engine === Engine.SNOWFLAKE && (
                    <div className="mt-2 textinfolabel">
                      {t("instance.sentence.proxy.snowflake")}
                    </div>
                  )}
                </>
              )}
            </div>

            {/* Port input */}
            {basicInfo.engine !== Engine.SPANNER &&
              basicInfo.engine !== Engine.BIGQUERY &&
              basicInfo.engine !== Engine.DATABRICKS &&
              basicInfo.engine !== Engine.COSMOSDB &&
              adminDataSource.authenticationType !==
                DataSource_AuthenticationType.GOOGLE_CLOUD_SQL_IAM && (
                <div className="sm:col-span-1">
                  <label htmlFor="port" className="textlabel block">
                    {t("instance.port")}
                  </label>
                  <Input
                    value={adminDataSource.port}
                    className="mt-1 w-full"
                    placeholder={defaultPort}
                    disabled={!allowEdit || !allowEditPort}
                    onChange={handlePortChange}
                  />
                </div>
              )}

            {/* MongoDB connection string schema */}
            {basicInfo.engine === Engine.MONGODB && (
              <div className="sm:col-span-4 sm:col-start-1">
                <label
                  htmlFor="connectionStringSchema"
                  className="textlabel flex flex-row items-center"
                >
                  {t("data-source.connection-string-schema")}
                </label>
                <div className="flex items-center gap-x-4 mt-1">
                  {MongoDBConnectionStringSchemaList.map((type) => (
                    <label
                      key={type}
                      className="flex items-center gap-x-2 cursor-pointer"
                    >
                      <input
                        type="radio"
                        checked={currentMongoDBConnectionSchema === type}
                        onChange={() =>
                          handleMongodbConnectionStringSchemaChange(type)
                        }
                      />
                      {type}
                    </label>
                  ))}
                </div>
              </div>
            )}

            {/* Redis connection type */}
            {basicInfo.engine === Engine.REDIS && (
              <div className="sm:col-span-4 sm:col-start-1 flex flex-col gap-y-2">
                <label
                  htmlFor="connectionStringSchema"
                  className="textlabel flex flex-row items-center"
                >
                  {t("data-source.connection-type")}
                </label>
                <div className="flex items-center gap-x-4">
                  {RedisConnectionType.map((type) => (
                    <label
                      key={type}
                      className="flex items-center gap-x-2 cursor-pointer"
                    >
                      <input
                        type="radio"
                        checked={currentRedisConnectionType === type}
                        onChange={() => handleRedisConnectionTypeChange(type)}
                      />
                      {type}
                    </label>
                  ))}
                </div>
              </div>
            )}

            {/* Additional addresses */}
            {showAdditionalAddresses && (
              <div className="sm:col-span-4 sm:col-start-1">
                <label
                  htmlFor="additionalAddresses"
                  className="textlabel flex flex-row items-center"
                >
                  {t("data-source.additional-node-addresses")}
                </label>
                <div className="mt-1 grid grid-cols-1 gap-y-1 gap-x-4 sm:grid-cols-12">
                  {adminDataSource.additionalAddresses.map((addr, index) => (
                    <div key={index} className="contents">
                      <div className="sm:col-span-8 sm:col-start-1">
                        {index === 0 && (
                          <label
                            htmlFor="additionalAddressesHost"
                            className="textlabel font-normal! flex flex-row items-center"
                          >
                            {t("instance.host-or-socket")}
                          </label>
                        )}
                        <Input
                          value={addr.host}
                          required
                          className="mt-1 w-full"
                          disabled={!allowEdit}
                          onChange={(e) =>
                            handleAdditionalAddressHostChange(
                              index,
                              e.target.value
                            )
                          }
                        />
                      </div>
                      <div className="sm:col-span-3">
                        {index === 0 && (
                          <label
                            htmlFor="additionalAddressesPort"
                            className="textlabel font-normal! flex flex-row items-center"
                          >
                            {t("instance.port")}
                          </label>
                        )}
                        <Input
                          value={addr.port}
                          className="mt-1 w-full"
                          placeholder={defaultPort}
                          disabled={!allowEdit || !allowEditPort}
                          onChange={(e) =>
                            handleAdditionalAddressPortChange(
                              index,
                              e.target.value
                            )
                          }
                        />
                      </div>
                      <div className="h-8.5 flex flex-row items-center self-end">
                        <button
                          type="button"
                          className="p-1 text-control-light hover:text-error disabled:opacity-50"
                          disabled={!allowEdit}
                          onClick={() => removeDSAdditionalAddress(index)}
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </div>
                  ))}
                  <div className="mt-1 sm:col-span-12 sm:col-start-1">
                    <Button
                      variant="outline"
                      size="sm"
                      className="ml-auto w-12!"
                      onClick={(e) => {
                        e.preventDefault();
                        addDSAdditionalAddress();
                      }}
                    >
                      {t("common.add")}
                    </Button>
                  </div>
                </div>
              </div>
            )}

            {/* MongoDB replica set */}
            {basicInfo.engine === Engine.MONGODB && !adminDataSource.srv && (
              <div className="sm:col-span-2 sm:col-start-1">
                <label htmlFor="replicaSet" className="textlabel">
                  {t("data-source.replica-set")}
                </label>
                <Input
                  value={adminDataSource.replicaSet}
                  required
                  className="mt-1 w-full"
                  disabled={!allowEdit}
                  onChange={(e) =>
                    updateAdminDS({ replicaSet: e.target.value })
                  }
                />
              </div>
            )}

            {/* MongoDB direct connection */}
            {basicInfo.engine === Engine.MONGODB &&
              !adminDataSource.srv &&
              adminDataSource.additionalAddresses.length === 0 && (
                <div className="sm:col-span-4 sm:col-start-1">
                  <label className="flex items-center gap-x-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={adminDataSource.directConnection}
                      disabled={!allowEdit}
                      onChange={(e) =>
                        updateAdminDS({
                          directConnection: e.target.checked,
                        })
                      }
                    />
                    {t("data-source.direct-connection")}
                  </label>
                </div>
              )}
          </div>

          {/* Credentials (auth method, username, password) */}
          {basicInfo.engine !== Engine.DYNAMODB && (
            <>
              <DataSourceSection
                hideOptions
                onOpenInfoPanel={onOpenInfoPanel}
              />

              {actuatorStore.isSaaSMode && (
                <div className="mt-4 rounded-sm border-none bg-blue-50 p-3">
                  <a
                    href="https://docs.bytebase.com/get-started/cloud#prerequisites"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="normal-link text-sm"
                  >
                    {t("instance.sentence.firewall-info")}
                  </a>
                </div>
              )}
            </>
          )}
        </div>

        {/* Connection Options Card */}
        {basicInfo.engine !== Engine.DYNAMODB && editingDataSource && (
          <div className="border border-block-border rounded-lg bg-white">
            <button
              type="button"
              className="w-full flex items-center justify-between gap-x-3 px-5 py-4 text-left transition-colors hover:bg-gray-50"
              onClick={() => setIsConnectionOptionsCollapsed((prev) => !prev)}
            >
              <h3 className="text-base font-medium text-main">
                {t("instance.connection-options")}
              </h3>
              <div className="shrink-0 text-control-light">
                {!isConnectionOptionsCollapsed ? (
                  <ChevronDown className="w-4 h-4" />
                ) : (
                  <ChevronRight className="w-4 h-4" />
                )}
              </div>
            </button>
            {!isConnectionOptionsCollapsed && (
              <div className="border-t border-block-border px-5 py-4">
                <DataSourceForm
                  dataSource={editingDataSource}
                  optionsOnly
                  onDataSourceChange={handleDataSourceChange}
                  onOpenInfoPanel={onOpenInfoPanel}
                />
              </div>
            )}
          </div>
        )}

        {/* Test Connection button (create only) */}
        {isCreating && !!editingDataSource && (
          <div className="flex justify-start">
            <Button
              variant="outline"
              disabled={!allowTestConnection || state.isTestingConnection}
              onClick={(e) => {
                e.preventDefault();
                testConnectionForCurrentEditingDS();
              }}
            >
              {state.isTestingConnection
                ? `${t("instance.test-connection")}...`
                : t("instance.test-connection")}
            </Button>
          </div>
        )}

        {/* Sync Databases Card (create only) */}
        {basicInfo.engine !== Engine.DYNAMODB && isCreating && (
          <div className="border border-block-border rounded-lg p-5 flex flex-col gap-y-1">
            <p className="w-full text-lg leading-6 font-medium text-gray-900">
              {t("instance.sync-databases.self")}
            </p>
            <SyncDatabases
              isCreating
              showLabel={false}
              allowEdit={allowEdit && !!allowCreate}
              syncDatabases={basicInfo.syncDatabases}
              onSyncDatabasesChange={handleChangeSyncDatabases}
            />
          </div>
        )}
      </div>
    </div>
  );
}
