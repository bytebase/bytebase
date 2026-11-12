import type { TFunction } from "i18next";
import {
  CalendarClock,
  Check,
  CircleHelp,
  CirclePause,
  CircleX,
  ExternalLink,
  Hand,
  Info,
  ListChecks,
  LoaderCircle,
} from "lucide-react";
import { type ReactNode, useId, useState } from "react";
import { useTranslation } from "react-i18next";
import { SecretInput } from "@/components/SecretInput";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { FormLabel } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { NeonOptionSelect } from "./NeonOptionSelect";
import { useEnvironmentTitle } from "./NeonRowSheet";
import { NeonWarning } from "./NeonWarning";
import {
  entryRetryable,
  failureKind,
  formatClock,
  isValidInstanceId,
  type NeonImportBatch,
  type NeonImportEntry,
  type NeonImportRun,
  type NeonResolvedDraft,
  type NeonScheduleMode,
  neonReasonText,
  rejectionText,
  summarizeRun,
  withRole,
} from "./neon-import";

const CREATED_PREVIEW = 5;
const NAME_PREVIEW = 8;

type NeonImportPhase = "running" | "done" | "stopped" | "paused" | "finished";

export function importPhase(
  run: NeonImportRun,
  stopped: boolean
): NeonImportPhase {
  const summary = summarizeRun(run);
  if (summary.awaiting > 0) return "running";
  if (summary.failed + summary.unknown + summary.unsent === 0) return "done";
  if (stopped) return "stopped";
  return summary.unknown + summary.unsent > 0 ? "paused" : "finished";
}

type DraftUpdate = (draft: NeonResolvedDraft) => NeonResolvedDraft;

