import { create } from "@bufbuild/protobuf";
import { Plus, X } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { issueServiceClientConnect } from "@/connect";
import { Checkbox } from "@/react/components/ui/checkbox";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/react/components/ui/popover";
import { cn } from "@/react/lib/utils";
import { extractUserEmail, pushNotification } from "@/store";
import { getTimeForPbTimestampProtoEs } from "@/types";
import {
  IssueSchema,
  IssueStatus,
  UpdateIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { hasProjectPermissionV2, humanizeTs } from "@/utils";
import { usePlanDetailContext } from "../shell/PlanDetailContext";

export function PlanDetailMeta() {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const { patchState } = page;
  const project = page.project;
  const [now, setNow] = useState(Date.now());

  useEffect(() => {
    if (page.isCreating) return;
    // Only consumer is the plan-creation timestamp, which never advances
    // faster than once a minute — no need for a 1s tick.
    const timer = window.setInterval(() => setNow(Date.now()), 60_000);
    return () => window.clearInterval(timer);
  }, [page.isCreating]);

  const creatorEmail = useMemo(
    () => extractUserEmail(page.plan.creator),
    [page.plan.creator]
  );
  const createdTimeDisplay = useMemo(() => {
    const ts = getTimeForPbTimestampProtoEs(page.plan.createTime, 0);
    if (!ts) return "";
    void now;
    return humanizeTs(ts / 1000);
  }, [now, page.plan.createTime]);
  const allowChangeLabels = useMemo(() => {
    if (!project || !page.issue || page.issue.status !== IssueStatus.OPEN) {
      return false;
    }
    return hasProjectPermissionV2(project, "bb.issues.update");
  }, [page.issue, project]);

  if (page.isCreating) {
    return null;
  }

  const handleLabelsUpdate = async (labels: string[]) => {
    if (!page.issue) return;
    const issuePatch = create(IssueSchema, page.issue);
    issuePatch.labels = labels;
    try {
      const response = await issueServiceClientConnect.updateIssue(
        create(UpdateIssueRequestSchema, {
          issue: issuePatch,
          updateMask: { paths: ["labels"] },
        })
      );
      patchState({ issue: response });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
      });
    }
  };

  return (
    <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-control-placeholder">
      <span>
        {t("plan.meta.created-by-at", {
          time: createdTimeDisplay,
          user: creatorEmail,
        })}
      </span>

      {page.issue && (
        <>
          <span aria-hidden="true">·</span>
          <InlineLabels
            allowChange={allowChangeLabels}
            issueLabels={project?.issueLabels ?? []}
            labels={page.issue.labels || []}
            onUpdate={handleLabelsUpdate}
          />
        </>
      )}
    </div>
  );
}

function InlineLabels({
  allowChange,
  issueLabels,
  labels,
  onUpdate,
}: {
  allowChange: boolean;
  issueLabels: Array<{ color: string; value: string }>;
  labels: string[];
  onUpdate: (labels: string[]) => Promise<void>;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [isUpdating, setIsUpdating] = useState(false);

  const toggleLabel = async (value: string) => {
    const next = labels.includes(value)
      ? labels.filter((label) => label !== value)
      : [...labels, value];
    try {
      setIsUpdating(true);
      await onUpdate(next);
    } finally {
      setIsUpdating(false);
    }
  };

  const chips = labels.map((value) => {
    const option = issueLabels.find((item) => item.value === value);
    return (
      <span
        key={value}
        className="inline-flex items-center gap-1 rounded-xs bg-control-bg px-1.5 py-0.5 text-xs text-control"
      >
        <span
          className="size-2.5 shrink-0 rounded-sm"
          style={{ backgroundColor: option?.color }}
        />
        <span className="truncate">{value}</span>
        {allowChange && (
          <button
            aria-label={t("common.remove")}
            className={cn(
              "inline-flex shrink-0 items-center justify-center text-control-placeholder transition-colors",
              isUpdating
                ? "cursor-not-allowed opacity-60"
                : "hover:text-control"
            )}
            disabled={isUpdating}
            onClick={() => void toggleLabel(value)}
            type="button"
          >
            <X className="size-3" />
          </button>
        )}
      </span>
    );
  });

  const triggerDisabled = !allowChange || isUpdating;

  return (
    <div className="flex flex-wrap items-center gap-1">
      {chips}
      <Popover open={open} onOpenChange={allowChange ? setOpen : undefined}>
        <PopoverTrigger
          render={
            <button
              className={cn(
                "inline-flex items-center gap-1 rounded-xs border border-dashed border-control-border px-1.5 py-0.5 text-xs text-control-placeholder transition-colors",
                allowChange &&
                  !isUpdating &&
                  "hover:border-control hover:text-control",
                triggerDisabled && "cursor-not-allowed opacity-60"
              )}
              disabled={triggerDisabled}
              type="button"
            />
          }
        >
          <Plus className="size-3" />
          <span>{t("issue.labels")}</span>
        </PopoverTrigger>
        <PopoverContent
          side="bottom"
          align="start"
          initialFocus={false}
          finalFocus={false}
          className="w-56 overflow-hidden bg-white p-0"
        >
          <div className="max-h-60 overflow-y-auto">
            {issueLabels.length === 0 ? (
              <div className="px-3 py-6 text-sm text-control-placeholder">
                {t("common.no-data")}
              </div>
            ) : (
              issueLabels.map((option) => {
                const isSelected = labels.includes(option.value);
                return (
                  <button
                    key={option.value}
                    className="flex w-full items-center gap-x-2 px-3 py-2 text-left text-sm transition-colors hover:bg-control-bg"
                    disabled={isUpdating}
                    onClick={() => void toggleLabel(option.value)}
                    type="button"
                  >
                    <Checkbox checked={isSelected} />
                    <span
                      className="size-4 shrink-0 rounded-sm"
                      style={{ backgroundColor: option.color }}
                    />
                    <span>{option.value}</span>
                  </button>
                );
              })
            )}
          </div>
        </PopoverContent>
      </Popover>
    </div>
  );
}
