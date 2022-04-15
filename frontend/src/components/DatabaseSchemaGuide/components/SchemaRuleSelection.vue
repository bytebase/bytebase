<template>
  <nav class="" aria-label="rules">
    <div v-for="category in categoryList" :key="category.id" class="relative">
      <div
        class="z-10 sticky top-0 border-t border-b border-gray-200 bg-gray-50 px-2 py-1 text-base font-medium text-gray-500"
      >
        <h3>{{ category.name }}</h3>
      </div>
      <ul role="list" class="relative z-0 divide-y divide-gray-200">
        <li v-for="rule in category.ruleList" :key="rule.id" class="bg-white">
          <div
            class="relative px-2 py-5 flex items-center space-x-3 hover:bg-gray-100 focus-within:ring-2 focus-within:ring-inset focus-within:ring-indigo-500"
            :class="isRuleSelected(rule) ? 'bg-gray-100' : 'cursor-pointer'"
            @click="onSelect(rule)"
          >
            <div class="flex-1 min-w-0">
              <div class="focus:outline-none flex items-center">
                <span class="absolute inset-0" aria-hidden="true" />
                <div class="flex-1">
                  <p class="text-base font-medium text-gray-900 mb-3 space-x-2">
                    {{ rule.id }}
                    <BBBadge
                      v-for="db in rule.database"
                      :key="`${rule.id}-${db}`"
                      :text="db"
                      :can-remove="false"
                    />
                  </p>
                  <p class="text-sm text-gray-500 truncate grid grid-cols-2">
                    {{ rule.description }}
                  </p>
                </div>
                <heroicons-solid:check-circle
                  class="w-7 h-7 text-gray-600"
                  v-if="isRuleSelected(rule)"
                />
              </div>
            </div>
          </div>
        </li>
      </ul>
    </div>
  </nav>
</template>

<script lang="ts" setup>
import { computed, PropType } from "vue";
import {
  ruleList,
  SchemaRule,
  convertToCategoryList,
} from "../../../types/schemaSystem";

const props = defineProps({
  selectedRuleIdList: {
    required: true,
    type: Array as PropType<string[]>,
  },
});

const emit = defineEmits(["select"]);

const categoryList = computed(() => {
  return convertToCategoryList(ruleList);
});

const selectedRuleIdSet = computed(() => new Set(props.selectedRuleIdList));

const isRuleSelected = (rule: SchemaRule): boolean => {
  return selectedRuleIdSet.value.has(rule.id);
};

const onSelect = (rule: SchemaRule) => {
  if (isRuleSelected(rule)) {
    return;
  }

  emit("select", rule);
};
</script>
