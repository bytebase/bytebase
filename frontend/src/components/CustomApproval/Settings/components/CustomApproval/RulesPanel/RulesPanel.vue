<template>
  <div class="w-full">
    <RiskFilter :hide-search="true" />

    <div class="space-y-4 my-4">
      <RulesSection
        v-for="source in selectedSourceList"
        :key="source"
        :source="source"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useSupportedSourceList } from "@/types";
import { Risk_Source } from "@/types/proto/api/v1alpha/risk_service";
import { RiskFilter, useRiskFilter } from "../../common";
import RulesSection from "./RulesSection.vue";

const filter = useRiskFilter();
const SupportedSourceList = useSupportedSourceList();

const selectedSourceList = computed(() => {
  if (filter.source.value === Risk_Source.SOURCE_UNSPECIFIED) {
    // "ALL"
    return SupportedSourceList.value;
  }
  return [filter.source.value];
});
</script>
