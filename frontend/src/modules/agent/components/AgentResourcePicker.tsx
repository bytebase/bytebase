import { Popover as BasePopover } from "@base-ui/react/popover";
import {
  ArrowLeft,
  ChevronRight,
  Database as DatabaseIcon,
  FolderKanban,
  LoaderCircle,
} from "lucide-react";
import {
  type KeyboardEvent,
  type RefObject,
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { DatabaseTargetDisplay } from "@/components/DatabaseTargetDisplay";
import { Button } from "@/components/ui/button";
import { getLayerRoot, LAYER_SURFACE_CLASS } from "@/components/ui/layer";
import { SearchInput } from "@/components/ui/search-input";
import { useAppStore } from "@/stores/app";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { extractDatabaseResourceName } from "@/utils/v1/database";
import { normalizeAgentResourceReference } from "../logic/resourceMention";
import {
  searchAgentDatabases,
  searchAgentProjects,
} from "../logic/resourceSearch";

interface AgentResourcePickerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  anchorRef: RefObject<HTMLTextAreaElement | null>;
  workspaceName: string;
  projectName?: string;
  selectedProjectName?: string;
  hasSelectedReference: boolean;
  onSelectProject: (project: Project) => void;
  onSelectDatabase: (database: Database) => void;
}

type PickerView = "categories" | "projects" | "database-projects" | "databases";
type SearchStatus =
  | "idle"
  | "loading"
  | "loaded"
  | "loading-more"
  | "error"
  | "more-error";

const PICKER_VIEWPORT_PADDING = 8;

const isSelectableDatabase = (
  database: Database,
  projectName?: string
): boolean =>
  (!projectName || database.project === projectName) &&
  Boolean(
    normalizeAgentResourceReference({
      kind: "database",
      name: database.name,
      project: database.project,
      label: extractDatabaseResourceName(database.name).databaseName,
    })
  );

