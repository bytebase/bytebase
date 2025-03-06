<template>
  <NDataTable
    key="database-table"
    size="small"
    :columns="columnList"
    :data="databaseList"
    :striped="true"
    :bordered="bordered"
    :loading="loading"
    :row-key="(data: ComposedDatabase) => data.name"
    :checked-row-keys="Array.from(state.selectedDatabaseNameList)"
    :row-props="rowProps"
    @update:checked-row-keys="
      (val) => (state.selectedDatabaseNameList = new Set(val as string[]))
    "
  />
</template>

<script setup lang="tsx">
import { NDataTable, type DataTableColumn } from "naive-ui";
import { computed, reactive } from "vue";
import { watch } from "vue";
import { useI18n } from "vue-i18n";
import { EnvironmentV1Name, InstanceV1Name } from "@/components/v2";
import type { ComposedDatabase } from "@/types";
import { hostPortOfInstanceV1 } from "@/utils";
import { DatabaseNameCell, ProjectNameCell, DatabaseLabelsCell } from "./cells";

interface LocalState {
  // The selected database name list.
  selectedDatabaseNameList: Set<string>;
}

type DatabaseDataTableColumn = DataTableColumn<ComposedDatabase> & {
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
    databaseList: ComposedDatabase[];
    bordered?: boolean;
    loading?: boolean;
    showSelection?: boolean;
    singleSelection?: boolean;
    schemaless?: boolean;
    customClick?: boolean;
    rowClickable?: boolean;
    selectedDatabaseNames?: string[];
    keyword?: string;
  }>(),
  {
    mode: "ALL",
    bordered: true,
    showSelection: true,
    rowClickable: true,
    keyword: undefined,
    selectedDatabaseNames: () => [],
  }
);

const emit = defineEmits<{
  (event: "row-click", e: MouseEvent, val: ComposedDatabase): void;
  (event: "update:selected-databases", val: Set<string>): void;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  selectedDatabaseNameList: new Set(props.selectedDatabaseNames),
});

const columnList = computed((): DatabaseDataTableColumn[] => {
  const SELECTION: DatabaseDataTableColumn = {
    type: "selection",
    multiple: !props.singleSelection,
    hide: !props.showSelection,
    cellProps: () => {
      return {
        onClick: (e: MouseEvent) => {
          e.stopPropagation();
        },
      };
    },
  };
  const NAME: DatabaseDataTableColumn = {
    key: "title",
    title: t("common.name"),
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
        environment={data.effectiveEnvironmentEntity}
        link={false}
        tag="div"
        keyword={props.keyword}
      />
    ),
  };
  const SCHEMA_VERSION: DatabaseDataTableColumn = {
    key: "schema-version",
    title: t("common.schema-version"),
    resizable: true,
    hide: props.schemaless,
    render: (data) => (data as ComposedDatabase).schemaVersion || "-",
  };
  const PROJECT: DatabaseDataTableColumn = {
    key: "project",
    title: t("common.project"),
    resizable: true,
    render: (data) => (
      <ProjectNameCell
        project={data.projectEntity}
        mode={props.mode}
        keyword={props.keyword}
      />
    ),
  };
  const INSTANCE: DatabaseDataTableColumn = {
    key: "instance",
    title: t("common.instance"),
    resizable: true,
    render: (data) => (
      <InstanceV1Name instance={data.instanceResource} link={false} tag="div" />
    ),
  };
  const ADDRESS: DatabaseDataTableColumn = {
    key: "address",
    title: t("common.address"),
    resizable: true,
    render: (data) => hostPortOfInstanceV1(data.instanceResource),
  };
  const DATABASE_LABELS: DatabaseDataTableColumn = {
    key: "labels",
    title: t("common.labels"),
    resizable: true,
    render: (data) => (
      <DatabaseLabelsCell labels={data.labels} showCount={1} placeholder="-" />
    ),
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
      ],
    ],
    ["ALL_SHORT", [NAME, ENVIRONMENT, SCHEMA_VERSION, PROJECT, INSTANCE]],
    ["ALL_TINY", [NAME, ENVIRONMENT, PROJECT, INSTANCE]],
    ["INSTANCE", [NAME, ENVIRONMENT, SCHEMA_VERSION, PROJECT, DATABASE_LABELS]],
    [
      "PROJECT",
      [NAME, ENVIRONMENT, SCHEMA_VERSION, INSTANCE, ADDRESS, DATABASE_LABELS],
    ],
    ["PROJECT_SHORT", [NAME, ENVIRONMENT, SCHEMA_VERSION, INSTANCE, ADDRESS]],
  ]);

  return [SELECTION, ...(columnsMap.get(props.mode) || [])].filter(
    (column) => !column.hide
  );
});

const rowProps = (database: ComposedDatabase) => {
  return {
    style: props.rowClickable ? "cursor: pointer;" : "",
    onClick: (e: MouseEvent) => {
      if (!props.rowClickable) {
        return;
      }

      if (props.customClick) {
        emit("row-click", e, database);
        return;
      }

      if (props.singleSelection) {
        state.selectedDatabaseNameList = new Set([database.name]);
      } else {
        const selectedDatabaseNameList = new Set(
          Array.from(state.selectedDatabaseNameList)
        );
        if (selectedDatabaseNameList.has(database.name)) {
          selectedDatabaseNameList.delete(database.name);
        } else {
          selectedDatabaseNameList.add(database.name);
        }
        state.selectedDatabaseNameList = selectedDatabaseNameList;
      }
    },
  };
};

watch(
  () => state.selectedDatabaseNameList,
  () => {
    emit("update:selected-databases", state.selectedDatabaseNameList);
  }
);
</script>
