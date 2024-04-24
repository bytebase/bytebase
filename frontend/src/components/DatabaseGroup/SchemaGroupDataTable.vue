<template>
  <NDataTable
    size="small"
    :columns="columnList"
    :data="data"
    :striped="true"
    :bordered="bordered"
    :loading="loading"
    :row-key="(data: ComposedSchemaGroup) => data.name"
    :checked-row-keys="Array.from(state.selectedSchemaGroupNameList)"
    :row-props="rowProps"
    :pagination="{ pageSize: 20 }"
    :paginate-single-page="false"
    @update:checked-row-keys="
        (val) => (state.selectedSchemaGroupNameList = new Set(val as string[]))
      "
  ></NDataTable>
</template>

<script lang="tsx" setup>
import { NButton, NDataTable, type DataTableColumn } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { ComposedSchemaGroup } from "@/types";

interface LocalState {
  // The selected schema group name list.
  selectedSchemaGroupNameList: Set<string>;
}

type SchemaGroupDataTableColumn = DataTableColumn<ComposedSchemaGroup> & {
  hide?: boolean;
};

const props = withDefaults(
  defineProps<{
    schemaGroupList: ComposedSchemaGroup[];
    bordered?: boolean;
    loading?: boolean;
    showSelection?: boolean;
    showEdit?: boolean;
    customClick?: boolean;
  }>(),
  {
    bordered: true,
    showEdit: true,
    customClick: true,
  }
);

const emit = defineEmits<{
  (event: "row-click", e: MouseEvent, schemaGroup: ComposedSchemaGroup): void;
  (event: "update:selected-schema-groups", val: Set<string>): void;
  (event: "edit", schemaGroup: ComposedSchemaGroup): void;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  selectedSchemaGroupNameList: new Set(),
});

const columnList = computed((): SchemaGroupDataTableColumn[] => {
  const SELECTION: SchemaGroupDataTableColumn = {
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
  const NAME: SchemaGroupDataTableColumn = {
    key: "title",
    title: t("common.name"),
    render: (data) => {
      return <span>{data.tablePlaceholder}</span>;
    },
  };
  const EDIT_BUTTON: SchemaGroupDataTableColumn = {
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

  // Maybe we can add more columns here. e.g. matched tables, etc.
  return [SELECTION, NAME, EDIT_BUTTON].filter((column) => !column.hide);
});

const data = computed(() => {
  return [...props.schemaGroupList];
});

const rowProps = (schemaGroup: ComposedSchemaGroup) => {
  return {
    style: "cursor: pointer;",
    onClick: (e: MouseEvent) => {
      if (props.customClick) {
        emit("row-click", e, schemaGroup);
        return;
      }

      state.selectedSchemaGroupNameList = new Set([schemaGroup.name]);
    },
  };
};

watch(
  () => state.selectedSchemaGroupNameList,
  () => {
    emit("update:selected-schema-groups", state.selectedSchemaGroupNameList);
  }
);
</script>
