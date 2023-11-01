<template>
  <DrawerContent :title="$t('schema-template.field-template.self')">
    <div class="space-y-6 divide-y divide-block-border">
      <div class="space-y-6">
        <!-- category -->
        <div class="sm:col-span-2 sm:col-start-1">
          <label for="category" class="textlabel">
            {{ $t("schema-template.form.category") }}
          </label>
          <p class="text-sm text-gray-500 mb-2">
            {{ $t("schema-template.form.category-desc") }}
          </p>
          <div class="relative flex flex-row justify-between items-center mt-1">
            <input
              v-model="state.category"
              required
              name="category"
              type="text"
              :placeholder="$t('schema-template.form.unclassified')"
              class="textfield w-full"
              :disabled="!allowEdit"
            />
            <NDropdown
              trigger="click"
              :options="categoryOptions"
              :disabled="!allowEdit"
              @select="(category: string) => (state.category = category)"
            >
              <button class="absolute right-5">
                <heroicons-solid:chevron-up-down
                  class="w-4 h-auto text-gray-400"
                />
              </button>
            </NDropdown>
          </div>
        </div>

        <div class="w-full mb-6 space-y-1">
          <label for="engine" class="textlabel">
            {{ $t("database.engine") }}
          </label>
          <div class="grid grid-cols-4 gap-2">
            <template v-for="engine in engineList" :key="engine">
              <div
                class="flex relative justify-start p-2 border rounded"
                :class="[
                  state.engine === engine && 'font-medium bg-control-bg-hover',
                  allowEdit
                    ? 'cursor-pointer hover:bg-control-bg-hover'
                    : 'cursor-not-allowed',
                ]"
                @click.capture="changeEngine(engine)"
              >
                <div class="flex flex-row justify-start items-center">
                  <input
                    type="radio"
                    class="btn mr-2"
                    :checked="state.engine === engine"
                    :disabled="!allowEdit"
                  />
                  <EngineIcon
                    :engine="engine"
                    custom-class="w-5 h-auto max-h-[20px] object-contain mr-1"
                  />
                  <p class="text-center text-sm">
                    {{ engineNameV1(engine) }}
                  </p>
                </div>
              </div>
            </template>
          </div>
        </div>
      </div>
      <div class="space-y-6 pt-6">
        <div class="grid grid-cols-1 gap-y-6 gap-x-4 sm:grid-cols-4">
          <!-- column name -->
          <div class="sm:col-span-2 sm:col-start-1">
            <label for="column-name" class="textlabel">
              {{ $t("schema-template.form.column-name") }}
              <span class="text-red-600 mr-2">*</span>
            </label>
            <input
              v-model="state.column!.name"
              required
              name="column-name"
              type="text"
              placeholder="column name"
              class="textfield mt-1 w-full"
              :disabled="!allowEdit"
            />
          </div>

          <div class="sm:col-span-1 sm:col-start-1">
            <label for="semantic-types" class="textlabel">
              {{ $t("settings.sensitive-data.semantic-types.self") }}
            </label>
            <div class="flex items-center gap-x-2 mt-3">
              {{ columnSemanticType?.title }}
              <div v-if="allowEdit" class="flex items-center">
                <button
                  v-if="columnSemanticType"
                  class="w-6 h-6 p-1 hover:bg-control-bg-hover rounded cursor-pointer disabled:cursor-not-allowed disabled:hover:bg-white disabled:text-gray-400"
                  @click.prevent="onSemanticTypeApply('')"
                >
                  <heroicons-outline:x class="w-4 h-4" />
                </button>
                <button
                  class="w-6 h-6 p-1 hover:bg-control-bg-hover rounded cursor-pointer disabled:cursor-not-allowed disabled:hover:bg-white disabled:text-gray-400"
                  @click.prevent="state.showSemanticTypesDrawer = true"
                >
                  <heroicons-outline:pencil class="w-4 h-4" />
                </button>
              </div>
            </div>
          </div>

          <div v-if="classificationConfig" class="sm:col-span-2">
            <label for="column-name" class="textlabel">
              {{ $t("schema-template.classification.self") }}
            </label>
            <div class="flex items-center gap-x-2 mt-3">
              <ClassificationLevelBadge
                :classification="state.column?.classification"
                :classification-config="classificationConfig"
              />
              <div v-if="allowEdit" class="flex items-center">
                <button
                  v-if="state.column?.classification"
                  class="w-6 h-6 p-1 hover:bg-control-bg-hover rounded cursor-pointer disabled:cursor-not-allowed disabled:hover:bg-white disabled:text-gray-400"
                  @click.prevent="state.column!.classification = ''"
                >
                  <heroicons-outline:x class="w-4 h-4" />
                </button>
                <button
                  class="w-6 h-6 p-1 hover:bg-control-bg-hover rounded cursor-pointer disabled:cursor-not-allowed disabled:hover:bg-white disabled:text-gray-400"
                  @click.prevent="state.showClassificationDrawer = true"
                >
                  <heroicons-outline:pencil class="w-4 h-4" />
                </button>
              </div>
            </div>
          </div>

          <!-- type -->
          <div class="sm:col-span-2 sm:col-start-1">
            <label for="column-type" class="textlabel">
              {{ $t("schema-template.form.column-type") }}
              <span class="text-red-600 mr-2">*</span>
            </label>
            <div
              class="relative flex flex-row justify-between items-center mt-1"
            >
              <BBSelect
                v-if="schemaTemplateColumnTypes.length > 0"
                :selected-item="state.column!.type"
                :item-list="schemaTemplateColumnTypes"
                :show-prefix-item="false"
                placeholder="column type"
                @select-item="(item: string) => state.column!.type = item"
              >
                <template #menuItem="{ item }">
                  {{ item }}
                </template>
              </BBSelect>
              <template v-else>
                <input
                  v-model="state.column!.type"
                  required
                  name="column-type"
                  type="text"
                  placeholder="column type"
                  class="textfield w-full"
                  :disabled="!allowEdit"
                />
                <NDropdown
                  trigger="click"
                  :options="dataTypeOptions"
                  :disabled="!allowEdit"
                  @select="(dataType: string) => (state.column!.type = dataType)"
                >
                  <button class="absolute right-5">
                    <heroicons-solid:chevron-up-down
                      class="w-4 h-auto text-gray-400"
                    />
                  </button>
                </NDropdown>
              </template>
            </div>
          </div>

          <!-- default value -->
          <div class="sm:col-span-2 sm:col-start-1">
            <label for="default-value" class="textlabel">
              {{ $t("schema-template.form.default-value") }}
            </label>
            <div class="flex flex-row items-center relative">
              <input
                class="textfield mt-1 w-full"
                type="text"
                :value="getColumnDefaultDisplayString(state.column!)"
                :disabled="!allowEdit"
                :placeholder="getColumnDefaultValuePlaceholder(state.column!)"
                @change="(e) => handleColumnDefaultInputChange(e)"
              />
              <NDropdown
                trigger="click"
                :disabled="!allowEdit"
                :options="getColumnDefaultValueOptions(state.engine, state.column!.type)"
                @select="(key: string) => handleColumnDefaultFieldChange(key)"
              >
                <button class="absolute right-5">
                  <heroicons-solid:chevron-up-down
                    class="w-4 h-auto text-gray-400"
                  />
                </button>
              </NDropdown>
            </div>
          </div>

          <!-- nullable -->
          <div class="sm:col-span-2 ml-0 sm:ml-3 flex flex-col">
            <label for="nullable" class="textlabel">
              {{ $t("schema-template.form.nullable") }}
            </label>
            <BBSwitch
              class="mt-4"
              :text="false"
              :value="state.column?.nullable"
              :disabled="!allowEdit"
              @toggle="(on: boolean) => state.column!.nullable = on"
            />
          </div>

          <!-- comment -->
          <div class="sm:col-span-4 sm:col-start-1">
            <label for="comment" class="textlabel">
              {{ $t("schema-template.form.comment") }}
            </label>
            <textarea
              v-model="state.column!.userComment"
              rows="3"
              class="textfield block w-full resize-none mt-1 text-sm text-control rounded-md whitespace-pre-wrap"
              :disabled="!allowEdit"
            />
          </div>
        </div>
      </div>
      <div class="space-y-1 pt-6">
        <label for="category" class="textlabel">
          {{ $t("common.labels") }}
        </label>
        <LabelListEditor
          ref="labelListEditorRef"
          v-model:kv-list="state.kvList"
          :readonly="!!readonly"
          :show-errors="dirty"
          class="max-w-[30rem]"
        />
      </div>
    </div>

    <template #footer>
      <div class="w-full flex justify-between items-center">
        <div class="w-full flex justify-end items-center gap-x-3">
          <NButton @click.prevent="$emit('dismiss')">
            {{ $t("common.cancel") }}
          </NButton>
          <NButton
            v-if="allowEdit"
            :disabled="sumbitDisabled"
            type="primary"
            @click.prevent="sumbit"
          >
            {{ create ? $t("common.create") : $t("common.update") }}
          </NButton>
        </div>
      </div>
    </template>
  </DrawerContent>

  <SelectClassificationDrawer
    v-if="classificationConfig"
    :show="state.showClassificationDrawer"
    :classification-config="classificationConfig"
    @dismiss="state.showClassificationDrawer = false"
    @apply="onClassificationSelect"
  />

  <ColumnDefaultValueExpressionModal
    v-if="state.showColumnDefaultValueExpressionModal"
    :expression="state.column!.defaultExpression"
    @close="state.showColumnDefaultValueExpressionModal = false"
    @update:expression="handleSelectedColumnDefaultValueExpressionChange"
  />

  <SemanticTypesDrawer
    :show="state.showSemanticTypesDrawer"
    :semantic-type-list="semanticTypeList"
    @dismiss="state.showSemanticTypesDrawer = false"
    @apply="onSemanticTypeApply($event)"
  />
