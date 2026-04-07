import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  AdvancedSearch,
  getValueFromScopes,
  type ScopeOption,
  type SearchParams,
} from "@/react/components/AdvancedSearch";
import { DatabaseTable } from "@/react/components/database";
import {
  InstanceActionDropdown,
  InstanceFormBody,
  InstanceFormButtons,
  InstanceFormProvider,
  InstanceRoleTable,
  InstanceSyncButton,
} from "@/react/components/instance";
import { EngineIconPath } from "@/react/components/instance/constants";
import {
  Tabs,
  TabsList,
  TabsPanel,
  TabsTrigger,
} from "@/react/components/ui/tabs";
import { useVueState } from "@/react/hooks/useVueState";
import { useDatabaseV1Store, useInstanceV1Store } from "@/store";
import {
  environmentNamePrefix,
  instanceNamePrefix,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import type { DatabaseFilter } from "@/store/modules/v1/database";
import { State } from "@/types/proto-es/v1/common_pb";
import { instanceV1Name, setDocumentTitle } from "@/utils";

const instanceHashList = ["overview", "databases", "users"] as const;
type InstanceHash = (typeof instanceHashList)[number];
const isInstanceHash = (x: unknown): x is InstanceHash =>
  instanceHashList.includes(x as InstanceHash);

export function InstanceDetailPage({ instanceId }: { instanceId: string }) {
  const { t } = useTranslation();
  const instanceStore = useInstanceV1Store();
  const databaseStore = useDatabaseV1Store();
  const instanceName = `${instanceNamePrefix}${instanceId}`;
  const instance = useVueState(() =>
    instanceStore.getInstanceByName(instanceName)
  );

  const [selectedTab, setSelectedTab] = useState<InstanceHash>("overview");
  const [searchParams, setSearchParams] = useState<SearchParams>({
    query: "",
    scopes: [{ id: "instance", value: instanceId, readonly: true }],
  });
  // Sync tab with URL hash
  useEffect(() => {
    const hash = window.location.hash.replace(/^#?/, "");
    if (isInstanceHash(hash)) {
      setSelectedTab(hash);
    }
  }, []);

  useEffect(() => {
    const query = new URLSearchParams(window.location.search);
    query.delete("qs");
    const url = `${window.location.pathname}?${query.toString()}#${selectedTab}`;
    window.history.replaceState(null, "", url);
  }, [selectedTab]);

  // Set document title
  useEffect(() => {
    if (instance.title) {
      setDocumentTitle(instance.title);
    }
  }, [instance.title]);

  const syncSchema = useCallback(
    async (enableFullSync: boolean) => {
      await instanceStore.syncInstance(instance.name, enableFullSync);
      databaseStore.removeCacheByInstance(instance.name);
    },
    [instance.name, instanceStore, databaseStore]
  );

  // Database filter
  const envVal = getValueFromScopes(searchParams, "environment");
  const selectedEnvironment = envVal
    ? `${environmentNamePrefix}${envVal}`
    : undefined;
  const projectVal = getValueFromScopes(searchParams, "project");
  const selectedProject = projectVal
    ? `${projectNamePrefix}${projectVal}`
    : undefined;
  const selectedLabels = searchParams.scopes
    .filter((s) => s.id === "label")
    .map((s) => s.value);

  const filter: DatabaseFilter = useMemo(
    () => ({
      environment: selectedEnvironment,
      project: selectedProject,
      query: searchParams.query,
      labels: selectedLabels.length > 0 ? selectedLabels : undefined,
    }),
    [selectedEnvironment, selectedProject, searchParams.query, selectedLabels]
  );

  const scopeOptions: ScopeOption[] = useMemo(
    () => [
      {
        id: "environment",
        title: t("common.environment"),
        description: t("common.environment"),
      },
      {
        id: "project",
        title: t("common.project"),
        description: t("common.project"),
      },
      {
        id: "label",
        title: t("common.labels"),
        description: t("common.labels"),
        allowMultiple: true,
      },
    ],
    [t]
  );

  const handleTabChange = useCallback((tab: string | number | null) => {
    if (typeof tab === "string" && isInstanceHash(tab)) {
      setSelectedTab(tab);
    }
  }, []);

  const engineIconSrc = EngineIconPath[instance.engine];

  return (
    <div className="pt-4 flex flex-col gap-y-2 px-6">
      {/* Archive banner */}
      {instance.state === State.DELETED && (
        <div className="bg-gray-700 text-white text-center py-2 rounded-sm text-sm font-medium">
          {t("common.archived")}
        </div>
      )}

      {/* No environment warning */}
      {!instance.environment && (
        <div className="w-full mb-4 rounded-sm bg-yellow-50 border border-yellow-200 p-3 text-sm text-yellow-800">
          {t("instance.no-environment")}
        </div>
      )}

      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-x-2">
          {engineIconSrc && (
            <img src={engineIconSrc} alt="" className="h-6 w-6" />
          )}
          <span className="text-lg font-medium">
            {instanceV1Name(instance)}
          </span>
        </div>
        <div className="flex items-center gap-x-2">
          {instance.state === State.ACTIVE && (
            <InstanceSyncButton
              instanceName={instance.name}
              instanceTitle={instance.title}
              onSyncSchema={syncSchema}
            />
          )}
          <InstanceActionDropdown instance={instance} />
        </div>
      </div>

      {/* Tabs */}
      <Tabs value={selectedTab} onValueChange={handleTabChange}>
        <TabsList>
          <TabsTrigger value="overview">{t("common.overview")}</TabsTrigger>
          <TabsTrigger value="databases">{t("common.databases")}</TabsTrigger>
          <TabsTrigger value="users">{t("instance.users")}</TabsTrigger>
        </TabsList>

        <TabsPanel value="overview">
          <InstanceFormProvider instance={instance}>
            <div className="-mt-2">
              <InstanceFormBody />
              <InstanceFormButtons className="sticky bottom-0 z-10" />
            </div>
          </InstanceFormProvider>
        </TabsPanel>

        <TabsPanel value="databases">
          <div className="flex flex-col gap-y-2">
            <AdvancedSearch
              params={searchParams}
              onParamsChange={setSearchParams}
              placeholder={t("database.filter-database")}
              scopeOptions={scopeOptions}
            />
            <DatabaseTable filter={filter} parent={instance.name} mode="ALL" />
          </div>
        </TabsPanel>

        <TabsPanel value="users">
          <InstanceRoleTable instanceRoleList={instance.roles ?? []} />
        </TabsPanel>
      </Tabs>
    </div>
  );
}
