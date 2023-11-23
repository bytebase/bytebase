<template>
  <div class="w-full space-y-4 text-sm">
    <div v-if="!readonly" class="space-y-4">
      <div class="flex items-center justify-between gap-x-6">
        <div class="flex-1 textinfolabel">
          {{ $t("schema-template.table-template.description") }}
        </div>
        <div>
          <NButton
            type="primary"
            :disabled="readonly"
            @click="createSchemaTemplate"
          >
            {{ $t("common.add") }}
          </NButton>
        </div>
      </div>
    </div>
    <div class="flex items-center justify-between gap-x-2 my-4 pb-5 border-b">
      <div class="flex flex-row items-center justify-start gap-x-4">
        <template v-if="showEngineFilter">
          <NCheckbox
            v-for="item in engineList"
            :key="item"
            :checked="state.selectedEngine.has(item)"
            @update:checked="toggleEngineCheck(item)"
          >
            <div class="flex items-center gap-x-1 text-sm text-gray-600">
              <EngineIcon :engine="item" class="ml-0 mr-1" />
              <span
                class="items-center text-xs px-2 py-0.5 rounded-full bg-gray-200 text-gray-800"
              >
                {{ countTemplateByEngine(item) }}
              </span>
            </div>
          </NCheckbox>
        </template>
      </div>
      <SearchBox
        v-model:value="state.searchText"
        :autofocus="true"
        :placeholder="$t('schema-template.search-by-name-or-comment')"
        style="width: 18rem"
      />
    </div>
    <TableTemplateView
      :engine="engine"
      :readonly="!!readonly"
      :template-list="filteredTemplateList"
      @view="editSchemaTemplate"
      @apply="$emit('apply', $event)"
    />
  </div>
  <Drawer v-model:show="state.showDrawer">
    <TableTemplateForm
      :readonly="!!readonly"
      :create="!state.template.table?.name"
      :template="state.template"
      @dismiss="state.showDrawer = false"
    />
  </Drawer>
</template>

<script lang="ts" setup>
import { NButton, NCheckbox } from "naive-ui";
import { v1 as uuidv1 } from "uuid";
import { reactive, computed, onMounted } from "vue";
import TableTemplateForm from "@/components/SchemaTemplate/TableTemplateForm.vue";
import TableTemplateView from "@/components/SchemaTemplate/TableTemplateView.vue";
import { engineList } from "@/components/SchemaTemplate/utils";
import { Drawer, SearchBox } from "@/components/v2";
import { useSettingV1Store } from "@/store";
import { Engine } from "@/types/proto/v1/common";
import { TableMetadata, TableConfig } from "@/types/proto/v1/database_service";
import { SchemaTemplateSetting_TableTemplate } from "@/types/proto/v1/setting_service";

interface LocalState {
  template: SchemaTemplateSetting_TableTemplate;
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
  (event: "apply", item: SchemaTemplateSetting_TableTemplate): void;
}>();

const initialTemplate = (): SchemaTemplateSetting_TableTemplate => ({
  id: uuidv1(),
  engine: props.engine ?? Engine.MYSQL,
  category: "",
  table: TableMetadata.fromPartial({
    name: "",
    comment: "",
    columns: [],
  }),
  config: TableConfig.fromPartial({}),
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

const editSchemaTemplate = (template: SchemaTemplateSetting_TableTemplate) => {
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
  return setting?.value?.schemaTemplateSettingValue?.tableTemplates ?? [];
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
  template: SchemaTemplateSetting_TableTemplate
) => {
  const keyword = state.searchText.trim().toLowerCase();
  if (!keyword) return true;
  if (template.table?.name.toLowerCase().includes(keyword)) {
    return true;
  }
  if (template.table?.userComment.toLowerCase().includes(keyword)) {
    return true;
  }
  return false;
};
</script>
