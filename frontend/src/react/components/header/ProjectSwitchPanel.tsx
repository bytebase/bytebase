import { Check, ChevronLeft, Plus } from "lucide-react";
import {
  type MouseEvent as ReactMouseEvent,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { useRecentProjects } from "@/components/Project/useRecentProjects";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import {
  Tabs,
  TabsList,
  TabsPanel,
  TabsTrigger,
} from "@/react/components/ui/tabs";
import { PagedTableFooter, usePagedData } from "@/react/hooks/usePagedData";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { PROJECT_V1_ROUTE_DETAIL } from "@/router/dashboard/projectV1";
import { WORKSPACE_ROUTE_LANDING } from "@/router/dashboard/workspaceRoutes";
import { useRecentVisit } from "@/router/useRecentVisit";
import { useProjectV1Store } from "@/store";
import { getProjectName, projectNamePrefix } from "@/store/modules/v1/common";
import { isDefaultProject, isValidProjectName } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  filterProjectV1ListByKeyword,
  hasWorkspacePermissionV2,
} from "@/utils";

export interface ProjectSwitchPanelProps {
  onClose: () => void;
  onRequestCreate: () => void;
}

type ProjectSwitchTab = "recent" | "all";

function ProjectRow({
  project,
  currentProjectName,
  keyword,
  onSelect,
}: {
  project: Project;
  currentProjectName?: string;
  keyword: string;
  onSelect: (
    project: Project,
    event: ReactMouseEvent<HTMLButtonElement>
  ) => void;
}) {
  const resourceId = getProjectName(project.name);
  const labels = Object.entries(project.labels ?? {}).slice(0, 3);
  const isCurrent = project.name === currentProjectName;

  return (
    <button
      type="button"
      className="flex w-full items-start gap-x-2.5 rounded-sm px-2 py-1.5 text-left hover:bg-control-bg"
      onClick={(event) => onSelect(project, event)}
    >
      <span className="mt-0.5 h-4 w-4 shrink-0">
        {isCurrent ? <Check className="h-4 w-4 text-accent" /> : null}
      </span>
      <span className="min-w-0 flex-1">
        <span className="block truncate font-medium text-control">
          {project.title || resourceId}
        </span>
        <span className="block truncate text-xs text-control-light">
          {resourceId}
        </span>
        {labels.length > 0 ? (
          <span className="mt-1 flex flex-wrap gap-1">
            {labels.map(([key, value]) => (
              <span
                key={`${key}:${value}`}
                className="rounded-full bg-control-bg px-2 py-0.5 text-[11px] text-control-light"
              >
                {key}:{value}
              </span>
            ))}
          </span>
        ) : null}
        {keyword.trim().length > 0 &&
        !project.title.toLowerCase().includes(keyword.toLowerCase()) &&
        !resourceId.toLowerCase().includes(keyword.toLowerCase()) ? (
          <span className="sr-only">{keyword}</span>
        ) : null}
      </span>
    </button>
  );
}