export function AgentResourcePicker({
  open,
  onOpenChange,
  anchorRef,
  workspaceName,
  projectName,
  selectedProjectName,
  hasSelectedReference,
  onSelectProject,
  onSelectDatabase,
}: AgentResourcePickerProps) {
  const { t } = useTranslation();
  const canListWorkspaceDatabases = useAppStore((state) =>
    state.hasWorkspacePermission("bb.databases.list")
  );
  const [view, setView] = useState<PickerView>("categories");
  const [query, setQuery] = useState("");
  const [projects, setProjects] = useState<Project[]>([]);
  const [databases, setDatabases] = useState<Database[]>([]);
  const [chosenProject, setChosenProject] = useState<{
    name: string;
    workspaceName: string;
  }>();
  const [nextPageToken, setNextPageToken] = useState("");
  const [status, setStatus] = useState<SearchStatus>("idle");
  const requestGeneration = useRef(0);
  const requestController = useRef<AbortController | null>(null);
  const previousWorkspaceName = useRef(workspaceName);
  const categoryRef = useRef<HTMLButtonElement>(null);
  const databaseCategoryRef = useRef<HTMLButtonElement>(null);
  const popupRef = useRef<HTMLDivElement>(null);
  const categoryTopOffsetRef = useRef<number | null>(null);
  const noticeRef = useRef<HTMLButtonElement>(null);
  const searchRef = useRef<HTMLInputElement>(null);
  const resultRefs = useRef<Array<HTMLButtonElement | null>>([]);
  const databaseProjectName =
    selectedProjectName ??
    projectName ??
    (chosenProject?.workspaceName === workspaceName
      ? chosenProject.name
      : undefined);
  const databaseParent =
    databaseProjectName ??
    (canListWorkspaceDatabases ? workspaceName : undefined);
  const isDatabaseView = view === "databases";
  const results = isDatabaseView ? databases : projects;

  const loadPage = useCallback(
    async (pageToken: string, append: boolean) => {
      requestController.current?.abort();
      const controller = new AbortController();
      requestController.current = controller;
      const generation = ++requestGeneration.current;
      if (append) {
        setStatus("loading-more");
      } else {
        setProjects([]);
        setDatabases([]);
        setNextPageToken("");
        setStatus("loading");
      }

      try {
        const page =
          view === "databases" && databaseParent
            ? {
                kind: "databases" as const,
                result: await searchAgentDatabases(
                  databaseParent,
                  query,
                  pageToken,
                  controller.signal
                ),
              }
            : {
                kind: "projects" as const,
                result: await searchAgentProjects(
                  query,
                  pageToken,
                  controller.signal,
                  view === "database-projects"
                ),
              };
        if (
          controller.signal.aborted ||
          generation !== requestGeneration.current
        )
          return;
        if (page.kind === "databases") {
          const items = page.result.items.filter((database) =>
            isSelectableDatabase(database, databaseProjectName)
          );
          setDatabases((previous) =>
            append ? [...previous, ...items] : items
          );
        } else {
          setProjects((previous) =>
            append ? [...previous, ...page.result.items] : page.result.items
          );
        }
        setNextPageToken(page.result.nextPageToken);
        setStatus("loaded");
      } catch {
        if (
          controller.signal.aborted ||
          generation !== requestGeneration.current
        )
          return;
        setStatus(append ? "more-error" : "error");
      }
    },
    [databaseParent, databaseProjectName, query, view]
  );

  useLayoutEffect(() => {
    if (previousWorkspaceName.current === workspaceName) return;
    previousWorkspaceName.current = workspaceName;
    if (selectedProjectName || projectName || view !== "databases") return;
    requestController.current?.abort();
    requestGeneration.current += 1;
    setChosenProject(undefined);
    setView(canListWorkspaceDatabases ? "databases" : "database-projects");
    setQuery("");
    setDatabases([]);
    setNextPageToken("");
    setStatus("idle");
  }, [
    canListWorkspaceDatabases,
    projectName,
    selectedProjectName,
    view,
    workspaceName,
  ]);

  useLayoutEffect(() => {
    if (
      view === "databases" &&
      !selectedProjectName &&
      !projectName &&
      !chosenProject &&
      !canListWorkspaceDatabases
    ) {
      setView("database-projects");
      setQuery("");
    }
  }, [
    canListWorkspaceDatabases,
    chosenProject,
    projectName,
    selectedProjectName,
    view,
  ]);

  useEffect(() => {
    if (!open || view === "categories" || hasSelectedReference) return;
    if (view === "databases" && !databaseParent) return;
    void loadPage("", false);
    return () => {
      requestController.current?.abort();
      requestGeneration.current += 1;
    };
  }, [
    databaseParent,
    hasSelectedReference,
    loadPage,
    open,
    view,
    workspaceName,
  ]);

  useEffect(() => {
    if (open) return;
    setView("categories");
    setQuery("");
    setProjects([]);
    setDatabases([]);
    setChosenProject(undefined);
    setNextPageToken("");
    setStatus("idle");
  }, [open]);

  useLayoutEffect(() => {
    if (!open) return;
    if (hasSelectedReference) noticeRef.current?.focus();
    else if (view !== "categories") searchRef.current?.focus();
    else if (selectedProjectName) databaseCategoryRef.current?.focus();
    else categoryRef.current?.focus();
  }, [hasSelectedReference, open, selectedProjectName, view]);

  const selectProject = (project: Project) => {
    if (hasSelectedReference) return;
    if (view === "database-projects") {
      setChosenProject({ name: project.name, workspaceName });
      setQuery("");
      setView("databases");
      return;
    }
    onSelectProject(project);
    onOpenChange(false);
  };

  const selectDatabase = (database: Database) => {
    if (
      hasSelectedReference ||
      !isSelectableDatabase(database, databaseProjectName)
    ) {
      return;
    }
    onSelectDatabase(database);
    onOpenChange(false);
  };

  const showView = (nextView: PickerView) => {
    const anchorTop = anchorRef.current?.getBoundingClientRect().top;
    const popupTop = popupRef.current?.getBoundingClientRect().top;
    if (anchorTop !== undefined && popupTop !== undefined) {
      categoryTopOffsetRef.current = anchorTop - popupTop;
    }
    setView(nextView);
  };

  const goBack = () => {
    if (
      view === "databases" &&
      !projectName &&
      chosenProject?.workspaceName === workspaceName
    ) {
      setChosenProject(undefined);
      setView("database-projects");
    } else {
      setChosenProject(undefined);
      setView("categories");
    }
    setQuery("");
  };

  const handleSearchKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.nativeEvent.isComposing) return;
    if (event.key === "ArrowDown" && results.length > 0) {
      event.preventDefault();
      resultRefs.current[0]?.focus();
    } else if (event.key === "Enter" && results.length > 0) {
      event.preventDefault();
      if (isDatabaseView) selectDatabase(databases[0]);
      else selectProject(projects[0]);
    }
  };

  const handleResultKeyDown = (
    event: KeyboardEvent<HTMLButtonElement>,
    index: number
  ) => {
    if (event.nativeEvent.isComposing) return;
    if (event.key === "ArrowDown" && index < results.length - 1) {
      event.preventDefault();
      resultRefs.current[index + 1]?.focus();
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      if (index === 0) searchRef.current?.focus();
      else resultRefs.current[index - 1]?.focus();
    } else if (event.key === "Enter") {
      event.preventDefault();
      if (isDatabaseView) selectDatabase(databases[index]);
      else selectProject(projects[index]);
    }
  };

  return (
    <BasePopover.Root open={open} onOpenChange={onOpenChange}>
      <BasePopover.Portal container={getLayerRoot("agent")}>
        <BasePopover.Positioner
          anchor={anchorRef}
          side="top"
          align="start"
          sideOffset={({ positioner }) => {
            if (
              view === "categories" ||
              categoryTopOffsetRef.current === null
            ) {
              return 6;
            }
            const anchorTop = anchorRef.current?.getBoundingClientRect().top;
            if (anchorTop === undefined) return 6;
            // Keep the expanded view at the category menu's top edge when it fits.
            const preferredTop = anchorTop - categoryTopOffsetRef.current;
            const top = Math.max(
              PICKER_VIEWPORT_PADDING,
              Math.min(
                preferredTop,
                window.innerHeight - positioner.height - PICKER_VIEWPORT_PADDING
              )
            );
            return anchorTop - top - positioner.height;
          }}
          collisionPadding={PICKER_VIEWPORT_PADDING}
          collisionAvoidance={{
            side: "shift",
            align: "shift",
            fallbackAxisSide: "none",
          }}
          className={LAYER_SURFACE_CLASS}
        >
          <BasePopover.Popup
            ref={popupRef}
            data-agent-resource-picker
            aria-label={t("agent.resource-picker.add-context")}
            initialFocus={
              hasSelectedReference
                ? noticeRef
                : view === "categories"
                  ? selectedProjectName
                    ? databaseCategoryRef
                    : categoryRef
                  : searchRef
            }
            finalFocus={anchorRef}
            onKeyDown={(event) => {
              if (event.key === "Escape") {
                event.preventDefault();
                onOpenChange(false);
              }
            }}
            className="w-80 max-w-[calc(100vw-2rem)] rounded-sm border border-control-border bg-background p-2 shadow-md focus:outline-hidden"
          >
            {hasSelectedReference ? (
              <div className="flex flex-col gap-3 px-2 py-1">
                <p role="status" className="text-sm text-control-light">
                  {t("agent.resource-picker.reference-limit")}
                </p>
                <Button
                  ref={noticeRef}
                  appearance="secondary"
                  size="sm"
                  className="self-end"
                  onClick={() => onOpenChange(false)}
                >
                  {t("common.close")}
                </Button>
              </div>
            ) : view === "categories" ? (
              <div className="flex flex-col gap-1">
                {!selectedProjectName && (
                  <Button
                    ref={categoryRef}
                    appearance="secondary"
                    size="sm"
                    className="h-auto w-full justify-start gap-2 px-2 py-2 text-left"
                    data-agent-resource-category="project"
                    onKeyDown={(event) => {
                      if (event.nativeEvent.isComposing) return;
                      if (event.key === "ArrowDown") {
                        event.preventDefault();
                        databaseCategoryRef.current?.focus();
                      } else if (event.key === "Enter") {
                        event.preventDefault();
                        showView("projects");
                      }
                    }}
                    onClick={() => showView("projects")}
                  >
                    <FolderKanban className="size-4 shrink-0" />
                    <span className="min-w-0 flex-1 truncate">
                      {t("agent.resource-picker.projects")}
                    </span>
                    <ChevronRight className="size-4 shrink-0" />
                  </Button>
                )}
                <Button
                  ref={databaseCategoryRef}
                  appearance="secondary"
                  size="sm"
                  className="h-auto w-full justify-start gap-2 px-2 py-2 text-left"
                  data-agent-resource-category="database"
                  onKeyDown={(event) => {
                    if (event.nativeEvent.isComposing) return;
                    if (event.key === "ArrowUp" && !selectedProjectName) {
                      event.preventDefault();
                      categoryRef.current?.focus();
                    } else if (event.key === "Enter") {
                      event.preventDefault();
                      showView(
                        databaseProjectName || canListWorkspaceDatabases
                          ? "databases"
                          : "database-projects"
                      );
                    }
                  }}
                  onClick={() =>
                    showView(
                      databaseProjectName || canListWorkspaceDatabases
                        ? "databases"
                        : "database-projects"
                    )
                  }
                >
                  <DatabaseIcon className="size-4 shrink-0" />
                  <span className="min-w-0 flex-1 truncate">
                    {t("agent.resource-picker.databases")}
                  </span>
                  <ChevronRight className="size-4 shrink-0" />
                </Button>
              </div>
            ) : (
              <div className="flex max-h-[min(20rem,calc(100dvh-2rem))] flex-col gap-2">
                <Button
                  appearance="secondary"
                  size="sm"
                  className="h-auto w-fit gap-1 px-2 py-1 text-control-light"
                  onClick={goBack}
                >
                  <ArrowLeft className="size-4" />
                  {t("common.back")}
                </Button>
                {view === "database-projects" && (
                  <p className="px-2 text-xs font-medium text-control-light">
                    {t("agent.resource-picker.choose-project-for-databases")}
                  </p>
                )}
                <SearchInput
                  ref={searchRef}
                  size="sm"
                  value={query}
                  onChange={(event) => {
                    const nextQuery = event.target.value;
                    if (nextQuery === query) {
                      void loadPage("", false);
                    } else {
                      setQuery(nextQuery);
                    }
                  }}
                  onChangeCapture={() => {
                    requestController.current?.abort();
                    requestGeneration.current += 1;
                    setProjects([]);
                    setDatabases([]);
                    setNextPageToken("");
                    setStatus("loading");
                  }}
                  onKeyDown={handleSearchKeyDown}
                  placeholder={
                    isDatabaseView
                      ? t("agent.resource-picker.search-databases")
                      : t("agent.resource-picker.search-projects")
                  }
                  aria-label={
                    isDatabaseView
                      ? t("agent.resource-picker.search-databases")
                      : t("agent.resource-picker.search-projects")
                  }
                />
                <div
                  className="min-h-0 overflow-y-auto"
                  data-agent-resource-results
                >
                  {!isDatabaseView &&
                    projects.map((project, index) => (
                      <Button
                        key={project.name}
                        ref={(element) => {
                          resultRefs.current[index] = element;
                        }}
                        appearance="secondary"
                        size="sm"
                        className="h-auto w-full justify-start gap-2 px-2 py-2 text-left focus-visible:bg-control-bg focus-visible:ring-inset focus-visible:ring-offset-0"
                        data-agent-resource-project={project.name}
                        onKeyDown={(event) => handleResultKeyDown(event, index)}
                        onClick={() => selectProject(project)}
                      >
                        <FolderKanban className="size-4 shrink-0" />
                        <span className="flex min-w-0 flex-1 flex-col">
                          <span className="truncate">
                            {project.title || project.name}
                          </span>{" "}
                          <span className="break-all whitespace-normal text-xs text-control-light">
                            {project.name}
                          </span>
                        </span>
                      </Button>
                    ))}
                  {isDatabaseView &&
                    databases.map((database, index) => (
                      <Button
                        key={database.name}
                        ref={(element) => {
                          resultRefs.current[index] = element;
                        }}
                        appearance="secondary"
                        size="sm"
                        className="h-auto w-full justify-start gap-2 px-2 py-2 text-left focus-visible:bg-control-bg focus-visible:ring-inset focus-visible:ring-offset-0"
                        data-agent-resource-database={database.name}
                        onKeyDown={(event) => handleResultKeyDown(event, index)}
                        onClick={() => selectDatabase(database)}
                      >
                        <DatabaseTargetDisplay
                          database={database}
                          keyword={query}
                          tooltipLayer="agent"
                          className="min-w-0 flex-1"
                        />
                      </Button>
                    ))}
                  {status === "loading" ? (
                    <p
                      className="flex items-center gap-2 px-2 py-2 text-xs text-control-light"
                      role="status"
                    >
                      <LoaderCircle
                        className="size-4 animate-spin"
                        aria-hidden
                      />
                      {t("common.loading")}
                    </p>
                  ) : null}
                  {status === "loaded" &&
                  results.length === 0 &&
                  !nextPageToken ? (
                    <p className="px-2 py-2 text-xs text-control-light">
                      {isDatabaseView
                        ? t("agent.resource-picker.no-databases")
                        : t("agent.resource-picker.no-projects")}
                    </p>
                  ) : null}
                  {status === "error" || status === "more-error" ? (
                    <div className="flex items-center justify-between gap-2 px-2 py-2">
                      <p className="text-xs text-error" role="alert">
                        {isDatabaseView
                          ? t("agent.resource-picker.load-databases-error")
                          : t("agent.resource-picker.load-error")}
                      </p>
                      <Button
                        appearance="secondary"
                        size="sm"
                        onClick={() =>
                          void loadPage(
                            status === "more-error" ? nextPageToken : "",
                            status === "more-error"
                          )
                        }
                      >
                        {t("agent.resource-picker.retry")}
                      </Button>
                    </div>
                  ) : null}
                  {nextPageToken &&
                  (status === "loaded" || status === "loading-more") ? (
                    <Button
                      appearance="secondary"
                      size="sm"
                      className="w-full justify-start"
                      disabled={status === "loading-more"}
                      onClick={() => void loadPage(nextPageToken, true)}
                    >
                      {status === "loading-more" && (
                        <LoaderCircle
                          className="size-4 animate-spin"
                          aria-hidden
                        />
                      )}
                      {status === "loading-more"
                        ? t("common.loading")
                        : t("common.load-more")}
                    </Button>
                  ) : null}
                </div>
              </div>
            )}
          </BasePopover.Popup>
        </BasePopover.Positioner>
      </BasePopover.Portal>
    </BasePopover.Root>
  );
}
