import { create } from "@bufbuild/protobuf";
import { Loader2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { releaseServiceClientConnect } from "@/connect";
import { HumanizeTs } from "@/react/components/HumanizeTs";
import { LearnMoreLink } from "@/react/components/LearnMoreLink";
import { Alert } from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import { PagedTableFooter, usePagedData } from "@/react/hooks/usePagedData";
import { cn } from "@/react/lib/utils";
import { router } from "@/router";
import { useReleaseStore } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { getTimeForPbTimestampProtoEs } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Release } from "@/types/proto-es/v1/release_service_pb";
import { ListReleaseCategoriesRequestSchema } from "@/types/proto-es/v1/release_service_pb";
import { buildCategoryFilter, buildCategoryQuery } from "@/utils/releaseFilter";

const MAX_SHOW_FILES_COUNT = 3;

export function ProjectReleaseDashboardPage({
  projectId,
}: {
  projectId: string;
}) {
  const { t } = useTranslation();
  const releaseStore = useReleaseStore();
  const projectName = `${projectNamePrefix}${projectId}`;

  // Category filter from URL
  const [selectedCategory, setSelectedCategory] = useState<string | undefined>(
    () => {
      const params = new URLSearchParams(window.location.search);
      return params.get("category") || undefined;
    }
  );

  // Fetch categories
  const [categories, setCategories] = useState<string[]>([]);
  const [categoriesLoading, setCategoriesLoading] = useState(false);

  useEffect(() => {
    setCategoriesLoading(true);
    const request = create(ListReleaseCategoriesRequestSchema, {
      parent: projectName,
    });
    releaseServiceClientConnect
      .listReleaseCategories(request)
      .then((resp) => setCategories(resp.categories))
      .catch(() => setCategories([]))
      .finally(() => setCategoriesLoading(false));
  }, [projectName]);

  // URL sync
  const isUpdatingUrl = useRef(false);
  useEffect(() => {
    if (isUpdatingUrl.current) return;
    isUpdatingUrl.current = true;
    const query = buildCategoryQuery(selectedCategory);
    router.replace({ query }).finally(() => {
      isUpdatingUrl.current = false;
    });
  }, [selectedCategory]);

  // Fetch releases
  const fetchReleaseList = useCallback(
    async (params: { pageSize: number; pageToken: string }) => {
      const filter = buildCategoryFilter(selectedCategory);
      const { nextPageToken, releases } =
        await releaseStore.fetchReleasesByProject(
          projectName,
          { pageSize: params.pageSize, pageToken: params.pageToken },
          false,
          filter
        );
      return { list: releases, nextPageToken };
    },
    [releaseStore, projectName, selectedCategory]
  );

  const paged = usePagedData<Release>({
    sessionKey: `project-${projectName}-releases`,
    fetchList: fetchReleaseList,
  });

  return (
    <div className="py-4 w-full flex flex-col">
      <div className="px-4 flex flex-col gap-y-2 pb-2">
        <Alert
          variant="info"
          description={
            <>
              <span>{t("release.usage-description")}</span>
              <LearnMoreLink
                href="https://docs.bytebase.com/gitops/migration-based-workflow/release/?source=console"
                className="ml-1"
              />
            </>
          }
        />

        {/* Category filter */}
        <div className="flex items-center gap-x-4">
          <CategorySelect
            value={selectedCategory}
            onChange={setSelectedCategory}
            categories={categories}
            loading={categoriesLoading}
          />
          {selectedCategory && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setSelectedCategory(undefined)}
            >
              {t("common.clear-filters")}
            </Button>
          )}
        </div>
      </div>

      {/* Release table */}
      <div className="mt-2">
        {paged.isLoading ? (
          <div className="flex justify-center py-8 text-control-light">
            <Loader2 className="size-5 animate-spin" />
          </div>
        ) : paged.dataList.length === 0 ? (
          <div className="flex justify-center py-8 text-control-light">
            {t("common.no-data")}
          </div>
        ) : (
          <ReleaseTable releases={paged.dataList} />
        )}

        {paged.dataList.length > 0 && (
          <div className="mt-4 mx-2">
            <PagedTableFooter
              pageSize={paged.pageSize}
              pageSizeOptions={paged.pageSizeOptions}
              onPageSizeChange={paged.onPageSizeChange}
              hasMore={paged.hasMore}
              isFetchingMore={paged.isFetchingMore}
              onLoadMore={paged.loadMore}
            />
          </div>
        )}
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// CategorySelect
// ---------------------------------------------------------------------------

function CategorySelect({
  value,
  onChange,
  categories,
  loading,
}: {
  value: string | undefined;
  onChange: (value: string | undefined) => void;
  categories: string[];
  loading: boolean;
}) {
  return (
    <select
      className="w-64 border border-control-border rounded-sm text-sm px-3 py-1.5 bg-background focus:outline-none focus:border-accent"
      value={value ?? ""}
      onChange={(e) => onChange(e.target.value || undefined)}
      disabled={loading}
    >
      <option value="">All</option>
      {categories.map((cat) => (
        <option key={cat} value={cat}>
          {cat}
        </option>
      ))}
    </select>
  );
}

// ---------------------------------------------------------------------------
// ReleaseTable
// ---------------------------------------------------------------------------

function ReleaseTable({ releases }: { releases: Release[] }) {
  const { t } = useTranslation();

  return (
    <div className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow className="bg-control-bg">
            <TableHead className="w-75">{t("common.name")}</TableHead>
            <TableHead>{t("release.files")}</TableHead>
            <TableHead className="w-32">{t("common.created-at")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {releases.map((release) => (
            <ReleaseRow key={release.name} release={release} />
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

// ---------------------------------------------------------------------------
// ReleaseRow
// ---------------------------------------------------------------------------

function ReleaseRow({ release }: { release: Release }) {
  const { t } = useTranslation();
  const isDeleted = release.state === State.DELETED;

  const releaseName = useMemo(() => {
    const parts = release.name.split("/");
    return parts[parts.length - 1] || release.name;
  }, [release.name]);

  const showFiles = release.files.slice(0, MAX_SHOW_FILES_COUNT);

  const createTimeTs =
    getTimeForPbTimestampProtoEs(release.createTime, 0) / 1000;

  const url = `/${release.name}`;

  const onRowClick = useCallback(
    (e: React.MouseEvent) => {
      if (e.ctrlKey || e.metaKey) {
        window.open(url, "_blank");
      } else {
        router.push(url);
      }
    },
    [url]
  );

  return (
    <TableRow className="cursor-pointer" onClick={onRowClick}>
      <TableCell>
        <span
          className={cn(
            "truncate",
            isDeleted && "text-control-light line-through"
          )}
        >
          {releaseName}
        </span>
      </TableCell>
      <TableCell>
        <div className="flex flex-col items-start gap-1">
          {showFiles.map((file, idx) => (
            <p key={idx} className="w-full truncate">
              {file.version && (
                <span className="mr-2 inline-flex items-center rounded-full bg-control-bg px-2 py-0.5 text-xs">
                  {file.version}
                </span>
              )}
              {file.path}
            </p>
          ))}
          {release.files.length > MAX_SHOW_FILES_COUNT && (
            <p className="text-control-placeholder text-xs italic">
              {t("release.total-files", { count: release.files.length })}
            </p>
          )}
        </div>
      </TableCell>
      <TableCell>
        <HumanizeTs ts={createTimeTs} />
      </TableCell>
    </TableRow>
  );
}
