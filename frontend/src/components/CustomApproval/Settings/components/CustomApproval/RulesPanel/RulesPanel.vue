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
import { SupportedSourceList } from "@/types";
import { Risk_Source } from "@/types/proto/v1/risk_service";
import { RiskFilter, useRiskFilter } from "../../common";
import RulesSection from "./RulesSection.vue";

const filter = useRiskFilter();

const selectedSourceList = computed(() => {
  if (filter.source.value === Risk_Source.SOURCE_UNSPECIFIED) {
    // "ALL"
    return SupportedSourceList;
  }
  return [filter.source.value];
});
</script>
