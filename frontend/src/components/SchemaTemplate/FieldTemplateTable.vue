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
import type { DataTableColumn } from "naive-ui";
import { NDataTable, NPopconfirm } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { getColumnDefaultValuePlaceholder } from "@/components/SchemaEditorLite";
import { MiniActionButton } from "@/components/v2";
import { LabelsCell } from "@/components/v2/Model/cells";
import { useSettingV1Store } from "@/store";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import type { SchemaTemplateSetting_FieldTemplate } from "@/types/proto-es/v1/setting_service_pb";
import {
  SchemaTemplateSettingSchema,
  Setting_SettingName,
  ValueSchema as SettingValueSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { EngineIcon } from "../Icon";

const props = defineProps<{
  engine?: Engine;
  readonly: boolean;
  templateList: SchemaTemplateSetting_FieldTemplate[];
}>();

const emit = defineEmits<{
  (event: "view", item: SchemaTemplateSetting_FieldTemplate): void;
  (event: "apply", item: SchemaTemplateSetting_FieldTemplate): void;
}>();

const { t } = useI18n();
const settingStore = useSettingV1Store();

const columns = computed(
  (): DataTableColumn<SchemaTemplateSetting_FieldTemplate>[] => {
    const cols: DataTableColumn<SchemaTemplateSetting_FieldTemplate>[] = [
      {
        title: t("schema-template.form.category"),
        key: "category",
        render: (item) => item.category || "-",
      },
      {
        title: t("schema-template.form.column-name"),
        key: "name",
        render: (item) => (
          <div class="flex justify-start items-center">
            <EngineIcon engine={item.engine} customClass="ml-0 mr-1" />
            {item.column?.name ?? ""}
          </div>
        ),
      },
    ];

    cols.push(
      {
        title: t("schema-template.form.column-type"),
        key: "type",
        render: (item) => item.column?.type ?? "",
      },
      {
        title: t("schema-template.form.default-value"),
        key: "default",
        render: (item) => getColumnDefaultValuePlaceholder(item.column!),
      },
      {
        title: t("schema-template.form.comment"),
        key: "comment",
        render: (item) => item.column?.comment ?? "",
      },
      {
        title: t("common.labels"),
        key: "labels",
        render: (item) => (
          <LabelsCell labels={item.catalog?.labels ?? {}} showCount={2} />
        ),
      }
    );

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
                    type="error"
                    onClick={(e: MouseEvent) => e.stopPropagation()}
                  >
                    <TrashIcon class="w-4 h-4" />
                  </MiniActionButton>
                ),
                default: () => (
                  <div class="whitespace-nowrap">
                    {t("common.delete")} '{item.column?.name}'?
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

const rowProps = (row: SchemaTemplateSetting_FieldTemplate) => {
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

  let settingValue = createProto(SchemaTemplateSettingSchema, {});
  if (
    setting?.value?.value &&
    setting.value.value.case === "schemaTemplateSettingValue"
  ) {
    settingValue = setting.value.value.value;
  }

  const index = settingValue.fieldTemplates.findIndex((t) => t.id === id);
  if (index >= 0) {
    pullAt(settingValue.fieldTemplates, index);

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
