<template>
  <NDataTable
    size="small"
    :columns="columnList"
    :data="data"
    :striped="true"
    :bordered="bordered"
    :loading="loading"
    :row-key="(data: ComposedDatabaseGroup) => data.name"
    :checked-row-keys="Array.from(state.selectedDatabaseGroupNameList)"
    :row-props="rowProps"
    :pagination="{ pageSize: 20 }"
    :paginate-single-page="false"
    @update:checked-row-keys="
        (val) => (state.selectedDatabaseGroupNameList = new Set(val as string[]))
      "
  ></NDataTable>
</template>

<script lang="tsx" setup>
import { NButton, NDataTable, NTag, type DataTableColumn } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { ComposedDatabaseGroup } from "@/types";

interface LocalState {
  // The selected database group name list.
  selectedDatabaseGroupNameList: Set<string>;
}

type DatabaseGroupDataTableColumn = DataTableColumn<ComposedDatabaseGroup> & {
  hide?: boolean;
};

const props = withDefaults(
  defineProps<{
    databaseGroupList: ComposedDatabaseGroup[];
    bordered?: boolean;
    loading?: boolean;
    showSelection?: boolean;
    showProject?: boolean;
    showEdit?: boolean;
    customClick?: boolean;
  }>(),
  {
    bordered: true,
    showSelection: true,
    showProject: true,
  }
);

const emit = defineEmits<{
  (
    event: "row-click",
    e: MouseEvent,
    databaseGroup: ComposedDatabaseGroup
  ): void;
  (event: "update:selected-database-groups", val: Set<string>): void;
  (event: "edit", databaseGroup: ComposedDatabaseGroup): void;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  selectedDatabaseGroupNameList: new Set(),
});

const columnList = computed((): DatabaseGroupDataTableColumn[] => {
  const SELECTION: DatabaseGroupDataTableColumn = {
    type: "selection",
    multiple: false,
    hide: !props.showSelection,
    cellProps: () => {
      return {
        onClick: (e: MouseEvent) => {
          e.stopPropagation();
        },
      };
    },
  };
  const NAME: DatabaseGroupDataTableColumn = {
    key: "title",
    title: t("common.name"),
    render: (data) => {
      return (
        <div class="space-x-2">
          <span>{data.databasePlaceholder}</span>
          {data.multitenancy && (
            <NTag round type="info" size="small">
              {t("database-group.multitenancy.self")}
            </NTag>
          )}
        </div>
      );
    },
  };
  const PROJECT: DatabaseGroupDataTableColumn = {
    key: "project",
    title: t("common.project"),
    hide: !props.showProject,
    render: (data) => {
      return <span>{data.projectEntity.title}</span>;
    },
  };
  const EDIT_BUTTON: DatabaseGroupDataTableColumn = {
    key: "edit",
    title: "",
    hide: !props.showEdit,
    width: 150,
    render: (data) => {
      return (
        <div class="flex justify-end">
          <NButton
            size="small"
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              emit("edit", data);
            }}
          >
            {t("common.configure")}
          </NButton>
        </div>
      );
    },
  };

  // Maybe we can add more columns here. e.g. matched databases, etc.
  return [SELECTION, NAME, PROJECT, EDIT_BUTTON].filter(
    (column) => !column.hide
  );
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

      state.selectedDatabaseGroupNameList = new Set([databaseGroup.name]);
    },
  };
};

watch(
  () => state.selectedDatabaseGroupNameList,
  () => {
    emit(
      "update:selected-database-groups",
      state.selectedDatabaseGroupNameList
    );
  }
);
</script>
