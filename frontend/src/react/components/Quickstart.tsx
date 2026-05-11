import { CheckCircle, X } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type { RouteLocationRaw } from "vue-router";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { useNavigate } from "@/react/router";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import {
  DATABASE_ROUTE_DASHBOARD,
  ENVIRONMENT_V1_ROUTE_DASHBOARD,
  INSTANCE_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DASHBOARD,
  WORKSPACE_ROUTE_USERS,
} from "@/router/dashboard/workspaceRoutes";
import { SQL_EDITOR_WORKSHEET_MODULE } from "@/router/sqlEditor";
import {
  pushNotification,
  useActuatorV1Store,
  useIssueV1Store,
  useProjectIamPolicyStore,
  useProjectV1Store,
  useUIStateStore,
  useWorkSheetStore,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import type { Permission } from "@/types";
import { isValidProjectName, UNKNOWN_PROJECT_NAME } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  extractProjectResourceName,
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
} from "@/utils";

const SAMPLE_PROJECT_NAME = "project-sample";
const SAMPLE_ISSUE_ID = "101";
const SAMPLE_SHEET_ID = "101";

interface IntroItem {
  name: string;
  link: RouteLocationRaw;
  done: boolean;
  hide?: boolean;
  requiredPermissions?: Permission[];
  needsProject?: boolean;
}

/**
 * React port of `frontend/src/components/Quickstart.vue`.
 *
 * Floating onboarding tracker pinned to the bottom of the workspace
 * shell. Renders a progress bar over a list of intro tasks (view an
 * issue, query data, visit project, visit environment / instance /
 * database, visit member). Each task is permission-gated so users only
 * see tasks they can perform; tasks that depend on the sample project
 * are filtered out when no sample project exists for this workspace.
 *
 * State sources (all Pinia, read via `useVueState`):
 *  - `actuatorStore.quickStartEnabled` — disabled in self-hosted /
 *    enterprise builds via the actuator config.
 *  - `uiStateStore.getIntroStateByKey(...)` — per-task done flags and
 *    the global `hidden` flag (toggled when the user dismisses).
 *
 * Async fetches (project / sample issue / sample worksheet) run on
 * mount and fall back to undefined when the sample data is missing
 * (e.g. an upgraded workspace that never bootstrapped the sample).
 */
