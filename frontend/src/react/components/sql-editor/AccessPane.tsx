import { Loader2 } from "lucide-react";
import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import type {
  ScopeOption,
  SearchParams,
} from "@/react/components/AdvancedSearch";
import { AdvancedSearch } from "@/react/components/AdvancedSearch";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import { Button } from "@/react/components/ui/button";
import { useVueState } from "@/react/hooks/useVueState";
import {
  hasFeature,
  useAccessGrantStore,
  useDatabaseV1Store,
  useIssueV1Store,
  useProjectV1Store,
  useSQLEditorStore,
  useSQLEditorUIStore,
} from "@/store";
import type { AccessFilter } from "@/store/modules/accessGrant";
import type { AccessGrant } from "@/types/proto-es/v1/access_grant_service_pb";
import { AccessGrant_Status } from "@/types/proto-es/v1/access_grant_service_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import type { AccessGrantFilterStatus } from "@/utils";
import { getDefaultPagination } from "@/utils";
import { sqlEditorEvents } from "@/views/sql-editor/events";
import { AccessGrantItem } from "./AccessGrantItem";
import { AccessGrantRequestDrawer } from "./AccessGrantRequestDrawer";

const PAGE_SIZE = getDefaultPagination();

const DEFAULT_SCOPES = [
  { id: "status", value: AccessGrant_Status[AccessGrant_Status.ACTIVE] },
  { id: "status", value: AccessGrant_Status[AccessGrant_Status.PENDING] },
];

