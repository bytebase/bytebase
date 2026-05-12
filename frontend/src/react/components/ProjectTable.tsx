import { Check } from "lucide-react";
import type { MouseEvent as ReactMouseEvent, ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { HighlightLabelText } from "@/react/components/HighlightLabelText";
import { Badge } from "@/react/components/ui/badge";
import { Checkbox } from "@/react/components/ui/checkbox";
import { EllipsisText } from "@/react/components/ui/ellipsis-text";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
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
    <Table className={className}>
      <TableHeader>
        <TableRow>
          {showSelection ? (
            <TableHead
              className="w-12 cursor-pointer"
              onClick={(e) => {
                e.stopPropagation();
                handleSelectAll();
              }}
            >
              <Checkbox
                checked={allSelected}
                aria-label={t("common.select-all")}
                onCheckedChange={handleSelectAll}
                onClick={(e) => e.stopPropagation()}
              />
            </TableHead>
          ) : showLeadingCheck ? (
            <TableHead className="w-8 px-2" />
          ) : null}
          <TableHead className="min-w-[128px]">{t("common.id")}</TableHead>
          <TableHead
            className="min-w-[200px]"
            sortable={!!onSortChange}
            sortActive={sortKey === "title"}
            sortDir={sortOrder}
            onSort={() => onSortChange?.("title")}
          >
            {t("project.table.name")}
          </TableHead>
          {showLabels ? (
            <TableHead className="min-w-[240px] hidden md:table-cell">
              {t("common.labels")}
            </TableHead>
          ) : null}
          {showActions ? <TableHead className="w-[50px]" /> : null}
        </TableRow>
      </TableHeader>
      <TableBody>
        {loading && projectList.length === 0 ? (
          <TableRow>
            <TableCell
              colSpan={totalColumns}
              className="px-4 py-8 text-center text-control-placeholder"
            >
              <div className="flex items-center justify-center gap-x-2">
                <div className="animate-spin size-4 border-2 border-accent border-t-transparent rounded-full" />
                {t("common.loading")}
              </div>
            </TableCell>
          </TableRow>
        ) : projectList.length === 0 ? (
          <TableRow>
            <TableCell
              colSpan={totalColumns}
              className="px-4 py-8 text-center text-control-placeholder"
            >
              {emptyContent ?? t("common.no-data")}
            </TableCell>
          </TableRow>
        ) : (
          projectList.map((project) => {
            const resourceId = getProjectName(project.name);
            const isDefault = resourceId === "default";
            const isCurrent = currentProject?.name === project.name;
            const isSelected = selectedProjectNames.includes(project.name);
            return (
              <TableRow
                key={project.name}
                className={cn(onRowClick && "cursor-pointer")}
                onClick={(event) => onRowClick?.(project, event)}
              >
                {showSelection ? (
                  <TableCell
                    className={cn("w-12", !isDefault && "cursor-pointer")}
                    onClick={(e) => {
                      e.stopPropagation();
                      if (isDefault) return;
                      handleToggleRow(project.name);
                    }}
                  >
                    <Checkbox
                      checked={isSelected}
                      aria-label={t("common.select")}
                      disabled={isDefault}
                      onCheckedChange={() => handleToggleRow(project.name)}
                      onClick={(e) => e.stopPropagation()}
                      className="disabled:opacity-50"
                    />
                  </TableCell>
                ) : showLeadingCheck ? (
                  <TableCell className="w-8 px-2">
                    {isCurrent ? (
                      <Check className="size-4 text-accent" />
                    ) : null}
                  </TableCell>
                ) : null}
                <TableCell>
                  <EllipsisText text={resourceId}>
                    <HighlightLabelText text={resourceId} keyword={keyword} />
                  </EllipsisText>
                </TableCell>
                <TableCell>
                  <div className="flex items-center gap-x-2 min-w-0">
                    <EllipsisText
                      text={project.title || resourceId}
                      className="min-w-0"
                    >
                      <HighlightLabelText
                        text={project.title || resourceId}
                        keyword={keyword}
                      />
                    </EllipsisText>
                    {project.state === State.DELETED ? (
                      <Badge variant="warning" className="text-xs shrink-0">
                        {t("common.archived")}
                      </Badge>
                    ) : null}
                  </div>
                </TableCell>
                {showLabels ? (
                  <TableCell className="hidden md:table-cell">
                    <LabelsCell labels={project.labels ?? {}} />
                  </TableCell>
                ) : null}
                {showActions ? (
                  <TableCell
                    className="w-[50px]"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <div className="flex justify-end">
                      {renderActions?.(project)}
                    </div>
                  </TableCell>
                ) : null}
              </TableRow>
            );
          })
        )}
      </TableBody>
    </Table>
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
