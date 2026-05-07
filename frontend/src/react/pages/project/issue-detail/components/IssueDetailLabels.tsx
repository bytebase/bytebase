import { create } from "@bufbuild/protobuf";
import { ChevronDown, ExternalLink, X } from "lucide-react";
import {
  type MouseEvent as ReactMouseEvent,
  useCallback,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { issueServiceClientConnect } from "@/connect";
import { LAYER_SURFACE_CLASS } from "@/react/components/ui/layer";
import { useClickOutside } from "@/react/hooks/useClickOutside";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import { PROJECT_V1_ROUTE_SETTINGS } from "@/router/dashboard/projectV1";
import { pushNotification, useProjectV1Store } from "@/store";
import { getProjectName, projectNamePrefix } from "@/store/modules/v1/common";
import {
  IssueSchema,
  IssueStatus,
  UpdateIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import { useIssueDetailContext } from "../context/IssueDetailContext";

type IssueLabelOption = {
  color: string;
  value: string;
};

export function IssueDetailLabels() {
  const { t } = useTranslation();
  const page = useIssueDetailContext();
  const projectStore = useProjectV1Store();
  const containerRef = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);
  const [isUpdating, setIsUpdating] = useState(false);
  const projectName = `${projectNamePrefix}${page.projectId}`;
  const project = useVueState(() => projectStore.getProjectByName(projectName));

  useClickOutside(containerRef, open, () => setOpen(false));

  const options = useMemo<IssueLabelOption[]>(() => {
    return (project?.issueLabels ?? []).map((label) => ({
      color: label.color,
      value: label.value,
    }));
  }, [project?.issueLabels]);
  const selected = useMemo(() => {
    const pool = new Set(options.map((label) => label.value));
    return (page.issue?.labels ?? []).filter((label) => pool.has(label));
  }, [options, page.issue?.labels]);
  const allowChange = useMemo(() => {
    if (!project || !page.issue || page.issue.status !== IssueStatus.OPEN) {
      return false;
    }
    return hasProjectPermissionV2(project, "bb.issues.update");
  }, [page.issue, project]);
  const settingsHref = useMemo(() => {
    if (!project) {
      return "";
    }
    return router.resolve({
      hash: "#issue-related",
      name: PROJECT_V1_ROUTE_SETTINGS,
      params: {
        projectId: getProjectName(project.name),
      },
    }).href;
  }, [project]);

  const updateLabels = useCallback(
    async (labels: string[]) => {
      if (!page.issue) {
        return;
      }
      try {
        setIsUpdating(true);
        const issuePatch = create(IssueSchema, {
          ...page.issue,
          labels,
        });
        const request = create(UpdateIssueRequestSchema, {
          issue: issuePatch,
          updateMask: { paths: ["labels"] },
        });
        const response = await issueServiceClientConnect.updateIssue(request);
        page.patchState({ issue: response });
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
      } finally {
        setIsUpdating(false);
      }
    },
    [page.issue, t]
  );

  const toggleLabel = useCallback(
    async (value: string) => {
      const next = selected.includes(value)
        ? selected.filter((label) => label !== value)
        : [...selected, value];
      await updateLabels(next);
    },
    [selected, updateLabels]
  );

  const removeSelectedLabel = useCallback(
    async (value: string, e: ReactMouseEvent) => {
      e.stopPropagation();
      await updateLabels(selected.filter((label) => label !== value));
    },
    [selected, updateLabels]
  );

  return (
    <div className="flex flex-col gap-y-1">
      <div className="flex items-center gap-x-1 textinfolabel">
        <span>{t("issue.labels")}</span>
        {project?.forceIssueLabels && <span className="text-error">*</span>}
      </div>
      <div ref={containerRef} className="relative">
        <button
          className={cn(
            "flex min-h-9 w-full items-center justify-between gap-2 rounded-sm border border-control-border bg-white px-3 py-1.5 text-left text-sm transition-colors",
            allowChange && !isUpdating && "hover:bg-control-bg",
            open && "border-accent shadow-[0_0_0_1px_var(--color-accent)]",
            (!allowChange || isUpdating) && "cursor-not-allowed opacity-60"
          )}
          disabled={!allowChange || isUpdating}
          onClick={() => setOpen((current) => !current)}
          type="button"
        >
          <div className="flex min-w-0 flex-1 flex-wrap items-center gap-1.5">
            {selected.length > 0 ? (
              selected.map((value) => {
                const option = options.find((item) => item.value === value);
                return (
                  <span
                    key={value}
                    className="inline-flex items-center gap-1 rounded-xs bg-control-bg px-1.5 py-0.5 text-xs"
                  >
                    <span
                      className="size-2.5 shrink-0 rounded-sm"
                      style={{ backgroundColor: option?.color }}
                    />
                    <span className="truncate">{value}</span>
                    {allowChange && !isUpdating && (
                      <button
                        className="text-control-placeholder hover:text-control"
                        onClick={(e) => {
                          void removeSelectedLabel(value, e);
                        }}
                        type="button"
                      >
                        <X className="size-3" />
                      </button>
                    )}
                  </span>
                );
              })
            ) : (
              <span className="text-control-placeholder">
                {t("common.select")}
              </span>
            )}
          </div>
          <ChevronDown
            className={cn(
              "size-4 shrink-0 text-control-placeholder transition-transform",
              open && "rotate-180"
            )}
          />
        </button>

        {open && (
          <div
            className={cn(
              "absolute mt-1 w-full overflow-hidden rounded-sm border border-control-border bg-white shadow-lg",
              LAYER_SURFACE_CLASS
            )}
          >
            <div className="max-h-60 overflow-y-auto">
              {options.length === 0 ? (
                <div className="flex flex-col items-center gap-y-3 px-3 py-6 text-sm text-control-placeholder">
                  <span>{t("common.no-data")}</span>
                  {settingsHref &&
                    project &&
                    hasProjectPermissionV2(project, "bb.projects.update") && (
                      <a
                        className="flex items-center gap-x-2 textinfolabel normal-link"
                        href={settingsHref}
                      >
                        <span>
                          {t(
                            "project.settings.issue-related.labels.configure-labels"
                          )}
                        </span>
                        <ExternalLink className="h-4 w-4" />
                      </a>
                    )}
                </div>
              ) : (
                options.map((option) => {
                  const isSelected = selected.includes(option.value);
                  return (
                    <button
                      key={option.value}
                      className="flex w-full items-center gap-x-2 px-3 py-2 text-left text-sm transition-colors hover:bg-control-bg"
                      disabled={isUpdating}
                      onClick={() => {
                        void toggleLabel(option.value);
                      }}
                      type="button"
                    >
                      <input
                        checked={isSelected}
                        className="accent-accent"
                        readOnly
                        type="checkbox"
                      />
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
          </div>
        )}
      </div>
    </div>
  );
}
