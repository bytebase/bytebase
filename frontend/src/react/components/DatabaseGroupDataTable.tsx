import { EllipsisVertical, ExternalLink } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/react/components/ui/dropdown-menu";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import { PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL } from "@/router/dashboard/projectV1";
import { getProjectNameAndDatabaseGroupName } from "@/store";
import type { DatabaseGroup } from "@/types/proto-es/v1/database_group_service_pb";

type Props = {
  readonly databaseGroupList: DatabaseGroup[];
  readonly loading?: boolean;
  readonly showSelection?: boolean;
  readonly showExternalLink?: boolean;
  readonly showActions?: boolean;
  readonly singleSelection?: boolean;
  readonly selectedDatabaseGroupNames?: string[];
  readonly onSelectedDatabaseGroupNamesChange?: (names: string[]) => void;
  readonly onRowClick?: (e: React.MouseEvent, group: DatabaseGroup) => void;
  readonly onDelete?: (group: DatabaseGroup) => void;
  /** Client-side pagination size. `0` or undefined disables pagination. */
  readonly pageSize?: number;
};

/**
 * Replaces frontend/src/components/DatabaseGroup/DatabaseGroupDataTable.vue.
 * Shared React port. Columns — in order — are: optional selection, title,
 * expression, optional external-link, optional actions (kebab menu with
 * Delete). Supports optional client-side pagination for the "manage
 * groups" admin use case.
 */
