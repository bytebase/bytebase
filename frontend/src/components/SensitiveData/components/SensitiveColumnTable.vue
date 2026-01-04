<template>
  <NDataTable
    key="sensitive-column-table"
    :columns="dataTableColumns"
    :data="columnList"
    :row-key="itemKey"
    :checked-row-keys="checkedItemKeys"
    size="small"
    @update:checked-row-keys="handleUpdateCheckedRowKeys($event as string[])"
  />
</template>

<script lang="tsx" setup>
import { create } from "@bufbuild/protobuf";
import { TrashIcon } from "lucide-vue-next";
import { type DataTableColumn, NDataTable, NPopconfirm } from "naive-ui";
import { computed, h, withModifiers } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink, useRouter } from "vue-router";
import ClassificationCell from "@/components/ColumnDataTable/ClassificationCell.vue";
import SemanticTypeCell from "@/components/ColumnDataTable/SemanticTypeCell.vue";
import type {
  MaskData,
  MaskDataTarget,
} from "@/components/SensitiveData/types";
import { MiniActionButton } from "@/components/v2";
import {
  pushNotification,
  useDatabaseCatalog,
  useDatabaseCatalogV1Store,
  useSettingV1Store,
} from "@/store";
import type { ComposedDatabase } from "@/types";
import type {
  ColumnCatalog,
  ObjectSchema,
  TableCatalog,
} from "@/types/proto-es/v1/database_catalog_service_pb";
import { DataClassificationSetting_DataClassificationConfigSchema } from "@/types/proto-es/v1/setting_service_pb";
import { autoDatabaseRoute } from "@/utils";

const props = defineProps<{
  database: ComposedDatabase;
  showOperation: boolean;
  rowClickable: boolean;
  rowSelectable: boolean;
  columnList: MaskData[];
  checkedColumnList: MaskData[];
}>();

const emit = defineEmits<{
  (event: "delete", item: MaskData): void;
  (event: "update:checkedColumnList", list: MaskData[]): void;
}>();

const { t } = useI18n();
const router = useRouter();
const settingStore = useSettingV1Store();
const dbCatalogStore = useDatabaseCatalogV1Store();

const itemKey = (item: MaskData) => {
  const parts = [];
  const { schema, table, column } = item;
  if (schema) {
    parts.push(schema);
  }
  parts.push(table, column);
  return parts.join("::");
};

const databaseCatalog = useDatabaseCatalog(props.database.name, false);

const classificationConfig = computed(() => {
  return (
    settingStore.getProjectClassification(
      props.database.projectEntity.dataClassificationConfigId
    ) ?? create(DataClassificationSetting_DataClassificationConfigSchema, {})
  );
});

const checkedItemKeys = computed(() => {
  return props.checkedColumnList.map(itemKey);
});

const dataTableColumns = computed(() => {
  const columns: DataTableColumn<MaskData>[] = [
    {
      key: "table",
      title: t("common.table"),
      resizable: true,
      width: "minmax(min-content, auto)",
      render(item) {
        return (
          <div>
            <RouterLink
              to={{
                ...autoDatabaseRoute(router, props.database),
                query: {
                  schema: item.schema,
                  table: item.table,
                },
                hash: "overview",
              }}
              class="normal-link"
              exactActiveClass=""
            >
              {item.schema ? `${item.schema}.${item.table}` : item.table}
            </RouterLink>
          </div>
        );
      },
    },
    {
      key: "column",
      title: t("database.column"),
      resizable: true,
      width: "minmax(min-content, auto)",
      render(item) {
        return item.column || "-";
      },
    },
    {
      key: "semantic-type",
      title: t("settings.sensitive-data.semantic-types.table.semantic-type"),
      width: "minmax(min-content, auto)",
      resizable: true,
      render(item) {
        return (
          <SemanticTypeCell
            database={props.database}
            semanticTypeId={item.semanticTypeId}
            readonly={!props.showOperation || item.disableSemanticType}
            onApply={(id: string) => onSemanticTypeApply(item, id)}
          />
        );
      },
    },
    {
      key: "classification",
      title: t("database.classification.self"),
      width: "minmax(min-content, auto)",
      resizable: true,
      render(item) {
        return (
          <ClassificationCell
            classification={item.classificationId}
            classificationConfig={classificationConfig.value}
            engine={props.database.instanceResource.engine}
            readonly={!props.showOperation || item.disableClassification}
            onApply={(id: string) => onClassificationIdApply(item, id)}
          />
        );
      },
    },
  ];
  if (props.showOperation) {
    columns.push({
      key: "operation",
      title: t("common.operation"),
      width: "6rem",
      render(item) {
        return h(
          NPopconfirm,
          {
            onPositiveClick: () => {
              onMaskingClear(item);
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

const hasSemanticType = (
  target: MaskDataTarget
): target is ColumnCatalog | ObjectSchema => {
  return "semanticType" in target;
};

const hasClassificationType = (
  target: MaskDataTarget
): target is ColumnCatalog | TableCatalog => {
  return "classification" in target;
};

const onSemanticTypeApply = async (item: MaskData, semanticType: string) => {
  if (!hasSemanticType(item.target)) {
    return;
  }
  item.target.semanticType = semanticType;
  await dbCatalogStore.updateDatabaseCatalog(databaseCatalog.value);

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};

const onClassificationIdApply = async (
  item: MaskData,
  classification: string
) => {
  if (!hasClassificationType(item.target)) {
    return;
  }
  item.target.classification = classification;
  await dbCatalogStore.updateDatabaseCatalog(databaseCatalog.value);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};

const onMaskingClear = async (item: MaskData) => {
  if (hasSemanticType(item.target)) {
    item.target.semanticType = "";
  }
  if (hasClassificationType(item.target)) {
    item.target.classification = "";
  }
  await dbCatalogStore.updateDatabaseCatalog(databaseCatalog.value);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.removed"),
  });
  emit("delete", item);
};

const handleUpdateCheckedRowKeys = (keys: string[]) => {
  const keysSet = new Set(keys);
  const checkedList: MaskData[] = [];
  for (const column of props.columnList) {
    const key = itemKey(column);
    if (keysSet.has(key)) {
      checkedList.push(column);
    }
  }
  emit("update:checkedColumnList", checkedList);
};
</script>
