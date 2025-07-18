<template>
  <NDataTable
    size="small"
    :columns="columns"
    :data="templateList"
    :striped="true"
    :bordered="true"
    :row-props="rowProps"
  />
</template>

<script lang="tsx" setup>
import { create as createProto } from "@bufbuild/protobuf";
import { pullAt } from "lodash-es";
import { PencilIcon, TrashIcon } from "lucide-vue-next";
import { NPopconfirm, NDataTable } from "naive-ui";
import type { DataTableColumn } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { MiniActionButton } from "@/components/v2";
import { useSettingV1Store } from "@/store";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { SchemaTemplateSetting_TableTemplate } from "@/types/proto-es/v1/setting_service_pb";
import {
  SchemaTemplateSettingSchema,
  Setting_SettingName,
  ValueSchema as SettingValueSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { EngineIcon } from "../Icon";
import ClassificationLevelBadge from "./ClassificationLevelBadge.vue";
import { classificationConfig } from "./utils";

const props = defineProps<{
  engine?: Engine;
  readonly: boolean;
  templateList: SchemaTemplateSetting_TableTemplate[];
}>();

const emit = defineEmits<{
  (event: "view", item: SchemaTemplateSetting_TableTemplate): void;
  (event: "apply", item: SchemaTemplateSetting_TableTemplate): void;
}>();

const { t } = useI18n();
const settingStore = useSettingV1Store();

const columns = computed(
  (): DataTableColumn<SchemaTemplateSetting_TableTemplate>[] => {
    const cols: DataTableColumn<SchemaTemplateSetting_TableTemplate>[] = [
      {
        title: t("schema-template.form.category"),
        key: "category",
        render: (item) => item.category || "-",
      },
      {
        title: t("schema-template.form.table-name"),
        key: "name",
        render: (item) => (
          <div class="flex justify-start items-center">
            <EngineIcon engine={item.engine} customClass="ml-0 mr-1" />
            {item.table?.name ?? ""}
          </div>
        ),
      },
    ];

    if (classificationConfig.value) {
      cols.push({
        title: t("schema-template.classification.self"),
        key: "classification",
        render: (item) => (
          <ClassificationLevelBadge
            classification={item.catalog?.classification}
            classificationConfig={classificationConfig.value!}
          />
        ),
      });
    }

    cols.push({
      title: t("schema-template.form.comment"),
      key: "comment",
      render: (item) => item.table?.userComment ?? "",
    });

    if (!props.readonly) {
      cols.push({
        title: t("common.operations"),
        key: "operations",
        width: 160,
        render: (item) => (
          <div class="flex items-center justify-start gap-x-2">
            <MiniActionButton
              onClick={(e: MouseEvent) => {
                e.stopPropagation();
                emit("view", item);
              }}
            >
              <PencilIcon class="w-4 h-4" />
            </MiniActionButton>
            <NPopconfirm onPositiveClick={() => deleteTemplate(item.id)}>
              {{
                trigger: () => (
                  <MiniActionButton
                    onClick={(e: MouseEvent) => e.stopPropagation()}
                  >
                    <TrashIcon class="w-4 h-4" />
                  </MiniActionButton>
                ),
                default: () => (
                  <div class="whitespace-nowrap">
                    {t("common.delete")} '{item.table?.name}'?
                  </div>
                ),
              }}
            </NPopconfirm>
          </div>
        ),
      });
    }

    return cols;
  }
);

const rowProps = (row: SchemaTemplateSetting_TableTemplate) => {
  if (row.engine === props.engine) {
    return {
      style: "cursor: pointer;",
      onClick: () => {
        emit("apply", row);
      },
    };
  }
  return {};
};

const deleteTemplate = async (id: string) => {
  const setting = await settingStore.fetchSettingByName(
    Setting_SettingName.SCHEMA_TEMPLATE
  );

  const existingValue =
    setting?.value?.value?.case === "schemaTemplateSettingValue"
      ? setting.value.value.value
      : undefined;
  const settingValue = createProto(SchemaTemplateSettingSchema, {
    fieldTemplates: existingValue?.fieldTemplates || [],
    columnTypes: existingValue?.columnTypes || [],
    tableTemplates: existingValue?.tableTemplates || [],
  });

  const index = settingValue.tableTemplates.findIndex((t) => t.id === id);
  if (index >= 0) {
    pullAt(settingValue.tableTemplates, index);

    await settingStore.upsertSetting({
      name: Setting_SettingName.SCHEMA_TEMPLATE,
      value: createProto(SettingValueSchema, {
        value: {
          case: "schemaTemplateSettingValue",
          value: settingValue,
        },
      }),
    });
  }
};
</script>
