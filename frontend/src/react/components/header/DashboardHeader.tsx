import {
  Bot,
  CircleDot,
  Menu,
  MessagesSquare,
  SquareTerminal,
} from "lucide-react";
import { useMemo, useSyncExternalStore } from "react";
import { useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import { BytebaseLogo } from "@/react/components/BytebaseLogo";
import { Button } from "@/react/components/ui/button";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import {
  WORKSPACE_ROUTE_LANDING,
  WORKSPACE_ROUTE_MY_ISSUES,
} from "@/router/dashboard/workspaceRoutes";
import {
  SQL_EDITOR_DATABASE_MODULE,
  SQL_EDITOR_HOME_MODULE,
  SQL_EDITOR_PROJECT_MODULE,
} from "@/router/sqlEditor";
import { useRecentVisit } from "@/router/useRecentVisit";
import { useSubscriptionV1Store } from "@/store";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";
import { STORAGE_KEY_MY_ISSUES_TAB } from "@/utils";
import { ProfileMenuTrigger } from "./ProfileMenuTrigger";
import { ProjectSwitchPopover } from "./ProjectSwitchPopover";

export interface DashboardHeaderProps {
  showLogo: boolean;
  showMobileSidebarToggle?: boolean;
  onOpenMobileSidebar?: () => void;
}

function subscribeToViewport(onStoreChange: () => void) {
  window.addEventListener("resize", onStoreChange);
  return () => window.removeEventListener("resize", onStoreChange);
}

function getWindowWidth() {
  return window.innerWidth;
}

export function DashboardHeader({
  showLogo,
  showMobileSidebarToggle = false,
  onOpenMobileSidebar,
}: DashboardHeaderProps) {
  const { t } = useTranslation();
  const { record } = useRecentVisit();
  const subscriptionStore = useSubscriptionV1Store();

  const currentPlan = useVueState(() => subscriptionStore.currentPlan);
  const route = useVueState(() => router.currentRoute.value);
  const windowWidth = useSyncExternalStore(
    subscribeToViewport,
    getWindowWidth,
    () => 1024
  );
  const isDesktopLabelVisible = windowWidth >= 640;
  const isLargeLabelVisible = windowWidth >= 1024;

  const sqlEditorHref = useMemo(() => {
    const projectId = route.params.projectId as string | undefined;
    const instanceId = route.params.instanceId as string | undefined;
    const databaseName = route.params.databaseName as string | undefined;

    if (projectId) {
      if (instanceId && databaseName) {
        return router.resolve({
          name: SQL_EDITOR_DATABASE_MODULE,
          params: {
            project: projectId,
            instance: instanceId,
            database: databaseName,
          },
        }).href;
      }
      return router.resolve({
        name: SQL_EDITOR_PROJECT_MODULE,
        params: {
          project: projectId,
        },
      }).href;
    }

    return router.resolve({
      name: SQL_EDITOR_HOME_MODULE,
    }).href;
  }, [
    route.params.databaseName,
    route.params.instanceId,
    route.params.projectId,
  ]);

  const myIssueHref = useMemo(
    () =>
      router.resolve({
        name: WORKSPACE_ROUTE_MY_ISSUES,
      }).href,
    []
  );

  return (
    <div className="my-1 flex h-10 items-center justify-between gap-x-1 px-3 sm:gap-x-3 sm:px-4">
      {showLogo ? (
        <BytebaseLogo className="h-10" redirect={WORKSPACE_ROUTE_LANDING} />
      ) : null}

      {showMobileSidebarToggle ? (
        <button
          type="button"
          className="p-1 text-gray-500 hover:text-gray-900 md:hidden"
          onClick={onOpenMobileSidebar}
        >
          <Menu className="h-4 w-4" />
        </button>
      ) : null}

      <ProjectSwitchPopover />

      <div className="flex flex-1 items-center justify-end gap-x-2">
        <Button
          size="sm"
          variant="outline"
          onClick={() => {
            void import("@/react/plugins/agent/store/agent").then(
              ({ useAgentStore }) => {
                useAgentStore.getState().toggle();
              }
            );
          }}
        >
          <Bot className="h-4 w-4" />
          {isDesktopLabelVisible ? <span>{t("agent.self")}</span> : null}
        </Button>

        {currentPlan === PlanType.FREE ? (
          <Button
            size="sm"
            variant="outline"
            className="border-success text-success hover:bg-success/10"
            onClick={() => {
              window.open(
                "https://docs.bytebase.com/faq#how-to-reach-us",
                "_blank"
              );
            }}
          >
            <MessagesSquare className="h-4 w-4" />
            {isLargeLabelVisible ? <span>{t("common.want-help")}</span> : null}
          </Button>
        ) : null}

        <Button
          size="sm"
          variant="outline"
          onClick={() => {
            window.open(sqlEditorHref, "_blank", "noopener,noreferrer");
          }}
        >
          <SquareTerminal className="h-4 w-4" />
          {isDesktopLabelVisible ? (
            <span className="whitespace-nowrap">{t("sql-editor.self")}</span>
          ) : null}
        </Button>

        <Button
          size="sm"
          variant="outline"
          onClick={() => {
            record(myIssueHref);
            localStorage.setItem(STORAGE_KEY_MY_ISSUES_TAB, uuidv4());
            void router.push({ name: WORKSPACE_ROUTE_MY_ISSUES });
          }}
        >
          <CircleDot className="h-4 w-4" />
          {isDesktopLabelVisible ? <span>{t("issue.my-issues")}</span> : null}
        </Button>

        <ProfileMenuTrigger size="medium" link />
      </div>
    </div>
  );
}
