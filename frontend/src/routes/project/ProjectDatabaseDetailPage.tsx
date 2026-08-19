import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { LoaderCircle } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { router } from "@/app/router";
import { ComponentPermissionGuard } from "@/components/ComponentPermissionGuard";
import { TransferProjectSheet } from "@/components/database";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import {
  BatchUpdateDatabasesRequestSchema,
  DatabaseSchema$,
  UpdateDatabaseRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { autoDatabaseRoute, getDatabaseProject } from "@/utils";
import { DatabaseDetailActions } from "./database-detail/DatabaseDetailActions";
import { DatabaseDetailHeader } from "./database-detail/DatabaseDetailHeader";
import { DatabaseCatalogPanel } from "./database-detail/panels/DatabaseCatalogPanel";
import { DatabaseChangelogPanel } from "./database-detail/panels/DatabaseChangelogPanel";
import { DatabaseOverviewPanel } from "./database-detail/panels/DatabaseOverviewPanel";
import { DatabaseRevisionPanel } from "./database-detail/panels/DatabaseRevisionPanel";
import { DatabaseSettingsPanel } from "./database-detail/panels/DatabaseSettingsPanel";
import {
  PROJECT_DATABASE_DETAIL_TAB_CATALOG,
  PROJECT_DATABASE_DETAIL_TAB_CHANGELOG,
  PROJECT_DATABASE_DETAIL_TAB_OVERVIEW,
  PROJECT_DATABASE_DETAIL_TAB_REVISION,
  PROJECT_DATABASE_DETAIL_TAB_SETTING,
  type ProjectDatabaseDetailTab,
  parseProjectDatabaseDetailTabHash,
} from "./database-detail/tabs";
import { useProjectDatabaseDetail } from "./database-detail/useProjectDatabaseDetail";

export interface ProjectDatabaseDetailPageProps {
  projectId: string;
  instanceId: string;
  databaseName: string;
  routeHash?: string;
  routeQuery?: Record<string, string | undefined>;
}

export function ProjectDatabaseDetailPage({
  projectId,
  instanceId,
  databaseName,
  routeHash: hash,
  routeQuery: query,
}: ProjectDatabaseDetailPageProps) {
  const { t } = useTranslation();
  const parent = query?.parent ?? `instances/${instanceId}`;
  const databaseRouteQuery = useMemo(
    () => ({ ...query, parent }),
    [parent, query]
  );
  const detail = useProjectDatabaseDetail({
    parent,
    projectId,
    instanceId,
    databaseName,
  });
  const [selectedTab, setSelectedTab] = useState<ProjectDatabaseDetailTab>(() =>
    parseProjectDatabaseDetailTabHash(hash)
  );
  const [showTransferDrawer, setShowTransferDrawer] = useState(false);

  const handleTabChange = useCallback(
    (tab: string | number | null) => {
      if (typeof tab !== "string") {
        return;
      }

      const nextTab = parseProjectDatabaseDetailTabHash(tab);
      setSelectedTab(nextTab);
      const databaseRoute = autoDatabaseRoute(detail.database);
      void router.replace({
        ...databaseRoute,
        hash: `#${nextTab}`,
        query: {
          ...databaseRouteQuery,
          ...databaseRoute.query,
        },
      });
    },
    [databaseRouteQuery, detail.database]
  );

  useEffect(() => {
    setSelectedTab(parseProjectDatabaseDetailTabHash(hash));
  }, [hash]);

  const handleSetEnvironment = useCallback(() => {
    handleTabChange(PROJECT_DATABASE_DETAIL_TAB_SETTING);
  }, [handleTabChange]);

  const handleTransferProject = useCallback(
    async (projectName: string) => {
      try {
        await useAppStore.getState().batchUpdateDatabases(
          create(BatchUpdateDatabasesRequestSchema, {
            parent: "-",
            requests: [
              create(UpdateDatabaseRequestSchema, {
                database: create(DatabaseSchema$, {
                  name: detail.database.name,
                  project: projectName,
                }),
                updateMask: create(FieldMaskSchema, {
                  paths: ["project"],
                }),
              }),
            ],
          })
        );
        const updatedDatabase = await useAppStore
          .getState()
          .getOrFetchDatabaseByName(detail.database.name);
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("database.successfully-transferred-databases"),
        });
        setShowTransferDrawer(false);
        const databaseRoute = autoDatabaseRoute(updatedDatabase);
        void router.replace({
          ...databaseRoute,
          hash: `#${selectedTab}`,
          query: {
            ...databaseRouteQuery,
            ...databaseRoute.query,
          },
        });
      } catch {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("common.failed"),
        });
      }
    },
    [databaseRouteQuery, detail.database, selectedTab, t]
  );

  if (!detail.ready) {
    return (
      <div className="flex min-h-full items-center justify-center p-4">
        <LoaderCircle className="h-4 w-4 animate-spin text-control-light" />
      </div>
    );
  }

  return (
    <div className="flex min-h-full flex-col gap-y-4 p-4">
      {!detail.database.effectiveEnvironment && (
        <Alert
          variant="warning"
          description={
            <div className="flex flex-row items-center justify-between gap-x-2">
              <div>{t("instance.no-environment")}</div>
              <Button size="sm" onClick={handleSetEnvironment}>
                {t("database.edit-environment")}
              </Button>
            </div>
          }
        />
      )}

      <div className="flex flex-col items-start gap-y-2 xl:flex-row xl:items-center xl:justify-between xl:gap-x-2">
        <DatabaseDetailHeader database={detail.database} />
        <DatabaseDetailActions
          database={detail.database}
          isDefaultProject={detail.isDefaultProject}
          onOpenTransferProject={() => setShowTransferDrawer(true)}
        />
      </div>

      <Tabs value={selectedTab} onValueChange={handleTabChange}>
        <TabsList>
          <TabsTrigger value={PROJECT_DATABASE_DETAIL_TAB_OVERVIEW}>
            {t("common.overview")}
          </TabsTrigger>
          <TabsTrigger value={PROJECT_DATABASE_DETAIL_TAB_CHANGELOG}>
            {t("common.changelog")}
          </TabsTrigger>
          <TabsTrigger value={PROJECT_DATABASE_DETAIL_TAB_REVISION}>
            {t("database.revision.self")}
          </TabsTrigger>
          <TabsTrigger value={PROJECT_DATABASE_DETAIL_TAB_CATALOG}>
            {t("common.catalog")}
          </TabsTrigger>
          <TabsTrigger value={PROJECT_DATABASE_DETAIL_TAB_SETTING}>
            {t("common.settings")}
          </TabsTrigger>
        </TabsList>
      </Tabs>

      {selectedTab === PROJECT_DATABASE_DETAIL_TAB_SETTING && (
        <DatabaseSettingsPanel database={detail.database} />
      )}
      {selectedTab === PROJECT_DATABASE_DETAIL_TAB_OVERVIEW && (
        <DatabaseOverviewPanel database={detail.database} />
      )}
      {selectedTab === PROJECT_DATABASE_DETAIL_TAB_CHANGELOG && (
        <ComponentPermissionGuard
          project={getDatabaseProject(detail.database)}
          permissions={["bb.changelogs.list"]}
        >
          <DatabaseChangelogPanel database={detail.database} />
        </ComponentPermissionGuard>
      )}
      {selectedTab === PROJECT_DATABASE_DETAIL_TAB_REVISION && (
        <ComponentPermissionGuard
          project={getDatabaseProject(detail.database)}
          permissions={["bb.revisions.list"]}
        >
          <DatabaseRevisionPanel database={detail.database} />
        </ComponentPermissionGuard>
      )}
      {selectedTab === PROJECT_DATABASE_DETAIL_TAB_CATALOG && (
        <ComponentPermissionGuard
          project={getDatabaseProject(detail.database)}
          permissions={["bb.databaseCatalogs.get"]}
        >
          <DatabaseCatalogPanel database={detail.database} />
        </ComponentPermissionGuard>
      )}

      <TransferProjectSheet
        open={showTransferDrawer}
        databases={[detail.database]}
        onClose={() => setShowTransferDrawer(false)}
        onTransfer={handleTransferProject}
      />
    </div>
  );
}
