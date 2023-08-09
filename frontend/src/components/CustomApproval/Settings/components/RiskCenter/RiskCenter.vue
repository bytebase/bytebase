<template>
  <div class="w-full">
    <RiskFilter class="my-4">
      <template #suffix>
        <NButton type="primary" :disabled="!allowAdmin" @click="addRisk">
          {{ $t("custom-approval.security-rule.add-rule") }}
        </NButton>
      </template>
    </RiskFilter>

    <div class="space-y-4 pb-4">
      <RiskSection
        v-for="{ source, riskList } in riskListGroupBySource"
        :key="source"
        :source="source"
        :risk-list="riskList"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { groupBy } from "lodash-es";
import { computed, watch } from "vue";
import { useRiskStore } from "@/store";
import { PresetRiskLevelList, SupportedSourceList } from "@/types";
import { Risk, Risk_Source } from "@/types/proto/v1/risk_service";
import { RiskFilter, orderByLevelDesc, useRiskFilter } from "../common";
import RiskSection from "./RiskSection.vue";
import { useRiskCenterContext } from "./context";

const riskStore = useRiskStore();
const context = useRiskCenterContext();
const filter = useRiskFilter();
const { allowAdmin, hasFeature, showFeatureModal } = context;

const filteredRiskList = computed(() => {
  let list = [...riskStore.riskList];
  const { source, levels } = filter;
  const search = filter.search.value.trim();
  // Risk_Source.SOURCE_UNSPECIFIED to "ALL"
  if (source.value !== Risk_Source.SOURCE_UNSPECIFIED) {
    list = list.filter((risk) => risk.source === source.value);
  }
  // empty to "ALL"
  if (levels.value.size > 0) {
    list = list.filter((risk) => levels.value.has(risk.level));
  }
  if (search) {
    list = list.filter((risk) => risk.title.includes(search));
  }
  return list;
});

const riskListGroupBySource = computed(() => {
  const groupBySource = groupBy(filteredRiskList.value, (risk) => risk.source);
  const groups = SupportedSourceList.map((source) => {
    const riskList = groupBySource[source] ?? [];
    riskList.sort(orderByLevelDesc);
    return { source, riskList };
  });
  if (filter.source.value === Risk_Source.SOURCE_UNSPECIFIED) {
    // Show "ALL" sources
    return groups;
  }

  return groups.filter((group) => {
    return group.riskList.length > 0 || group.source === filter.source.value;
  });
});

const addRisk = () => {
  const risk = Risk.fromJSON({
    level: PresetRiskLevelList[0].level,
    source: filter.source.value || SupportedSourceList[0],
    active: true,
  });
  if (!hasFeature.value) {
    showFeatureModal.value = true;
    return;
  }
  context.dialog.value = {
    mode: "CREATE",
    risk,
  };
};

watch(
  context.dialog,
  (dialog) => {
    const source = dialog?.risk?.source;
    if (!source) return;
    if (filter.source.value !== Risk_Source.SOURCE_UNSPECIFIED) {
      filter.source.value = source;
    }
  },
  { immediate: true }
);
</script>
