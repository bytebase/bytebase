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
            <DropdownInput
              v-model:value="state.category"
              :options="categoryOptions"
              :placeholder="$t('schema-template.form.unclassified')"
              :disabled="!allowEdit"
              :consistent-menu-width="true"
            />
          </div>
        </div>

        <div class="w-full mb-6 space-y-1">
          <label for="engine" class="textlabel">
            {{ $t("database.engine") }}
          </label>
          <div class="grid grid-cols-4 gap-2">
            <NButton
              v-for="engine in engineList"
              :key="engine"
              size="large"
              ghost
              class="column-template-engine-button"
              :type="state.engine === engine ? 'primary' : 'default'"
              @click="changeEngine(engine)"
            >
              <NRadio
                :checked="state.engine === engine"
                size="large"
                class="btn mr-2 pointer-events-none"
              />
              <EngineIcon
                :engine="engine"
                class="w-5 h-auto max-h-[20px] object-contain mr-1 ml-0"
              />
              <RichEngineName
                :engine="engine"
                tag="p"
                class="text-center text-sm !text-main"
              />
            </NButton>
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
            <NInput
              v-model:value="state.column!.name"
              placeholder="column name"
              :disabled="!allowEdit"
            />
          </div>

          <div class="sm:col-span-1 sm:col-start-1">
            <label for="semantic-types" class="textlabel">
              {{ $t("settings.sensitive-data.semantic-types.self") }}
            </label>
            <div class="flex items-center gap-x-2 mt-3 text-sm">
              {{ columnSemanticType?.title }}
              <div v-if="allowEdit" class="flex items-center">
                <MiniActionButton
                  v-if="columnSemanticType"
                  @click.prevent="onSemanticTypeApply('')"
                >
                  <XIcon class="w-4 h-4" />
                </MiniActionButton>
                <MiniActionButton
                  @click.prevent="state.showSemanticTypesDrawer = true"
                >
                  <PencilIcon class="w-4 h-4" />
                </MiniActionButton>
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
                <MiniActionButton
                  v-if="state.column?.classification"
                  @click.prevent="state.column!.classification = ''"
                >
                  <XIcon class="w-4 h-4" />
                </MiniActionButton>
                <MiniActionButton
                  @click.prevent="state.showClassificationDrawer = true"
                >
                  <PencilIcon class="w-4 h-4" />
                </MiniActionButton>
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
                :disabled="!allowEdit"
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
              <DropdownInput
                :value="getColumnDefaultDisplayString(state.column!)||null"
                :options="defaultValueOptions"
                :disabled="!allowEdit"
                :placeholder="getColumnDefaultValuePlaceholder(state.column!)"
                @update:value="handleColumnDefaultChange"
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
                :disabled="!allowEdit"
              />
            </div>
          </div>

          <!-- comment -->
          <div class="sm:col-span-4 sm:col-start-1">
            <label for="comment" class="textlabel">
              {{ $t("schema-template.form.comment") }}
            </label>
            <NInput
              v-model:value="state.column!.userComment"
              type="textarea"
              :autosize="{ minRows: 3, maxRows: 3 }"
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
import { XIcon, PencilIcon } from "lucide-vue-next";
import { NButton, NInput, NRadio, NSwitch, SelectOption } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { EngineIcon } from "@/components/Icon";
import { LabelListEditor } from "@/components/Label";
import {
  getColumnDefaultDisplayString,
  getColumnDefaultValuePlaceholder,
  getDefaultValueByKey,
  getColumnDefaultValueOptions,
  isTextOfColumnType,
} from "@/components/SchemaEditorV1/utils/columnDefaultValue";
import {
  DrawerContent,
  DropdownInput,
  MiniActionButton,
  RichEngineName,
} from "@/components/v2";
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
  useWorkspacePermissionV1,
  convertKVListToLabels,
  convertLabelsToKVList,
} from "@/utils";
import ColumnDefaultValueExpressionModal from "../SchemaEditorV1/Modals/ColumnDefaultValueExpressionModal.vue";
import SemanticTypesDrawer from "../SensitiveData/components/SemanticTypesDrawer.vue";
import { engineList, categoryList, classificationConfig } from "./utils";

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

const defaultValueOptions = computed(() => {
  if (!state.column) return [];
  return getColumnDefaultValueOptions(
    state.engine,
    state.column.type
  ).map<SelectOption>((opt) => ({
    value: opt.key,
    label: opt.label as string,
    defaultValue: opt.value,
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
const schemaTemplateColumnTypeOptions = computed(() => {
  return schemaTemplateColumnTypes.value.map<SelectOption>((type) => ({
    label: type,
    value: type,
  }));
});

const changeEngine = (engine: Engine) => {
  if (allowEdit.value) {
    state.engine = engine;
  }
};

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

const handleColumnDefaultChange = (key: string) => {
  const value = getDefaultValueByKey(key);
  if (value) {
    handleColumnDefaultSelect(key);
    return;
  }

  handleColumnDefaultInput(key);
};
const handleColumnDefaultInput = (value: string) => {
  const { column } = state;
  if (!column) return;

  column.hasDefault = true;
  column.defaultNull = undefined;
  // If column is text type or has default string, we will treat user's input as string.
  if (
    isTextOfColumnType(state.engine, column.type) ||
    column.defaultString !== undefined
  ) {
    column.defaultString = value;
    column.defaultExpression = undefined;
    return;
  }
  // Otherwise we will treat user's input as expression.
  column.defaultExpression = value;
};
const handleColumnDefaultSelect = (key: string) => {
  const { column } = state;
  if (!column) return;

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

<style lang="postcss" scoped>
.column-template-engine-button :deep(.n-button__content) {
  @apply w-full justify-start;
}
</style>