</template>

<script lang="ts" setup>
import { isEqual, cloneDeep } from "lodash-es";
import { NDropdown } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import {
  getColumnDefaultDisplayString,
  getColumnDefaultValuePlaceholder,
  getDefaultValueByKey,
  getColumnDefaultValueOptions,
} from "@/components/SchemaEditorV1/utils/columnDefaultValue";
import { DrawerContent } from "@/components/v2";
import { useSettingV1Store, useNotificationStore } from "@/store";
import { Engine } from "@/types/proto/v1/common";
import {
  ColumnConfig,
  ColumnMetadata,
} from "@/types/proto/v1/database_service";
import {
  SchemaTemplateSetting,
  SchemaTemplateSetting_FieldTemplate,
} from "@/types/proto/v1/setting_service";
import {
  getDataTypeSuggestionList,
  engineNameV1,
  useWorkspacePermissionV1,
  convertKVListToLabels,
  convertLabelsToKVList,
} from "@/utils";
import { engineList, caregoryList, classificationConfig } from "./utils";

const props = defineProps<{
  create: boolean;
  readonly?: boolean;
  template: SchemaTemplateSetting_FieldTemplate;
}>();

const emit = defineEmits(["dismiss"]);

interface LocalState extends SchemaTemplateSetting_FieldTemplate {
  showClassificationDrawer: boolean;
  showSemanticTypesDrawer: boolean;
  showColumnDefaultValueExpressionModal: boolean;
  kvList: { key: string; value: string }[];
}

