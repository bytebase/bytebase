<template>
  <div class="w-full flex gap-x-2 items-start">
    <aside
      class="w-auto min-h-[16rem] border border-control-border p-1 flex flex-col gap-y-1"
    >
      <div
        v-for="source in riskSourceList"
        :key="source"
        class="pl-2 pr-8 py-1 rounded-sm cursor-pointer"
        :class="[
          source === selectedSource ? 'bg-gray-200' : 'hover:bg-gray-100',
        ]"
        @click="selectedSource = source"
      >
        {{ sourceText(source) }}
      </div>
    </aside>
    <div class="flex-1 space-y-6 mb-6">
      <div class="flex items-start justify-between gap-x-2">
        <div class="text-sm text-control-light space-x-1">
          <span>{{ $t("custom-approval.security-rule.description") }}</span>
          <LearnMoreLink url="https://www.bytebase.com/404" />
        </div>
        <div class="flex items-center justify-end">
          <NButton type="primary" :disabled="!allowAdmin" @click="addRisk">
            {{ $t("custom-approval.security-rule.add-rule") }}
          </NButton>
        </div>
      </div>

      <div class="space-y-4">
        <RiskSection
          v-for="group in selectedRiskListGroupByLevel"
          :key="group.level"
          :level="group.level"
          :risk-list="group.riskList"
        />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from "vue";

import LearnMoreLink from "@/components/LearnMoreLink.vue";
import RiskSection from "./RiskSection.vue";
import { useRiskCenterContext } from "./context";
import { sourceText } from "./common";
import { Risk, Risk_Source } from "@/types/proto/v1/risk_service";
import { PresetRiskLevelList } from "@/types";
import { useRiskStore } from "@/store";

const riskStore = useRiskStore();

const riskSourceList = [
  Risk_Source.DDL,
  Risk_Source.DML,
  Risk_Source.CREATE_DATABASE,
];

const selectedSource = ref<Risk_Source>(riskSourceList[0]);
const context = useRiskCenterContext();
const { allowAdmin } = context;
const selectedRiskList = computed(() => {
  return riskStore.riskList.filter(
    (risk) => risk.source === selectedSource.value
  );
});

const selectedRiskListGroupByLevel = computed(() => {
  return PresetRiskLevelList.map(({ level }) => {
    const riskList = selectedRiskList.value.filter(
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
    source: selectedSource.value,
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
    if (dialog) {
      const { risk } = dialog;
      selectedSource.value = risk.source;
    }
  },
  { immediate: true }
);
</script>
