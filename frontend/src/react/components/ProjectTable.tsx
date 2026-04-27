import { Check, ChevronDown } from "lucide-react";
import type { MouseEvent as ReactMouseEvent, ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { HighlightLabelText } from "@/react/components/HighlightLabelText";
import { Badge } from "@/react/components/ui/badge";
import { Tooltip } from "@/react/components/ui/tooltip";
import { getProjectName } from "@/react/lib/resourceName";
import { cn } from "@/react/lib/utils";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

const DEFAULT_LABELS_PLACEHOLDER = "-";
const DEFAULT_LABELS_PREVIEW = 3;

export type ProjectTableSortKey = "title";
export type ProjectTableSortDirection = "asc" | "desc";

type RenderActions = (project: Project) => ReactNode;

export interface ProjectTableProps {
  /** Rows to render. */
  readonly projectList: readonly Project[];
  /**
   * When set, the row whose `name` matches gets a leading check icon
   * (mirrors Vue's `:current-project` mode used in the project switcher).
   */
  readonly currentProject?: Project;
  /** Highlight keyword for the id and title cells. */
  readonly keyword?: string;
  /**
   * Show the loading spinner row instead of `emptyContent` when there
   * are no projects. Mirrors NDataTable's `:loading` prop.
   */
  readonly loading?: boolean;
  /**
   * Custom empty-state node. Defaults to a "No data" placeholder.
   */
  readonly emptyContent?: ReactNode;
  /** Mirrors Vue's `:show-selection`. Adds a leading checkbox column. */
  readonly showSelection?: boolean;
  /** Mirrors Vue's `:show-labels`. Defaults to true. */
  readonly showLabels?: boolean;
  /**
   * Mirrors Vue's `:show-actions`. When set, the trailing column renders
   * `renderActions(project)` (typically the project action dropdown).
   */
  readonly showActions?: boolean;
  readonly renderActions?: RenderActions;
  /** Currently-checked rows (selection mode). */
  readonly selectedProjectNames?: readonly string[];
  /** Selection-change callback. */
  readonly onSelectedChange?: (names: string[]) => void;
  /** Sortable title column — when active, the header shows the indicator. */
  readonly sortKey?: string | null;
  readonly sortOrder?: ProjectTableSortDirection;
  readonly onSortChange?: (key: ProjectTableSortKey) => void;
  /**
   * Click handler. Receives the original event so callers can detect
   * `ctrl`/`meta` modifiers for "open in new tab".
   */
  readonly onRowClick?: (
    project: Project,
    event: ReactMouseEvent<HTMLTableRowElement>
  ) => void;
  readonly className?: string;
}

/**
 * React port of `frontend/src/components/v2/Model/ProjectV1Table.vue`.
 *
 * Renders the standard project listing — id / title / labels columns,
 * with optional leading current-project check, leading selection
 * checkboxes, and trailing action-dropdown slot. Matches the prop
 * shape of the Vue component so call sites read the same way.
 *
 * Two surfaces consume this today:
 *   - `ProjectsPage` (settings) — `showSelection` + `showActions` +
 *     server-side sort by title.
 *   - `ProjectSwitchPanel` (header popover) — `showLabels=false` +
 *     `currentProject` set to the active project.
 *
 * Uses plain HTML table elements (not the shadcn `Table` primitives)
 * to preserve the visual styling that ProjectsPage shipped with —
 * specifically: `bg-control-bg` header row, `px-4 py-2` cell padding,
 * and a rotating `ChevronDown` sort indicator. Switching to the
 * shadcn primitives would change all of those at once.
 */
export function ProjectTable({
  projectList,
  currentProject,
  keyword = "",
  loading = false,
  emptyContent,
  showSelection = false,
  showLabels = true,
  showActions = false,
  renderActions,
  selectedProjectNames = [],
  onSelectedChange,
  sortKey,
  sortOrder,
  onSortChange,
  onRowClick,
  className,
}: ProjectTableProps) {
  const { t } = useTranslation();
  const showLeadingCheck = !!currentProject && !showSelection;
  const selectableProjects = projectList.filter(
    (p) => getProjectName(p.name) !== "default"
  );
  const allSelectableNames = selectableProjects.map((p) => p.name);
  const allSelected =
    showSelection &&
    allSelectableNames.length > 0 &&
    allSelectableNames.every((name) => selectedProjectNames.includes(name));

  const totalColumns =
    (showSelection || showLeadingCheck ? 1 : 0) +
    1 + // id
    1 + // title
    (showLabels ? 1 : 0) +
    (showActions ? 1 : 0);

  const handleSelectAll = () => {
    if (!onSelectedChange) return;
    onSelectedChange(allSelected ? [] : allSelectableNames);
  };

  const handleToggleRow = (name: string) => {
    if (!onSelectedChange) return;
    const next = selectedProjectNames.includes(name)
      ? selectedProjectNames.filter((n) => n !== name)
      : [...selectedProjectNames, name];
    onSelectedChange(next as string[]);
  };

  return (
    <table className={cn("w-full text-sm", className)}>
      <thead>
        <tr className="bg-control-bg border-b border-control-border">
          {showSelection ? (
            <th className="w-12 px-4 py-2">
              <input
                type="checkbox"
                aria-label={t("common.select-all")}
                checked={allSelected}
                onChange={handleSelectAll}
                className="rounded-xs border-control-border"
              />
            </th>
          ) : showLeadingCheck ? (
            <th className="w-8 px-2 py-2" />
          ) : null}
          <th className="px-4 py-2 text-left font-medium min-w-[128px]">
            {t("common.id")}
          </th>
          <th
            className={cn(
              "px-4 py-2 text-left font-medium min-w-[200px]",
              onSortChange && "cursor-pointer select-none"
            )}
            onClick={() => onSortChange?.("title")}
          >
            <div className="flex items-center gap-x-1">
              {t("project.table.name")}
              {onSortChange ? (
                <SortIndicator
                  active={sortKey === "title"}
                  direction={sortOrder}
                />
              ) : null}
            </div>
          </th>
          {showLabels ? (
            <th className="px-4 py-2 text-left font-medium min-w-[240px] hidden md:table-cell">
              {t("common.labels")}
            </th>
          ) : null}
          {showActions ? <th className="w-[50px]" /> : null}
        </tr>
      </thead>
      <tbody>
        {loading && projectList.length === 0 ? (
          <tr>
            <td
              colSpan={totalColumns}
              className="px-4 py-8 text-center text-control-placeholder"
            >
              <div className="flex items-center justify-center gap-x-2">
                <div className="animate-spin size-4 border-2 border-accent border-t-transparent rounded-full" />
                {t("common.loading")}
              </div>
            </td>
          </tr>
        ) : projectList.length === 0 ? (
          <tr>
            <td
              colSpan={totalColumns}
              className="px-4 py-8 text-center text-control-placeholder"
            >
              {emptyContent ?? t("common.no-data")}
            </td>
          </tr>
        ) : (
          projectList.map((project, i) => {
            const resourceId = getProjectName(project.name);
            const isDefault = resourceId === "default";
            const isCurrent = currentProject?.name === project.name;
            const isSelected = selectedProjectNames.includes(project.name);
            return (
              <tr
                key={project.name}
                className={cn(
                  "border-b last:border-b-0",
                  onRowClick && "cursor-pointer hover:bg-control-bg",
                  i % 2 === 1 && "bg-control-bg/50"
                )}
                onClick={(event) => onRowClick?.(project, event)}
              >
                {showSelection ? (
                  <td
                    className="w-12 px-4 py-2"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <input
                      type="checkbox"
                      aria-label={t("common.select")}
                      checked={isSelected}
                      disabled={isDefault}
                      onChange={() => handleToggleRow(project.name)}
                      className="rounded-xs border-control-border disabled:opacity-50"
                    />
                  </td>
                ) : showLeadingCheck ? (
                  <td className="w-8 px-2 py-2">
                    {isCurrent ? (
                      <Check className="size-4 text-accent" />
                    ) : null}
                  </td>
                ) : null}
                <td className="px-4 py-2">
                  <HighlightLabelText text={resourceId} keyword={keyword} />
                </td>
                <td className="px-4 py-2">
                  <div className="flex items-center gap-x-2">
                    <HighlightLabelText
                      text={project.title || resourceId}
                      keyword={keyword}
                    />
                    {project.state === State.DELETED ? (
                      <Badge variant="warning" className="text-xs">
                        {t("common.archived")}
                      </Badge>
                    ) : null}
                  </div>
                </td>
                {showLabels ? (
                  <td className="px-4 py-2 hidden md:table-cell">
                    <LabelsCell labels={project.labels ?? {}} />
                  </td>
                ) : null}
                {showActions ? (
                  <td
                    className="w-[50px] px-4 py-2"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <div className="flex justify-end">
                      {renderActions?.(project)}
                    </div>
                  </td>
                ) : null}
              </tr>
            );
          })
        )}
      </tbody>
    </table>
  );
}

/**
 * ProjectsPage's original sort indicator: a single ChevronDown that
 * rotates 180° when the column is sorted ascending. Inactive state
 * shows the chevron in `text-control-border` so it reads as "you
 * could click this". Matches the pre-extraction look.
 */
function SortIndicator({
  active,
  direction,
}: {
  active: boolean;
  direction?: ProjectTableSortDirection;
}) {
  if (!active) {
    return <ChevronDown className="size-3 text-control-border" />;
  }
  return (
    <ChevronDown
      className={cn(
        "size-3 text-accent transition-transform",
        direction === "asc" && "rotate-180"
      )}
    />
  );
}

/**
 * Mirrors Vue's `LabelsCell` — show up to N labels inline, "..." for
 * the rest, and a tooltip on hover that lists all of them.
 */
function LabelsCell({ labels }: { labels: { [key: string]: string } }) {
  const entries = Object.entries(labels);
  if (entries.length === 0) {
    return (
      <span className="text-control-placeholder">
        {DEFAULT_LABELS_PLACEHOLDER}
      </span>
    );
  }
  const visible = entries.slice(0, DEFAULT_LABELS_PREVIEW);
  const hasMore = entries.length > DEFAULT_LABELS_PREVIEW;
  const trigger = (
    <span className="inline-flex items-center gap-x-1">
      {visible.map(([key, value]) => (
        <span
          key={key}
          className="rounded-xs bg-control-bg py-0.5 px-2 text-sm"
        >
          {key}:{value}
        </span>
      ))}
      {hasMore ? <span>…</span> : null}
    </span>
  );
  if (!hasMore) return trigger;
  return (
    <Tooltip
      content={
        <div className="text-sm flex flex-col gap-y-1">
          {entries.map(([key, value]) => (
            <div key={key}>
              {key}:{value}
            </div>
          ))}
        </div>
      }
    >
      {trigger}
    </Tooltip>
  );
}
