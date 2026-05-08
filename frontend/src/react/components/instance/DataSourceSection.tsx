import { Plus } from "lucide-react";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";
import { useInstanceV1Store } from "@/store";
import { DATASOURCE_READONLY_USER_NAME } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { DataSourceType } from "@/types/proto-es/v1/instance_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import type { EditDataSource } from "./common";
import { wrapEditDataSource } from "./common";
import { DataSourceForm } from "./DataSourceForm";
import { useInstanceFormContext } from "./InstanceFormContext";
import type { InfoSection } from "./info-content";

interface DataSourceSectionProps {
  hideOptions?: boolean;
  onOpenInfoPanel?: (section: InfoSection) => void;
}

export function DataSourceSection({
  hideOptions = false,
  onOpenInfoPanel,
}: DataSourceSectionProps) {
  const { t } = useTranslation();
  const ctx = useInstanceFormContext();
  const {
    instance,
    isCreating,
    allowEdit,
    basicInfo,
    dataSourceEditState,
    setDataSourceEditState,
    adminDataSource,
    editingDataSource,
    readonlyDataSourceList,
    hasReadOnlyDataSource,
  } = ctx;
  const instanceStore = useInstanceV1Store();

  const allowUpdate = hasWorkspacePermissionV2("bb.instances.update");

  const handleCreateRODataSource = useCallback(() => {
    if (isCreating) return;
    const ds = {
      ...wrapEditDataSource(undefined),
      type: DataSourceType.READ_ONLY,
      host: adminDataSource.host,
      port: adminDataSource.port,
      database: adminDataSource.database,
      username: DATASOURCE_READONLY_USER_NAME,
    };
    if (
      basicInfo.engine === Engine.SPANNER ||
      basicInfo.engine === Engine.BIGQUERY ||
      basicInfo.engine === Engine.DYNAMODB
    ) {
      ds.host = adminDataSource.host;
    }
    setDataSourceEditState((prev) => ({
      dataSources: [...prev.dataSources, ds],
      editingDataSourceId: ds.id,
    }));
  }, [isCreating, adminDataSource, basicInfo.engine, setDataSourceEditState]);

  const handleDeleteDataSource = useCallback(
    async (ds: EditDataSource) => {
      if (instance && !ds.pendingCreate) {
        await instanceStore.deleteDataSource(instance, ds);
      }
      setDataSourceEditState((prev) => {
        const dataSources = prev.dataSources.filter((d) => d.id !== ds.id);
        let editingDataSourceId = prev.editingDataSourceId;
        if (ds.id === editingDataSourceId) {
          const index = prev.dataSources.findIndex((d) => d.id === ds.id);
          const siblingIndex = Math.min(index, dataSources.length - 1);
          editingDataSourceId = dataSources[siblingIndex]?.id;
        }
        return { dataSources, editingDataSourceId };
      });
    },
    [instance, instanceStore, setDataSourceEditState]
  );

  const handleDataSourceChange = useCallback(
    (updated: EditDataSource) => {
      setDataSourceEditState((prev) => ({
        ...prev,
        dataSources: prev.dataSources.map((ds) =>
          ds.id === updated.id ? updated : ds
        ),
      }));
    },
    [setDataSourceEditState]
  );

  const handleTabChange = useCallback(
    (tabId: string | number | null) => {
      if (typeof tabId === "string") {
        setDataSourceEditState((prev) => ({
          ...prev,
          editingDataSourceId: tabId,
        }));
      }
    },
    [setDataSourceEditState]
  );

  // Show RO tips when not creating and no RO data source
  const showROTips = !isCreating && !hasReadOnlyDataSource;

  return (
    <>
      {showROTips && (
        <Alert
          variant="warning"
          className="my-4"
          description={
            <div className="flex items-center justify-between gap-x-2">
              <span>{t("data-source.no-read-only-data-source")}</span>
              <Button
                variant="outline"
                size="sm"
                onClick={handleCreateRODataSource}
              >
                {t("common.create")}
              </Button>
            </div>
          }
        />
      )}

      <div className="mt-2 gap-y-2 gap-x-4 border-none">
        {/* Data source tabs */}
        {!isCreating && (
          <div className="mb-4 flex items-center gap-x-2 border-b border-block-border">
            <button
              type="button"
              className={`pb-2 px-1 text-sm font-medium border-b-2 transition-colors ${
                dataSourceEditState.editingDataSourceId === adminDataSource.id
                  ? "border-accent text-accent"
                  : "border-transparent text-control-light hover:text-main"
              }`}
              onClick={() => handleTabChange(adminDataSource.id)}
            >
              {t("common.admin")}
            </button>
            {readonlyDataSourceList.map((ds) => (
              <div key={ds.id} className="flex items-center">
                <button
                  type="button"
                  className={`pb-2 px-1 text-sm font-medium border-b-2 transition-colors ${
                    dataSourceEditState.editingDataSourceId === ds.id
                      ? "border-accent text-accent"
                      : "border-transparent text-control-light hover:text-main"
                  }`}
                  onClick={() => handleTabChange(ds.id)}
                >
                  {t("common.read-only")}
                </button>
                {hasReadOnlyDataSource && (
                  <button
                    type="button"
                    className="ml-1 text-red-500 hover:text-red-700 text-xs pb-2"
                    disabled={!allowUpdate}
                    onClick={() => {
                      if (
                        ds.pendingCreate ||
                        window.confirm(
                          `${t("data-source.delete-read-only-data-source")}?`
                        )
                      ) {
                        handleDeleteDataSource(ds);
                      }
                    }}
                  >
                    ✕
                  </button>
                )}
              </div>
            ))}
            {!hasReadOnlyDataSource && (
              <span className="pb-2 px-1 text-sm text-control-light">
                {t("common.read-only")}
              </span>
            )}
            {allowEdit && (
              <button
                type="button"
                className="pb-2 px-1 text-control-light hover:text-main disabled:opacity-50"
                disabled={!allowUpdate}
                onClick={handleCreateRODataSource}
              >
                <Plus className="w-4 h-4" />
              </button>
            )}
          </div>
        )}

        {editingDataSource && (
          <DataSourceForm
            dataSource={editingDataSource}
            hideOptions={hideOptions}
            onDataSourceChange={handleDataSourceChange}
            onOpenInfoPanel={onOpenInfoPanel}
          />
        )}
      </div>
    </>
  );
}
