import { create } from "@bufbuild/protobuf";
import { GitCommitHorizontal, Loader2, Settings2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { planServiceClientConnect } from "@/api";
import { router } from "@/app/router";
import {
  PROJECT_V1_ROUTE_GITOPS,
  PROJECT_V1_ROUTE_PLAN_DETAIL,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT,
} from "@/app/router/handles";
import {
  markListScrollRestorationEntry,
  useListScrollRestorationKey,
  useListScrollRestorationLoadMore,
} from "@/app/router/NavigationScrollRestoration";
import azureDevOpsIcon from "@/assets/git/azure-devops.svg";
import bitbucketIcon from "@/assets/git/bitbucket.svg";
import githubIcon from "@/assets/git/github.svg";
import gitlabIcon from "@/assets/git/gitlab.svg";
import {
  AdvancedSearch,
  type ScopeOption,
  type SearchParams,
} from "@/components/AdvancedSearch";
import { HumanizeTs } from "@/components/HumanizeTs";
import {
  PermissionGuard,
  usePermissionCheck,
} from "@/components/PermissionGuard";
import {
  ProjectPageContent,
  ProjectPageFooter,
  ProjectPageInfo,
  ProjectPageLayout,
} from "@/components/ProjectPageLayout";
import { TimeRangePicker } from "@/components/TimeRangePicker";
import { Button } from "@/components/ui/button";
import { useMediaQuery } from "@/hooks/useMediaQuery";
import { PagedTableFooter, usePagedData } from "@/hooks/usePagedData";
import { useProjectByName } from "@/hooks/useProjectByName";
import {
  createAdvancedSearchParser,
  serializeAdvancedSearch,
  useURLSearchParam,
} from "@/hooks/useURLSearchParam";
import { projectPlansPagedDataCacheScope } from "@/lib/projectPagedDataCache";
import { useAppStore } from "@/stores/app";
import {
  buildPlanFilter,
  buildPlanFindBySearchParams,
} from "@/stores/app/plan";
import { projectNamePrefix } from "@/stores/modules/v1/common";
import type { Permission } from "@/types";
import { State, VCSType } from "@/types/proto-es/v1/common_pb";
import {
  ListPlansRequestSchema,
  type Plan,
} from "@/types/proto-es/v1/plan_service_pb";
import { extractPlanUID } from "@/utils";
import {
  createPlanStagesColumn,
  type PlanColumn,
  PlanLifecycleBadges,
  PlanTable,
} from "./PlanTable";
import { planStateScopeOption, planStatusScopeOption } from "./planListSearch";

// Below Tailwind's `sm` breakpoint (640px) the table keeps a compact Release
// column and scrolls sideways.
const MOBILE_MEDIA_QUERY = "(max-width: 639px)";
// `release` is not offered in the scope menu; the retired release detail URL
// redirects here with it set.
const parseReleaseSearchParams = createAdvancedSearchParser([
  "state",
  "status",
  "created",
  "release",
]);
const GITOPS_PLAN_FIND = {
  specType: "change_database_config",
  hasRelease: true,
} as const;
const GITOPS_SETUP_PERMISSIONS: Permission[] = [
  "bb.workloadIdentities.list",
  "bb.databases.list",
];
const GET_RELEASE_PERMISSIONS: Permission[] = ["bb.releases.get"];
const GITOPS_GUIDE_URL =
  "https://docs.bytebase.com/gitops/overview?source=console";

export function ProjectReleaseDashboardPage({
  projectId,
}: {
  projectId: string;
}) {
  const { t } = useTranslation();
  const listScrollRestorationKey = useListScrollRestorationKey();
  const projectName = `${projectNamePrefix}${projectId}`;
  const project = useProjectByName(projectName);
  const isMobile = useMediaQuery(MOBILE_MEDIA_QUERY);
  const [canSetUpGitOps] = usePermissionCheck(
    GITOPS_SETUP_PERMISSIONS,
    project
  );

  const defaultSearchParams = useMemo<SearchParams>(
    () => ({
      query: "",
      scopes: [{ id: "state", value: "ACTIVE" }],
    }),
    []
  );
  const [searchParams, setSearchParams] = useURLSearchParam({
    param: "q",
    parse: parseReleaseSearchParams,
    serialize: serializeAdvancedSearch,
    defaultValue: defaultSearchParams,
  });
  const serializedSearch = serializeAdvancedSearch(searchParams);
  const isDefaultSearch =
    serializedSearch === serializeAdvancedSearch(defaultSearchParams);

  const scopeOptions: ScopeOption[] = useMemo(
    () => [planStateScopeOption(t), planStatusScopeOption(t)],
    [t]
  );

  const planFind = useMemo(
    () =>
      buildPlanFindBySearchParams(
        {
          query: searchParams.query.trim().toLowerCase(),
          scopes: [
            ...searchParams.scopes.map((s) => ({ id: s.id, value: s.value })),
            { id: "project", value: projectId },
          ],
        },
        GITOPS_PLAN_FIND
      ),
    [searchParams, projectId]
  );

  const fetchPlanList = useCallback(
    async (params: { pageSize: number; pageToken: string }) => {
      const { nextPageToken, plans } = await useAppStore.getState().listPlans({
        find: planFind,
        pageSize: params.pageSize,
        pageToken: params.pageToken,
      });
      return { list: plans, nextPageToken };
    },
    [planFind]
  );

  const paged = usePagedData<Plan>({
    sessionKey: `bb.${projectName}.release-plan-table`,
    cacheKey: JSON.stringify([
      "project-releases",
      projectName,
      serializedSearch,
    ]),
    cacheScope: projectPlansPagedDataCacheScope(projectId),
    cacheRestoreToken: listScrollRestorationKey,
    fetchList: fetchPlanList,
  });
  useListScrollRestorationLoadMore(paged);

  const isEmpty = !paged.isLoading && paged.dataList.length === 0;
  const projectHasGitOpsPlan = useProjectHasGitOpsPlan(projectName, isEmpty);
  const hasGitOpsPlan = paged.dataList.length > 0 || projectHasGitOpsPlan;

  const columns = useReleaseColumns(isMobile);
  const releaseNames = useMemo(
    () => paged.dataList.map(getPlanReleaseName).filter(Boolean),
    [paged.dataList]
  );
  // A role that can list plans but not read releases sees "-" in Source.
  const [canGetReleases] = usePermissionCheck(GET_RELEASE_PERMISSIONS, project);
  useFetchReleases(canGetReleases ? releaseNames : []);

  const gitOpsSetupUrl = router.resolve({
    name: PROJECT_V1_ROUTE_GITOPS,
    params: { projectId },
  }).fullPath;
  const openGitOpsSetup = () => router.push(gitOpsSetupUrl);

  let content;
  if (paged.isLoading || (isEmpty && projectHasGitOpsPlan === undefined)) {
    content = (
      <div className="flex justify-center py-8 text-control-light">
        <Loader2 className="size-5 animate-spin" />
      </div>
    );
  } else if (isEmpty && !projectHasGitOpsPlan) {
    content = (
      <div
        data-testid="release-setup-empty-state"
        className="flex flex-col items-center gap-y-4 rounded-sm border border-dashed border-block-border px-4 py-16 text-center"
      >
        <div className="flex flex-col items-center gap-y-1">
          <h2 className="text-base font-semibold text-main">
            {t("release.empty.title")}
          </h2>
          <p className="max-w-md text-sm text-control-light">
            {t("release.empty.description")}
          </p>
        </div>
        <PermissionGuard
          permissions={GITOPS_SETUP_PERMISSIONS}
          project={project}
        >
          <Button disabled={!canSetUpGitOps} onClick={openGitOpsSetup}>
            {t("release.empty.set-up")}
          </Button>
        </PermissionGuard>
        <a
          href={GITOPS_GUIDE_URL}
          target="_blank"
          rel="noopener noreferrer"
          className="text-sm text-accent hover:underline"
        >
          {t("release.empty.read-guide")}
        </a>
      </div>
    );
  } else if (isEmpty) {
    content = (
      <div className="flex flex-col items-center gap-y-2 py-8 text-control-light">
        <span>{t("common.no-data")}</span>
        {!isDefaultSearch && (
          <Button
            appearance="secondary"
            size="sm"
            onClick={() => setSearchParams(defaultSearchParams)}
          >
            {t("common.clear-filters")}
          </Button>
        )}
      </div>
    );
  } else {
    content = (
      <PlanTable
        plans={paged.dataList}
        projectId={projectId}
        columns={columns}
        isMobile={isMobile}
        resolvePlanUrl={resolveReleasePlanUrl}
        onOpenPlan={markListScrollRestorationEntry}
      />
    );
  }

  return (
    <ProjectPageLayout>
      <div className="flex flex-col gap-y-2">
        <ProjectPageInfo description={t("release.list-description")} />
        <div className="flex flex-wrap items-center gap-2">
          <div className="min-w-0 basis-80 grow">
            <AdvancedSearch
              params={searchParams}
              onParamsChange={setSearchParams}
              scopeOptions={scopeOptions}
            />
          </div>
          <TimeRangePicker
            params={searchParams}
            onParamsChange={setSearchParams}
          />
          {hasGitOpsPlan && (
            <PermissionGuard
              permissions={GITOPS_SETUP_PERMISSIONS}
              project={project}
            >
              <Button
                data-testid="release-gitops-setup"
                appearance="outline"
                disabled={!canSetUpGitOps}
                onClick={openGitOpsSetup}
              >
                <Settings2 className="size-4 mr-1" />
                {t("release.gitops-setup")}
              </Button>
            </PermissionGuard>
          )}
        </div>
      </div>

      <ProjectPageContent>
        {content}
        {paged.dataList.length > 0 && (
          <ProjectPageFooter>
            <PagedTableFooter
              pageSize={paged.pageSize}
              pageSizeOptions={paged.pageSizeOptions}
              onPageSizeChange={paged.onPageSizeChange}
              hasMore={paged.hasMore}
              isFetchingMore={paged.isFetchingMore}
              onLoadMore={paged.loadMore}
            />
          </ProjectPageFooter>
        )}
      </ProjectPageContent>
    </ProjectPageLayout>
  );
}

// Only asked when the filtered page is empty: it decides between the setup
// empty state and the ordinary "No data" state.
function useProjectHasGitOpsPlan(
  projectName: string,
  enabled: boolean
): boolean | undefined {
  const [result, setResult] = useState<{
    projectName: string;
    hasPlan: boolean;
  }>();
  useEffect(() => {
    if (!enabled || result?.projectName === projectName) {
      return;
    }
    let canceled = false;
    planServiceClientConnect
      .listPlans(
        create(ListPlansRequestSchema, {
          parent: projectName,
          filter: buildPlanFilter({
            project: projectName,
            ...GITOPS_PLAN_FIND,
          }),
          pageSize: 1,
        })
      )
      // On failure, fall back to "No data" rather than inviting a second setup.
      .then(
        (resp) => resp.plans.length > 0,
        () => true
      )
      .then((hasPlan) => {
        if (!canceled) setResult({ projectName, hasPlan });
      });
    return () => {
      canceled = true;
    };
  }, [enabled, projectName, result?.projectName]);
  return result?.projectName === projectName ? result.hasPlan : undefined;
}

const getPlanReleaseName = (plan: Plan): string => {
  for (const spec of plan.specs) {
    if (
      spec.config?.case === "changeDatabaseConfig" &&
      spec.config.value.release
    ) {
      return spec.config.value.release;
    }
  }
  return "";
};

const lastPathSegment = (value: string): string => {
  const path = value.split(/[?#]/)[0].replace(/\/+$/, "");
  return path.slice(path.lastIndexOf("/") + 1);
};

const resolveReleasePlanUrl = (plan: Plan, projectId: string): string =>
  router.resolve({
    name: plan.hasRollout
      ? PROJECT_V1_ROUTE_PLAN_ROLLOUT
      : PROJECT_V1_ROUTE_PLAN_DETAIL,
    params: { projectId, planId: extractPlanUID(plan.name) },
  }).fullPath;

// Each distinct release is fetched once per page; a failed fetch is not retried
// and its row shows "-".
function useFetchReleases(releaseNames: string[]) {
  const fetchRelease = useAppStore((s) => s.fetchRelease);
  const [failed] = useState(() => new Set<string>());
  const key = [...new Set(releaseNames)].sort().join(",");
  useEffect(() => {
    if (!key) return;
    const { releasesByName } = useAppStore.getState();
    for (const name of key.split(",")) {
      if (releasesByName[name] || failed.has(name)) continue;
      fetchRelease(name, /* silent */ true).catch(() => failed.add(name));
    }
  }, [key, fetchRelease, failed]);
}

function useReleaseColumns(isMobile: boolean): PlanColumn[] {
  const { t } = useTranslation();
  return useMemo<PlanColumn[]>(
    () => [
      {
        key: "release",
        title: t("release.self"),
        defaultWidth: isMobile ? 200 : 360,
        minWidth: 200,
        resizable: !isMobile,
        render: (plan, ctx) => (
          <div className="flex items-center gap-x-2 overflow-hidden">
            <span className="truncate min-w-0">
              {lastPathSegment(getPlanReleaseName(plan))}
            </span>
            <PlanLifecycleBadges ctx={ctx} />
          </div>
        ),
      },
      {
        key: "source",
        title: t("release.source"),
        defaultWidth: 160,
        minWidth: 120,
        resizable: true,
        render: (plan) => (
          <ReleaseSourceCell releaseName={getPlanReleaseName(plan)} />
        ),
      },
      createPlanStagesColumn(t),
      {
        key: "created",
        title: t("common.created-at"),
        defaultWidth: 152,
        minWidth: 100,
        resizable: true,
        render: (_plan, ctx) =>
          ctx.createTimeMs === undefined ? (
            <span className="text-control-light">-</span>
          ) : (
            <HumanizeTs
              tsMs={ctx.createTimeMs}
              className="text-control-light whitespace-nowrap"
            />
          ),
      },
    ],
    [t, isMobile]
  );
}

const VCS_ICONS: Partial<Record<VCSType, string>> = {
  [VCSType.GITHUB]: githubIcon,
  [VCSType.GITLAB]: gitlabIcon,
  [VCSType.BITBUCKET]: bitbucketIcon,
  [VCSType.AZURE_DEVOPS]: azureDevOpsIcon,
};

function ReleaseSourceCell({ releaseName }: { releaseName: string }) {
  const release = useAppStore((s) => s.releasesByName[releaseName]);
  const url = release?.vcsSource?.url ?? "";
  const commit = lastPathSegment(url);
  if (!release || release.state === State.DELETED || !commit) {
    return <span className="text-control-light">-</span>;
  }
  const icon =
    VCS_ICONS[release.vcsSource?.vcsType ?? VCSType.VCS_TYPE_UNSPECIFIED];
  return (
    <a
      href={url}
      target="_blank"
      rel="noopener noreferrer"
      title={url}
      className="inline-flex items-center gap-x-1 text-accent hover:underline"
      onClick={(e) => e.stopPropagation()}
    >
      {icon ? (
        <img src={icon} alt="" className="size-4" />
      ) : (
        <GitCommitHorizontal className="size-4" />
      )}
      <span className="font-mono text-sm">{commit.slice(0, 7)}</span>
    </a>
  );
}
