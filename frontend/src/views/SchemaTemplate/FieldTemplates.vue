<template>
  <div class="w-full space-y-4 text-sm">
    <FeatureAttention
      feature="bb.feature.schema-template"
      custom-class="my-4"
    />
    <div class="space-y-4">
      <div class="flex items-center justify-between gap-x-6">
        <div class="flex-1 textinfolabel">
          {{ $t("schema-template.description") }}
        </div>
        <div>
          <NButton
            type="primary"
            :disabled="!hasPermission"
            @click="createSchemaTemplate"
          >
            {{ $t("schema-template.add-field") }}
          </NButton>
        </div>
      </div>
    </div>
    <div class="flex items-center gap-x-5 my-5 pb-5 border-b">
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
      <BBTableSearch
        ref="searchField"
        class="ml-auto"
        :placeholder="$t('schema-template.search-by-name-or-comment')"
        @change-text="(val: string) => state.searchText = val"
      />
    </div>
    <FieldTemplateView
      :engine="engine"
      :readonly="!hasPermission"
      :template-list="filteredTemplateList"
      @view="editSchemaTemplate"
      @apply="(template: SchemaTemplateSetting_FieldTemplate) => $emit('apply', template)"
    />
  </div>
  <Drawer :show="state.showDrawer" @close="state.showDrawer = false">
    <FieldTemplateForm
      :create="!state.template.column?.name"
      :template="state.template"
      @dismiss="state.showDrawer = false"
    />
  </Drawer>
  <FeatureModal
    feature="bb.feature.schema-template"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { reactive, computed, onMounted } from "vue";
import { NButton } from "naive-ui";
import { Drawer } from "@/components/v2";
import { v1 as uuidv1 } from "uuid";

import { featureToRef, useSchemaEditorStore } from "@/store";
import { useWorkspacePermissionV1 } from "@/utils";
import { SchemaTemplateSetting_FieldTemplate } from "@/types/proto/v1/setting_service";
import { Engine } from "@/types/proto/v1/common";
import { engineList } from "@/components/SchemaTemplate/utils";
import { ColumnMetadata } from "@/types/proto/v1/database_service";

interface LocalState {
  template: SchemaTemplateSetting_FieldTemplate;
  showDrawer: boolean;
  showFeatureModal: boolean;
  searchText: string;
  selectedEngine: Set<Engine>;
}

const props = defineProps<{
  engine?: Engine;
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
});

const state = reactive<LocalState>({
  showDrawer: false,
  showFeatureModal: false,
  template: initialTemplate(),
  searchText: "",
  selectedEngine: new Set<Engine>(),
});
const store = useSchemaEditorStore();

onMounted(() => {
  if (props.engine) {
    state.selectedEngine.add(props.engine);
  }
});

const hasFeature = featureToRef("bb.feature.schema-template");
const hasPermission = useWorkspacePermissionV1(
  "bb.permission.workspace.manage-general"
);

const createSchemaTemplate = () => {
  if (!hasFeature.value) {
    state.showFeatureModal = true;
    return;
  }
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

const countTemplateByEngine = (engine: Engine) => {
  return store.schemaTemplateList.filter(
    (template) => template.engine === engine
  ).length;
};

const filteredTemplateList = computed(() => {
  if (state.selectedEngine.size === 0) {
    return store.schemaTemplateList.filter(filterTemplateByKeyword);
  }
  return store.schemaTemplateList.filter(
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
  if (template.column?.comment.toLowerCase().includes(keyword)) {
    return true;
  }
  return false;
};
</script>