const state = reactive<LocalState>({
  id: props.template.id,
  engine: props.template.engine,
  category: props.template.category,
  column: ColumnMetadata.fromPartial({
    ...(props.template.column ?? {}),
  }),
  showClassificationDrawer: false,
  showSemanticTypesDrawer: false,
  showColumnDefaultValueExpressionModal: false,
  config: ColumnConfig.fromPartial({
    ...(props.template.config ?? {}),
  }),
  kvList: [],
});
const { t } = useI18n();
const settingStore = useSettingV1Store();
const allowEdit = computed(() => {
  return (
    useWorkspacePermissionV1("bb.permission.workspace.manage-general").value &&
    !props.readonly
  );
});

const semanticTypeList = computed(() => {
  return (
    settingStore.getSettingByName("bb.workspace.semantic-types")?.value
      ?.semanticTypeSettingValue?.types ?? []
  );
});

const columnSemanticType = computed(() => {
  if (!state.config?.semanticTypeId) {
    return;
  }
  return semanticTypeList.value.find(
    (data) => data.id === state.config?.semanticTypeId
  );
});

const convert = () => {
  return convertLabelsToKVList(
    props.template.config?.labels ?? {},
    true /* sort */
  );
};

watch(
  () => props.template.config?.labels,
  () => {
    state.kvList = convert();
  },
  {
    immediate: true,
    deep: true,
  }
);

