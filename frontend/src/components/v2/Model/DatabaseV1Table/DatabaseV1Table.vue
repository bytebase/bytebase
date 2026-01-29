<template>
  <div ref="tableRef">
    <NDataTable
      key="database-table"
      size="small"
      :columns="columnList"
      :data="databaseList"
      :striped="true"
      :bordered="bordered"
      :pagination="pagination"
      :loading="loading"
      :scroll-x="scrollX"
      :row-key="(data: Database) => data.name"
      :checked-row-keys="props.selectedDatabaseNames"
      :row-props="rowProps"
      @update:checked-row-keys="
        (val) => $emit('update:selected-database-names', val as string[])
      "
      @update:sorter="$emit('update:sorters', $event)"
    />
  </div>
</template>

<script setup lang="tsx">
import { useElementSize } from "@vueuse/core";
import { CheckCircleIcon, XCircleIcon } from "lucide-vue-next";
import {
  type DataTableColumn,
  type DataTableSortState,
  NDataTable,
  NTooltip,
} from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { EnvironmentV1Name, InstanceV1Name } from "@/components/v2";
import {
  DatabaseNameCell,
  LabelsCell,
  ProjectNameCell,
} from "@/components/v2/Model/cells";
import {
  type Database,
  SyncStatus,
} from "@/types/proto-es/v1/database_service_pb";
import {
  getDatabaseEnvironment,
  getDatabaseProject,
  getInstanceResource,
  hostPortOfInstanceV1,
  TailwindBreakpoints,
} from "@/utils";
import { extractReleaseUID } from "@/utils/v1/release";
import { mapSorterStatus } from "../utils";

type DatabaseDataTableColumn = DataTableColumn<Database> & {
  hide?: boolean;
};

export type Mode =
  | "ALL"
  | "ALL_SHORT"
  | "ALL_TINY"
  | "INSTANCE"
  | "PROJECT"
  | "PROJECT_SHORT";

const props = withDefaults(
  defineProps<{
    mode?: Mode;
    databaseList: Database[];
    bordered?: boolean;
    loading?: boolean;
    showSelection?: boolean;
    singleSelection?: boolean;
    schemaless?: boolean;
    rowClickable?: boolean;
    selectedDatabaseNames?: string[];
    keyword?: string;
    rowClick?: (e: MouseEvent, val: Database) => void;
    selectDisabled?: (db: Database) => boolean;
    pagination?:
      | false
      | {
          defaultPageSize: number;
          disabled: boolean;
        };
    sorters?: DataTableSortState[];
  }>(),
  {
    mode: "ALL",
    bordered: true,
    showSelection: true,
    rowClickable: true,
    keyword: undefined,
    selectedDatabaseNames: () => [],
    pagination: false,
    selectDisabled: (_: Database) => false,
  }
);

const emit = defineEmits<{
  (event: "update:selected-database-names", val: string[]): void;
  (event: "update:sorters", sorters: DataTableSortState[]): void;
}>();

const { t } = useI18n();
const tableRef = ref<HTMLDivElement>();
const { width: tableWidth } = useElementSize(tableRef);
const showExtendedColumns = computed(
  () => tableWidth.value > TailwindBreakpoints.md
);

