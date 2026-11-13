import { create } from "@bufbuild/protobuf";
import { DurationSchema, TimestampSchema } from "@bufbuild/protobuf/wkt";
import dayjs from "dayjs";
import { Loader2 } from "lucide-react";
import {
  lazy,
  Suspense,
  type SyntheticEvent,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { accessGrantServiceClientConnect } from "@/api";
import { router } from "@/app/router";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/app/router/handles";
import { DatabaseSelect } from "@/components/DatabaseSelect";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Combobox } from "@/components/ui/combobox";
import { ExpirationPicker } from "@/components/ui/expiration-picker";
import { FormError, FormField } from "@/components/ui/form";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Textarea } from "@/components/ui/textarea";
import {
  monacoThemeName,
  themeColorScheme,
  themeToCssVars,
} from "@/modules/sql-editor/components/theme/derive";
import { SQLEditorThemeScope } from "@/modules/sql-editor/components/theme/SQLEditorThemeScope";
import type { SQLEditorTheme } from "@/modules/sql-editor/components/theme/types";
import { useAppStore } from "@/stores/app";
import type { AccessGrant } from "@/types/proto-es/v1/access_grant_service_pb";
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

const MonacoEditor = lazy(() =>
  import("@/components/monaco/MonacoEditor").then((module) => ({
    default: module.MonacoEditor,
  }))
);

interface Props {
  readonly project: string;
  readonly reason?: string;
  readonly theme?: SQLEditorTheme;
  readonly onCreated?: (grant: AccessGrant) => void;
  readonly targets?: string[];
  readonly query?: string;
  readonly unmask?: boolean;
  readonly export?: boolean;
  readonly schema?: string;
  readonly container?: string;
  readonly onClose: () => void;
}

interface AccessGrantRequestDrawerInnerProps {
  readonly stableProps: Props;
  readonly theme?: SQLEditorTheme;
  readonly onClose: () => void;
}