export function Quickstart() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const projectStore = useProjectV1Store();
  const uiStateStore = useUIStateStore();
  const actuatorStore = useActuatorV1Store();
  const issueStore = useIssueV1Store();
  const worksheetStore = useWorkSheetStore();
  const projectIamPolicyStore = useProjectIamPolicyStore();

  const quickStartEnabled = useVueState(() => actuatorStore.quickStartEnabled);
  const hidden = useVueState(() => uiStateStore.getIntroStateByKey("hidden"));

  // Subscribe to each task's done flag so the line-through + progress
  // bar update live.
  const issueVisited = useVueState(() =>
    uiStateStore.getIntroStateByKey("issue.visit")
  );
  const dataQueried = useVueState(() =>
    uiStateStore.getIntroStateByKey("data.query")
  );
  const projectVisited = useVueState(() =>
    uiStateStore.getIntroStateByKey("project.visit")
  );
  const environmentVisited = useVueState(() =>
    uiStateStore.getIntroStateByKey("environment.visit")
  );
  const instanceVisited = useVueState(() =>
    uiStateStore.getIntroStateByKey("instance.visit")
  );
  const databaseVisited = useVueState(() =>
    uiStateStore.getIntroStateByKey("database.visit")
  );
  const memberVisited = useVueState(() =>
    uiStateStore.getIntroStateByKey("member.visit")
  );

  // ---- async sample fetches --------------------------------------------

  const [sampleProject, setSampleProject] = useState<Project | undefined>();
  const [sampleIssueExists, setSampleIssueExists] = useState(false);
  const [sampleSheetExists, setSampleSheetExists] = useState(false);

  useEffect(() => {
    if (!quickStartEnabled) {
      setSampleProject(undefined);
      return;
    }
    let cancelled = false;
    void (async () => {
      const project = await projectStore.getOrFetchProjectByName(
        `${projectNamePrefix}${SAMPLE_PROJECT_NAME}`,
        true /* silent */
      );
      if (cancelled) return;
      if (!isValidProjectName(project.name)) {
        setSampleProject(undefined);
        return;
      }
      await projectIamPolicyStore.getOrFetchProjectIamPolicy(project.name);
      if (cancelled) return;
      setSampleProject(project);
    })();
    return () => {
      cancelled = true;
    };
  }, [quickStartEnabled, projectStore, projectIamPolicyStore]);

  useEffect(() => {
    if (!sampleProject) {
      setSampleIssueExists(false);
      setSampleSheetExists(false);
      return;
    }
    let cancelled = false;
    void (async () => {
      const issue = await issueStore.fetchIssueByName(
        `${sampleProject.name}/issues/${SAMPLE_ISSUE_ID}`,
        true /* silent */
      );
      if (!cancelled) setSampleIssueExists(!!issue);
    })();
    void (async () => {
      const sheet = await worksheetStore.getOrFetchWorksheetByName(
        `${sampleProject.name}/sheets/${SAMPLE_SHEET_ID}`,
        true /* silent */
      );
      if (!cancelled) setSampleSheetExists(!!sheet);
    })();
    return () => {
      cancelled = true;
    };
  }, [sampleProject, issueStore, worksheetStore]);

  // ---- intro list (memoized + permission-filtered) ----------------------

  const introList = useMemo<IntroItem[]>(() => {
    const sampleProjectId = sampleProject
      ? extractProjectResourceName(sampleProject.name)
      : extractProjectResourceName(UNKNOWN_PROJECT_NAME);
    const all: IntroItem[] = [
      {
        name: t("quick-start.view-an-issue"),
        link: {
          name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
          params: { projectId: sampleProjectId, issueId: SAMPLE_ISSUE_ID },
        },
        done: issueVisited,
        hide: !sampleIssueExists,
        requiredPermissions: ["bb.issues.get", "bb.projects.get"],
        needsProject: true,
      },
      {
        name: t("quick-start.query-data"),
        link: {
          name: SQL_EDITOR_WORKSHEET_MODULE,
          params: {
            project: SAMPLE_PROJECT_NAME,
            sheet: SAMPLE_SHEET_ID,
          },
        },
        done: dataQueried,
        hide: !sampleSheetExists,
        requiredPermissions: [
          "bb.sql.select",
          "bb.projects.get",
          "bb.worksheets.get",
        ],
        needsProject: true,
      },
      {
        name: t("quick-start.visit-project"),
        link: { name: PROJECT_V1_ROUTE_DASHBOARD },
        done: projectVisited,
        requiredPermissions: ["bb.projects.list"],
      },
      {
        name: t("quick-start.visit-environment"),
        link: { name: ENVIRONMENT_V1_ROUTE_DASHBOARD },
        done: environmentVisited,
        requiredPermissions: ["bb.settings.getEnvironment"],
      },
      {
        name: t("quick-start.visit-instance"),
        link: { name: INSTANCE_ROUTE_DASHBOARD },
        done: instanceVisited,
        requiredPermissions: ["bb.instances.list"],
      },
      {
        name: t("quick-start.visit-database"),
        link: { name: DATABASE_ROUTE_DASHBOARD },
        done: databaseVisited,
        requiredPermissions: ["bb.databases.list"],
      },
      {
        name: t("quick-start.visit-member"),
        link: { name: WORKSPACE_ROUTE_USERS },
        done: memberVisited,
        requiredPermissions: ["bb.workspaces.getIamPolicy", "bb.users.list"],
      },
    ];
    return all.filter((item) => {
      if (item.hide) return false;
      const perms = item.requiredPermissions ?? [];
      if (item.needsProject) {
        if (!sampleProject) return false;
        return perms.every((p) => hasProjectPermissionV2(sampleProject, p));
      }
      return perms.every(hasWorkspacePermissionV2);
    });
  }, [
    t,
    sampleProject,
    sampleIssueExists,
    sampleSheetExists,
    issueVisited,
    dataQueried,
    projectVisited,
    environmentVisited,
    instanceVisited,
    databaseVisited,
    memberVisited,
  ]);

  const showQuickstart = quickStartEnabled && !hidden;
  const currentStep = useMemo(() => {
    let i = 0;
    while (i < introList.length && introList[i].done) i++;
    return i;
  }, [introList]);

  const isTaskActive = (index: number) => {
    for (let i = index - 1; i >= 0; i--) {
      if (!introList[i].done) return false;
    }
    return !introList[index].done;
  };

  const percent = useMemo(() => {
    const total = introList.length;
    if (total === 0) return "0%";
    if (currentStep === 0) return "3rem";
    if (currentStep === total - 1) return "calc(100% - 3rem)";
    const offset = 0.5;
    const unit = 100 / total;
    return `${Math.min((currentStep + offset) * unit, 100)}%`;
  }, [currentStep, introList.length]);

  const handleHide = (silent = false) => {
    void uiStateStore
      .saveIntroStateByKey({ key: "hidden", newState: true })
      .then(() => {
        if (!silent) {
          pushNotification({
            module: "bytebase",
            style: "INFO",
            title: t("quick-start.notice.title"),
            description: t("quick-start.notice.desc"),
            manualHide: true,
          });
        }
      });
  };

  if (!showQuickstart || introList.length === 0) {
    return null;
  }

  return (
    <div className="py-2 px-4 w-full shrink-0 border-t border-block-border hidden lg:block bg-yellow-50">
      <p className="text-sm font-medium text-gray-900 flex items-center justify-between">
        <span>
          🎈 {t("quick-start.self")} - {t("quick-start.guide")}
        </span>
        <button
          type="button"
          className="rounded-xs p-1 hover:bg-control-bg cursor-pointer"
          aria-label={t("common.close")}
          onClick={() => handleHide()}
        >
          <X className="w-4 h-4" />
        </button>
      </p>
      <div className="mt-2" aria-hidden="true">
        <div className="overflow-hidden rounded-full bg-gray-200">
          <div
            className="h-2 rounded-full bg-indigo-600"
            style={{ width: percent }}
          />
        </div>
        <div
          className="mt-2 hidden text-sm font-medium text-gray-600 sm:grid"
          style={{
            gridTemplateColumns: `repeat(${introList.length}, minmax(0, 1fr))`,
          }}
        >
          {introList.map((intro, index) => {
            const active = isTaskActive(index);
            return (
              <button
                key={`${intro.name}-${index}`}
                type="button"
                className={cn(
                  "group cursor-pointer flex items-center gap-x-1 text-sm font-medium",
                  index === 0 && "justify-start",
                  index > 0 && index < introList.length - 1 && "justify-center",
                  index === introList.length - 1 && "justify-end",
                  active
                    ? "text-indigo-600"
                    : "text-control-light group-hover:text-control-light-hover",
                  intro.done && "line-through"
                )}
                onClick={() => {
                  void navigate.push(intro.link);
                }}
              >
                <span className="relative h-5 w-5 inline-flex items-center justify-center">
                  {intro.done ? (
                    <CheckCircle className="w-4 h-auto text-success group-hover:text-success-hover" />
                  ) : active ? (
                    <span className="relative flex h-3 w-3">
                      <span
                        className="absolute inline-flex h-full w-full rounded-full bg-indigo-400 opacity-75"
                        style={{
                          animation:
                            "ping 2s cubic-bezier(0, 0, 0.2, 1) infinite",
                        }}
                      />
                      <span className="relative inline-flex rounded-full h-3 w-3 bg-indigo-500" />
                    </span>
                  ) : (
                    <span className="h-2 w-2 bg-gray-300 rounded-full group-hover:bg-gray-400" />
                  )}
                </span>
                <span className="inline-flex">{intro.name}</span>
              </button>
            );
          })}
        </div>
      </div>
    </div>
  );
}
