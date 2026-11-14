import type { TFunction } from "i18next";
import { CircleCheck, SlidersHorizontal, TriangleAlert } from "lucide-react";
import {
  type ReactNode,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import { EnvironmentSelect } from "@/components/EnvironmentSelect";
import { SecretInput } from "@/components/SecretInput";
import { SelectionActionBar } from "@/components/SelectionActionBar";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/components/ui/dialog";
import { FormField, FormLabel } from "@/components/ui/form";
import { Progress } from "@/components/ui/progress";
import { SegmentedControl } from "@/components/ui/segmented-control";
import { Switch } from "@/components/ui/switch";
import {
  Table,
  TableBody,
  TableCell,
  TableEmptyView,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tooltip } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { type NeonOption, NeonOptionSelect } from "./NeonOptionSelect";
import {
  intervalOptions,
  rowEnvironmentFromValue,
  rowEnvironmentValue,
  rowIntervalFromValue,
  SHARED,
  useEnvironmentTitle,
  useRowEnvironmentOptions,
} from "./NeonRowSheet";
import {
  CHECKBOX_HIT_AREA,
  CHECKBOX_HIT_AREA_DENSE,
  groupByNeonProject,
} from "./NeonSelectStep";
import {
  branchKey,
  countOverrides,
  draftNeed,
  duplicateInstanceIds,
  hasOverride,
  type NeonDraftNeed,
  type NeonImportDraft,
  type NeonScheduleMode,
  type NeonSharedKey,
  type NeonSharedSettings,
  resolveDraft,
  syncScheduleText,
  syncScopeText,
  withRole,
} from "./neon-import";

type Tab = "all" | "needs" | "changed";
type BulkAction = "environment" | "interval" | "remove";
type DraftUpdate = (draft: NeonImportDraft) => NeonImportDraft;

const checkedState = (selected: number, total: number) =>
  selected === 0 ? false : selected === total ? true : "indeterminate";

const COLUMN_COUNT = 6;

export function NeonConfigureStep({
  drafts,
  shared,
  capacity,
  schedule,
  actionBarContainer,
  onSharedChange,
  onDraftsChange,
  onResetOverride,
  onRemove,
  onOpenRow,
}: {
  // The selected rows, in discovery order.
  drafts: readonly NeonImportDraft[];
  shared: NeonSharedSettings;
  capacity?: number;
  schedule: NeonScheduleMode;
  // Where the selection action bar is mounted so it clears the page footer.
  actionBarContainer?: HTMLElement | null;
  onSharedChange: (change: Partial<NeonSharedSettings>) => void;
  onDraftsChange: (keys: readonly string[], update: DraftUpdate) => void;
  onResetOverride: (setting: NeonSharedKey) => void;
  onRemove: (keys: readonly string[]) => void;
  onOpenRow: (key: string) => void;
}) {
  const { t } = useTranslation();
  const [tab, setTab] = useState<Tab>("all");
  const [checked, setChecked] = useState<ReadonlySet<string>>(new Set());
  const [bulk, setBulk] = useState<BulkAction>();
  const environmentTitle = useEnvironmentTitle();
  const rowEnvironmentOptions = useRowEnvironmentOptions(shared);

  const duplicateIds = useMemo(() => duplicateInstanceIds(drafts), [drafts]);
  const needs = useMemo(
    () =>
      new Map(
        drafts.flatMap((draft): [string, NeonDraftNeed][] => {
          const need = draftNeed(draft, duplicateIds);
          return need ? [[branchKey(draft.branch), need]] : [];
        })
      ),
    [drafts, duplicateIds]
  );
  const overrides = countOverrides(drafts);
  const changedCount = drafts.filter(hasOverride).length;
  const shown = drafts.filter((draft) =>
    tab === "needs"
      ? needs.has(branchKey(draft.branch))
      : tab === "changed"
        ? hasOverride(draft)
        : true
  );
  const shownKeys = shown.map((draft) => branchKey(draft.branch));
  // Bulk actions reach only the checked rows the current tab shows.
  const selectedKeys = shownKeys.filter((key) => checked.has(key));
  const shownChecked = selectedKeys.length;
  const over =
    capacity === undefined ? 0 : Math.max(0, drafts.length - capacity);
  const sharedIntervalId = useId();

  const check = (keys: readonly string[], on: boolean) =>
    setChecked((previous) => {
      const next = new Set(previous);
      for (const key of keys) {
        if (on) next.add(key);
        else next.delete(key);
      }
      return next;
    });

  const differHint = (setting: NeonSharedKey, fallback: ReactNode) =>
    overrides[setting] > 0 ? (
      <span className="inline-flex flex-wrap items-center gap-x-1">
        {t("instance.neon.configure.differ", { count: overrides[setting] })}
        <Button
          appearance="link"
          size="xs"
          className="h-auto p-0"
          onClick={() => onResetOverride(setting)}
        >
          {t("common.reset")}
        </Button>
      </span>
    ) : (
      fallback
    );

  const actionBar = (
    <SelectionActionBar
      placement="container"
      count={selectedKeys.length}
      label={t("instance.neon.count.rows-selected", {
        count: selectedKeys.length,
      })}
      allSelected={shownKeys.length > 0 && shownChecked === shownKeys.length}
      onToggleSelectAll={() =>
        check(shownKeys, shownChecked !== shownKeys.length)
      }
      actions={[
        {
          key: "environment",
          label: t("instance.neon.configure.bulk.set-environment"),
          onClick: () => setBulk("environment"),
        },
        {
          key: "interval",
          label: t("instance.neon.configure.bulk.set-interval"),
          hidden: schedule !== "editable",
          onClick: () => setBulk("interval"),
        },
        {
          key: "shared",
          label: t("instance.neon.configure.bulk.use-shared"),
          onClick: () =>
            onDraftsChange(selectedKeys, (draft) => ({
              ...draft,
              environment: undefined,
              syncAll: undefined,
              syncSeconds: undefined,
            })),
        },
        {
          key: "remove",
          label: t("instance.neon.configure.bulk.remove"),
          tone: "destructive",
          onClick: () => setBulk("remove"),
        },
      ]}
    />
  );

  return (
    <div className="flex w-full flex-col gap-6">
      <section className="flex flex-col gap-3">
        <div>
          <h2 className="text-base font-semibold leading-6 text-main">
            {t("instance.neon.configure.shared.title")}
          </h2>
          <p className="text-xs leading-4 text-control-light">
            {t("instance.neon.configure.shared.description", {
              count: drafts.length,
            })}
          </p>
        </div>
        <div className="grid grid-cols-1 gap-x-6 gap-y-4 rounded-sm border border-block-border p-4 sm:grid-cols-2 xl:grid-cols-4">
          <FormField layout="vertical" title={t("common.environment")}>
            <EnvironmentSelect
              portal
              className="w-full"
              placeholder={t("instance.neon.configure.no-environment")}
              value={shared.environment}
              onChange={(environment) => onSharedChange({ environment })}
            />
            <p className="text-xs leading-4 text-control-light">
              {differHint(
                "environment",
                shared.environment
                  ? ""
                  : t("instance.neon.configure.environment-hint")
              )}
            </p>
          </FormField>
          <FormField
            layout="vertical"
            title={t("instance.sync-databases.sync-all")}
          >
            <div className="flex h-9 items-center">
              <Switch
                aria-label={t("instance.sync-databases.sync-all")}
                checked={shared.syncAll}
                onCheckedChange={(syncAll) => onSharedChange({ syncAll })}
              />
            </div>
            <p className="text-xs leading-4 text-control-light">
              {differHint(
                "syncAll",
                shared.syncAll
                  ? t("instance.neon.sheet.scope-all")
                  : t("instance.neon.configure.sync-all-off")
              )}
            </p>
          </FormField>
          {schedule === "editable" ? (
            <FormField
              layout="vertical"
              title={
                <FormLabel htmlFor={sharedIntervalId}>
                  {t("instance.neon.configure.interval")}
                </FormLabel>
              }
            >
              <NeonOptionSelect
                id={sharedIntervalId}
                value={String(shared.syncSeconds)}
                options={intervalOptions(t)}
                onChange={(value) =>
                  onSharedChange({ syncSeconds: Number(value) })
                }
              />
              <p className="text-xs leading-4 text-control-light">
                {differHint(
                  "syncSeconds",
                  t("instance.neon.configure.interval-hint")
                )}
              </p>
            </FormField>
          ) : (
            <FormField
              layout="vertical"
              title={t("instance.neon.configure.schedule")}
            >
              <p className="text-sm leading-5 text-control">
                {schedule === "off"
                  ? t("instance.neon.sync.unscheduled-help")
                  : t("instance.neon.sync.default-help")}
              </p>
            </FormField>
          )}
          <FormField
            layout="vertical"
            title={t("instance.neon.configure.limit.title")}
          >
            {capacity === undefined ? (
              <p className="flex min-h-9 items-center text-sm text-main">
                {t("instance.neon.configure.limit.unlimited")}
              </p>
            ) : (
              <Progress
                className="min-h-9 justify-center"
                value={Math.min(drafts.length, capacity)}
                max={Math.max(capacity, 1)}
                label={
                  <span className="text-sm text-main">
                    {t("instance.neon.configure.limit.usage", {
                      count: drafts.length,
                      capacity,
                    })}
                  </span>
                }
              />
            )}
            <p
              className={cn(
                "text-xs leading-4",
                over > 0 ? "text-error" : "text-control-light"
              )}
            >
              {over > 0
                ? t("instance.neon.configure.limit.over", {
                    instances: t("instance.neon.count.instances", {
                      count: over,
                    }),
                  })
                : schedule !== "off"
                  ? t("instance.neon.configure.limit.activated")
                  : ""}
            </p>
          </FormField>
        </div>
      </section>

      <NeedsPanel
        drafts={drafts}
        needs={needs}
        onDraftsChange={onDraftsChange}
        onOpenRow={onOpenRow}
      />

      <section className="flex flex-col gap-3">
        <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
          <h2 className="text-base font-semibold leading-6 text-main">
            {t("common.instances")}
          </h2>
          <SegmentedControl
            size="sm"
            appearance="soft"
            ariaLabel={t("instance.neon.select.view-label")}
            value={tab}
            onValueChange={setTab}
            options={[
              {
                value: "all",
                label: (
                  <span className="px-2 text-xs font-medium">
                    {t("instance.neon.configure.tab-all", {
                      count: drafts.length,
                    })}
                  </span>
                ),
              },
              {
                value: "needs",
                label: (
                  <span className="px-2 text-xs font-medium">
                    {t("instance.neon.configure.tab-needs", {
                      count: needs.size,
                    })}
                  </span>
                ),
              },
              {
                value: "changed",
                label: (
                  <span className="px-2 text-xs font-medium">
                    {t("instance.neon.configure.tab-changed", {
                      count: changedCount,
                    })}
                  </span>
                ),
              },
            ]}
          />
          <span className="text-xs text-control-light">
            {t("instance.neon.configure.check-hint")}
          </span>
        </div>
        <div className="overflow-x-auto rounded-sm border border-block-border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-10">
                  <Checkbox
                    className={CHECKBOX_HIT_AREA_DENSE}
                    aria-label={t("instance.neon.configure.select-shown")}
                    checked={checkedState(shownChecked, shownKeys.length)}
                    disabled={shownKeys.length === 0}
                    onCheckedChange={(on) => check(shownKeys, on)}
                  />
                </TableHead>
                <TableHead>{t("common.instance")}</TableHead>
                <TableHead className="hidden sm:table-cell">
                  {t("common.environment")}
                </TableHead>
                <TableHead className="hidden md:table-cell">
                  <span className="whitespace-nowrap">
                    {t("instance.neon.sheet.role")}
                  </span>{" "}
                  <span className="font-normal whitespace-nowrap">
                    {t("instance.neon.configure.role-note")}
                  </span>
                </TableHead>
                <TableHead className="hidden lg:table-cell">
                  {t("instance.neon.sheet.sync")}
                </TableHead>
                <TableHead className="w-10">
                  <span className="sr-only">{t("common.edit")}</span>
                </TableHead>
              </TableRow>
            </TableHeader>
            {groupByNeonProject(shown).map((group) => {
              const keys = group.rows.map((row) => branchKey(row.branch));
              const groupChecked = keys.filter((key) => checked.has(key));
              return (
                <TableBody key={group.projectId} striped={false}>
                  <TableRow className="bg-control-bg/70 hover:bg-control-bg/70">
                    <TableCell className="w-10 py-2">
                      <Checkbox
                        className={CHECKBOX_HIT_AREA_DENSE}
                        aria-label={t(
                          "instance.neon.configure.group-select-label",
                          { project: group.projectTitle }
                        )}
                        checked={checkedState(groupChecked.length, keys.length)}
                        onCheckedChange={(on) => check(keys, on)}
                      />
                    </TableCell>
                    <TableCell colSpan={COLUMN_COUNT - 1} className="py-2">
                      <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
                        <span className="text-xs text-control-light">
                          {t("instance.neon.neon-project")}
                        </span>
                        <span className="font-medium text-main">
                          {group.projectTitle}
                        </span>
                        <span className="hidden font-mono text-xs text-control-light sm:inline">
                          {group.projectId}
                        </span>
                        <span className="ml-auto text-xs text-control-light">
                          {t("instance.neon.count.instances", {
                            count: group.rows.length,
                          })}
                        </span>
                      </div>
                    </TableCell>
                  </TableRow>
                  {group.rows.map((draft) => {
                    const key = branchKey(draft.branch);
                    const need = needs.get(key);
                    const resolved = resolveDraft(draft, shared);
                    // Without activation the schedule is stated once, above.
                    const scheduleText =
                      schedule === "off"
                        ? ""
                        : syncScheduleText(t, resolved.syncSeconds);
                    return (
                      <TableRow
                        key={key}
                        data-state={checked.has(key) ? "selected" : undefined}
                      >
                        <TableCell className="align-top">
                          <Checkbox
                            className={CHECKBOX_HIT_AREA}
                            aria-label={t(
                              "instance.neon.select.row-select-label",
                              { name: draft.title }
                            )}
                            checked={checked.has(key)}
                            onCheckedChange={(on) => check([key], on)}
                          />
                        </TableCell>
                        <TableCell className="w-full max-w-0 align-top">
                          <Button
                            appearance="secondary"
                            className="block h-auto max-w-full p-0 text-left font-medium text-main wrap-anywhere whitespace-normal hover:bg-transparent hover:text-accent"
                            onClick={() => onOpenRow(key)}
                          >
                            {draft.title || draft.branch.branchTitle}
                          </Button>
                          <div className="font-mono text-xs leading-4 wrap-anywhere text-control-light">
                            {draft.instanceId}
                          </div>
                          {need && (
                            <Badge variant="attention" className="mt-1">
                              <TriangleAlert className="size-3" />
                              {needLabel(t, need)}
                            </Badge>
                          )}
                          <div className="mt-1 text-xs leading-4 text-control-light md:hidden">
                            <span className="sm:hidden">
                              {environmentTitle(resolved.environment)}
                              {" · "}
                            </span>
                            {draft.role || t("instance.neon.configure.no-role")}
                            {scheduleText && ` · ${scheduleText}`}
                          </div>
                        </TableCell>
                        <TableCell className="hidden align-top sm:table-cell">
                          <div className="flex w-48 items-center gap-1.5">
                            <NeonOptionSelect
                              size="sm"
                              aria-label={t(
                                "instance.neon.configure.row-environment-label",
                                { name: draft.title }
                              )}
                              value={rowEnvironmentValue(draft, true)}
                              options={rowEnvironmentOptions}
                              onChange={(value) =>
                                onDraftsChange([key], (row) => ({
                                  ...row,
                                  environment: rowEnvironmentFromValue(value),
                                }))
                              }
                            />
                            {draft.environment !== undefined && (
                              <Tooltip
                                content={t(
                                  "instance.neon.configure.set-on-row"
                                )}
                              >
                                <span
                                  role="img"
                                  aria-label={t(
                                    "instance.neon.configure.set-on-row"
                                  )}
                                  className="block size-1.5 shrink-0 rounded-full bg-accent"
                                />
                              </Tooltip>
                            )}
                          </div>
                        </TableCell>
                        <TableCell className="hidden align-top md:table-cell">
                          <NeonOptionSelect
                            size="sm"
                            className="w-40"
                            aria-label={t(
                              "instance.neon.configure.row-role-label",
                              { name: draft.title }
                            )}
                            value={draft.role}
                            invalid={!draft.role}
                            placeholder={t("instance.neon.choose-role")}
                            options={draft.branch.roles.map((role) => ({
                              value: role,
                              label: role,
                            }))}
                            onChange={(role) =>
                              onDraftsChange([key], (row) =>
                                withRole(row, role)
                              )
                            }
                          />
                          {(!draft.role ||
                            draft.passwordSource === "typed") && (
                            <div
                              className={cn(
                                "mt-1 text-xs leading-4",
                                need === "password"
                                  ? "text-error"
                                  : "text-control-light"
                              )}
                            >
                              {!draft.role
                                ? t("instance.neon.configure.owners-differ")
                                : draft.password
                                  ? t("instance.neon.configure.password-typed")
                                  : t(
                                      "instance.neon.configure.password-needed"
                                    )}
                            </div>
                          )}
                        </TableCell>
                        <TableCell className="hidden align-top whitespace-nowrap lg:table-cell">
                          <div className="text-main">
                            {syncScopeText(t, resolved)}
                          </div>
                          {scheduleText && (
                            <div className="flex items-center gap-1.5 text-xs leading-4 text-control-light">
                              {scheduleText}
                              {draft.syncSeconds !== undefined && (
                                <Tooltip
                                  content={t(
                                    "instance.neon.configure.set-on-row"
                                  )}
                                >
                                  <span
                                    role="img"
                                    aria-label={t(
                                      "instance.neon.configure.set-on-row"
                                    )}
                                    className="block size-1.5 shrink-0 rounded-full bg-accent"
                                  />
                                </Tooltip>
                              )}
                            </div>
                          )}
                        </TableCell>
                        <TableCell className="px-2 align-top">
                          <Button
                            appearance="secondary"
                            size="sm"
                            aria-label={t(
                              "instance.neon.configure.edit-label",
                              { name: draft.title }
                            )}
                            onClick={() => onOpenRow(key)}
                          >
                            <SlidersHorizontal className="size-4" />
                          </Button>
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              );
            })}
            {shown.length === 0 && (
              <TableBody striped={false}>
                <TableEmptyView
                  colSpan={COLUMN_COUNT}
                  contentClassName="min-h-24 flex-col gap-2"
                >
                  {tab === "needs"
                    ? t("instance.neon.configure.empty-needs")
                    : t("instance.neon.configure.empty")}
                  {tab !== "all" && (
                    <Button
                      appearance="link"
                      size="sm"
                      className="h-auto p-0"
                      onClick={() => setTab("all")}
                    >
                      {t("instance.neon.configure.show-all")}
                    </Button>
                  )}
                </TableEmptyView>
              </TableBody>
            )}
          </Table>
        </div>
      </section>

      {/* Lets the last rows scroll clear of the action bar floating above the
          footer. */}
      {selectedKeys.length > 0 && (
        <div aria-hidden="true" className="h-16 max-sm:h-28" />
      )}

      {actionBarContainer === undefined ? (
        <div className="relative">{actionBar}</div>
      ) : (
        actionBarContainer && createPortal(actionBar, actionBarContainer)
      )}

      <Dialog
        open={bulk === "environment" || bulk === "interval"}
        onOpenChange={(open) => !open && setBulk(undefined)}
      >
        <DialogContent className="w-[26rem] max-w-[calc(100vw-2rem)]">
          <BulkSettingForm
            key={bulk}
            setting={bulk === "interval" ? "interval" : "environment"}
            count={selectedKeys.length}
            environmentOptions={rowEnvironmentOptions}
            onCancel={() => setBulk(undefined)}
            onApply={(change) => {
              onDraftsChange(selectedKeys, (draft) => ({
                ...draft,
                ...change,
              }));
              setBulk(undefined);
            }}
          />
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={bulk === "remove"}
        onOpenChange={(open) => !open && setBulk(undefined)}
      >
        <AlertDialogContent>
          <AlertDialogTitle>
            {t("instance.neon.configure.bulk.remove-title", {
              count: selectedKeys.length,
            })}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t("instance.neon.configure.bulk.remove-body")}
          </AlertDialogDescription>
          <AlertDialogFooter>
            <Button appearance="outline" onClick={() => setBulk(undefined)}>
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              onClick={() => {
                onRemove(selectedKeys);
                setChecked(new Set());
                setBulk(undefined);
              }}
            >
              {t("instance.neon.configure.bulk.remove")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

function needLabel(t: TFunction, need: NeonDraftNeed): string {
  switch (need) {
    case "role":
      return t("instance.neon.configure.needs-role");
    case "password":
      return t("instance.neon.configure.needs-password");
    case "title":
      return t("instance.neon.configure.needs-name");
    case "instance-id":
      return t("instance.neon.configure.needs-id");
  }
}

const ROLE_NAME_PREVIEW = 3;

const databaseOwners = (draft: NeonImportDraft) =>
  draft.branch.databases
    .map((database) => `${database.name}: ${database.owner}`)
    .sort((a, b) => a.localeCompare(b));

// Rows are asked once when they share a Neon project, the same role choices,
// and the same database owners, so the owners named in the question hold for
// every row it sets.
function roleQuestions(rows: readonly NeonImportDraft[]) {
  const questions = new Map<string, NeonImportDraft[]>();
  for (const row of rows) {
    const key = JSON.stringify([
      row.branch.projectId,
      [...row.branch.roles].sort((a, b) => a.localeCompare(b)),
      databaseOwners(row),
    ]);
    questions.set(key, [...(questions.get(key) ?? []), row]);
  }
  return [...questions.entries()];
}

function NeedsPanel({
  drafts,
  needs,
  onDraftsChange,
  onOpenRow,
}: {
  drafts: readonly NeonImportDraft[];
  needs: ReadonlyMap<string, NeonDraftNeed>;
  onDraftsChange: (keys: readonly string[], update: DraftUpdate) => void;
  onOpenRow: (key: string) => void;
}) {
  const { t } = useTranslation();
  const id = useId();
  const listRef = useRef<HTMLDivElement>(null);
  const headerRef = useRef<HTMLDivElement>(null);
  // Once a question is answered here the panel stays to say so, and focus
  // moves on from the control the answer removed.
  const [answered, setAnswered] = useState(false);
  const answeredRef = useRef<{ key: string; index: number }>(undefined);
  // A password prompt stays for the rest of this visit once shown. Dropping
  // it at the first typed character would take the field away mid-word.
  const [prompted, setPrompted] = useState<ReadonlySet<string>>(new Set());
  const missingPasswords = drafts
    .filter((draft) => needs.get(branchKey(draft.branch)) === "password")
    .map((draft) => branchKey(draft.branch))
    .join("\n");
  useEffect(() => {
    if (!missingPasswords) return;
    setPrompted((previous) => {
      const keys = missingPasswords.split("\n");
      return keys.every((key) => previous.has(key))
        ? previous
        : new Set([...previous, ...keys]);
    });
  }, [missingPasswords]);
  const passwordRows = drafts.filter((draft) => {
    const key = branchKey(draft.branch);
    return (
      needs.get(key) === "password" ||
      (prompted.has(key) && !needs.has(key) && draft.passwordSource === "typed")
    );
  });
  const withNeed = (need: NeonDraftNeed) =>
    drafts.filter((draft) => needs.get(branchKey(draft.branch)) === need);
  const questions = roleQuestions(withNeed("role"));
  const editRows = [...withNeed("title"), ...withNeed("instance-id")];
  useEffect(() => {
    const last = answeredRef.current;
    if (!last || questions.some(([key]) => key === last.key)) return;
    answeredRef.current = undefined;
    const items = listRef.current?.children;
    const next = items?.[Math.min(last.index, items.length - 1)];
    (
      next?.querySelector<HTMLElement>("button, input") ?? headerRef.current
    )?.focus();
  });
  if (needs.size === 0 && passwordRows.length === 0 && !answered) return null;
  const itemClass =
    "flex flex-col gap-3 p-4 lg:flex-row lg:items-start lg:justify-between lg:gap-6";
  const complete = needs.size === 0;
  const empty = questions.length + passwordRows.length + editRows.length === 0;

  return (
    <section
      aria-label={t("instance.neon.configure.needs.label")}
      className={cn(
        "rounded-sm border",
        complete ? "border-block-border" : "border-warning/50"
      )}
    >
      <div
        ref={headerRef}
        tabIndex={-1}
        role="status"
        className={cn(
          "flex items-center gap-2 px-4 py-2 text-sm font-medium focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-inset",
          !empty && "border-b",
          complete
            ? "border-block-border text-control"
            : "border-warning/50 bg-warning/10 text-warning-hover"
        )}
      >
        {complete ? (
          <CircleCheck className="size-4 shrink-0 text-success" />
        ) : (
          <TriangleAlert className="size-4 shrink-0" />
        )}
        {complete
          ? t("instance.neon.configure.empty-needs")
          : t("instance.neon.configure.needs.title", { count: needs.size })}
      </div>
      <div ref={listRef} className="divide-y divide-block-border">
        {questions.map(([questionKey, rows], index) => {
          const selectId = `${id}-role-${questionKey}`;
          const names = rows
            .slice(0, ROLE_NAME_PREVIEW)
            .map((row) => row.title)
            .join(", ");
          return (
            <div key={questionKey} className={itemClass}>
              <div className="min-w-0">
                <div className="font-medium break-words text-main">
                  {t("instance.neon.configure.needs.role-title", {
                    names:
                      rows.length > ROLE_NAME_PREVIEW
                        ? `${names} ${t("instance.neon.import.and-more", { count: rows.length - ROLE_NAME_PREVIEW })}`
                        : names,
                  })}
                </div>
                <p className="max-w-3xl text-xs leading-4 text-control">
                  {t("instance.neon.configure.needs.role-body", {
                    owners: databaseOwners(rows[0]).join(", "),
                  })}
                </p>
              </div>
              <div className="flex shrink-0 flex-col gap-1.5">
                <FormLabel htmlFor={selectId}>
                  {t("instance.neon.configure.needs.role-label", {
                    count: rows.length,
                  })}
                </FormLabel>
                <NeonOptionSelect
                  id={selectId}
                  className="w-56"
                  value=""
                  placeholder={t("instance.neon.choose-role")}
                  options={rows[0].branch.roles.map((role) => ({
                    value: role,
                    label: role,
                  }))}
                  onChange={(role) => {
                    answeredRef.current = { key: questionKey, index };
                    setAnswered(true);
                    onDraftsChange(
                      rows.map((row) => branchKey(row.branch)),
                      (row) => withRole(row, role)
                    );
                  }}
                />
              </div>
            </div>
          );
        })}
        {passwordRows.map((draft) => {
          const key = branchKey(draft.branch);
          const inputId = `${id}-password-${key}`;
          return (
            <div key={key} className={itemClass}>
              <div className="min-w-0">
                <div className="font-medium break-words text-main">
                  {t("instance.neon.configure.needs.password-title", {
                    role: draft.role,
                    name: draft.title,
                  })}
                </div>
                <p className="max-w-3xl text-xs leading-4 text-control">
                  {t("instance.neon.configure.needs.password-body")}
                </p>
              </div>
              <div className="flex shrink-0 flex-col gap-1.5">
                <FormLabel htmlFor={inputId}>
                  {t("instance.neon.password-for", { role: draft.role })}
                </FormLabel>
                <SecretInput
                  id={inputId}
                  isCreating
                  className="w-56"
                  value={draft.password}
                  onValueChange={(password) =>
                    onDraftsChange([key], (row) => ({ ...row, password }))
                  }
                />
                <Button
                  appearance="link"
                  size="xs"
                  className="h-auto self-start p-0"
                  onClick={() =>
                    onDraftsChange([key], (row) => ({
                      ...row,
                      passwordSource: "neon",
                      password: "",
                    }))
                  }
                >
                  {t("instance.neon.configure.needs.use-neon")}
                </Button>
              </div>
            </div>
          );
        })}
        {editRows.map((draft) => {
          const key = branchKey(draft.branch);
          return (
            <div key={key} className={itemClass}>
              <div className="min-w-0">
                <div className="font-medium break-words text-main">
                  {needs.get(key) === "title"
                    ? t("instance.neon.configure.needs.name-title", {
                        id: draft.instanceId,
                      })
                    : t("instance.neon.configure.needs.id-title", {
                        name: draft.title,
                      })}
                </div>
                <p className="max-w-3xl text-xs leading-4 text-control">
                  {needs.get(key) === "title"
                    ? t("instance.neon.sheet.title-required")
                    : t("instance.neon.configure.needs.id-body")}
                </p>
              </div>
              <Button
                appearance="outline"
                size="sm"
                className="shrink-0 self-start"
                onClick={() => onOpenRow(key)}
              >
                {t("instance.neon.edit-settings")}
              </Button>
            </div>
          );
        })}
      </div>
    </section>
  );
}

function BulkSettingForm({
  setting,
  count,
  environmentOptions,
  onCancel,
  onApply,
}: {
  setting: "environment" | "interval";
  count: number;
  environmentOptions: readonly NeonOption[];
  onCancel: () => void;
  onApply: (change: Partial<NeonImportDraft>) => void;
}) {
  const { t } = useTranslation();
  const selectId = useId();
  // Empty until chosen, so Apply cannot set a value nobody picked.
  const [environment, setEnvironment] = useState("");
  const [interval, setChosenInterval] = useState("");
  return (
    <div className="flex flex-col gap-4">
      <DialogTitle>
        {setting === "environment"
          ? t("instance.neon.configure.bulk.environment-title", { count })
          : t("instance.neon.configure.bulk.interval-title", { count })}
      </DialogTitle>
      <DialogDescription>
        {t("instance.neon.configure.bulk.description")}
      </DialogDescription>
      {setting === "environment" ? (
        <FormField
          layout="vertical"
          title={
            <FormLabel htmlFor={selectId}>{t("common.environment")}</FormLabel>
          }
        >
          <NeonOptionSelect
            id={selectId}
            value={environment}
            placeholder={t("environment.select")}
            options={environmentOptions}
            onChange={setEnvironment}
          />
        </FormField>
      ) : (
        <FormField
          layout="vertical"
          title={
            <FormLabel htmlFor={selectId}>
              {t("instance.neon.configure.interval")}
            </FormLabel>
          }
        >
          <NeonOptionSelect
            id={selectId}
            value={interval}
            placeholder={t("instance.neon.configure.bulk.choose-interval")}
            options={[
              ...intervalOptions(t),
              {
                value: SHARED,
                label: t("instance.neon.configure.bulk.use-shared-setting"),
              },
            ]}
            onChange={setChosenInterval}
          />
        </FormField>
      )}
      <div className="flex flex-wrap justify-end gap-2">
        <Button appearance="outline" onClick={onCancel}>
          {t("common.cancel")}
        </Button>
        <Button
          disabled={setting === "environment" ? !environment : !interval}
          onClick={() =>
            onApply(
              setting === "environment"
                ? { environment: rowEnvironmentFromValue(environment) }
                : { syncSeconds: rowIntervalFromValue(interval) }
            )
          }
        >
          {t("instance.neon.configure.bulk.apply")}
        </Button>
      </div>
    </div>
  );
}