export function NeonImportStep({
  run,
  stopped,
  checking,
  schedule,
  duplicateIds,
  destination,
  instancesHref,
  context,
  onCheck,
  onRetryUnknown,
  onRetryFailed,
  onContinue,
  onEntryChange,
  onEditEntry,
}: {
  run: NeonImportRun;
  // The user stopped the import and has not continued it.
  stopped: boolean;
  checking: boolean;
  schedule: NeonScheduleMode;
  // Instance IDs more than one entry of the run uses.
  duplicateIds: ReadonlySet<string>;
  destination: string;
  instancesHref: string;
  // The engine, destination, and API-key summary shown under the title.
  context?: ReactNode;
  onCheck: () => void;
  onRetryUnknown: (key: string) => void;
  onRetryFailed: () => void;
  onContinue: () => void;
  onEntryChange: (key: string, update: DraftUpdate) => void;
  onEditEntry: (key: string) => void;
}) {
  const { t } = useTranslation();
  const environmentTitle = useEnvironmentTitle();
  const [showAllCreated, setShowAllCreated] = useState(false);
  const summary = summarizeRun(run);
  const phase = importPhase(run, stopped);
  const running = phase === "running";
  const by = (state: NeonImportEntry["state"]) =>
    run.entries.filter((entry) => entry.state === state);
  const unknown = by("unknown");
  const failed = by("failed");
  const awaiting = by("awaiting");
  const unsent = by("unsent");
  const created = by("created");
  const entries = (count: number) =>
    t("instance.neon.count.entries", { count });

  const batchStates = (index: number) =>
    run.entries
      .filter((entry) => entry.batch === index)
      .map((entry) => entry.state);
  const activeBatch = run.batches.findIndex((_, index) =>
    run.entries.some(
      (entry) =>
        entry.batch === index && entry.state === "awaiting" && !entry.retrying
    )
  );
  const batchLabel = (batch: NeonImportBatch, index: number) => {
    const states = batchStates(index);
    if (states.includes("awaiting")) {
      return t("instance.neon.import.batch.awaiting");
    }
    if (states.every((state) => state === "unsent")) {
      return t("instance.neon.import.batch.unsent");
    }
    if (states.includes("unknown")) {
      return batch.cause === "lost"
        ? t("instance.neon.import.batch.lost")
        : batch.cause === "stopped"
          ? t("instance.neon.import.batch.stopped")
          : t("instance.neon.import.batch.unknown");
    }
    return t("instance.neon.import.batch.returned");
  };
  const batchTone = (index: number) => {
    const states = batchStates(index);
    if (states.includes("awaiting")) return "bg-accent/40 animate-pulse";
    if (states.every((state) => state === "unsent")) {
      return "bg-control-bg-hover";
    }
    return states.includes("unknown") ? "bg-warning" : "bg-accent";
  };

  const lostBatch = run.batches.some((batch) => batch.cause === "lost");
  const rejection = failed.find((entry) => entry.rejection)?.rejection;
  const retryable = failed.filter((entry) =>
    entryRetryable(entry, duplicateIds)
  );
  const limited = failed.filter(
    (entry) => failureKind(entry.reason) === "limit"
  );
  const scheduled = created.filter(
    (entry) => schedule !== "off" && entry.draft.syncSeconds > 0
  ).length;
  const manual = created.length - scheduled;
  const uncheckedUnknown = unknown.every((entry) => !entry.checkedAt);

  const title = {
    running: t("instance.neon.import.title.running", { count: summary.total }),
    done: t("instance.neon.import.title.done", { count: summary.total }),
    stopped: t("instance.neon.import.title.stopped"),
    paused: t("instance.neon.import.title.paused"),
    finished: t("instance.neon.import.title.finished"),
  }[phase];

  const banner = (() => {
    switch (phase) {
      case "running":
        return (
          <p className="flex items-start gap-1.5 text-xs leading-4 text-control-light">
            <Info className="mt-px size-3.5 shrink-0" />
            <span>{t("instance.neon.import.banner.running")}</span>
          </p>
        );
      case "paused":
        if (unknown.length === 0) {
          return (
            <NeonWarning
              title={
                rejection
                  ? t("instance.neon.import.banner.rejected-title")
                  : t("instance.neon.import.banner.paused-resolved-title")
              }
              description={[
                rejection
                  ? [
                      rejectionText(t, rejection),
                      t("instance.neon.import.banner.rejected-body"),
                    ].join(" ")
                  : t("instance.neon.import.banner.paused-resolved-body"),
                t("instance.neon.import.banner.unsent", {
                  count: unsent.length,
                }),
              ].join(" ")}
            />
          );
        }
        return (
          <NeonWarning
            title={
              lostBatch
                ? t("instance.neon.import.banner.paused-lost-title")
                : t("instance.neon.import.banner.paused-unknown-title", {
                    count: unknown.length,
                  })
            }
            description={[
              lostBatch
                ? t("instance.neon.import.banner.paused-lost-body")
                : t("instance.neon.import.banner.paused-unknown-body"),
              unsent.length > 0
                ? t("instance.neon.import.banner.paused-note")
                : t("instance.neon.import.banner.paused-note-all-sent"),
            ].join(" ")}
          />
        );
      case "stopped":
        return (
          <NeonWarning
            title={t("instance.neon.import.banner.stopped-title")}
            description={[
              t("instance.neon.import.banner.stopped-created"),
              unknown.length > 0 &&
                t("instance.neon.import.banner.stopped-unknown", {
                  count: unknown.length,
                }),
              unsent.length > 0 &&
                t("instance.neon.import.banner.stopped-unsent", {
                  count: unsent.length,
                }),
            ]
              .filter(Boolean)
              .join(" ")}
          />
        );
      case "finished":
        return rejection ? (
          <NeonWarning
            title={t("instance.neon.import.banner.rejected-title")}
            description={[
              rejectionText(t, rejection),
              t("instance.neon.import.banner.rejected-body"),
            ].join(" ")}
          />
        ) : (
          <Alert
            variant="info"
            title={t("instance.neon.import.banner.finished-title")}
            description={t("instance.neon.import.banner.finished-body", {
              count: failed.length,
            })}
          />
        );
      case "done":
        return (
          <Alert
            variant="info"
            title={t("instance.neon.import.banner.done-title")}
            description={t("instance.neon.import.banner.done-body")}
          />
        );
    }
  })();

  return (
    <div className="flex w-full max-w-5xl flex-col gap-4">
      <h1
        data-step-heading
        tabIndex={-1}
        className="text-lg font-semibold leading-7 text-main focus:outline-hidden"
      >
        {title}
      </h1>
      {context}

      <section
        aria-label={t("instance.neon.import.progress-label")}
        className="flex flex-col gap-3 rounded-sm border border-block-border p-4"
      >
        <div className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
          <div
            role="status"
            className="text-base font-semibold leading-6 text-main"
          >
            {t("instance.neon.import.progress", {
              processed: summary.processed,
              total: summary.total,
            })}
          </div>
          <div className="text-xs leading-4 text-control-light">
            {activeBatch >= 0
              ? t("instance.neon.import.progress-batch", {
                  index: activeBatch + 1,
                  total: run.batches.length,
                  time: formatClock(
                    run.batches[activeBatch].sentAt ?? new Date()
                  ),
                  count: run.batches[activeBatch].size,
                })
              : running
                ? t("instance.neon.import.progress-retry")
                : t("instance.neon.import.progress-batches", {
                    count: run.batches.length,
                  })}
          </div>
        </div>
        <div className="flex flex-col gap-1.5">
          <div className="flex gap-1" aria-hidden="true">
            {run.batches.map((batch, index) => (
              <div
                key={index}
                className={cn("h-2 rounded-xs", batchTone(index))}
                style={{ flex: `${batch.size} 1 0%` }}
              />
            ))}
          </div>
          <ul className="flex flex-wrap gap-x-4 gap-y-1 text-xs leading-4 text-control-light">
            {run.batches.map((batch, index) => (
              <li key={index} className="whitespace-nowrap">
                {t("instance.neon.import.batch-label", {
                  index: index + 1,
                  size: batch.size,
                  state: batchLabel(batch, index),
                })}
              </li>
            ))}
          </ul>
        </div>
        <dl className="flex flex-wrap gap-x-4 gap-y-1 text-xs leading-4">
          <Count
            tone="bg-success"
            label={t("instance.neon.import.state.created")}
            count={summary.created}
          />
          <Count
            tone="bg-error"
            label={t("instance.neon.import.state.failed")}
            count={summary.failed}
          />
          <Count
            tone="bg-warning"
            label={t("instance.neon.import.state.unknown")}
            count={summary.unknown}
          />
          <Count
            tone="bg-accent/40"
            label={t("instance.neon.import.state.awaiting")}
            count={summary.awaiting}
          />
          <Count
            tone="bg-control-bg-hover"
            label={t("instance.neon.import.state.unsent")}
            count={summary.unsent}
          />
        </dl>
      </section>

      {banner}

      {unknown.length > 0 && (
        <Group
          tone="border-warning/60"
          badge={
            <Badge variant="attention">
              <CircleHelp className="size-3" />
              {t("instance.neon.import.state.unknown")}
            </Badge>
          }
          heading={entries(unknown.length)}
          description={t("instance.neon.import.unknown.description")}
          actions={
            <>
              <Button
                size="sm"
                appearance={uncheckedUnknown ? "solid" : "outline"}
                disabled={checking || running}
                onClick={onCheck}
              >
                {checking ? (
                  <LoaderCircle className="size-3.5 animate-spin" />
                ) : (
                  <ListChecks className="size-3.5" />
                )}
                {checking
                  ? t("instance.neon.import.unknown.checking")
                  : uncheckedUnknown
                    ? t("instance.neon.import.unknown.check")
                    : t("instance.neon.import.unknown.check-again")}
              </Button>
              <a
                className="inline-flex items-center gap-1 text-xs font-medium text-accent hover:underline"
                href={instancesHref}
                target="_blank"
                rel="noreferrer"
              >
                {t("instance.neon.import.open-list")}
                <ExternalLink className="size-3.5" />
              </a>
            </>
          }
        >
          <ul className="divide-y divide-block-border">
            {unknown.map((entry) => (
              <Entry
                key={entry.key}
                entry={entry}
                side={
                  entry.checkedAt && (
                    <Button
                      size="sm"
                      appearance="outline"
                      disabled={running || checking}
                      onClick={() => onRetryUnknown(entry.key)}
                    >
                      {t("instance.neon.import.unknown.retry")}
                    </Button>
                  )
                }
              >
                <p
                  className={cn(
                    "text-xs leading-4",
                    entry.checkedAt ? "text-main" : "text-control-light"
                  )}
                >
                  {entry.checkedAt
                    ? t("instance.neon.import.unknown.checked", {
                        time: formatClock(entry.checkedAt),
                      })
                    : entry.unknownCause === "stopped"
                      ? t("instance.neon.import.unknown.stopped")
                      : entry.unknownCause === "lost"
                        ? t("instance.neon.import.unknown.lost")
                        : t("instance.neon.import.unknown.reported")}
                </p>
                {entry.checkFailed && (
                  <p className="text-xs leading-4 text-error">
                    {t("instance.neon.import.unknown.check-failed")}
                  </p>
                )}
              </Entry>
            ))}
          </ul>
        </Group>
      )}

      {failed.length > 0 && (
        <Group
          tone="border-error/40"
          badge={
            <Badge variant="destructive">
              <CircleX className="size-3" />
              {t("instance.neon.import.state.failed")}
            </Badge>
          }
          heading={entries(failed.length)}
          description={t("instance.neon.import.failed.description")}
          actions={
            <Button
              size="sm"
              // One primary action at a time: checking unknown outcomes and
              // sending the unsent entries both come before a retry.
              appearance={
                (unknown.length > 0 && uncheckedUnknown) ||
                (unsent.length > 0 && !running)
                  ? "outline"
                  : "solid"
              }
              disabled={retryable.length === 0 || running || checking}
              onClick={onRetryFailed}
            >
              {retryable.length > 0
                ? t("instance.neon.import.failed.retry", {
                    count: retryable.length,
                  })
                : t("instance.neon.import.failed.retry-none")}
            </Button>
          }
        >
          <ul className="divide-y divide-block-border">
            {failed
              .filter((entry) => failureKind(entry.reason) !== "limit")
              .map((entry) => (
                <FailedEntry
                  key={entry.key}
                  entry={entry}
                  duplicateIds={duplicateIds}
                  destination={destination}
                  disabled={running}
                  onChange={(update) => onEntryChange(entry.key, update)}
                  onEdit={() => onEditEntry(entry.key)}
                />
              ))}
            {limited.length > 0 && (
              <li className="flex flex-col gap-1 px-4 py-3">
                <div className="font-medium text-main">
                  {t("instance.neon.import.failed.limit-title", {
                    count: limited.length,
                  })}
                </div>
                <p className="text-xs leading-4 text-error">
                  {t("instance.neon.import.failed.limit-error")}
                </p>
                <p className="text-xs leading-4 text-control-light">
                  {t("instance.neon.import.failed.limit-help")}
                </p>
                <Names entries={limited} max={6} />
              </li>
            )}
          </ul>
        </Group>
      )}

      {awaiting.length > 0 && (
        <Group
          badge={
            <Badge variant="secondary">
              <LoaderCircle className="size-3 animate-spin" />
              {t("instance.neon.import.state.awaiting")}
            </Badge>
          }
          heading={entries(awaiting.length)}
          description={
            activeBatch >= 0
              ? t("instance.neon.import.awaiting.description", {
                  index: activeBatch + 1,
                  time: formatClock(
                    run.batches[activeBatch].sentAt ?? new Date()
                  ),
                  count: awaiting.length,
                })
              : t("instance.neon.import.awaiting.retrying")
          }
        >
          <Names entries={awaiting} max={NAME_PREVIEW} className="px-4 py-3" />
        </Group>
      )}

      {unsent.length > 0 && (
        <Group
          badge={
            <Badge>
              <CirclePause className="size-3" />
              {t("instance.neon.import.state.unsent")}
            </Badge>
          }
          heading={entries(unsent.length)}
          description={
            running
              ? t("instance.neon.import.unsent.waiting")
              : t("instance.neon.import.unsent.description", {
                  count: unsent.length,
                })
          }
          actions={
            !running && (
              <Button
                size="sm"
                appearance={unknown.length > 0 ? "outline" : "solid"}
                disabled={checking}
                onClick={onContinue}
              >
                {t("instance.neon.import.unsent.continue", {
                  count: unsent.length,
                })}
              </Button>
            )
          }
        >
          <Names entries={unsent} max={NAME_PREVIEW} className="px-4 py-3" />
        </Group>
      )}

      {created.length > 0 && (
        <Group
          badge={
            <Badge variant="success">
              <Check className="size-3" />
              {t("instance.neon.import.state.created")}
            </Badge>
          }
          heading={t("instance.neon.count.instances", {
            count: created.length,
          })}
          description={t("instance.neon.import.created.description", {
            destination,
          })}
        >
          {schedule === "off" ? (
            <div className="flex gap-2 border-b border-block-border px-4 py-3 text-xs leading-4 text-control">
              <Hand className="size-4 shrink-0 text-control-light" />
              {t("instance.neon.sync.unscheduled-help")}
            </div>
          ) : (
            <div
              className={cn(
                "grid grid-cols-1 border-b border-block-border text-xs leading-4",
                scheduled > 0 &&
                  manual > 0 &&
                  "md:grid-cols-2 md:divide-x md:divide-block-border max-md:divide-y max-md:divide-block-border"
              )}
            >
              {scheduled > 0 && (
                <div className="flex gap-2 px-4 py-3">
                  <CalendarClock className="size-4 shrink-0 text-control-light" />
                  <div>
                    <span className="font-medium text-main">
                      {t("instance.neon.import.created.scheduled", {
                        count: scheduled,
                      })}
                    </span>{" "}
                    <span className="text-control">
                      {t("instance.neon.import.created.scheduled-help")}
                    </span>
                  </div>
                </div>
              )}
              {manual > 0 && (
                <div className="flex gap-2 px-4 py-3">
                  <Hand className="size-4 shrink-0 text-control-light" />
                  <div>
                    <span className="font-medium text-main">
                      {t("instance.neon.import.created.manual", {
                        count: manual,
                      })}
                    </span>{" "}
                    <span className="text-control">
                      {t("instance.neon.import.created.manual-help", {
                        count: manual,
                      })}
                    </span>
                  </div>
                </div>
              )}
            </div>
          )}
          <ul className="divide-y divide-block-border">
            {created
              .slice(0, showAllCreated ? created.length : CREATED_PREVIEW)
              .map((entry) => (
                <Entry
                  key={entry.key}
                  entry={entry}
                  side={
                    <div className="flex flex-wrap items-center gap-2">
                      <Badge>{environmentTitle(entry.draft.environment)}</Badge>
                      {schedule !== "off" &&
                        (entry.draft.syncSeconds > 0 ? (
                          <Badge>
                            <CalendarClock className="size-3" />
                            {t("instance.neon.import.created.badge-scheduled")}
                          </Badge>
                        ) : (
                          <Badge variant="attention">
                            <Hand className="size-3" />
                            {t("instance.neon.import.created.badge-manual")}
                          </Badge>
                        ))}
                      {entry.instance && (
                        <a
                          className="inline-flex items-center gap-1 text-xs font-medium text-accent hover:underline"
                          href={`/${entry.instance.name}`}
                          target="_blank"
                          rel="noreferrer"
                          aria-label={t("instance.neon.import.created.open", {
                            name: entry.draft.title,
                          })}
                        >
                          {t("common.open")}
                          <ExternalLink className="size-3.5" />
                        </a>
                      )}
                    </div>
                  }
                >
                  {entry.resolvedBy && (
                    <p className="text-xs leading-4 text-control-light">
                      {entry.resolvedBy === "check"
                        ? t("instance.neon.import.created.by-check")
                        : t("instance.neon.import.created.by-retry")}
                    </p>
                  )}
                </Entry>
              ))}
          </ul>
          {created.length > CREATED_PREVIEW && (
            <div className="border-t border-block-border px-4 py-2">
              <Button
                appearance="link"
                size="xs"
                className="h-auto p-0"
                onClick={() => setShowAllCreated((shown) => !shown)}
              >
                {showAllCreated
                  ? t("instance.neon.import.created.show-fewer")
                  : t("instance.neon.import.created.show-all", {
                      count: created.length,
                    })}
              </Button>
            </div>
          )}
        </Group>
      )}
    </div>
  );
}

