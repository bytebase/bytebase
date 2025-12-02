<template>
  <DrawerContent :title="$t('schema-template.field-template.self')">
    <div>
      <div class="flex flex-col gap-y-6 pb-6">
        <!-- category -->
        <div class="sm:col-span-2 sm:col-start-1">
          <label for="category" class="textlabel">
            {{ $t("schema-template.form.category") }}
          </label>
          <p class="text-sm text-gray-500 mb-2">
            {{ $t("schema-template.form.category-desc") }}
          </p>
          <div class="relative flex flex-row justify-between items-center mt-1">
            <DropdownInput
              v-model:value="state.category"
              :options="categoryOptions"
              :placeholder="$t('schema-template.form.unclassified')"
              :disabled="readonly"
              :consistent-menu-width="true"
              :allow-filter="true"
            />
          </div>
        </div>

        <div class="w-full flex flex-col gap-y-1">
          <label for="engine" class="textlabel">
            {{ $t("database.engine") }}
          </label>
          <InstanceEngineRadioGrid
            v-model:engine="state.engine"
            :engine-list="engineList"
            :disabled="readonly"
            class="grid-cols-4 gap-2"
          />
        </div>
      </div>
      <div class="flex flex-col gap-y-6 border-t border-block-border pt-6">
        <div class="grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-4">
          <!-- column name -->
          <div class="sm:col-span-2 sm:col-start-1">
            <label for="column-name" class="textlabel">
              {{ $t("schema-template.form.column-name") }}
              <RequiredStar />
            </label>
            <NInput
              v-model:value="state.column!.name"
              placeholder="column name"
              :disabled="readonly"
            />
          </div>

          <!-- type -->
          <div class="sm:col-span-2 sm:col-start-1">
            <label for="column-type" class="textlabel">
              {{ $t("schema-template.form.column-type") }}
              <RequiredStar />
            </label>
            <div
              class="relative flex flex-row justify-between items-center mt-1"
            >
              <DropdownInput
                :value="state.column!.type || null"
                :allow-input-value="
                  schemaTemplateColumnTypeOptions.length === 0
                "
                :options="
                  schemaTemplateColumnTypeOptions.length > 0
                    ? schemaTemplateColumnTypeOptions
                    : dataTypeOptions
                "
                :disabled="readonly"
                :allow-filter="true"
                placeholder="column type"
                @update:value="state.column!.type = $event"
              />
            </div>
          </div>

          <!-- default value -->
          <div class="sm:col-span-2 sm:col-start-1">
            <label for="default-value" class="textlabel">
              {{ $t("schema-template.form.default-value") }}
            </label>
            <div class="flex flex-row items-center relative">
              <DefaultValueCell
                :column="state.column"
                :disabled="readonly"
                border="1px solid rgb(var(--color-control-border))"
                @update="handleColumnDefaultSelect"
              />
            </div>
          </div>

          <!-- nullable -->
          <div class="sm:col-span-2 ml-0 sm:ml-3">
            <label for="nullable" class="textlabel">
              {{ $t("schema-template.form.nullable") }}
            </label>
            <div class="flex flex-row items-center h-[34px]">
              <NSwitch
                v-model:value="state.column!.nullable"
                :text="false"
                :disabled="readonly"
              />
            </div>
          </div>

          <!-- on update -->
          <div class="sm:col-span-2 sm:col-start-1">
            <label for="default-value" class="textlabel">
              {{ $t("schema-template.form.on-update") }}
            </label>
            <div class="flex flex-row items-center relative">
              <NInput v-model:value="state.column!.onUpdate" />
            </div>
          </div>

          <!-- comment -->
          <div class="sm:col-span-4 sm:col-start-1">
            <label for="comment" class="textlabel">
              {{ $t("schema-template.form.comment") }}
            </label>
            <NInput
              v-model:value="state.column!.comment"
              type="textarea"
              :autosize="{ minRows: 3, maxRows: 3 }"
              :disabled="readonly"
            />
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="w-full flex justify-between items-center">
        <div class="w-full flex justify-end items-center gap-x-2">
          <NButton @click.prevent="$emit('dismiss')">
            {{ $t("common.cancel") }}
          </NButton>
          <NButton
            v-if="!readonly"
            :disabled="submitDisabled"
            type="primary"
            @click.prevent="submit"
          >
            {{ create ? $t("common.create") : $t("common.update") }}
          </NButton>
        </div>
      </div>
    </template>
  </DrawerContent>
</template>

