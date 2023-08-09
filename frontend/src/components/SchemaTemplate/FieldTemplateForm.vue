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

          <!-- type -->
          <div class="sm:col-span-2 sm:col-start-1">
            <label for="column-type" class="textlabel">
              {{ $t("schema-template.form.column-type") }}
              <span class="text-red-600 mr-2">*</span>
            </label>
            <div
              class="relative flex flex-row justify-between items-center mt-1"
            >
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
            </div>
          </div>

          <!-- default value -->
          <div class="sm:col-span-2 sm:col-start-1">
            <label for="default-value" class="textlabel">
              {{ $t("schema-template.form.default-value") }}
            </label>
            <input
              v-model="state.column!.default"
              required
              name="default-value"
              type="text"
              class="textfield mt-1 w-full"
              :placeholder="getDefaultValue(state.column)"
              :disabled="!allowEdit"
            />
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
              v-model="state.column!.comment"
              rows="3"
              class="textfield block w-full resize-none mt-1 text-sm text-control rounded-md whitespace-pre-wrap"
              :disabled="!allowEdit"
            />
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="w-full flex justify-between items-center">
        <div class="w-full flex justify-end items-center gap-x-3">
          <NButton @click.prevent="$emit('dismiss')">
            {{ $t("common.cancel") }}
          </NButton>
          <NButton
            v-if="!readonly && allowEdit"
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
</template>

<script lang="ts" setup>
import { isEqual } from "lodash-es";
import { computed, reactive } from "vue";
import { DrawerContent } from "@/components/v2";
import { useSchemaEditorStore } from "@/store";
import { Engine } from "@/types/proto/v1/common";
import { SchemaTemplateSetting_FieldTemplate } from "@/types/proto/v1/setting_service";
import {
  getDataTypeSuggestionList,
  engineNameV1,
  useWorkspacePermissionV1,
} from "@/utils";
import { engineList, getDefaultValue } from "./utils";

const props = defineProps<{
  create: boolean;
  readonly?: boolean;
  template: SchemaTemplateSetting_FieldTemplate;
}>();

const emit = defineEmits(["dismiss"]);

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface LocalState extends SchemaTemplateSetting_FieldTemplate {}

const state = reactive<LocalState>({
  id: props.template.id,
  engine: props.template.engine,
  category: props.template.category,
  column: Object.assign({}, props.template.column),
});
const store = useSchemaEditorStore();
const allowEdit = computed(() => {
  return (
    useWorkspacePermissionV1("bb.permission.workspace.manage-general").value &&
    !props.readonly
  );
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
  const options = [];
  for (const category of new Set(
    store.schemaTemplateList.map((template) => template.category)
  ).values()) {
    if (!category) {
      continue;
    }
    options.push({
      label: category,
      key: category,
    });
  }
  return options;
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
  await store.upsertSchemaTemplate(state);
  emit("dismiss");
};
</script>