export function AccessPane() {
  const { t } = useTranslation();

  const projectStore = useProjectV1Store();
  const editorStore = useSQLEditorStore();
  const accessGrantStore = useAccessGrantStore();
  const issueStore = useIssueV1Store();
  const databaseStore = useDatabaseV1Store();
  const uiStore = useSQLEditorUIStore();

  const [showDrawer, setShowDrawer] = useState(false);
  const [loading, setLoading] = useState(false);
  const [pendingCreate, setPendingCreate] = useState<AccessGrant | undefined>(
    undefined
  );
  const [accessGrantList, setAccessGrantList] = useState<AccessGrant[]>([]);
  const [nextPageToken, setNextPageToken] = useState("");
  const nextPageTokenRef = useRef(nextPageToken);
  nextPageTokenRef.current = nextPageToken;
  const [issueByGrantName, setIssueByGrantName] = useState<Map<string, Issue>>(
    new Map()
  );
  const [useSmallLayout, setUseSmallLayout] = useState(false);

  const containerRef = useRef<HTMLDivElement>(null);

  const [searchParams, setSearchParams] = useState<SearchParams>({
    query: "",
    scopes: DEFAULT_SCOPES,
  });

  const projectName = useVueState(() => editorStore.project);
  const highlightAccessGrantName = useVueState(
    () => uiStore.highlightAccessGrantName
  );

  const project = useMemo(() => {
    if (!projectName) return undefined;
    return projectStore.getProjectByName(projectName as string);
  }, [projectStore, projectName]);

  const hasJITFeature = useMemo(() => hasFeature(PlanFeature.FEATURE_JIT), []);

  // Build scope options for AdvancedSearch (React-compatible, no Vue renderers)
  const scopeOptions = useMemo((): ScopeOption[] => {
    return [
      {
        id: "status",
        title: t("common.status"),
        description: t("sql-editor.access-search.scope.status.description"),
        allowMultiple: true,
        options: [
          {
            value: AccessGrant_Status[AccessGrant_Status.ACTIVE],
            keywords: ["active"],
            render: () => t("common.active"),
          },
          {
            value: AccessGrant_Status[AccessGrant_Status.PENDING],
            keywords: ["pending"],
            render: () => t("common.pending"),
          },
          {
            value: "EXPIRED",
            keywords: ["expired"],
            render: () => t("sql-editor.expired"),
          },
          {
            value: AccessGrant_Status[AccessGrant_Status.REVOKED],
            keywords: ["revoked"],
            render: () => t("common.revoked"),
          },
        ],
      },
      {
        id: "database",
        title: t("common.database"),
        description: t("sql-editor.access-search.scope.database.description"),
        onSearch: async (keyword: string) => {
          const parent = projectName as string | undefined;
          if (!parent) return [];
          const result = await databaseStore.fetchDatabases({
            parent,
            filter: { query: keyword },
            pageSize: getDefaultPagination(),
            silent: true,
          });
          return result.databases.map((db) => ({
            value: db.name,
            keywords: [db.name],
          }));
        },
      },
    ];
  }, [t, projectName, databaseStore]);

  // Build AccessFilter from React SearchParams
  const filter = useMemo((): AccessFilter => {
    const selectedStatuses = searchParams.scopes
      .filter((s) => s.id === "status")
      .map((s) => s.value) as AccessGrantFilterStatus[];

    const databaseScope = searchParams.scopes.find((s) => s.id === "database");

    const f: AccessFilter = {
      status: selectedStatuses,
    };
    if (databaseScope?.value) {
      f.target = databaseScope.value;
    }
    const queryText = searchParams.query.trim();
    if (queryText) {
      f.statement = queryText;
    }
    return f;
  }, [searchParams]);

  const fetchIssuesForPendingGrants = useCallback(
    async (grants: AccessGrant[]) => {
      const pendingWithIssue = grants.filter(
        (g) => g.status === AccessGrant_Status.PENDING && g.issue
      );
      const results = await Promise.all(
        pendingWithIssue.map(async (g) => {
          try {
            const issue = await issueStore.fetchIssueByName(g.issue, true);
            return { grantName: g.name, issue };
          } catch {
            return undefined;
          }
        })
      );
      setIssueByGrantName((prev) => {
        const next = new Map(prev);
        for (const r of results) {
          if (r) {
            next.set(r.grantName, r.issue);
          }
        }
        return next;
      });
    },
    [issueStore]
  );

  const fetchAccessGrants = useCallback(
    async (resetList = true) => {
      const parent = projectName as string | undefined;
      if (!parent) return;

      setLoading(true);
      try {
        const response = await accessGrantStore.searchMyAccessGrants({
          parent,
          filter,
          pageSize: PAGE_SIZE,
          pageToken: resetList ? undefined : nextPageTokenRef.current,
        });
        if (resetList) {
          setAccessGrantList(response.accessGrants);
          setIssueByGrantName(new Map());
        } else {
          setAccessGrantList((prev) => [...prev, ...response.accessGrants]);
        }
        setNextPageToken(response.nextPageToken);
        await fetchIssuesForPendingGrants(response.accessGrants);
      } finally {
        setLoading(false);
      }
    },
    [projectName, filter, accessGrantStore, fetchIssuesForPendingGrants]
  );

  // Re-fetch when project or filter changes
  useEffect(() => {
    void fetchAccessGrants(true);
  }, [projectName, filter]);

  // Re-fetch + clear highlight when highlightAccessGrantName changes
  useEffect(() => {
    const name = highlightAccessGrantName;
    if (!name) return;
    void fetchAccessGrants(true);
    const timer = setTimeout(() => {
      if (uiStore.highlightAccessGrantName === name) {
        uiStore.highlightAccessGrantName = undefined;
      }
    }, 3000);
    return () => clearTimeout(timer);
  }, [highlightAccessGrantName]);

  // Responsive small layout detection
  useLayoutEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const observer = new ResizeObserver((entries) => {
      const width = entries[0]?.contentRect.width ?? 0;
      setUseSmallLayout(width > 0 && width < 250);
    });
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  const handleDrawerClose = useCallback(() => {
    setShowDrawer(false);
    setPendingCreate(undefined);
    void fetchAccessGrants(true);
  }, [fetchAccessGrants]);

  const handleRequest = useCallback((grant: AccessGrant) => {
    setPendingCreate(grant);
    setShowDrawer(true);
  }, []);

  const handleRun = useCallback(
    async (grant: AccessGrant) => {
      const database = grant.targets[0] ?? "";
      const instanceName = database.replace(/\/databases\/.*$/, "");
      await databaseStore.getOrFetchDatabaseByName(database);
      await sqlEditorEvents.emit("execute-sql", {
        connection: { instance: instanceName, database },
        statement: grant.query,
        batchQueryContext: { databases: grant.targets },
      });
    },
    [databaseStore]
  );

  return (
    <div className="relative w-full h-full flex flex-col justify-start items-start gap-y-1">
      <div
        ref={containerRef}
        className="w-full px-1 flex flex-wrap items-center gap-x-2 gap-y-2"
      >
        <AdvancedSearch
          params={searchParams}
          scopeOptions={scopeOptions}
          placeholder={t("issue.advanced-search.filter")}
          onParamsChange={setSearchParams}
        />
        <PermissionGuard
          permissions={["bb.accessGrants.create"]}
          project={project}
        >
          {({ disabled }) => (
            <Button
              variant="default"
              size="sm"
              style={{ width: useSmallLayout ? "100%" : "auto" }}
              disabled={!hasJITFeature || disabled}
              onClick={() => setShowDrawer(true)}
              className="ml-auto"
            >
              {!hasJITFeature && (
                <FeatureBadge
                  clickable={false}
                  feature={PlanFeature.FEATURE_JIT}
                  className="mr-1"
                />
              )}
              {t("sql-editor.request-access")}
            </Button>
          )}
        </PermissionGuard>
      </div>

      <div className="w-full border-t" />

      <div className="w-full flex flex-col justify-start items-start overflow-y-auto">
        {accessGrantList.map((grant) => (
          <AccessGrantItem
            key={grant.name}
            grant={grant}
            highlight={grant.name === highlightAccessGrantName}
            issue={issueByGrantName.get(grant.name)}
            onRun={(g) => void handleRun(g)}
            onRequest={handleRequest}
          />
        ))}
        {nextPageToken && (
          <div className="w-full flex flex-col items-center my-2">
            <Button
              variant="ghost"
              size="sm"
              disabled={loading}
              onClick={() => void fetchAccessGrants(false)}
            >
              {loading && <Loader2 className="size-4 mr-1 animate-spin" />}
              <span className="textinfolabel">{t("common.load-more")}</span>
            </Button>
          </div>
        )}
      </div>

      {accessGrantList.length === 0 &&
        (loading ? (
          <div className="absolute inset-0 flex items-center justify-center bg-background/75">
            <Loader2 className="size-6 animate-spin text-accent" />
          </div>
        ) : (
          <div className="w-full flex items-center justify-center py-8 textinfolabel">
            {t("sql-editor.no-access-requests")}
          </div>
        ))}

      {showDrawer && (
        <AccessGrantRequestDrawer
          query={pendingCreate?.query}
          unmask={pendingCreate?.unmask}
          targets={pendingCreate?.targets}
          onClose={handleDrawerClose}
        />
      )}
    </div>
  );
}