function Count({
  tone,
  label,
  count,
}: {
  tone: string;
  label: string;
  count: number;
}) {
  return (
    <div
      className={cn(
        "inline-flex items-center gap-1.5",
        count > 0 ? "text-main" : "text-control-light"
      )}
    >
      <span className={cn("size-2 rounded-full", tone)} aria-hidden="true" />
      <dt>{label}</dt>
      <dd className="font-medium">{count}</dd>
    </div>
  );
}

function Group({
  tone = "border-block-border",
  badge,
  heading,
  description,
  actions,
  children,
}: {
  tone?: string;
  badge: ReactNode;
  heading: string;
  description: string;
  actions?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className={cn("overflow-hidden rounded-sm border", tone)}>
      <div className="flex flex-col gap-2 border-b border-block-border px-4 py-3 md:flex-row md:items-start md:justify-between md:gap-6">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            {badge}
            <h2 className="text-sm font-semibold text-main">{heading}</h2>
          </div>
          <p className="mt-1 max-w-3xl text-xs leading-4 text-control">
            {description}
          </p>
        </div>
        {actions && (
          <div className="flex shrink-0 flex-wrap items-center gap-2">
            {actions}
          </div>
        )}
      </div>
      {children}
    </section>
  );
}

function Entry({
  entry,
  side,
  children,
}: {
  entry: NeonImportEntry;
  side?: ReactNode;
  children?: ReactNode;
}) {
  return (
    <li className="flex flex-col gap-2 px-4 py-3 md:flex-row md:items-start md:justify-between md:gap-6">
      <div className="flex min-w-0 flex-col gap-1">
        <div>
          <div className="font-medium wrap-anywhere text-main">
            {entry.draft.title}
          </div>
          <div className="font-mono text-xs leading-4 wrap-anywhere text-control-light">
            {entry.draft.instanceId}
          </div>
        </div>
        {children}
      </div>
      {side && (
        <div className="flex shrink-0 flex-col items-start gap-1.5 md:items-end">
          {side}
        </div>
      )}
    </li>
  );
}