<script lang="ts" setup>
import { create as createProto } from "@bufbuild/protobuf";
import { cloneDeep, isEqual } from "lodash-es";
import type { SelectOption } from "naive-ui";
import { NButton, NInput, NSwitch } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import RequiredStar from "@/components/RequiredStar.vue";
import { DefaultValueCell } from "@/components/SchemaEditorLite/Panels/TableColumnEditor/components";
import type { DefaultValue } from "@/components/SchemaEditorLite/utils";
import {
  DrawerContent,
  DropdownInput,
  InstanceEngineRadioGrid,
} from "@/components/v2";
import { useNotificationStore, useSettingV1Store } from "@/store";
import { ColumnCatalogSchema } from "@/types/proto-es/v1/database_catalog_service_pb";
import {
  type ColumnMetadata,
  ColumnMetadataSchema,
} from "@/types/proto-es/v1/database_service_pb";
import type { SchemaTemplateSetting_FieldTemplate } from "@/types/proto-es/v1/setting_service_pb";
import {
  SchemaTemplateSetting_FieldTemplateSchema,
  SchemaTemplateSettingSchema,
  Setting_SettingName,
  ValueSchema as SettingValueSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { getDataTypeSuggestionList } from "@/utils";
import { categoryList, engineList } from "./utils";

const props = defineProps<{
  create: boolean;
  readonly?: boolean;
  template: SchemaTemplateSetting_FieldTemplate;
}>();

const emit = defineEmits(["dismiss"]);

interface LocalState extends SchemaTemplateSetting_FieldTemplate {
  column: ColumnMetadata;
}

const state = reactive<LocalState>({
  ...createProto(SchemaTemplateSetting_FieldTemplateSchema, {
    id: props.template.id,
    engine: props.template.engine,
    category: props.template.category,
    catalog: createProto(ColumnCatalogSchema, props.template.catalog ?? {}),
  }),
  column: createProto(ColumnMetadataSchema, props.template.column ?? {}),
});
const { t } = useI18n();
const settingStore = useSettingV1Store();

const dataTypeOptions = computed(() => {
  return getDataTypeSuggestionList(state.engine).map<SelectOption>(
    (dataType) => {
      return {
        label: dataType,
        value: dataType,
      };
    }
  );
});

const categoryOptions = computed(() => {
  return categoryList.value.map<SelectOption>((category) => ({
    label: category,
    value: category,
  }));
});

const schemaTemplateColumnTypes = computed(() => {
  const setting = settingStore.getSettingByName(
    Setting_SettingName.SCHEMA_TEMPLATE
  );
  if (!setting?.value?.value) return [];
  const value = setting.value.value;
  if (value.case !== "schemaTemplateSettingValue") return [];
  const columnTypes = value.value.columnTypes;
  if (columnTypes && columnTypes.length > 0) {
    const columnType = columnTypes.find(
      (columnType) => columnType.engine === state.engine
    );
    if (columnType && columnType.enabled) {
      return columnType.types;
    }
  }
  return [];
});

const schemaTemplateColumnTypeOptions = computed(() => {
  return schemaTemplateColumnTypes.value.map<SelectOption>((type) => ({
    label: type,
    value: type,
  }));
});

const submitDisabled = computed(() => {
  if (!state.column?.name || !state.column?.type) {
    return true;
  }
  if (!props.create && isEqual(props.template, state)) {
    return true;
  }
  return false;
});

const submit = async () => {
  const template = createProto(SchemaTemplateSetting_FieldTemplateSchema, {
    id: state.id,
    engine: state.engine,
    category: state.category,
    column: state.column,
    catalog: createProto(ColumnCatalogSchema, {}),
  });
  const setting = await settingStore.fetchSettingByName(
    Setting_SettingName.SCHEMA_TEMPLATE
  );

  let settingValue = createProto(SchemaTemplateSettingSchema, {});
  if (
    setting?.value?.value &&
    setting.value.value.case === "schemaTemplateSettingValue"
  ) {
    settingValue = cloneDeep(setting.value.value.value);
  }

  const index = settingValue.fieldTemplates.findIndex(
    (t) => t.id === template.id
  );
  if (index >= 0) {
    settingValue.fieldTemplates[index] = template;
  } else {
    settingValue.fieldTemplates.push(template);
  }

  await settingStore.upsertSetting({
    name: Setting_SettingName.SCHEMA_TEMPLATE,
    value: createProto(SettingValueSchema, {
      value: {
        case: "schemaTemplateSettingValue",
        value: settingValue,
      },
    }),
  });
  useNotificationStore().pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
  emit("dismiss");
};

const handleColumnDefaultSelect = (defaultValue: DefaultValue) => {
  state.column.hasDefault = defaultValue.hasDefault;
  state.column.default = defaultValue.default;
};
</script>
