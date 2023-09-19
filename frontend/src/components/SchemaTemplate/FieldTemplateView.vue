<template>
  <div v-if="templateList.length === 0" class="bg-white rounded-lg">
    <div
      class="border-4 border-dashed border-gray-200 rounded-lg h-96 flex justify-center items-center"
    >
      <div class="text-center flex flex-col justify-center items-center">
        <img src="../../assets/illustration/no-data.webp" class="w-52" />
      </div>
    </div>
  </div>
  <div v-else class="flex">
    <div
      class="hidden sm:flex w-1/5 max-w-xs flex-col space-y-3 border-r mr-5 pr-5"
    >
      <p class="text-lg">
        {{ $t("schema-template.form.category") }}
      </p>
      <div class="space-y-2">
        <label
          v-for="item in categoryList"
          :key="item.id"
          class="flex items-center gap-x-1 text-sm text-gray-600"
        >
          <input
            type="checkbox"
            :value="item"
            :checked="state.selectedCategory.has(item.id)"
            class="h-4 w-4 border-gray-300 rounded text-indigo-600 focus:ring-indigo-500"
            @input="toggleCategoryCheck(item.id)"
          />
          <span class="text-ellipsis whitespace-nowrap overflow-hidden">
            {{ item.text }}
          </span>
          <span
            class="items-center text-xs px-2 py-0.5 rounded-full bg-gray-200 text-gray-800"
          >
            {{ item.count }}
          </span>
        </label>
      </div>
    </div>
    <FieldTemplateTable
      :engine="engine"
      :readonly="readonly"
      :template-list="filteredTemplateList"
      class="flex-1"
      @view="(template: SchemaTemplateSetting_FieldTemplate) => $emit('view', template)"
      @apply="(template: SchemaTemplateSetting_FieldTemplate) => $emit('apply', template)"
    />
  </div>
</template>

<script lang="ts" setup>
import { reactive, computed } from "vue";
import { useI18n } from "vue-i18n";
import { Engine } from "@/types/proto/v1/common";
import { SchemaTemplateSetting_FieldTemplate } from "@/types/proto/v1/setting_service";

interface LocalState {
  selectedCategory: Set<string>;
}

const props = defineProps<{
  engine?: Engine;
  readonly: boolean;
  templateList: SchemaTemplateSetting_FieldTemplate[];
}>();

defineEmits<{
  (event: "view", item: SchemaTemplateSetting_FieldTemplate): void;
  (event: "apply", item: SchemaTemplateSetting_FieldTemplate): void;
}>();

const state = reactive<LocalState>({
  selectedCategory: new Set<string>(),
});
const { t } = useI18n();

const toggleCategoryCheck = (category: string) => {
  if (state.selectedCategory.has(category)) {
    state.selectedCategory.delete(category);
  } else {
    state.selectedCategory.add(category);
  }
};

const categoryList = computed(() => {
  const categoryMap = new Map<string, number>();
  for (const template of props.templateList) {
    const num = categoryMap.get(template.category) ?? 0;
    categoryMap.set(template.category, num + 1);
  }

  const resp = [];
  for (const [category, count] of categoryMap.entries()) {
    resp.push({
      id: category,
      text: category || t("schema-template.form.unclassified"),
      count,
    });
  }
  return resp;
});

const filteredTemplateList = computed(() => {
  if (state.selectedCategory.size === 0) {
    return props.templateList;
  }
  return props.templateList.filter((template) =>
    state.selectedCategory.has(template.category)
  );
});
</script>