export function ProjectSwitchPanel({
  onClose,
  onRequestCreate,
}: ProjectSwitchPanelProps) {
  const { t } = useTranslation();
  const projectStore = useProjectV1Store();
  const { record } = useRecentVisit();
  const { recentViewProjects } = useRecentProjects();
  const [searchText, setSearchText] = useState("");
  const [selectedTab, setSelectedTab] = useState<ProjectSwitchTab>("all");
  const currentProjectName = useVueState(() => {
    const projectId = router.currentRoute.value.params.projectId as
      | string
      | undefined;
    return projectId ? `${projectNamePrefix}${projectId}` : "";
  });
  const currentProject = useVueState(() =>
    projectStore.getProjectByName(currentProjectName)
  );
  const recentProjectList = useVueState(() => recentViewProjects.value ?? []);
  const allowToCreateProject = hasWorkspacePermissionV2("bb.projects.create");

  useEffect(() => {
    if (recentProjectList.length > 0) {
      setSelectedTab((previous) => (previous === "all" ? "recent" : previous));
    }
  }, [recentProjectList.length]);

  const filteredRecentProjectList = useMemo(() => {
    return filterProjectV1ListByKeyword(
      recentProjectList.filter((project) => !isDefaultProject(project.name)),
      searchText
    );
  }, [recentProjectList, searchText]);

  const actualSelectedTab = useMemo<ProjectSwitchTab>(() => {
    if (
      selectedTab === "recent" &&
      searchText.trim().length > 0 &&
      filteredRecentProjectList.length === 0
    ) {
      return "all";
    }
    return selectedTab;
  }, [filteredRecentProjectList.length, searchText, selectedTab]);

  const {
    dataList: allProjects,
    isLoading,
    isFetchingMore,
    hasMore,
    loadMore,
    pageSize,
    pageSizeOptions,
    onPageSizeChange,
  } = usePagedData<Project>({
    sessionKey: "bb.project-switch",
    fetchList: useCallback(
      async ({ pageSize, pageToken }) => {
        const { projects, nextPageToken } = await projectStore.fetchProjectList(
          {
            pageSize,
            pageToken,
            filter: {
              query: searchText,
              excludeDefault: true,
              state: State.ACTIVE,
            },
            orderBy: "title",
            cache: true,
          }
        );
        return {
          list: projects,
          nextPageToken,
        };
      },
      [projectStore, searchText]
    ),
  });

  const handleProjectSelect = useCallback(
    (project: Project, event: ReactMouseEvent<HTMLButtonElement>) => {
      const route = router.resolve({
        name: PROJECT_V1_ROUTE_DETAIL,
        params: {
          projectId: getProjectName(project.name),
        },
      });
      record(route.fullPath);

      if (event.ctrlKey || event.metaKey) {
        window.open(route.fullPath, "_blank");
      } else {
        void router.push(route);
      }

      onClose();
    },
    [onClose, record]
  );

  const handleGotoWorkspace = useCallback(
    (event: ReactMouseEvent<HTMLButtonElement>) => {
      const route = router.resolve({
        name: WORKSPACE_ROUTE_LANDING,
      });
      record(route.fullPath);

      if (event.ctrlKey || event.metaKey) {
        window.open(route.fullPath, "_blank");
      } else {
        void router.push(route.fullPath);
      }

      onClose();
    },
    [onClose, record]
  );

  return (
    <div className="flex w-full max-h-[calc(100vh-10rem)] flex-col">
      {isValidProjectName(currentProject.name) ? (
        <Button
          variant="ghost"
          size="sm"
          className="mb-2 h-7 w-fit px-0 text-control-light hover:bg-transparent hover:text-control"
          onClick={handleGotoWorkspace}
        >
          <ChevronLeft className="h-4 w-4 opacity-80" />
          {t("common.back-to-workspace")}
        </Button>
      ) : null}

      <Tabs
        value={actualSelectedTab}
        onValueChange={(value) => setSelectedTab(value as ProjectSwitchTab)}
      >
        <div className="mb-2 flex items-center justify-between gap-x-3">
          <TabsList className="gap-x-5">
            <TabsTrigger className="pb-1.5" value="recent">
              {t("common.recent")}
            </TabsTrigger>
            <TabsTrigger className="pb-1.5" value="all">
              {t("common.all")}
            </TabsTrigger>
          </TabsList>

          <div className="flex items-center gap-x-1.5">
            <Input
              size="sm"
              className="w-36"
              value={searchText}
              placeholder={t("common.filter-by-name")}
              onChange={(event) => setSearchText(event.target.value)}
            />
            {allowToCreateProject ? (
              <Button
                size="sm"
                variant="outline"
                aria-label={t("quick-action.new-project")}
                className="h-8 w-8 px-0"
                onClick={onRequestCreate}
              >
                <Plus className="h-4 w-4" />
              </Button>
            ) : null}
          </div>
        </div>

        <TabsPanel value="recent" className="mt-0">
          <div className="flex max-h-[26rem] flex-col overflow-auto">
            {filteredRecentProjectList.length === 0 ? (
              <div className="px-3 py-8 text-center text-sm text-control-light">
                {searchText.trim().length > 0
                  ? t("common.no-data")
                  : t("common.no-data")}
              </div>
            ) : (
              filteredRecentProjectList.map((project) => (
                <ProjectRow
                  key={project.name}
                  project={project}
                  currentProjectName={currentProjectName}
                  keyword={searchText}
                  onSelect={handleProjectSelect}
                />
              ))
            )}
          </div>
        </TabsPanel>

        <TabsPanel value="all" className="mt-0">
          <div className="flex max-h-[26rem] flex-col overflow-auto">
            {isLoading && allProjects.length === 0 ? (
              <div className="px-3 py-8 text-center text-sm text-control-light">
                {t("common.loading")}
              </div>
            ) : allProjects.length === 0 ? (
              <div className="px-3 py-8 text-center text-sm text-control-light">
                {t("common.no-data")}
              </div>
            ) : (
              allProjects.map((project) => (
                <ProjectRow
                  key={project.name}
                  project={project}
                  currentProjectName={currentProjectName}
                  keyword={searchText}
                  onSelect={handleProjectSelect}
                />
              ))
            )}
          </div>

          <div className="mt-2 border-t border-control-border pt-2">
            <PagedTableFooter
              pageSize={pageSize}
              pageSizeOptions={pageSizeOptions}
              onPageSizeChange={onPageSizeChange}
              hasMore={hasMore}
              isFetchingMore={isFetchingMore}
              onLoadMore={loadMore}
            />
          </div>
        </TabsPanel>
      </Tabs>
    </div>
  );
}
