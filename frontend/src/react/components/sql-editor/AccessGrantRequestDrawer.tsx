import { create } from "@bufbuild/protobuf";
import { DurationSchema, TimestampSchema } from "@bufbuild/protobuf/wkt";
import { Loader2 } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { accessGrantServiceClientConnect } from "@/connect";
import { MonacoEditor } from "@/react/components/monaco/MonacoEditor";
import { Alert } from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import { Combobox } from "@/react/components/ui/combobox";
import { ExpirationPicker } from "@/react/components/ui/expiration-picker";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { Textarea } from "@/react/components/ui/textarea";
import { useVueState } from "@/react/hooks/useVueState";
import { useSQLEditorStore } from "@/react/stores/sqlEditor";
import { router } from "@/router";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import {
  pushNotification,
  useCurrentUserV1,
  useDatabaseV1Store,
  useSQLEditorStore as useSQLEditorPiniaStore,
  useSQLEditorTabStore,
} from "@/store";
import {
  AccessGrant_Status,
  AccessGrantSchema,
  CreateAccessGrantRequestSchema,
} from "@/types/proto-es/v1/access_grant_service_pb";
import {
  extractDatabaseResourceName,
  extractIssueUID,
  extractProjectResourceName,
} from "@/utils";

// Multi-select database combobox
function MultiDatabaseSelect({
  value,
  projectName,
  onChange,
}: {
  value: string[];
  projectName: string;
  onChange: (val: string[]) => void;
}) {
  const { t } = useTranslation();
  const [databases, setDatabases] = useState<
    { name: string; displayName: string }[]
  >([]);

  const databaseStore = useDatabaseV1Store();

  useEffect(() => {
    databaseStore
      .fetchDatabases({
        parent: projectName,
        filter: { query: "" },
        pageSize: 100,
        silent: true,
      })
      .then((result) => {
        setDatabases(
          result.databases.map((db) => {
            const { databaseName } = extractDatabaseResourceName(db.name);
            return { name: db.name, displayName: databaseName };
          })
        );
      })
      .catch(() => {
        /* ignore */
      });
  }, [databaseStore, projectName]);

  return (
    <Combobox
      multiple={true}
      value={value}
      onChange={onChange}
      placeholder={t("database.select")}
      noResultsText={t("common.no-data")}
      options={databases.map((db) => ({
        value: db.name,
        label: db.displayName,
      }))}
    />
  );
}

interface Props {
  readonly targets?: string[];
  readonly query?: string;
  readonly unmask?: boolean;
  readonly onClose: () => void;
}

interface AccessGrantRequestDrawerInnerProps extends Props {
  readonly stableProps: Props;
}

