<template>
  <div class="flex gap-x-20">
    <SQLReviewSidebar :selected-rule-list="selectedRuleList" />
    <div class="flex-1">
      <div v-if="name" class="mb-5">
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
        :title="$t('sql-review.create.basic-info.no-linked-environments')"
      />
      <div v-for="category in categoryList" :key="category.id" class="pb-4">
        <a
          :id="category.id.replace(/\./g, '-')"
          :href="`#${category.id.replace(/\./g, '-')}`"
          class="text-left text-2xl text-indigo-600 font-semibold hover:underline"
        >
          {{ $t(`sql-review.category.${category.id.toLowerCase()}`) }}
        </a>
        <div v-for="rule in category.ruleList" :key="rule.type" class="py-4">
          <div class="sm:flex sm:items-center sm:space-x-4">
            <a
              :id="rule.type.replace(/\./g, '-')"
              :href="`#${rule.type.replace(/\./g, '-')}`"
              class="text-left text-xl hover:underline whitespace-nowrap"
            >
              {{ getRuleLocalization(rule.type).title }}
            </a>
            <div class="mt-3 flex items-center space-x-2 sm:mt-0">
              <SQLRuleLevelBadge :level="rule.level" />
              <img
                v-for="engine in rule.engineList"
                :key="engine"
                class="h-4 w-auto"
                :src="getEngineIcon(engine)"
              />
              <a
                :href="`https://www.bytebase.com/docs/sql-review/review-rules/supported-rules#${rule.type}`"
                target="__blank"
                class="flex flex-row space-x-2 items-center text-base text-gray-500 hover:text-gray-900"
              >
                <heroicons-outline:external-link class="w-4 h-4" />
              </a>
            </div>
          </div>
          <p class="py-2 text-gray-400">
            {{ getRuleLocalization(rule.type).description }}
          </p>
          <ul
            role="list"
            :class="[
              'space-y-2 list-disc list-inside',
              rule.componentList.length > 0 ? 'mt-3' : '',
            ]"
          >
            <li
              v-for="(component, i) in rule.componentList"
              :key="i"
              class="leading-8"
            >
              {{
                $t(
                  `sql-review.rule.${getRuleLocalizationKey(
                    rule.type
                  )}.component.${component.key}.title`
                )
              }}:
              <span
                v-if="
                  component.payload.type === 'STRING' ||
                  component.payload.type === 'NUMBER' ||
                  component.payload.type === 'BOOLEAN' ||
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
import {
  RuleTemplate,
  getRuleLocalization,
  getRuleLocalizationKey,
  convertToCategoryList,
  Environment,
  SchemaRuleEngineType,
} from "@/types";
import { environmentName } from "@/utils";

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

const getEngineIcon = (engine: SchemaRuleEngineType) =>
  new URL(`../../assets/db-${engine.toLowerCase()}.png`, import.meta.url).href;
</script>
