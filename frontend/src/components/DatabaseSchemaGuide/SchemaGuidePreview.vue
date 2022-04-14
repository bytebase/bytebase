<template>
  <div class="flex gap-x-20">
    <aside class="hidden lg:block">
      <div class="space-y-6">
        <h1 class="text-left text-2xl font-semibold">Rules</h1>
        <fieldset v-for="(category, index) in categoryList" :key="index">
          <div class="block text-sm font-medium text-gray-900">
            {{ category.name }}
          </div>
          <div
            v-for="(rule, ruleIndex) in category.ruleList"
            :key="ruleIndex"
            class="pt-2 flex items-center text-sm group"
          >
            <a
              :href="`#${rule.id.replace(/\./g, '-')}`"
              class="text-gray-600 hover:underline cursor-pointer"
            >
              {{ rule.id }}
            </a>
          </div>
        </fieldset>
      </div>
    </aside>
    <div class="flex-1">
      <div class="mb-9" v-if="name">
        <h1 class="text-left text-3xl font-bold mb-2">
          {{ name }}
        </h1>
      </div>
      <div v-for="category in categoryList" :key="category.id" class="pb-4">
        <a
          :href="`#${category.id.replace(/\./g, '-')}`"
          :id="category.id.replace(/\./g, '-')"
          class="text-left text-2xl text-indigo-600 font-semibold hover:underline"
        >
          {{ category.name }}
        </a>
        <div v-for="rule in category.ruleList" :key="rule.id" class="py-4">
          <div class="sm:flex sm:items-center sm:space-x-4">
            <a
              :href="`#${rule.id.replace(/\./g, '-')}`"
              :id="rule.id.replace(/\./g, '-')"
              class="text-left text-xl text-gray-600 hover:underline whitespace-nowrap"
            >
              {{ rule.id }}
            </a>
            <div class="mt-3 flex items-center space-x-2 sm:mt-0">
              <SchemaRuleLevelBadge :level="rule.level" />
              <BBBadge
                v-for="db in rule.database"
                :key="`${rule.id}-${db}`"
                :text="db"
                :can-remove="false"
              />
            </div>
          </div>
          <p class="py-2 text-gray-400">{{ rule.description }}</p>
          <div v-if="rule.payload" class="mt-1">
            <ul role="list" class="space-y-4 list-disc list-inside">
              <li
                v-for="key in Object.keys(rule.payload)"
                :key="key"
                class="leading-8"
              >
                {{ key }}:
                <span
                  v-if="
                    rule.payload[key].type === 'string' ||
                    rule.payload[key].type === 'template'
                  "
                  class="bg-gray-100 rounded text-sm p-2"
                >
                  {{ rule.payload[key].value || rule.payload[key].default }}
                </span>
                <div
                  v-else-if="rule.payload[key].type === 'string[]'"
                  class="flex flex-wrap gap-3 ml-5 mt-3"
                >
                  <span
                    v-for="(val, i) in rule.payload[key].value ||
                    rule.payload[key].default"
                    :key="i"
                    class="bg-gray-100 rounded text-sm p-2"
                  >
                    {{ val }}
                  </span>
                </div>
              </li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { PropType, computed } from "vue";
import { SelectedRule, convertToCategoryList } from "../../types/schemaSystem";

const props = defineProps({
  name: {
    required: false,
    default: "",
    type: String,
  },
  ruleList: {
    required: true,
    type: Array as PropType<SelectedRule[]>,
  },
});

const categoryList = computed(() => {
  return convertToCategoryList(props.ruleList);
});
</script>
