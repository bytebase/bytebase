<template>
  <div class="w-full">
    <RiskFilter class="my-4">
      <template #suffix>
        <NButton type="primary" :disabled="!allowCreateRisk" @click="addRisk">
          <template #icon>
            <PlusIcon class="h-4 w-4" />
          </template>
          {{ $t("common.add") }}
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
import { PlusIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, watch } from "vue";
import { useRiskStore } from "@/store";
import { PresetRiskLevelList, useSupportedSourceList } from "@/types";
import { Risk, Risk_Source } from "@/types/proto/v1/risk_service";
import { hasWorkspacePermissionV2 } from "@/utils";
import { RiskFilter, orderByLevelDesc, useRiskFilter } from "../common";
import RiskSection from "./RiskSection.vue";
import { useRiskCenterContext } from "./context";

const context = useRiskCenterContext();
const riskStore = useRiskStore();
const filter = useRiskFilter();
const { hasFeature, showFeatureModal } = context;
const supportedSourceList = useSupportedSourceList();

const allowCreateRisk = computed(() => {
  return hasWorkspacePermissionV2("bb.risks.create");
});

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
  const groups = supportedSourceList.value.map((source) => {
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
  let source = filter.source.value;
  if (source === Risk_Source.SOURCE_UNSPECIFIED) {
    source = supportedSourceList.value[0];
  }
  const risk = Risk.fromPartial({
    level: PresetRiskLevelList[0].level,
    source,
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
