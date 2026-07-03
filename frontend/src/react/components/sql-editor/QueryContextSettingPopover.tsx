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
import { RadioGroup, RadioGroupItem } from "@/react/components/ui/radio-group";
import { Tooltip } from "@/react/components/ui/tooltip";
import {
  useConnectionOfCurrentSQLEditorTab,
  useSQLEditorQueryDataPolicy,
} from "@/react/hooks/useSQLEditorBridge";
import { cn } from "@/react/lib/utils";
import {
  getSQLEditorEditorState,
  useSQLEditorEditorState,
} from "@/react/stores/sqlEditor/editor";
import {
  getSQLEditorTabsState,
  useSQLEditorTabState,
} from "@/react/stores/sqlEditor/tab";
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
  const { database, connection: connectionRef } =
    useConnectionOfCurrentSQLEditorTab();
  const project = useSQLEditorEditorState((s) => s.project);

  const currentTabMode = useSQLEditorTabState(
    (s) => s.tabsById.get(s.currentTabId)?.mode
  );
  const resultRowsLimit = useSQLEditorEditorState((s) => s.resultRowsLimit);
  const redisCommandOption = useSQLEditorEditorState(
    (s) => s.redisCommandOption
  );
  const queryDataPolicy = useSQLEditorQueryDataPolicy(project);

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
    getSQLEditorTabsState().updateCurrentTab({ connection: nextConnection });
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
            <RadioGroup
              className="max-w-44 flex-col items-stretch gap-1"
              value={selectedDataSourceId}
              onValueChange={(value) => onDataSourceSelected(String(value))}
            >
              <RadioGroupItem value="">
                {t("data-source.automatic-query-data-source")}
              </RadioGroupItem>
              {dataSources.map((ds) => {
                const reason = dataSourceUnaccessibleReason(ds);
                const radio = (
                  <RadioGroupItem
                    value={ds.id}
                    disabled={Boolean(reason)}
                    contentClassName="min-w-0"
                  >
                    <div className="max-w-36 flex items-center">
                      <span className="text-xs opacity-60 shrink-0">
                        {readableDataSourceType(ds.type)}
                      </span>
                      <span className="ml-1 truncate">{ds.username}</span>
                    </div>
                  </RadioGroupItem>
                );
                return (
                  <Tooltip key={ds.id} content={reason} side="right">
                    {radio}
                  </Tooltip>
                );
              })}
            </RadioGroup>
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
                <RadioGroup
                  className="max-w-44 flex-col items-stretch gap-1"
                  value={String(redisCommandOption)}
                  onValueChange={(value) => {
                    getSQLEditorEditorState().setRedisCommandOption(
                      Number(value) as QueryOption_RedisRunCommandsOn
                    );
                  }}
                  aria-disabled={
                    selectedDataSource?.redisType !==
                    DataSource_RedisType.CLUSTER
                  }
                >
                  <RadioGroupItem
                    value={String(QueryOption_RedisRunCommandsOn.SINGLE_NODE)}
                    disabled={
                      selectedDataSource?.redisType !==
                      DataSource_RedisType.CLUSTER
                    }
                  >
                    {t("sql-editor.redis-command.single-node")}
                  </RadioGroupItem>
                  <RadioGroupItem
                    value={String(QueryOption_RedisRunCommandsOn.ALL_NODES)}
                    disabled={
                      selectedDataSource?.redisType !==
                      DataSource_RedisType.CLUSTER
                    }
                  >
                    {t("sql-editor.redis-command.all-nodes")}
                  </RadioGroupItem>
                </RadioGroup>
              </div>
            </Tooltip>
          )}

          {/* Max row count */}
          <div className="border-t pt-1">
            <MaxRowCountSelect
              value={resultRowsLimit ?? 1000}
              onChange={(n) => {
                getSQLEditorEditorState().setResultRowsLimit(n);
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
