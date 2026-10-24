import { Popover as BasePopover } from "@base-ui/react/popover";
import { ArrowLeft, ChevronRight, FolderKanban } from "lucide-react";
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
import { Button } from "@/components/ui/button";
import { getLayerRoot, LAYER_SURFACE_CLASS } from "@/components/ui/layer";
import { SearchInput } from "@/components/ui/search-input";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { searchAgentProjects } from "../logic/resourceSearch";

interface AgentResourcePickerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  anchorRef: RefObject<HTMLTextAreaElement | null>;
  workspaceName: string;
  hasSelectedReference: boolean;
  onSelectProject: (project: Project) => void;
}

type PickerView = "categories" | "projects";
type SearchStatus =
  | "idle"
  | "loading"
  | "loaded"
  | "loading-more"
  | "error"
  | "more-error";

const PICKER_VIEWPORT_PADDING = 8;

export function AgentResourcePicker({
  open,
  onOpenChange,
  anchorRef,
  workspaceName,
  hasSelectedReference,
  onSelectProject,
}: AgentResourcePickerProps) {
  const { t } = useTranslation();
  const [view, setView] = useState<PickerView>("categories");
  const [query, setQuery] = useState("");
  const [projects, setProjects] = useState<Project[]>([]);
  const [nextPageToken, setNextPageToken] = useState("");
  const [status, setStatus] = useState<SearchStatus>("idle");
  const requestGeneration = useRef(0);
  const requestController = useRef<AbortController | null>(null);
  const categoryRef = useRef<HTMLButtonElement>(null);
  const popupRef = useRef<HTMLDivElement>(null);
  const categoryTopOffsetRef = useRef<number | null>(null);
  const noticeRef = useRef<HTMLButtonElement>(null);
  const searchRef = useRef<HTMLInputElement>(null);
  const projectRefs = useRef<Array<HTMLButtonElement | null>>([]);

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
        setNextPageToken("");
        setStatus("loading");
      }

      try {
        const result = await searchAgentProjects(
          query,
          pageToken,
          controller.signal
        );
        if (
          controller.signal.aborted ||
          generation !== requestGeneration.current
        )
          return;
        setProjects((previous) =>
          append ? [...previous, ...result.items] : result.items
        );
        setNextPageToken(result.nextPageToken);
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
    [query]
  );

  useEffect(() => {
    if (!open || view !== "projects" || hasSelectedReference) return;
    void loadPage("", false);
    return () => {
      requestController.current?.abort();
      requestGeneration.current += 1;
    };
  }, [hasSelectedReference, loadPage, open, view, workspaceName]);

  useEffect(() => {
    if (open) return;
    setView("categories");
    setQuery("");
    setProjects([]);
    setNextPageToken("");
    setStatus("idle");
  }, [open]);

  useLayoutEffect(() => {
    if (!open) return;
    if (hasSelectedReference) noticeRef.current?.focus();
    else if (view === "projects") searchRef.current?.focus();
    else categoryRef.current?.focus();
  }, [hasSelectedReference, open, view]);

  const selectProject = (project: Project) => {
    if (hasSelectedReference) return;
    onSelectProject(project);
    onOpenChange(false);
  };

  const showProjects = () => {
    const anchorTop = anchorRef.current?.getBoundingClientRect().top;
    const popupTop = popupRef.current?.getBoundingClientRect().top;
    if (anchorTop !== undefined && popupTop !== undefined) {
      categoryTopOffsetRef.current = anchorTop - popupTop;
    }
    setView("projects");
  };

  const handleSearchKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.nativeEvent.isComposing) return;
    if (event.key === "ArrowDown" && projects.length > 0) {
      event.preventDefault();
      projectRefs.current[0]?.focus();
    } else if (event.key === "Enter" && projects.length > 0) {
      event.preventDefault();
      selectProject(projects[0]);
    }
  };

  const handleProjectKeyDown = (
    event: KeyboardEvent<HTMLButtonElement>,
    index: number
  ) => {
    if (event.nativeEvent.isComposing) return;
    if (event.key === "ArrowDown" && index < projects.length - 1) {
      event.preventDefault();
      projectRefs.current[index + 1]?.focus();
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      if (index === 0) searchRef.current?.focus();
      else projectRefs.current[index - 1]?.focus();
    } else if (event.key === "Enter") {
      event.preventDefault();
      selectProject(projects[index]);
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
            if (view !== "projects" || categoryTopOffsetRef.current === null) {
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
                : view === "projects"
                  ? searchRef
                  : categoryRef
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
                  {t("agent.resource-picker.one-reference-per-message")}
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
              <Button
                ref={categoryRef}
                appearance="secondary"
                size="sm"
                className="h-auto w-full justify-start gap-2 px-2 py-2 text-left"
                data-agent-resource-category="project"
                onKeyDown={(event) => {
                  if (event.nativeEvent.isComposing) return;
                  if (event.key === "Enter") {
                    event.preventDefault();
                    showProjects();
                  }
                }}
                onClick={showProjects}
              >
                <FolderKanban className="size-4 shrink-0" />
                <span className="min-w-0 flex-1 truncate">
                  {t("agent.resource-picker.projects")}
                </span>
                <ChevronRight className="size-4 shrink-0" />
              </Button>
            ) : (
              <div className="flex max-h-[min(20rem,calc(100dvh-2rem))] flex-col gap-2">
                <Button
                  appearance="secondary"
                  size="sm"
                  className="h-auto w-fit gap-1 px-2 py-1 text-control-light"
                  onClick={() => {
                    setView("categories");
                    setQuery("");
                  }}
                >
                  <ArrowLeft className="size-4" />
                  {t("common.back")}
                </Button>
                <SearchInput
                  ref={searchRef}
                  size="sm"
                  value={query}
                  onChange={(event) => setQuery(event.target.value)}
                  onChangeCapture={() => {
                    requestController.current?.abort();
                    requestGeneration.current += 1;
                    setProjects([]);
                    setNextPageToken("");
                    setStatus("idle");
                  }}
                  onKeyDown={handleSearchKeyDown}
                  placeholder={t("agent.resource-picker.search-projects")}
                  aria-label={t("agent.resource-picker.search-projects")}
                />
                <div
                  className="min-h-0 overflow-y-auto"
                  data-agent-resource-results
                >
                  {projects.map((project, index) => (
                    <Button
                      key={project.name}
                      ref={(element) => {
                        projectRefs.current[index] = element;
                      }}
                      appearance="secondary"
                      size="sm"
                      className="h-auto w-full justify-start gap-2 px-2 py-2 text-left focus-visible:bg-control-bg focus-visible:ring-inset focus-visible:ring-offset-0"
                      data-agent-resource-project={project.name}
                      onKeyDown={(event) => handleProjectKeyDown(event, index)}
                      onClick={() => selectProject(project)}
                    >
                      <FolderKanban className="size-4 shrink-0" />
                      <span className="flex min-w-0 flex-1 flex-col">
                        <span className="truncate">
                          {project.title || project.name}
                        </span>{" "}
                        <span className="break-all text-xs text-control-light">
                          {project.name}
                        </span>
                      </span>
                    </Button>
                  ))}
                  {status === "loading" ? (
                    <p
                      className="px-2 py-2 text-xs text-control-light"
                      role="status"
                    >
                      {t("common.loading")}
                    </p>
                  ) : null}
                  {status === "loaded" && projects.length === 0 ? (
                    <p className="px-2 py-2 text-xs text-control-light">
                      {t("agent.resource-picker.no-projects")}
                    </p>
                  ) : null}
                  {status === "error" || status === "more-error" ? (
                    <div className="flex items-center justify-between gap-2 px-2 py-2">
                      <p className="text-xs text-error" role="alert">
                        {t("agent.resource-picker.load-error")}
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
