import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { LoaderCircle } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import type { LocationQueryRaw } from "vue-router";
import { TransferProjectDrawer } from "@/react/components/database";
import { Alert } from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { Tabs, TabsList, TabsTrigger } from "@/react/components/ui/tabs";
import { router } from "@/router";
import { PROJECT_V1_ROUTE_DATABASE_DETAIL } from "@/router/dashboard/projectV1";
import { pushNotification, useDatabaseV1Store } from "@/store";
import {
  BatchUpdateDatabasesRequestSchema,
  DatabaseSchema$,
  UpdateDatabaseRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { DatabaseDetailActions } from "./database-detail/DatabaseDetailActions";
import { DatabaseDetailHeader } from "./database-detail/DatabaseDetailHeader";
import { DatabaseChangelogPanel } from "./database-detail/panels/DatabaseChangelogPanel";
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

const buildDatabaseDetailRoute = (
  database: {
    name: string;
    project: string;
  },
  options?: {
    hash?: string;
    query?: LocationQueryRaw;
  }
) => {
  const databaseMatches = database.name.match(
    /(?:^|\/)instances\/(?<instanceId>[^/]+)\/databases\/(?<databaseName>[^/]+)(?:$|\/)/
  );
  const projectMatches = database.project.match(
    /(?:^|\/)projects\/(?<projectId>[^/]+)(?:$|\/)/
  );

  return {
    name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
    params: {
      projectId: projectMatches?.groups?.projectId ?? "",
      instanceId: databaseMatches?.groups?.instanceId ?? "",
      databaseName: databaseMatches?.groups?.databaseName ?? "",
    },
    hash: options?.hash,
    query: options?.query,
  };
};

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
  const databaseStore = useDatabaseV1Store();
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
  const [showTransferDrawer, setShowTransferDrawer] = useState(false);
  const [showIncorrectProjectModal, setShowIncorrectProjectModal] =
    useState(false);

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

  const handleSetEnvironment = useCallback(() => {
    handleTabChange(PROJECT_DATABASE_DETAIL_TAB_SETTING);
  }, [handleTabChange]);

  const handleSQLEditorFailed = useCallback(() => {
    setShowIncorrectProjectModal(true);
  }, []);

  const handleTransferProject = useCallback(
    async (projectName: string) => {
      try {
        await databaseStore.batchUpdateDatabases(
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
        const updatedDatabase = await databaseStore.getOrFetchDatabaseByName(
          detail.database.name
        );
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("database.successfully-transferred-databases"),
        });
        setShowTransferDrawer(false);
        void router.replace(
          buildDatabaseDetailRoute(updatedDatabase, {
            hash: `#${selectedTab}`,
            query: query ?? {},
          })
        );
      } catch {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("common.failed"),
        });
      }
    },
    [databaseStore, detail.database, query, selectedTab, t]
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
        <Alert variant="warning" className="items-center justify-between">
          <div>{t("instance.no-environment")}</div>
          <Button variant="link" size="sm" onClick={handleSetEnvironment}>
            {t("database.edit-environment")}
          </Button>
        </Alert>
      )}

      <div className="flex flex-col items-start gap-4 xl:flex-row xl:items-start xl:justify-between">
        <DatabaseDetailHeader
          database={detail.database}
          allowAlterSchema={detail.allowAlterSchema}
          onSQLEditorFailed={handleSQLEditorFailed}
        />
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
      {selectedTab === PROJECT_DATABASE_DETAIL_TAB_CHANGELOG && (
        <DatabaseChangelogPanel database={detail.database} />
      )}
      {selectedTab === PROJECT_DATABASE_DETAIL_TAB_REVISION && (
        <DatabaseRevisionPanel database={detail.database} />
      )}

      <Dialog
        open={showIncorrectProjectModal}
        onOpenChange={setShowIncorrectProjectModal}
      >
        <DialogContent className="p-6">
          <DialogTitle>{t("common.warning")}</DialogTitle>
          <p className="mt-3 text-sm text-control-light">
            {t("common.missing-required-permission", {
              permissions: "bb.sql.select",
            })}
          </p>
          <div className="mt-6 flex justify-end gap-x-2">
            <Button
              variant="outline"
              onClick={() => setShowIncorrectProjectModal(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button
              onClick={() => {
                setShowIncorrectProjectModal(false);
                setShowTransferDrawer(true);
              }}
            >
              {t("database.transfer-project")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      <TransferProjectDrawer
        open={showTransferDrawer}
        databases={[detail.database]}
        onClose={() => setShowTransferDrawer(false)}
        onTransfer={handleTransferProject}
      />
    </div>
  );
}