const columnList = computed((): DatabaseDataTableColumn[] => {
  const SELECTION: DatabaseDataTableColumn = {
    type: "selection",
    multiple: !props.singleSelection,
    hide: !props.showSelection,
    disabled: props.selectDisabled,
    cellProps: () => {
      return {
        onClick: (e: MouseEvent) => {
          e.stopPropagation();
        },
      };
    },
  };
  const NAME: DatabaseDataTableColumn = {
    key: "name",
    title: t("common.name"),
    minWidth: 120,
    resizable: true,
    render: (data) => {
      return <DatabaseNameCell database={data} keyword={props.keyword} />;
    },
  };
  const ENVIRONMENT: DatabaseDataTableColumn = {
    key: "environment",
    title: t("common.environment"),
    minWidth: 120,
    resizable: true,
    render: (data) => (
      <EnvironmentV1Name
        environment={getDatabaseEnvironment(data)}
        link={false}
        showColor={true}
        keyword={props.keyword}
        nullEnvironmentPlaceholder="Null"
      />
    ),
  };
  const SCHEMA_VERSION: DatabaseDataTableColumn = {
    key: "release",
    title: t("common.release"),
    minWidth: 140,
    resizable: true,
    hide: props.schemaless || !showExtendedColumns.value,
    render: (data) => {
      const release = data.release;
      return release ? extractReleaseUID(release) : "-";
    },
  };
  const PROJECT: DatabaseDataTableColumn = {
    key: "project",
    title: t("common.project"),
    resizable: true,
    ellipsis: true,
    hide: !showExtendedColumns.value,
    render: (data) => (
      <ProjectNameCell
        project={getDatabaseProject(data)}
        keyword={props.keyword}
      />
    ),
  };
  const INSTANCE: DatabaseDataTableColumn = {
    key: "instance",
    title: t("common.instance"),
    minWidth: 120,
    resizable: true,
    render: (data) => (
      <InstanceV1Name
        instance={getInstanceResource(data)}
        link={false}
        tag="div"
      />
    ),
  };
  const ADDRESS: DatabaseDataTableColumn = {
    key: "address",
    title: t("common.address"),
    resizable: true,
    hide: !showExtendedColumns.value,
    ellipsis: {
      tooltip: true,
    },
    render: (data) => hostPortOfInstanceV1(getInstanceResource(data)),
  };
  const DATABASE_LABELS: DatabaseDataTableColumn = {
    key: "labels",
    title: t("common.labels"),
    resizable: true,
    hide: !showExtendedColumns.value,
    render: (data) => (
      <LabelsCell labels={data.labels} showCount={1} placeholder="-" />
    ),
  };
  const SYNC_STATUS: DatabaseDataTableColumn = {
    key: "sync-status",
    title: t("database.sync-status"),
    minWidth: 100,
    render: (data) => {
      if (data.syncStatus === SyncStatus.FAILED) {
        return (
          <NTooltip>
            {{
              trigger: () => <XCircleIcon class="w-4 h-4 text-error" />,
              default: () => data.syncError || t("database.sync-status-failed"),
            }}
          </NTooltip>
        );
      }
      return <CheckCircleIcon class="w-4 h-4 text-success" />;
    },
  };

  const columnsMap = new Map<Mode, DatabaseDataTableColumn[]>([
    [
      "ALL",
      [
        NAME,
        ENVIRONMENT,
        SCHEMA_VERSION,
        PROJECT,
        INSTANCE,
        ADDRESS,
        DATABASE_LABELS,
        SYNC_STATUS,
      ],
    ],
    [
      "ALL_SHORT",
      [NAME, ENVIRONMENT, SCHEMA_VERSION, PROJECT, INSTANCE, SYNC_STATUS],
    ],
    ["ALL_TINY", [NAME, ENVIRONMENT, PROJECT, INSTANCE, SYNC_STATUS]],
    [
      "INSTANCE",
      [
        NAME,
        ENVIRONMENT,
        SCHEMA_VERSION,
        PROJECT,
        DATABASE_LABELS,
        SYNC_STATUS,
      ],
    ],
    [
      "PROJECT",
      [
        NAME,
        ENVIRONMENT,
        SCHEMA_VERSION,
        INSTANCE,
        ADDRESS,
        DATABASE_LABELS,
        SYNC_STATUS,
      ],
    ],
    [
      "PROJECT_SHORT",
      [NAME, ENVIRONMENT, SCHEMA_VERSION, INSTANCE, ADDRESS, SYNC_STATUS],
    ],
  ]);

  const columns: DatabaseDataTableColumn[] = (
    [
      SELECTION,
      ...(columnsMap.get(props.mode) || []),
    ] as DatabaseDataTableColumn[]
  ).filter((column) => !column.hide);
  return mapSorterStatus(columns, props.sorters);
});

const scrollX = computed(() => {
  return columnList.value.reduce((sum, col) => {
    return sum + ((col as { minWidth?: number }).minWidth ?? 100);
  }, 0);
});

const rowProps = (database: Database) => {
  return {
    style: props.rowClickable ? "cursor: pointer;" : "",
    onClick: (e: MouseEvent) => {
      if (!props.rowClickable) {
        return;
      }

      if (props.singleSelection) {
        emit("update:selected-database-names", [database.name]);
      } else {
        const selectedDatabaseNameList = new Set(props.selectedDatabaseNames);
        if (selectedDatabaseNameList.has(database.name)) {
          selectedDatabaseNameList.delete(database.name);
        } else {
          selectedDatabaseNameList.add(database.name);
        }
        emit("update:selected-database-names", [...selectedDatabaseNameList]);
      }

      if (props.rowClick) {
        props.rowClick(e, database);
      }
    },
  };
};
</script>
