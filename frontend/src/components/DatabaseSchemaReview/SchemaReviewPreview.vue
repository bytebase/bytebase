<template>
  <div class="flex gap-x-20">
    <aside class="hidden lg:block">
      <div class="space-y-6">
        <h1 class="text-left text-2xl font-semibold">Rules</h1>
        <fieldset v-for="(category, index) in categoryList" :key="index">
          <div class="block text-sm font-medium text-gray-900">
            {{
              $t(`schema-review-policy.category.${category.id.toLowerCase()}`)
            }}
          </div>
          <div
            v-for="(rule, ruleIndex) in category.ruleList"
            :key="ruleIndex"
            class="pt-2 flex items-center text-sm group"
          >
            <a
              :href="`#${rule.type.replace(/\./g, '-')}`"
              class="text-gray-600 hover:underline cursor-pointer"
            >
              {{ rule.type }}
            </a>
          </div>
        </fieldset>
      </div>
    </aside>
    <div class="flex-1">
      <div class="mb-5" v-if="name">
        <h1 class="text-left text-3xl font-bold mb-2">
          {{ name }}
        </h1>
      </div>
      <div v-if="selectedEnvironment" class="flex flex-wrap gap-x-3 mb-9">
        <span class="font-semibold">{{ $t("common.environment") }}</span>
        <BBBadge
          :text="environmentName(selectedEnvironment)"
          :can-remove="false"
        />
      </div>
      <BBAttention
        v-else-if="isPreviewStep"
        class="my-5"
        :style="`WARN`"
        :title="
          $t('schema-review-policy.create.basic-info.no-linked-environments')
        "
      />
      <div v-for="category in categoryList" :key="category.id" class="pb-4">
        <a
          :href="`#${category.id.replace(/\./g, '-')}`"
          :id="category.id.replace(/\./g, '-')"
          class="text-left text-2xl text-indigo-600 font-semibold hover:underline"
        >
          {{ $t(`schema-review-policy.category.${category.id.toLowerCase()}`) }}
        </a>
        <div v-for="rule in category.ruleList" :key="rule.type" class="py-4">
          <div class="sm:flex sm:items-center sm:space-x-4">
            <a
              :href="`#${rule.type.replace(/\./g, '-')}`"
              :id="rule.type.replace(/\./g, '-')"
              class="text-left text-xl hover:underline whitespace-nowrap"
            >
              {{ rule.type }}
            </a>
            <div class="mt-3 flex items-center space-x-2 sm:mt-0">
              <SchemaRuleLevelBadge :level="rule.level" />
              <BBBadge
                :text="$t(`engine.${rule.engine.toLowerCase()}`)"
                :can-remove="false"
              />
            </div>
          </div>
          <p class="py-2 text-gray-400">{{ rule.description }}</p>
          <ul role="list" class="space-y-4 list-disc list-inside">
            <li
              v-for="(component, i) in rule.componentList"
              :key="i"
              class="leading-8"
            >
              {{ component.title }}:
              <span
                v-if="
                  component.payload.type === 'STRING' ||
                  component.payload.type === 'TEMPLATE'
                "
                class="bg-gray-100 rounded text-sm font-semibold p-2"
              >
                {{ component.payload.value ?? component.payload.default }}
              </span>
              <div
                v-else-if="component.payload.type === 'STRING_ARRAY'"
                class="flex flex-wrap gap-3 ml-5 mt-3"
              >
                <span
                  v-for="(val, j) in component.payload.value ??
                  component.payload.default"
                  :key="`${i}-${j}`"
                  class="bg-gray-100 rounded text-sm font-semibold p-2"
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
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { RuleTemplate, convertToCategoryList, Environment } from "../../types";
import { environmentName } from "../../utils";

const props = withDefaults(
  defineProps<{
    name?: string;
    isPreviewStep?: boolean;
    selectedEnvironment?: Environment;
    selectedRuleList?: RuleTemplate[];
  }>(),
  {
    name: "",
    isPreviewStep: false,
    selectedEnvironment: undefined,
    selectedRuleList: () => [],
  }
);

const categoryList = computed(() => {
  return convertToCategoryList(props.selectedRuleList);
});
</script>