const dirty = computed(() => {
  const original = convert();
  const local = state.kvList;
  return !isEqual(original, local);
});

const dataTypeOptions = computed(() => {
  return getDataTypeSuggestionList(state.engine).map((dataType) => {
    return {
      label: dataType,
      key: dataType,
    };
  });
});

const categoryOptions = computed(() => {
  return caregoryList.value.map((category) => ({
    label: category,
    key: category,
  }));
});

const schemaTemplateColumnTypes = computed(() => {
  const setting = settingStore.getSettingByName("bb.workspace.schema-template");
  const columnTypes = setting?.value?.schemaTemplateSettingValue?.columnTypes;
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

const changeEngine = (engine: Engine) => {
  if (allowEdit.value) {
    state.engine = engine;
  }
};

const sumbitDisabled = computed(() => {
  if (!state.column?.name || !state.column?.type) {
    return true;
  }
  if (!props.create && isEqual(props.template, state)) {
    return true;
  }
  return false;
});

const sumbit = async () => {
  const template = SchemaTemplateSetting_FieldTemplate.fromPartial({
    ...state,
    config: ColumnConfig.fromPartial({
      ...state.config,
      name: state.column?.name,
      labels: convertKVListToLabels(state.kvList, false /* !omitEmpty */),
    }),
  });
  const setting = await settingStore.fetchSettingByName(
    "bb.workspace.schema-template"
  );

  const settingValue = SchemaTemplateSetting.fromPartial({});
  if (setting?.value?.schemaTemplateSettingValue) {
    Object.assign(
      settingValue,
      cloneDeep(setting.value.schemaTemplateSettingValue)
    );
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
    name: "bb.workspace.schema-template",
    value: {
      schemaTemplateSettingValue: settingValue,
    },
  });
  useNotificationStore().pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
  emit("dismiss");
};

const onClassificationSelect = (id: string) => {
  if (!state.column) {
    return;
  }
  state.column.classification = id;
};

const handleColumnDefaultInputChange = (event: Event) => {
  const value = (event.target as HTMLInputElement).value;
  if (!state.column) {
    return;
  }
  state.column.hasDefault = true;
  state.column.defaultNull = undefined;
  if (state.column.defaultString !== undefined) {
    state.column.defaultString = value;
    return;
  }
  // By default, user input is treated as expression.
  state.column.defaultExpression = value;
};

const handleColumnDefaultFieldChange = (key: string) => {
  if (key === "expression") {
    state.showColumnDefaultValueExpressionModal = true;
    return;
  }

  const defaultValue = getDefaultValueByKey(key);
  if (!defaultValue || !state.column) {
    return;
  }

  state.column.hasDefault = defaultValue.hasDefault;
  state.column.defaultNull = defaultValue.defaultNull;
  state.column.defaultString = defaultValue.defaultString;
  state.column.defaultExpression = defaultValue.defaultExpression;
  if (state.column.hasDefault && state.column.defaultNull) {
    state.column.nullable = true;
  }
};

const handleSelectedColumnDefaultValueExpressionChange = (
  expression: string
) => {
  if (!state.column) {
    return;
  }
  state.column.hasDefault = true;
  state.column.defaultNull = undefined;
  state.column.defaultString = undefined;
  state.column.defaultExpression = expression;
  state.showColumnDefaultValueExpressionModal = false;
};

const onSemanticTypeApply = async (semanticTypeId: string) => {
  state.config = ColumnConfig.fromPartial({
    ...state.config,
    semanticTypeId,
  });
};
</script>