function Names({
  entries,
  max,
  className,
}: {
  entries: readonly NeonImportEntry[];
  max: number;
  className?: string;
}) {
  const { t } = useTranslation();
  return (
    <p className={cn("text-xs leading-5 text-control-light", className)}>
      {entries
        .slice(0, max)
        .map((entry) => entry.draft.title)
        .join(" · ")}
      {entries.length > max &&
        ` · ${t("instance.neon.import.and-more", { count: entries.length - max })}`}
    </p>
  );
}

const failureText = (t: TFunction, entry: NeonImportEntry) => {
  const attempted = entry.attempted;
  switch (failureKind(entry.reason)) {
    case "password":
      return {
        error: t("instance.neon.import.failed.password-error", {
          role: attempted?.role,
        }),
        help: t("instance.neon.import.failed.password-help"),
      };
    case "role":
      return {
        error: t("instance.neon.import.failed.role-error", {
          role: attempted?.role,
        }),
        help: t("instance.neon.import.failed.role-help"),
      };
    case "database":
      return {
        error: t("instance.neon.import.failed.database-error", {
          database: attempted?.database,
        }),
        help: t("instance.neon.import.failed.database-help"),
      };
    case "instance-id":
      return {
        error: t("instance.neon.import.failed.id-error", {
          id: attempted?.instanceId,
        }),
        help: t("instance.neon.import.failed.id-help"),
      };
    default:
      return {
        error: entry.rejection
          ? rejectionText(t, entry.rejection)
          : neonReasonText(t, entry.reason, entry.message, entry.category),
        help: t("instance.neon.import.failed.other-help"),
      };
  }
};