export function DatabaseGroupDataTable({
  databaseGroupList,
  loading,
  showSelection,
  showExternalLink,
  showActions,
  singleSelection,
  selectedDatabaseGroupNames = [],
  onSelectedDatabaseGroupNamesChange,
  onRowClick,
  onDelete,
  pageSize = 0,
}: Props) {
  const { t } = useTranslation();
  const [page, setPage] = useState(0);

  // Reset paging whenever the underlying list shrinks/grows so we never
  // land on an out-of-range page.
  useEffect(() => {
    setPage(0);
  }, [databaseGroupList.length]);

  const totalPages =
    pageSize > 0 ? Math.ceil(databaseGroupList.length / pageSize) : 1;
  const visibleList = useMemo(() => {
    if (pageSize <= 0) return databaseGroupList;
    return databaseGroupList.slice(page * pageSize, (page + 1) * pageSize);
  }, [databaseGroupList, page, pageSize]);

  const selectedSet = useMemo(
    () => new Set(selectedDatabaseGroupNames),
    [selectedDatabaseGroupNames]
  );

  const toggleSingle = useCallback(
    (name: string, checked: boolean) => {
      if (!onSelectedDatabaseGroupNamesChange) return;
      if (singleSelection) {
        onSelectedDatabaseGroupNamesChange(checked ? [name] : []);
        return;
      }
      const next = new Set(selectedSet);
      if (checked) {
        next.add(name);
      } else {
        next.delete(name);
      }
      onSelectedDatabaseGroupNamesChange([...next]);
    },
    [onSelectedDatabaseGroupNamesChange, selectedSet, singleSelection]
  );

  const handleRowClick = (e: React.MouseEvent, group: DatabaseGroup) => {
    if (onRowClick) {
      onRowClick(e, group);
      return;
    }
    const isSelected = selectedSet.has(group.name);
    toggleSingle(group.name, !isSelected);
  };

  const handleExternalLink = (e: React.MouseEvent, group: DatabaseGroup) => {
    e.stopPropagation();
    const [projectId, databaseGroupName] = getProjectNameAndDatabaseGroupName(
      group.name
    );
    const route = router.resolve({
      name: PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
      params: {
        projectId,
        databaseGroupName,
      },
    });
    window.open(route.fullPath, "_blank");
  };

  const colCount =
    1 /* title */ +
    1 /* expression */ +
    (showSelection ? 1 : 0) +
    (showExternalLink ? 1 : 0) +
    (showActions ? 1 : 0);

  return (
    <div className="flex flex-col gap-y-3">
      <div className="border rounded-sm">
        <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow className="bg-gray-50">
                {showSelection && <TableHead className="w-10" />}
                <TableHead className="w-64">{t("common.name")}</TableHead>
                <TableHead>{t("database.expression")}</TableHead>
                {showExternalLink && <TableHead className="w-12" />}
                {showActions && <TableHead className="w-12" />}
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading && databaseGroupList.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={colCount}
                    className="py-8 text-center text-control-placeholder"
                  >
                    <div className="flex items-center justify-center gap-x-2">
                      <div className="animate-spin h-4 w-4 border-2 border-accent border-t-transparent rounded-full" />
                      {t("common.loading")}
                    </div>
                  </TableCell>
                </TableRow>
              ) : databaseGroupList.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={colCount}
                    className="py-8 text-center text-control-placeholder"
                  >
                    {t("common.no-data")}
                  </TableCell>
                </TableRow>
              ) : (
                visibleList.map((group) => {
                  const isSelected = selectedSet.has(group.name);
                  return (
                    <TableRow
                      key={group.name}
                      className={cn("cursor-pointer")}
                      data-state={isSelected ? "selected" : undefined}
                      onClick={(e) => handleRowClick(e, group)}
                    >
                      {showSelection && (
                        <TableCell onClick={(e) => e.stopPropagation()}>
                          <input
                            type={singleSelection ? "radio" : "checkbox"}
                            className="rounded-xs border-control-border"
                            checked={isSelected}
                            onChange={(e) =>
                              toggleSingle(group.name, e.target.checked)
                            }
                          />
                        </TableCell>
                      )}
                      <TableCell className="max-w-[16rem] truncate">
                        {group.title}
                      </TableCell>
                      <TableCell className="truncate">
                        {group.databaseExpr?.expression ? (
                          group.databaseExpr.expression
                        ) : (
                          <span className="italic text-control-placeholder">
                            {t("common.empty")}
                          </span>
                        )}
                      </TableCell>
                      {showExternalLink && (
                        <TableCell>
                          <button
                            type="button"
                            className={cn(
                              "flex items-center justify-end cursor-pointer size-6 p-1 opacity-60",
                              "hover:opacity-100 hover:bg-white hover:shadow-xs rounded-sm"
                            )}
                            onClick={(e) => handleExternalLink(e, group)}
                            aria-label={t("common.view-details")}
                          >
                            <ExternalLink className="size-4" />
                          </button>
                        </TableCell>
                      )}
                      {showActions && (
                        <TableCell onClick={(e) => e.stopPropagation()}>
                          <ActionDropdown group={group} onDelete={onDelete} />
                        </TableCell>
                      )}
                    </TableRow>
                  );
                })
              )}
            </TableBody>
          </Table>
        </div>
      </div>

      {pageSize > 0 && totalPages > 1 && (
        <div className="flex justify-end items-center gap-x-2">
          <Button
            variant="outline"
            size="sm"
            disabled={page === 0}
            onClick={() => setPage((p) => Math.max(0, p - 1))}
          >
            {t("common.previous")}
          </Button>
          <span className="text-sm text-control-light">
            {page + 1} / {totalPages}
          </span>
          <Button
            variant="outline"
            size="sm"
            disabled={page >= totalPages - 1}
            onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
          >
            {t("common.next")}
          </Button>
        </div>
      )}
    </div>
  );
}

function ActionDropdown({
  group,
  onDelete,
}: {
  group: DatabaseGroup;
  onDelete?: (group: DatabaseGroup) => void;
}) {
  const { t } = useTranslation();
  return (
    <div className="flex justify-end">
      <DropdownMenu>
        <DropdownMenuTrigger
          className="p-1 rounded-xs hover:bg-control-bg outline-hidden"
          onClick={(e) => e.stopPropagation()}
          aria-label={t("common.actions")}
        >
          <EllipsisVertical className="size-4" />
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuItem
            className="text-error"
            onClick={(e) => {
              e.stopPropagation();
              onDelete?.(group);
            }}
          >
            {t("common.delete")}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