function AccessGrantRequestDrawerInner({
  stableProps,
  onClose,
}: AccessGrantRequestDrawerInnerProps) {
  const { t } = useTranslation();
  const currentUser = useCurrentUserV1();
  const editorStore = useSQLEditorPiniaStore();
  const tabStore = useSQLEditorTabStore();
  const setAsidePanelTab = useSQLEditorStore((s) => s.setAsidePanelTab);
  const setHighlightAccessGrantName = useSQLEditorStore(
    (s) => s.setHighlightAccessGrantName
  );

  const currentUserEmail = useVueState(() => currentUser.value.email);
  const project = useVueState(() => editorStore.project);

  const defaultTargets = useMemo(() => {
    if (stableProps.targets && stableProps.targets.length > 0) {
      return stableProps.targets;
    }
    const database = tabStore.currentTab?.connection?.database;
    return database ? [database] : [];
  }, [stableProps.targets, tabStore]);

  const [targets, setTargets] = useState<string[]>(defaultTargets);
  const [query, setQuery] = useState(stableProps.query ?? "");
  const [unmask, setUnmask] = useState(stableProps.unmask ?? false);
  const [duration, setDuration] = useState<number>(4);
  const [customExpireTime, setCustomExpireTime] = useState<string | undefined>(
    undefined
  );
  const [reason, setReason] = useState("");
  const [isRequesting, setIsRequesting] = useState(false);

  const autoCompleteContext = useMemo(() => {
    const db = targets[0];
    if (!db) return undefined;
    const { instance } = extractDatabaseResourceName(db);
    return { instance, database: db };
  }, [targets]);

  const durationOptions = useMemo(
    () => [
      { value: "1", label: t("sql-editor.duration-hour", { hours: 1 }) },
      { value: "4", label: t("sql-editor.duration-hours", { hours: 4 }) },
      { value: "24", label: t("sql-editor.duration-day", { days: 1 }) },
      { value: "168", label: t("sql-editor.duration-days", { days: 7 }) },
      { value: "-1", label: t("common.custom") },
    ],
    [t]
  );

  const today = new Date();
  const minDate = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, "0")}-${String(today.getDate()).padStart(2, "0")}T00:00`;

  const allowSubmit = useMemo(() => {
    if (targets.length === 0) return false;
    if (!query.trim()) return false;
    if (!reason.trim()) return false;
    if (duration === -1 && !customExpireTime) return false;
    return true;
  }, [targets, query, reason, duration, customExpireTime]);

  const handleSubmit = async () => {
    if (isRequesting || !allowSubmit) return;
    setIsRequesting(true);
    try {
      const expiration =
        duration === -1 && customExpireTime
          ? (() => {
              const ms = new Date(customExpireTime).getTime();
              return {
                case: "expireTime" as const,
                value: create(TimestampSchema, {
                  seconds: BigInt(Math.floor(ms / 1000)),
                  nanos: (ms % 1000) * 1000000,
                }),
              };
            })()
          : {
              case: "ttl" as const,
              value: create(DurationSchema, {
                seconds: BigInt(duration * 3600),
              }),
            };

      const accessGrant = create(AccessGrantSchema, {
        creator: `users/${currentUserEmail}`,
        targets,
        query,
        unmask,
        reason,
        expiration,
      });

      const response = await accessGrantServiceClientConnect.createAccessGrant(
        create(CreateAccessGrantRequestSchema, {
          parent: project as string,
          accessGrant,
        })
      );

      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.created"),
      });

      if (response.status === AccessGrant_Status.PENDING && response.issue) {
        const route = router.resolve({
          name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
          params: {
            projectId: extractProjectResourceName(response.issue),
            issueId: extractIssueUID(response.issue),
          },
        });
        window.open(route.fullPath, "_blank");
      } else {
        setAsidePanelTab("ACCESS");
        setHighlightAccessGrantName(response.name);
      }
      onClose();
    } finally {
      setIsRequesting(false);
    }
  };

  return (
    <>
      <SheetHeader>
        <SheetTitle>{t("sql-editor.request-data-access")}</SheetTitle>
      </SheetHeader>
      <SheetBody>
        <div className="flex flex-col gap-y-6">
          {/* Databases */}
          <div className="flex flex-col gap-y-2">
            <div className="text-sm font-medium text-control">
              {t("common.databases")}
              <span className="text-error ml-0.5">*</span>
            </div>
            <MultiDatabaseSelect
              value={targets}
              projectName={project as string}
              onChange={setTargets}
            />
          </div>

          {/* Statement */}
          <div className="flex flex-col gap-y-2">
            <div className="text-sm font-medium text-control">
              {t("common.statement")}
              <span className="text-error ml-0.5">*</span>
            </div>
            <Alert
              variant="info"
              description={t("sql-editor.only-select-allowed")}
            />
            <MonacoEditor
              className="border rounded-[3px] h-40"
              content={query}
              language="sql"
              autoCompleteContext={autoCompleteContext}
              onChange={setQuery}
            />
          </div>

          {/* Unmask */}
          <div className="flex flex-col gap-y-2">
            <div className="text-sm font-medium text-control">
              {t("sql-editor.grant-type-unmask")}
            </div>
            <label className="flex items-center gap-2 cursor-pointer">
              <Checkbox
                checked={unmask}
                onCheckedChange={(checked) => setUnmask(checked)}
              />
              <span>{t("sql-editor.access-type-unmask")}</span>
            </label>
          </div>

          {/* Expiration */}
          <div className="flex flex-col gap-y-2">
            <div className="text-sm font-medium text-control">
              {t("common.expiration")}
              <span className="text-error ml-0.5">*</span>
            </div>
            <Combobox
              value={String(duration)}
              onChange={(val) => setDuration(Number(val))}
              options={durationOptions}
            />
            {duration === -1 && (
              <ExpirationPicker
                value={customExpireTime}
                onChange={setCustomExpireTime}
                minDate={minDate}
              />
            )}
          </div>

          {/* Reason */}
          <div className="flex flex-col gap-y-2">
            <div className="text-sm font-medium text-control">
              {t("common.reason")}
              <span className="text-error ml-0.5">*</span>
            </div>
            <Textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              rows={3}
              placeholder=""
            />
          </div>
        </div>
      </SheetBody>
      <SheetFooter>
        <Button variant="outline" onClick={onClose} disabled={isRequesting}>
          {t("common.cancel")}
        </Button>
        <Button
          variant="default"
          disabled={!allowSubmit || isRequesting}
          onClick={() => void handleSubmit()}
          data-submit-btn
        >
          {isRequesting && <Loader2 className="size-4 mr-1 animate-spin" />}
          {t("common.submit")}
        </Button>
      </SheetFooter>
    </>
  );
}

export function AccessGrantRequestDrawer({
  targets,
  query,
  unmask,
  onClose,
}: Props) {
  const propsRef = useRef({ targets, query, unmask, onClose });
  // Freeze props while drawer is open so inner form stays stable during close animation
  const stableProps = propsRef.current;

  return (
    <Sheet open={true} onOpenChange={(next) => !next && onClose()}>
      <SheetContent width="standard">
        <AccessGrantRequestDrawerInner
          key={`${targets?.join(",")}-${query}-${unmask}`}
          stableProps={stableProps}
          onClose={onClose}
        />
      </SheetContent>
    </Sheet>
  );
}
