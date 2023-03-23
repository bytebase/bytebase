<template>
  <div class="w-full">
    <div class="flex justify-end">
      <NButton type="primary" :disabled="!allowAdmin" @click="addRisk">
        {{ $t("custom-approval.security-rule.add-rule") }}
      </NButton>
    </div>

    <RiskNavigation class="my-4" />

    <div class="space-y-4">
      <RiskSection
        v-for="group in selectedRiskListGroupByLevel"
        :key="group.level"
        :level="group.level"
        :risk-list="group.riskList"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, watch } from "vue";

import RiskNavigation from "./RiskNavigation";
import RiskSection from "./RiskSection.vue";
import { useRiskCenterContext } from "./context";
import { Risk, Risk_Source } from "@/types/proto/v1/risk_service";
import { PresetRiskLevelList, SupportedSourceList } from "@/types";
import { useRiskStore } from "@/store";

const riskStore = useRiskStore();
const context = useRiskCenterContext();
const { allowAdmin, navigation } = context;

const filteredRiskList = computed(() => {
  let list = [...riskStore.riskList];
  const { source, levels } = navigation.value;
  const search = navigation.value.search.trim();
  // Risk_Source.SOURCE_UNSPECIFIED to "ALL"
  if (source !== Risk_Source.SOURCE_UNSPECIFIED) {
    list = list.filter((risk) => risk.source === source);
  }
  // empty to "ALL"
  if (levels.size > 0) {
    list = list.filter((risk) => levels.has(risk.level));
  }
  if (search) {
    list = list.filter((risk) => risk.title.includes(search));
  }
  return list;
});

const selectedRiskListGroupByLevel = computed(() => {
  return PresetRiskLevelList.map(({ level }) => {
    const riskList = filteredRiskList.value.filter(
      (risk) => risk.level === level
    );
    return {
      level,
      riskList,
    };
  });
});

const addRisk = () => {
  const risk = Risk.fromJSON({
    level: PresetRiskLevelList[0].level,
    source: navigation.value.source ?? SupportedSourceList[0],
    active: true,
  });
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
    if (navigation.value.source !== Risk_Source.SOURCE_UNSPECIFIED) {
      navigation.value.source = source;
    }
  },
  { immediate: true }
);
</script>