function AccessGrantRequestDrawerInner({
  stableProps,
  theme: active,
  onClose,
}: AccessGrantRequestDrawerInnerProps) {
  const { t } = useTranslation();
  const project = stableProps.project;
  const monacoOptions = useMemo(
    () => (active ? { theme: monacoThemeName(active) } : undefined),
    [active]
  );

  const [targets, setTargets] = useState<string[]>(stableProps.targets ?? []);
  const [query, setQuery] = useState(stableProps.query ?? "");
  const [unmask, setUnmask] = useState(stableProps.unmask ?? false);
  const [exportResult, setExportResult] = useState(stableProps.export ?? false);
  const [schema] = useState(stableProps.schema ?? "");
  const [container] = useState(stableProps.container ?? "");
  const [duration, setDuration] = useState<number>(4);
  const [customExpireTime, setCustomExpireTime] = useState<string | undefined>(
    undefined
  );
  const [reason, setReason] = useState(stableProps.reason ?? "");
  const [isRequesting, setIsRequesting] = useState(false);

  const autoCompleteContext = useMemo(() => {
    const db = targets[0];
    if (!db) return undefined;
    const { instance } = extractDatabaseResourceName(db);
    return { instance, database: db };
  }, [targets]);

  // Keep the workspace expiration cap in seconds to honor sub-day limits.
  const workspaceProfile = useAppStore((state) => state.getWorkspaceProfile());
  const maximumExpirationSeconds = useMemo(() => {
    const seconds = workspaceProfile.maximumRequestExpiration?.seconds;
    if (!seconds) return undefined;
    return Number(seconds);
  }, [workspaceProfile]);
  const expirationCapped = maximumExpirationSeconds !== undefined;
  // Whole-day cap value for the day-worded hint/error copy; undefined for a
  // sub-day cap, where the bounded picker enforces the limit on its own.
  const maximumExpirationDays =
    maximumExpirationSeconds !== undefined &&
    maximumExpirationSeconds % (60 * 60 * 24) === 0
      ? maximumExpirationSeconds / (60 * 60 * 24)
      : undefined;

  const durationOptions = useMemo(() => {
    const presets = [
      {
        hours: 1,
        label: t("sql-editor.duration-hour", { hours: 1 }),
      },
      {
        hours: 4,
        label: t("sql-editor.duration-hours", { hours: 4 }),
      },
      {
        hours: 24,
        label: t("sql-editor.duration-day", { days: 1 }),
      },
      {
        hours: 168,
        label: t("sql-editor.duration-days", { days: 7 }),
      },
    ];
    return [
      ...presets
        .filter(
          (o) =>
            maximumExpirationSeconds === undefined ||
            o.hours * 60 * 60 <= maximumExpirationSeconds
        )
        .map(({ hours, label }) => ({ value: `${hours}`, label })),
      { value: "-1", label: t("common.custom") },
    ];
  }, [t, maximumExpirationSeconds]);

  const today = new Date();
  const minDate = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, "0")}-${String(today.getDate()).padStart(2, "0")}T00:00`;

  // A previously selected preset (e.g. the default 4h) may exceed a newly
  // applied cap; fall back to the largest still-valid preset, or to the custom
  // picker when the cap is shorter than every preset.
  useEffect(() => {
    if (duration === -1) return;
    if (durationOptions.some((o) => o.value === String(duration))) return;
    const presets = durationOptions.filter((o) => o.value !== "-1");
    setDuration(
      presets.length > 0 ? Number(presets[presets.length - 1].value) : -1
    );
  }, [durationOptions, duration]);

  // Bind the picker's min to the current minute and its max to the configured
  // cap, matching the request-role drawer.
  const minDatetime = dayjs().format("YYYY-MM-DDTHH:mm");
  const maxDatetime =
    maximumExpirationSeconds === undefined
      ? undefined
      : dayjs()
          .add(maximumExpirationSeconds, "second")
          .format("YYYY-MM-DDTHH:mm");

  const expirationIsInPast =
    duration === -1 &&
    !!customExpireTime &&
    dayjs(customExpireTime).unix() <= dayjs().unix();
  const expirationExceedsMax =
    duration === -1 &&
    !!customExpireTime &&
    maximumExpirationSeconds !== undefined &&
    dayjs(customExpireTime).isAfter(
      dayjs().add(maximumExpirationSeconds, "second")
    );

  const allowSubmit = useMemo(() => {
    if (targets.length === 0) return false;
    if (!query.trim()) return false;
    if (!reason.trim()) return false;
    if (duration === -1) {
      return !!customExpireTime && !expirationIsInPast && !expirationExceedsMax;
    }
    return true;
  }, [
    targets,
    query,
    reason,
    duration,
    customExpireTime,
    expirationIsInPast,
    expirationExceedsMax,
  ]);

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
        targets,
        query,
        unmask,
        export: exportResult,
        reason,
        expiration,
        schema,
        container,
      });

      const response = await accessGrantServiceClientConnect.createAccessGrant(
        create(CreateAccessGrantRequestSchema, {
          parent: project,
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
      }
      stableProps.onCreated?.(response);
      onClose();
    } finally {
      setIsRequesting(false);
    }
  };

  return (
    <>
      <SheetHeader>
        <SheetTitle>{t("sql-editor.request-access-grant")}</SheetTitle>
      </SheetHeader>
      <SheetBody>
        <div className="flex flex-col gap-y-6">
          {/* Databases */}
          <FormField
            title={
              <>
                {t("common.databases")}
                <span className="text-error ml-0.5">*</span>
              </>
            }
          >
            <DatabaseSelect
              multiple
              value={targets}
              onChange={setTargets}
              projectName={project as string}
            />
          </FormField>

          {/* Statement */}
          <FormField
            title={
              <>
                {t("common.statement")}
                <span className="text-error ml-0.5">*</span>
              </>
            }
          >
            <Alert
              variant="info"
              description={t("sql-editor.only-select-allowed")}
            />
            <Suspense
              fallback={
                <div className="border rounded-xs h-40 bg-background flex items-center justify-center text-sm text-control-light">
                  {t("common.loading")}
                </div>
              }
            >
              <MonacoEditor
                // Scroll long statements internally instead of covering fields below.
                autoHeight={false}
                // The portal needs its own themed background outside SQL Editor.
                className="border rounded-xs h-40 bg-background sqleditor--monaco-transparent"
                content={query}
                language="sql"
                autoCompleteContext={autoCompleteContext}
                onChange={setQuery}
                options={monacoOptions}
              />
            </Suspense>
          </FormField>

          {/* Unmask */}
          <FormField title={<>{t("sql-editor.grant-type-unmask")}</>}>
            <label className="flex items-center gap-2 cursor-pointer">
              <Checkbox
                checked={unmask}
                onCheckedChange={(checked) => setUnmask(checked)}
              />
              <span>{t("sql-editor.access-type-unmask")}</span>
            </label>
          </FormField>

          {/* Export */}
          <FormField title={<>{t("sql-editor.grant-type-export")}</>}>
            <label className="flex items-center gap-2 cursor-pointer">
              <Checkbox
                checked={exportResult}
                onCheckedChange={(checked) => setExportResult(checked)}
              />
              <span>{t("sql-editor.access-type-export")}</span>
            </label>
          </FormField>

          {/* Expiration */}
          <FormField
            title={
              <>
                {t("common.expiration")}
                <span className="text-error ml-0.5">*</span>
              </>
            }
            description={
              maximumExpirationDays !== undefined
                ? t("common.expiration-max-hint", {
                    days: maximumExpirationDays,
                  })
                : undefined
            }
          >
            <Combobox
              value={String(duration)}
              onChange={(val) => setDuration(Number(val))}
              options={durationOptions}
            />
            {duration === -1 && (
              <ExpirationPicker
                className="w-full"
                value={customExpireTime}
                onChange={setCustomExpireTime}
                minDate={expirationCapped ? minDatetime : minDate}
                maxDate={maxDatetime}
              />
            )}
            {expirationIsInPast && (
              <FormError>{t("common.expiration-must-be-future")}</FormError>
            )}
            {expirationExceedsMax && maximumExpirationDays !== undefined && (
              <FormError>
                {t("common.expiration-exceeds-max", {
                  days: maximumExpirationDays,
                })}
              </FormError>
            )}
          </FormField>

          {/* Reason */}
          <FormField
            title={
              <>
                {t("common.reason")}
                <span className="text-error ml-0.5">*</span>
              </>
            }
          >
            <Textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              rows={3}
              placeholder=""
            />
          </FormField>
        </div>
      </SheetBody>
      <SheetFooter>
        <Button appearance="outline" onClick={onClose} disabled={isRequesting}>
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
  project,
  reason,
  theme,
  onCreated,
  targets,
  query,
  unmask,
  export: exportResult,
  schema,
  container,
  onClose,
}: Props) {
  const propsRef = useRef({
    project,
    reason,
    theme,
    onCreated,
    targets,
    query,
    unmask,
    export: exportResult,
    schema,
    container,
    onClose,
  });
  // Freeze props while drawer is open so inner form stays stable during close animation
  const stableProps = propsRef.current;

  const sheetStyle = useMemo(
    () =>
      theme
        ? {
            ...themeToCssVars(theme.tokens),
            colorScheme: themeColorScheme(theme),
          }
        : undefined,
    [theme]
  );
  const stopDrawerEvent = (event: SyntheticEvent) => {
    event.stopPropagation();
  };
  const content = (
    <AccessGrantRequestDrawerInner
      stableProps={stableProps}
      theme={theme}
      onClose={onClose}
    />
  );

  return (
    <Sheet open={true} onOpenChange={(next) => !next && onClose()}>
      {/* text-main gives the drawer a themed default text color (it portals
          outside the SQL Editor wrapper that sets one), so un-classed text like
          checkbox labels and selected values follow the theme. */}
      <SheetContent
        width="standard"
        style={sheetStyle}
        className="text-main"
        onClick={stopDrawerEvent}
        onPointerDown={stopDrawerEvent}
      >
        {theme ? (
          <SQLEditorThemeScope theme={theme} asContents>
            {content}
          </SQLEditorThemeScope>
        ) : (
          content
        )}
      </SheetContent>
    </Sheet>
  );
}
