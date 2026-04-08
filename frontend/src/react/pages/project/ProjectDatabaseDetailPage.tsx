import { LoaderCircle } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import type { LocationQueryRaw } from "vue-router";
import { Tabs, TabsList, TabsTrigger } from "@/react/components/ui/tabs";
import { router } from "@/router";
import { PROJECT_V1_ROUTE_DATABASE_DETAIL } from "@/router/dashboard/projectV1";
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
  hash?: string;
  query?: LocationQueryRaw;
}

export function ProjectDatabaseDetailPage({
  projectId,
  instanceId,
  databaseName,
  hash,
  query,
}: ProjectDatabaseDetailPageProps) {
  const { t } = useTranslation();
  const detail = useProjectDatabaseDetail({
    projectId,
    instanceId,
    databaseName,
    hash,
    query,
  });
  const [selectedTab, setSelectedTab] = useState<ProjectDatabaseDetailTab>(() =>
    parseProjectDatabaseDetailTabHash(hash)
  );

  const handleTabChange = useCallback(
    (tab: string | number | null) => {
      if (typeof tab !== "string") {
        return;
      }

      const nextTab = parseProjectDatabaseDetailTabHash(tab);
      setSelectedTab(nextTab);
      void router.replace({
        name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
        params: {
          projectId,
          instanceId,
          databaseName,
        },
        hash: `#${nextTab}`,
        query: query ?? {},
      });
    },
    [databaseName, instanceId, projectId, query]
  );

  useEffect(() => {
    setSelectedTab(parseProjectDatabaseDetailTabHash(hash));
  }, [hash]);

  if (!detail.ready) {
    return (
      <div className="flex min-h-full items-center justify-center p-4">
        <LoaderCircle className="h-4 w-4 animate-spin text-control-light" />
      </div>
    );
  }

  return (
    <div className="flex min-h-full flex-col p-4">
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
    </div>
  );
}