function FailedEntry({
  entry,
  duplicateIds,
  destination,
  disabled,
  onChange,
  onEdit,
}: {
  entry: NeonImportEntry;
  duplicateIds: ReadonlySet<string>;
  destination: string;
  disabled: boolean;
  onChange: (update: DraftUpdate) => void;
  onEdit: () => void;
}) {
  const { t } = useTranslation();
  const id = useId();
  const { draft, attempted } = entry;
  const kind = failureKind(entry.reason);
  const ready = entryRetryable(entry, duplicateIds);
  const duplicate = duplicateIds.has(draft.instanceId);
  const text = failureText(t, entry);

  const correction = (() => {
    switch (kind) {
      case "password":
        return (
          <>
            <FormLabel htmlFor={id}>
              {t("instance.neon.password-for", { role: draft.role })}
            </FormLabel>
            <SecretInput
              id={id}
              isCreating
              className="w-56"
              disabled={disabled}
              value={draft.passwordSource === "typed" ? draft.password : ""}
              onValueChange={(password) =>
                onChange((row) => ({
                  ...row,
                  passwordSource: password ? "typed" : "neon",
                  password,
                }))
              }
            />
            <Hint ready={ready}>{t("instance.neon.sheet.password-help")}</Hint>
          </>
        );
      case "role":
        return (
          <>
            <FormLabel htmlFor={id}>{t("instance.neon.sheet.role")}</FormLabel>
            <NeonOptionSelect
              id={id}
              size="sm"
              className="w-56"
              disabled={disabled}
              value={draft.role === attempted?.role ? "" : draft.role}
              placeholder={t("instance.neon.choose-role")}
              options={draft.branch.roles
                .filter((role) => role !== attempted?.role)
                .map((role) => ({ value: role, label: role }))}
              onChange={(role) =>
                onChange((row) => ({
                  ...withRole(row, role),
                  passwordSource: "neon",
                }))
              }
            />
            <Hint ready={ready}>
              {t("instance.neon.import.failed.role-password")}
            </Hint>
          </>
        );
      case "database":
        return (
          <>
            <FormLabel htmlFor={id}>
              {t("instance.neon.sheet.database")}
            </FormLabel>
            <NeonOptionSelect
              id={id}
              size="sm"
              className="w-56"
              disabled={disabled}
              value={
                draft.database === attempted?.database ? "" : draft.database
              }
              placeholder={t("instance.neon.import.failed.choose-database")}
              options={draft.branch.databases
                .filter((database) => database.name !== attempted?.database)
                .map((database) => ({
                  value: database.name,
                  label: database.name,
                }))}
              onChange={(database) =>
                onChange((row) => ({
                  ...row,
                  database,
                  databases: row.databases.filter(
                    (name) => name !== attempted?.database
                  ),
                }))
              }
            />
            <Hint ready={ready} />
          </>
        );
      case "instance-id":
        return (
          <>
            <FormLabel htmlFor={id}>
              {t("instance.neon.sheet.instance-id")}
            </FormLabel>
            <Input
              id={id}
              size="sm"
              className="w-full font-mono sm:w-96"
              spellCheck={false}
              disabled={disabled}
              value={draft.instanceId}
              aria-invalid={
                !isValidInstanceId(draft.instanceId) || duplicate || undefined
              }
              aria-describedby={duplicate ? `${id}-duplicate` : undefined}
              onChange={(event) => {
                const instanceId = event.target.value.trim();
                onChange((row) => ({ ...row, instanceId }));
              }}
            />
            <Hint ready={ready}>
              {t("instance.neon.import.failed.id-scope", { destination })}
            </Hint>
          </>
        );
      default:
        return (
          <Button
            size="sm"
            appearance="outline"
            disabled={disabled}
            onClick={onEdit}
          >
            {t("instance.neon.edit-settings")}
          </Button>
        );
    }
  })();

  return (
    <Entry entry={entry} side={correction}>
      <div>
        <p className="text-xs leading-4 text-error">{text.error}</p>
        <p className="text-xs leading-4 text-control-light">{text.help}</p>
        {duplicate && (
          <p id={`${id}-duplicate`} className="text-xs leading-4 text-error">
            {t("instance.neon.sheet.id-duplicate")}
          </p>
        )}
      </div>
    </Entry>
  );
}

function Hint({ ready, children }: { ready: boolean; children?: ReactNode }) {
  const { t } = useTranslation();
  return (
    <span className="text-xs leading-4 text-control-light">
      {ready && (
        <span className="font-medium text-success">
          {t("instance.neon.import.failed.ready")}{" "}
        </span>
      )}
      {children}
    </span>
  );
}
