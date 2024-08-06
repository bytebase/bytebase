<template>
  <NDataTable
    key="sensitive-column-table"
    :columns="dataTableColumns"
    :data="columnList"
    :row-props="rowProps"
    :row-key="itemKey"
    :checked-row-keys="checkedItemKeys"
    size="small"
    @update:checked-row-keys="handleUpdateCheckedRowKeys($event as string[])"
  />
</template>

<script lang="ts" setup>
import { PencilIcon } from "lucide-vue-next";
import { TrashIcon } from "lucide-vue-next";
import { NDataTable, NPopconfirm, type DataTableColumn } from "naive-ui";
import { computed, h, ref, watch } from "vue";
import { withModifiers } from "vue";
import { useI18n } from "vue-i18n";
import {
  DatabaseV1Name,
  EnvironmentV1Name,
  InstanceV1Name,
  MiniActionButton,
  ProjectV1Name,
} from "@/components/v2";
import type { MaskingLevel } from "@/types/proto/v1/common";
import { maskingLevelToJSON } from "@/types/proto/v1/common";
import type { SensitiveColumn } from "../types";

const props = defineProps<{
  showOperation: boolean;
  rowClickable: boolean;
  rowSelectable: boolean;
  columnList: SensitiveColumn[];
  checkedColumnIndexList: number[];
}>();

const emit = defineEmits<{
  (
    event: "click",
    item: SensitiveColumn,
    row: number,
    action: "VIEW" | "DELETE" | "EDIT"
  ): void;
  (event: "checked:update", list: number[]): void;
}>();

const { t } = useI18n();
const checkedColumnIndex = ref<Set<number>>(
  new Set(props.checkedColumnIndexList)
);

watch(
  () => props.columnList,
  () => (checkedColumnIndex.value = new Set()),
  { deep: true }
);
watch(
  () => props.checkedColumnIndexList,
  (val) => (checkedColumnIndex.value = new Set(val)),
  { deep: true }
);

const itemKey = (item: SensitiveColumn) => {
  const parts = [item.database.name];
  const { schema, table, column } = item.maskData;
  if (schema) {
    parts.push(schema);
  }
  parts.push(table, column);
  return parts.join("::");
};

const getMaskingLevelText = (maskingLevel: MaskingLevel) => {
  const level = maskingLevelToJSON(maskingLevel);
  return t(`settings.sensitive-data.masking-level.${level.toLowerCase()}`);
};

const checkedItemKeys = computed(() => {
  const keys: string[] = [];
  props.checkedColumnIndexList.forEach((index) => {
    const item = props.columnList[index];
    if (item) {
      keys.push(itemKey(item));
    }
  });
  return keys;
});

const dataTableColumns = computed(() => {
  const columns: DataTableColumn<SensitiveColumn>[] = [
    {
      key: "masking-level",
      title: t("settings.sensitive-data.masking-level.self"),
      render(item) {
        return getMaskingLevelText(item.maskData.maskingLevel);
      },
    },
    {
      key: "column",
      title: t("database.column"),
      render(item) {
        return item.maskData.column;
      },
    },
    {
      key: "table",
      title: t("common.table"),
      render(item) {
        return item.maskData.schema
          ? `${item.maskData.schema}.${item.maskData.table}`
          : item.maskData.table;
      },
    },
    {
      key: "database",
      title: t("common.database"),
      render(item) {
        return h(DatabaseV1Name, { database: item.database, link: false });
      },
    },
    {
      key: "instance",
      title: t("common.instance"),
      render(item) {
        return h(InstanceV1Name, {
          instance: item.database.instanceResource,
          link: false,
        });
      },
    },
    {
      key: "environment",
      title: t("common.environment"),
      render(item) {
        return h(EnvironmentV1Name, {
          environment: item.database.effectiveEnvironmentEntity,
          link: false,
        });
      },
    },
    {
      key: "project",
      title: t("common.project"),
      render(item) {
        return h(ProjectV1Name, {
          project: item.database.projectEntity,
          link: false,
        });
      },
    },
  ];
  if (props.showOperation) {
    columns.push({
      key: "operation",
      title: t("common.operation"),
      render(item, index) {
        const editButton = h(
          MiniActionButton,
          {
            onClick: withModifiers(() => {
              emit("click", item, index, "EDIT");
            }, ["prevent", "stop"]),
          },
          {
            default: () => h(PencilIcon, { class: "w-4 h-4" }),
          }
        );
        const deleteButton = h(
          NPopconfirm,
          {
            onPositiveClick: () => {
              emit("click", item, index, "DELETE");
            },
          },
          {
            trigger: () => {
              return h(
                MiniActionButton,
                {
                  tag: "div",
                  onClick: withModifiers(() => {
                    // noop
                  }, ["stop", "prevent"]),
                },
                {
                  default: () => h(TrashIcon, { class: "w-4 h-4" }),
                }
              );
            },
            default: () =>
              h(
                "div",
                { class: "whitespace-nowrap" },
                t("settings.sensitive-data.remove-sensitive-column-tips")
              ),
          }
        );
        return [editButton, deleteButton];
      },
    });
  }
  if (props.rowSelectable) {
    columns.unshift({
      type: "selection",
      multiple: true,
      cellProps: () => {
        return {
          onClick: (e: MouseEvent) => {
            e.stopPropagation();
          },
        };
      },
    });
  }
  return columns;
});

const handleUpdateCheckedRowKeys = (keys: string[]) => {
  const keysSet = new Set(keys);
  const checkedIndexList: number[] = [];
  props.columnList.forEach((item, index) => {
    const key = itemKey(item);
    if (keysSet.has(key)) {
      checkedIndexList.push(index);
    }
  });
  emit("checked:update", checkedIndexList);
};

const rowProps = (item: SensitiveColumn, index: number) => {
  return {
    style: props.rowClickable ? "cursor: pointer;" : "",
    onClick: () => {
      emit("click", item, index, "VIEW");
    },
  };
};
</script>
