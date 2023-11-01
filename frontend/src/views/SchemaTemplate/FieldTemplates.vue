<template>
  <div class="w-full space-y-4 text-sm">
    <div v-if="!readonly" class="space-y-4">
      <div class="flex items-center justify-between gap-x-6">
        <div class="flex-1 textinfolabel">
          {{ $t("schema-template.field-template.description") }}
        </div>
        <div>
          <NButton type="primary" @click="createSchemaTemplate">
            {{ $t("schema-template.field-template.add") }}
          </NButton>
        </div>
      </div>
    </div>
    <div class="flex items-center gap-x-5 my-4 pb-5 border-b">
      <template v-if="showEngineFilter">
        <label
          v-for="item in engineList"
          :key="item"
          class="flex items-center gap-x-1 text-sm text-gray-600"
        >
          <input
            type="checkbox"
            :value="item"
            :checked="state.selectedEngine.has(item)"
            class="h-4 w-4 border-gray-300 rounded text-indigo-600 focus:ring-indigo-500"
            @input="toggleEngineCheck(item)"
          />
          <EngineIcon :engine="item" custom-class="ml-0 mr-1" />
          <span
            class="items-center text-xs px-2 py-0.5 rounded-full bg-gray-200 text-gray-800"
          >
            {{ countTemplateByEngine(item) }}
          </span>
        </label>
      </template>
      <BBTableSearch
        ref="searchField"
        class="ml-auto w-72"
        :placeholder="$t('schema-template.search-by-name-or-comment')"
        @change-text="(val: string) => state.searchText = val"
      />
    </div>
    <FieldTemplateView
      :engine="engine"
      :readonly="!!readonly"
      :template-list="filteredTemplateList"
      @view="editSchemaTemplate"
      @apply="$emit('apply', $event)"
    />
  </div>
  <Drawer :show="state.showDrawer" @close="state.showDrawer = false">
    <FieldTemplateForm
      :readonly="!!readonly"
      :create="!state.template.column?.name"
      :template="state.template"
      @dismiss="state.showDrawer = false"
    />
  </Drawer>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { v1 as uuidv1 } from "uuid";
import { reactive, computed, onMounted } from "vue";
import { engineList } from "@/components/SchemaTemplate/utils";
import { Drawer } from "@/components/v2";
import { useSettingV1Store } from "@/store";
import { Engine } from "@/types/proto/v1/common";
import {
  ColumnMetadata,
  ColumnConfig,
} from "@/types/proto/v1/database_service";
import { SchemaTemplateSetting_FieldTemplate } from "@/types/proto/v1/setting_service";

interface LocalState {
  template: SchemaTemplateSetting_FieldTemplate;
  showDrawer: boolean;
  searchText: string;
  selectedEngine: Set<Engine>;
}

const props = defineProps<{
  engine?: Engine;
  readonly?: boolean;
  showEngineFilter?: boolean;
}>();

defineEmits<{
  (event: "apply", item: SchemaTemplateSetting_FieldTemplate): void;
}>();

const initialTemplate = () => ({
  id: uuidv1(),
  engine: props.engine ?? Engine.MYSQL,
  category: "",
  column: ColumnMetadata.fromPartial({
    name: "",
    type: "",
    nullable: false,
    comment: "",
    position: 0,
    characterSet: "",
    collation: "",
  }),
  config: ColumnConfig.fromPartial({}),
});

const state = reactive<LocalState>({
  showDrawer: false,
  template: initialTemplate(),
  searchText: "",
  selectedEngine: new Set<Engine>(),
});

onMounted(() => {
  if (props.engine) {
    state.selectedEngine.add(props.engine);
  }
});

const createSchemaTemplate = () => {
  state.template = initialTemplate();
  state.showDrawer = true;
};

const editSchemaTemplate = (template: SchemaTemplateSetting_FieldTemplate) => {
  state.template = template;
  state.showDrawer = true;
};

const toggleEngineCheck = (engine: Engine) => {
  if (state.selectedEngine.has(engine)) {
    state.selectedEngine.delete(engine);
  } else {
    state.selectedEngine.add(engine);
  }
};

const settingStore = useSettingV1Store();

const schemaTemplateList = computed(() => {
  const setting = settingStore.getSettingByName("bb.workspace.schema-template");
  return setting?.value?.schemaTemplateSettingValue?.fieldTemplates ?? [];
});

const countTemplateByEngine = (engine: Engine) => {
  return schemaTemplateList.value.filter(
    (template) => template.engine === engine
  ).length;
};

const filteredTemplateList = computed(() => {
  if (state.selectedEngine.size === 0) {
    return schemaTemplateList.value.filter(filterTemplateByKeyword);
  }
  return schemaTemplateList.value.filter(
    (template) =>
      state.selectedEngine.has(template.engine) &&
      filterTemplateByKeyword(template)
  );
});

const filterTemplateByKeyword = (
  template: SchemaTemplateSetting_FieldTemplate
) => {
  const keyword = state.searchText.trim().toLowerCase();
  if (!keyword) return true;
  if (template.column?.name.toLowerCase().includes(keyword)) {
    return true;
  }
  if (template.column?.userComment.toLowerCase().includes(keyword)) {
    return true;
  }
  return false;
};
</script>
