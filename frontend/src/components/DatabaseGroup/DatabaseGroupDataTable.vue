<template>
  <NDataTable
    size="small"
    :columns="columnList"
    :data="data"
    :striped="true"
    :bordered="bordered"
    :loading="loading"
    :row-key="(data: ComposedDatabaseGroup) => data.name"
    :checked-row-keys="selectedDatabaseGroupNames"
    :row-props="rowProps"
    :pagination="{ pageSize: 20 }"
    :paginate-single-page="false"
    @update:checked-row-keys="
      (val) => $emit('update:selected-database-group-names', val as string[])
    "
  />
</template>

<script lang="tsx" setup>
import { NDataTable, NEllipsis, type DataTableColumn } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { ComposedDatabaseGroup } from "@/types";

type DatabaseGroupDataTableColumn = DataTableColumn<ComposedDatabaseGroup> & {
  hide?: boolean;
};

const props = withDefaults(
  defineProps<{
    databaseGroupList: ComposedDatabaseGroup[];
    bordered?: boolean;
    loading?: boolean;
    showProject?: boolean;
    customClick?: boolean;
    showSelection?: boolean;
    showActions?: boolean;
    singleSelection?: boolean;
    selectedDatabaseGroupNames?: string[];
  }>(),
  {
    bordered: true,
    selectedDatabaseGroupNames: () => [],
  }
);

const emit = defineEmits<{
  (
    event: "row-click",
    e: MouseEvent,
    databaseGroup: ComposedDatabaseGroup
  ): void;
  (event: "update:selected-database-group-names", val: string[]): void;
}>();

const { t } = useI18n();

const columnList = computed((): DatabaseGroupDataTableColumn[] => {
  const rawColumnList: DatabaseGroupDataTableColumn[] = [
    {
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
    },
    {
      key: "title",
      title: t("common.name"),
      minWidth: 128,
      render: (data) => {
        return (
          <div class="space-x-2">
            <span>{data.title}</span>
          </div>
        );
      },
    },
    {
      key: "project",
      title: t("common.project"),
      minWidth: 128,
      hide: !props.showProject,
      render: (data) => {
        return <span>{data.projectEntity.title}</span>;
      },
    },
    {
      key: "expression",
      title: t("database.expression"),
      ellipsis: true,
      render: (data) => {
        if (!data.databaseExpr || data.databaseExpr.expression === "") {
          return <span class="textinfolabel italic">{t("common.empty")}</span>;
        }
        return <NEllipsis>{data.databaseExpr.expression}</NEllipsis>;
      },
    },
  ];

  return rawColumnList.filter((column) => !column.hide);
});

const data = computed(() => {
  return [...props.databaseGroupList];
});

const rowProps = (databaseGroup: ComposedDatabaseGroup) => {
  return {
    style: "cursor: pointer;",
    onClick: (e: MouseEvent) => {
      if (props.customClick) {
        emit("row-click", e, databaseGroup);
        return;
      }

      if (props.singleSelection) {
        emit("update:selected-database-group-names", [databaseGroup.name]);
      } else {
        const selectedDatabaseGroupNameList = new Set(
          props.selectedDatabaseGroupNames
        );
        if (selectedDatabaseGroupNameList.has(databaseGroup.name)) {
          selectedDatabaseGroupNameList.delete(databaseGroup.name);
        } else {
          selectedDatabaseGroupNameList.add(databaseGroup.name);
        }
        emit("update:selected-database-group-names", [
          ...selectedDatabaseGroupNameList,
        ]);
      }
    },
  };
};
</script>
