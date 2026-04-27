import type { ReactNode } from "react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { DatabaseGroupDataTable } from "@/react/components/DatabaseGroupDataTable";
import { SearchInput } from "@/react/components/ui/search-input";
import { useDBGroupStore } from "@/store/modules";
import type { DatabaseGroup } from "@/types/proto-es/v1/database_group_service_pb";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";

type Props = {
  readonly projectName: string;
  readonly view: DatabaseGroupView;
  /** Leading caption shown to the left of the search box. */
  readonly leadingLabel?: string;
  /** Optional element rendered to the right of the search box (e.g. Create). */
  readonly trailingAction?: ReactNode;
  // --- DataTable forwards ---
  readonly showSelection?: boolean;
  readonly showExternalLink?: boolean;
  readonly showActions?: boolean;
  readonly singleSelection?: boolean;
  readonly selectedDatabaseGroupNames?: string[];
  readonly onSelectedDatabaseGroupNamesChange?: (names: string[]) => void;
  readonly onRowClick?: (e: React.MouseEvent, group: DatabaseGroup) => void;
  readonly onDelete?: (group: DatabaseGroup) => void;
  readonly pageSize?: number;
  /**
   * Controls who owns the fetched list. When set, the wrapper skips its
   * internal fetch and renders whatever the caller passes. Use this when
   * the page already owns the list (e.g. needs to mutate it after delete).
   */
  readonly externalList?: DatabaseGroup[];
  readonly externalLoading?: boolean;
  readonly searchPlaceholder?: string;
};

/**
 * Shared wrapper: SearchInput + `DatabaseGroupDataTable`. Either:
 *  - owns the fetch (default — used by SQL Editor ConnectionPane); or
 *  - renders `externalList` verbatim (used by pages that already manage
 *    the list state, e.g. to remove a row after delete without refetching).
 *
 * All selection / action / pagination concerns are delegated to the inner
 * `DatabaseGroupDataTable`.
 */
export function DatabaseGroupTable({
  projectName,
  view,
  leadingLabel,
  trailingAction,
  showSelection,
  showExternalLink,
  showActions,
  singleSelection,
  selectedDatabaseGroupNames,
  onSelectedDatabaseGroupNamesChange,
  onRowClick,
  onDelete,
  pageSize,
  externalList,
  externalLoading,
  searchPlaceholder,
}: Props) {
  const { t } = useTranslation();
  const dbGroupStore = useDBGroupStore();

  const [search, setSearch] = useState("");
  const [internalList, setInternalList] = useState<DatabaseGroup[]>([]);
  const [internalReady, setInternalReady] = useState(false);

  // Only fetch when an external list is not provided.
  useEffect(() => {
    if (externalList !== undefined) return;
    let cancelled = false;
    setInternalReady(false);
    setInternalList([]);
    void dbGroupStore
      .fetchDBGroupListByProjectName(projectName, view)
      .then((groups) => {
        if (cancelled) return;
        setInternalList(groups);
        setInternalReady(true);
      });
    return () => {
      cancelled = true;
    };
  }, [dbGroupStore, projectName, view, externalList]);

  const sourceList = externalList ?? internalList;
  const loading =
    externalList !== undefined ? !!externalLoading : !internalReady;

  const filtered = useMemo(() => {
    const query = search.trim().toLowerCase();
    if (!query) return sourceList;
    return sourceList.filter(
      (g) =>
        g.name.toLowerCase().includes(query) ||
        g.title.toLowerCase().includes(query)
    );
  }, [sourceList, search]);

  return (
    <div className="flex flex-col gap-y-3">
      {(leadingLabel || trailingAction) && (
        <div className="w-full flex flex-row justify-between items-center gap-x-2">
          {leadingLabel ? (
            <p className="text-control-light text-sm">{leadingLabel}</p>
          ) : (
            <span />
          )}
          <div className="flex items-center gap-x-2">
            <SearchInput
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={searchPlaceholder ?? t("common.filter-by-name")}
              wrapperClassName="max-w-[16rem]"
            />
            {trailingAction}
          </div>
        </div>
      )}
      {!leadingLabel && !trailingAction && (
        <SearchInput
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder={searchPlaceholder ?? t("common.filter-by-name")}
          wrapperClassName="max-w-[16rem]"
        />
      )}
      <DatabaseGroupDataTable
        databaseGroupList={filtered}
        loading={loading}
        showSelection={showSelection}
        showExternalLink={showExternalLink}
        showActions={showActions}
        singleSelection={singleSelection}
        selectedDatabaseGroupNames={selectedDatabaseGroupNames}
        onSelectedDatabaseGroupNamesChange={onSelectedDatabaseGroupNamesChange}
        onRowClick={onRowClick}
        onDelete={onDelete}
        pageSize={pageSize}
      />
    </div>
  );
}
