<template>
  <aside class="hidden lg:block">
    <div class="space-y-6">
      <h1 class="text-left text-2xl font-semibold">
        {{ $t("schema-review-policy.rules") }}
      </h1>
      <fieldset v-for="(category, index) in categoryList" :key="index">
        <div class="block text-sm font-medium text-gray-900">
          {{ $t(`schema-review-policy.category.${category.id.toLowerCase()}`) }}
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
            {{ getRuleLocalization(rule.type).title }}
          </a>
        </div>
      </fieldset>
    </div>
  </aside>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import {
  RuleTemplate,
  getRuleLocalization,
  convertToCategoryList,
} from "@/types";

const props = withDefaults(
  defineProps<{
    selectedRuleList?: RuleTemplate[];
  }>(),
  {
    selectedRuleList: () => [],
  }
);

const categoryList = computed(() => {
  return convertToCategoryList(props.selectedRuleList);
});
</script>
