import { create } from "@bufbuild/protobuf";
import { DurationSchema, TimestampSchema } from "@bufbuild/protobuf/wkt";
import { Loader2 } from "lucide-react";
import { useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { accessGrantServiceClientConnect } from "@/connect";
import { DatabaseSelect } from "@/react/components/DatabaseSelect";
import { MonacoEditor } from "@/react/components/monaco/MonacoEditor";
import {
  monacoThemeName,
  themeColorScheme,
  themeToCssVars,
} from "@/react/components/sql-editor/theme/derive";
import { SQLEditorThemeScope } from "@/react/components/sql-editor/theme/SQLEditorThemeScope";
import { useActiveSQLEditorTheme } from "@/react/components/sql-editor/theme/useActiveSQLEditorTheme";
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
import { useCurrentUser } from "@/react/hooks/useAppState";
import { router } from "@/react/router";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/react/router/handles";
import { useAppStore } from "@/react/stores/app";
import { useSQLEditorStore } from "@/react/stores/sqlEditor";
import { useSQLEditorEditorState } from "@/react/stores/sqlEditor/editor";
import { getSQLEditorTabsState } from "@/react/stores/sqlEditor/tab";
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

interface Props {
  readonly targets?: string[];
  readonly query?: string;
  readonly unmask?: boolean;
  readonly export?: boolean;
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
  const currentUserEmail = useCurrentUser().email;
  const setAsidePanelTab = useSQLEditorStore((s) => s.setAsidePanelTab);
  const setHighlightAccessGrantName = useSQLEditorStore(
    (s) => s.setHighlightAccessGrantName
  );

  const project = useSQLEditorEditorState((s) => s.project);

  // Re-theme the drawer chrome since the Sheet portals outside the SQL Editor
  // chrome DOM subtree, so the chrome CSS vars don't cascade here. Use the
  // ACTIVE theme (the dark admin fallback in admin mode), not the selected
  // root theme — otherwise opening the drawer from an admin tab renders light
  // chrome over the dark terminal. This also keeps the chrome consistent with
  // the embedded Monaco below, which already uses the active theme.
  const active = useActiveSQLEditorTheme();

  // The embedded Monaco MUST carry the active SQL Editor theme. Monaco's
  // setTheme is global: a <MonacoEditor> mounting with no theme resets the
  // shared Monaco theme to bb-light, flipping the whole editor to light the
  // moment the drawer opens. Passing the active theme keeps it consistent.
  const monacoOptions = useMemo(
    () => ({ theme: monacoThemeName(active) }),
    [active]
  );

  const defaultTargets = useMemo(() => {
    if (stableProps.targets && stableProps.targets.length > 0) {
      return stableProps.targets;
    }
    const tabsState = getSQLEditorTabsState();
    const database = tabsState.tabsById.get(tabsState.currentTabId)?.connection
      ?.database;
    return database ? [database] : [];
  }, [stableProps.targets]);

  const [targets, setTargets] = useState<string[]>(defaultTargets);
  const [query, setQuery] = useState(stableProps.query ?? "");
  const [unmask, setUnmask] = useState(stableProps.unmask ?? false);
  const [exportResult, setExportResult] = useState(stableProps.export ?? false);
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
        export: exportResult,
        reason,
        expiration,
      });

      const response = await accessGrantServiceClientConnect.createAccessGrant(
        create(CreateAccessGrantRequestSchema, {
          parent: project as string,
          accessGrant,
        })
      );

      useAppStore.getState().notify({
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
    <SQLEditorThemeScope theme={active} asContents>
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
            <DatabaseSelect
              multiple
              value={targets}
              onChange={setTargets}
              projectName={project as string}
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
              // Drawer Monaco portals outside `.sqleditor--wrapper`, so opt the
              // canvas into the transparent-background rule and back it with the
              // themed `bg-background` (from `sheetStyle`'s `--color-background`)
              // so it matches the active theme like the worksheet editor.
              className="border rounded-[3px] h-40 bg-background sqleditor--monaco-transparent"
              content={query}
              language="sql"
              autoCompleteContext={autoCompleteContext}
              onChange={setQuery}
              options={monacoOptions}
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

          {/* Export */}
          <div className="flex flex-col gap-y-2">
            <div className="text-sm font-medium text-control">
              {t("sql-editor.grant-type-export")}
            </div>
            <label className="flex items-center gap-2 cursor-pointer">
              <Checkbox
                checked={exportResult}
                onCheckedChange={(checked) => setExportResult(checked)}
              />
              <span>{t("sql-editor.access-type-export")}</span>
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
    </SQLEditorThemeScope>
  );
}

export function AccessGrantRequestDrawer({
  targets,
  query,
  unmask,
  export: exportResult,
  onClose,
}: Props) {
  const propsRef = useRef({
    targets,
    query,
    unmask,
    export: exportResult,
    onClose,
  });
  // Freeze props while drawer is open so inner form stays stable during close animation
  const stableProps = propsRef.current;

  // The Sheet portals to the app-global overlay root, so the SQL Editor scope's
  // CSS vars don't cascade to it. Apply them directly on SheetContent so the
  // panel background AND its form contents follow the active theme. ACTIVE (not
  // the selected root theme) so an admin tab's dark fallback themes the drawer;
  // these inline vars would otherwise override the dark vars that
  // useSQLEditorOverlayTheme writes to the overlay root.
  const active = useActiveSQLEditorTheme();
  const sheetStyle = useMemo(
    () => ({
      ...themeToCssVars(active.tokens),
      // Native controls (date pickers, scrollbars) follow color-scheme.
      colorScheme: themeColorScheme(active),
    }),
    [active]
  );

  return (
    <Sheet open={true} onOpenChange={(next) => !next && onClose()}>
      {/* text-main gives the drawer a themed default text color (it portals
          outside the SQL Editor wrapper that sets one), so un-classed text like
          checkbox labels and selected values follow the theme. */}
      <SheetContent width="standard" style={sheetStyle} className="text-main">
        <AccessGrantRequestDrawerInner
          key={`${targets?.join(",")}-${query}-${unmask}-${exportResult}`}
          stableProps={stableProps}
          onClose={onClose}
        />
      </SheetContent>
    </Sheet>
  );
}
