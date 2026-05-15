import { orderBy } from "lodash-es";
import { ChevronDown } from "lucide-react";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { MaxRowCountSelect } from "@/react/components/MaxRowCountSelect";
import { Button } from "@/react/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/react/components/ui/popover";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { useSQLEditorVueState } from "@/react/stores/sqlEditor/editor-vue-state";
import {
  useConnectionOfCurrentSQLEditorTab,
  useSQLEditorTabStore,
} from "@/react/stores/sqlEditor/tab-vue-state";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { DataSource } from "@/types/proto-es/v1/instance_service_pb";
import {
  DataSource_RedisType,
  DataSourceType,
} from "@/types/proto-es/v1/instance_service_pb";
import { QueryOption_RedisRunCommandsOn } from "@/types/proto-es/v1/sql_service_pb";
import { getInstanceResource, readableDataSourceType } from "@/utils";

type Props = {
  readonly disabled?: boolean;
};

/**
 * Replaces frontend/src/views/sql-editor/EditorCommon/QueryContextSettingPopover.vue.
 * Popover for selecting query data source, Redis command mode, and max row count.
 */
export function QueryContextSettingPopover({ disabled = false }: Props) {
  const { t } = useTranslation();
  const tabStore = useSQLEditorTabStore();
  const editorStore = useSQLEditorVueState();
  const connection = useConnectionOfCurrentSQLEditorTab();

  const currentTabMode = useVueState(() => tabStore.currentTab?.mode);
  const database = useVueState(() => connection.database.value);
  const connectionRef = useVueState(() => connection.connection.value);
  const resultRowsLimit = useVueState(() => editorStore.resultRowsLimit);
  const redisCommandOption = useVueState(() => editorStore.redisCommandOption);
  const queryDataPolicy = useVueState(() => editorStore.queryDataPolicy);

  const show = currentTabMode !== "ADMIN";

  const instance = useMemo(() => getInstanceResource(database), [database]);

  const showRedisConfig = useMemo(
    () => instance.engine === Engine.REDIS,
    [instance]
  );

  const selectedDataSourceId = useMemo(
    () => connectionRef?.dataSourceId ?? "",
    [connectionRef]
  );

  const dataSources = useMemo(
    () => orderBy(instance.dataSources, "type"),
    [instance]
  );

  const selectedDataSource = useMemo(
    () => instance.dataSources.find((ds) => ds.id === selectedDataSourceId),
    [instance, selectedDataSourceId]
  );

  const dataSourceUnaccessibleReason = (
    dataSource: DataSource
  ): string | undefined => {
    if (!queryDataPolicy || queryDataPolicy.allowAdminDataSource) {
      return undefined;
    }
    if (dataSource.type !== DataSourceType.ADMIN) {
      return undefined;
    }
    const readOnlyDataSources = dataSources.filter(
      (ds) => ds.type === DataSourceType.READ_ONLY
    );
    if (readOnlyDataSources.length === 0) {
      return undefined;
    }
    return t(
      "sql-editor.query-context.admin-data-source-is-disallowed-to-query"
    );
  };

  const onDataSourceSelected = (dataSourceId: string) => {
    if (!connectionRef) return;
    const nextConnection = { ...connectionRef };
    if (dataSourceId) {
      nextConnection.dataSourceId = dataSourceId;
    } else {
      delete nextConnection.dataSourceId;
    }
    tabStore.updateCurrentTab({ connection: nextConnection });
  };

  if (!show) return null;

  return (
    <Popover>
      <PopoverTrigger
        render={
          <Button
            variant="default"
            size="sm"
            disabled={disabled}
            className={cn("h-7 px-1 gap-0", "rounded-none rounded-r-xs")}
          />
        }
      >
        <ChevronDown className="size-4" />
      </PopoverTrigger>
      <PopoverContent side="bottom" align="end">
        <div className="flex flex-col gap-1">
          {/* Data source radio group */}
          <div>
            <p className="mb-1 textinfolabel">
              {t("data-source.select-query-data-source")}
            </p>
            <div className="max-w-44 flex flex-col gap-1">
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="radio"
                  name="data-source"
                  checked={selectedDataSourceId === ""}
                  onChange={() => onDataSourceSelected("")}
                />
                <span>{t("data-source.automatic-query-data-source")}</span>
              </label>
              {dataSources.map((ds) => {
                const reason = dataSourceUnaccessibleReason(ds);
                const radio = (
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="radio"
                      name="data-source"
                      checked={selectedDataSourceId === ds.id}
                      disabled={Boolean(reason)}
                      onChange={() => onDataSourceSelected(ds.id)}
                    />
                    <div className="max-w-36 flex items-center">
                      <span className="text-xs opacity-60 shrink-0">
                        {readableDataSourceType(ds.type)}
                      </span>
                      <span className="ml-1 truncate">{ds.username}</span>
                    </div>
                  </label>
                );
                return (
                  <Tooltip key={ds.id} content={reason} side="right">
                    {radio}
                  </Tooltip>
                );
              })}
            </div>
          </div>

          {/* Redis cluster config */}
          {showRedisConfig && (
            <Tooltip
              content={
                selectedDataSource?.redisType === DataSource_RedisType.CLUSTER
                  ? null
                  : t("sql-editor.redis-command.only-for-cluster")
              }
              side="right"
            >
              <div className="border-t pt-1">
                <p className="mb-1 textinfolabel">
                  {t("sql-editor.redis-command.self")}
                </p>
                <div
                  className="max-w-44 flex flex-col gap-1"
                  aria-disabled={
                    selectedDataSource?.redisType !==
                    DataSource_RedisType.CLUSTER
                  }
                >
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="radio"
                      name="redis-command"
                      disabled={
                        selectedDataSource?.redisType !==
                        DataSource_RedisType.CLUSTER
                      }
                      checked={
                        redisCommandOption ===
                        QueryOption_RedisRunCommandsOn.SINGLE_NODE
                      }
                      onChange={() => {
                        editorStore.redisCommandOption =
                          QueryOption_RedisRunCommandsOn.SINGLE_NODE;
                      }}
                    />
                    <span>{t("sql-editor.redis-command.single-node")}</span>
                  </label>
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="radio"
                      name="redis-command"
                      disabled={
                        selectedDataSource?.redisType !==
                        DataSource_RedisType.CLUSTER
                      }
                      checked={
                        redisCommandOption ===
                        QueryOption_RedisRunCommandsOn.ALL_NODES
                      }
                      onChange={() => {
                        editorStore.redisCommandOption =
                          QueryOption_RedisRunCommandsOn.ALL_NODES;
                      }}
                    />
                    <span>{t("sql-editor.redis-command.all-nodes")}</span>
                  </label>
                </div>
              </div>
            </Tooltip>
          )}

          {/* Max row count */}
          <div className="border-t pt-1">
            <MaxRowCountSelect
              value={resultRowsLimit ?? 1000}
              onChange={(n) => {
                editorStore.resultRowsLimit = n;
              }}
              maximum={queryDataPolicy?.maximumResultRows ?? Number.MAX_VALUE}
              quaternary
            />
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
