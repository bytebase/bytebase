import { ChevronLeft, Plus } from "lucide-react";
import {
  type MouseEvent as ReactMouseEvent,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { ProjectTable } from "@/react/components/ProjectTable";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/react/components/ui/select";
import {
  Tabs,
  TabsList,
  TabsPanel,
  TabsTrigger,
} from "@/react/components/ui/tabs";
import {
  projectMatchesKeyword,
  useProject,
  useProjectList,
  useRecentProjects,
  useRecentVisit,
  useWorkspacePermission,
} from "@/react/hooks/useAppState";
import {
  getProjectName,
  isValidProjectName,
  projectNamePrefix,
} from "@/react/lib/resourceName";
import {
  PROJECT_V1_ROUTE_DETAIL,
  useCurrentRoute,
  useNavigate,
  WORKSPACE_ROUTE_LANDING,
} from "@/react/router";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

export interface ProjectSwitchPanelProps {
  onClose: () => void;
  onRequestCreate: () => void;
}

type ProjectSwitchTab = "recent" | "all";

function ProjectSwitchFooter({
  pageSize,
  pageSizeOptions,
  hasMore,
  isFetchingMore,
  onPageSizeChange,
  onLoadMore,
}: {
  pageSize: number;
  pageSizeOptions: number[];
  hasMore: boolean;
  isFetchingMore: boolean;
  onPageSizeChange: (size: number) => void;
  onLoadMore: () => void;
}) {
  const { t } = useTranslation();
  return (
    <div className="flex items-center justify-end gap-x-3 text-sm text-control-light">
      <span className="whitespace-nowrap">{t("common.rows-per-page")}</span>
      <Select
        value={String(pageSize)}
        onValueChange={(value) => onPageSizeChange(Number(value))}
      >
        <SelectTrigger size="sm" className="w-20 text-control">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {pageSizeOptions.map((option) => (
            <SelectItem key={option} value={String(option)}>
              {option}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      {hasMore ? (
        <Button
          size="sm"
          variant="outline"
          disabled={isFetchingMore}
          onClick={onLoadMore}
        >
          {isFetchingMore ? t("common.loading") : t("common.load-more")}
        </Button>
      ) : null}
    </div>
  );
}

export function ProjectSwitchPanel({
  onClose,
  onRequestCreate,
}: ProjectSwitchPanelProps) {
  const { t } = useTranslation();
  const { record } = useRecentVisit();
  const navigate = useNavigate();
  const route = useCurrentRoute();
  const { projects: recentProjectList } = useRecentProjects();
  const [searchText, setSearchText] = useState("");
  const [selectedTab, setSelectedTab] = useState<ProjectSwitchTab>(() =>
    recentProjectList.length > 0 ? "recent" : "all"
  );
  // The recent list loads async (localStorage read in useEffect), so the
  // initializer above may see an empty list. Switch to "recent" once loaded.
  useEffect(() => {
    if (recentProjectList.length > 0) {
      setSelectedTab((prev) => (prev === "all" ? "recent" : prev));
    }
  }, [recentProjectList.length]);
  const projectId = route.params.projectId as string | undefined;
  const currentProjectName = projectId
    ? `${projectNamePrefix}${projectId}`
    : "";
  const currentProject = useProject(currentProjectName);
  const allowToCreateProject = useWorkspacePermission("bb.projects.create");

  const filteredRecentProjectList = useMemo(() => {
    return recentProjectList.filter((project) =>
      projectMatchesKeyword(project, searchText)
    );
  }, [recentProjectList, searchText]);

  // Mirrors Vue's `actualSelectedTab` — force the All view when a search
  // keyword is active but yields no recent matches, regardless of which
  // tab the user explicitly clicked. Keeping the trigger highlight in
  // sync is the consumer's responsibility (we pass `actualSelectedTab`
  // to `<Tabs value=…>` so both indicator and panel agree).
  const actualSelectedTab = useMemo<ProjectSwitchTab>(() => {
    if (
      searchText.trim().length > 0 &&
      filteredRecentProjectList.length === 0
    ) {
      return "all";
    }
    return selectedTab;
  }, [filteredRecentProjectList.length, searchText, selectedTab]);

  const {
    projects: allProjects,
    isLoading,
    isFetchingMore,
    hasMore,
    loadMore,
    pageSize,
    pageSizeOptions,
    onPageSizeChange,
  } = useProjectList(searchText);

  const handleProjectSelect = useCallback(
    (project: Project, event: ReactMouseEvent<HTMLElement>) => {
      const route = navigate.resolve({
        name: PROJECT_V1_ROUTE_DETAIL,
        params: {
          projectId: getProjectName(project.name),
        },
      });
      record(route.fullPath);

      if (event.ctrlKey || event.metaKey) {
        window.open(route.fullPath, "_blank");
      } else {
        void navigate.push({
          name: PROJECT_V1_ROUTE_DETAIL,
          params: {
            projectId: getProjectName(project.name),
          },
        });
      }

      onClose();
    },
    [navigate, onClose, record]
  );

  const handleGotoWorkspace = useCallback(
    (event: ReactMouseEvent<HTMLButtonElement>) => {
      const route = navigate.resolve({
        name: WORKSPACE_ROUTE_LANDING,
      });
      record(route.fullPath);

      if (event.ctrlKey || event.metaKey) {
        window.open(route.fullPath, "_blank");
      } else {
        void navigate.push({ name: WORKSPACE_ROUTE_LANDING });
      }

      onClose();
    },
    [navigate, onClose, record]
  );

  return (
    <div className="flex w-full max-h-[calc(100vh-10rem)] flex-col">
      {isValidProjectName(currentProject?.name) ? (
        <Button
          variant="ghost"
          size="sm"
          className="mb-2 mx-3 w-fit px-0 text-control-light hover:bg-transparent hover:text-control"
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
        <div className="mb-2 mx-3 flex items-center justify-between gap-x-3">
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
                onClick={onRequestCreate}
              >
                <Plus className="h-4 w-4" />
              </Button>
            ) : null}
          </div>
        </div>

        <TabsPanel value="recent" className="mt-0">
          <div className="max-h-[26rem] overflow-auto">
            <ProjectTable
              projectList={filteredRecentProjectList}
              currentProject={currentProject}
              keyword={searchText}
              showLabels={false}
              onRowClick={handleProjectSelect}
            />
          </div>
        </TabsPanel>

        <TabsPanel value="all" className="mt-0">
          <div className="max-h-[26rem] overflow-auto">
            <ProjectTable
              projectList={allProjects}
              currentProject={currentProject}
              keyword={searchText}
              loading={isLoading}
              showLabels={false}
              onRowClick={handleProjectSelect}
            />
          </div>

          <div className="mt-2 mx-3">
            <ProjectSwitchFooter
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
