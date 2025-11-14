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

    <div class="flex flex-col gap-y-4 pb-4">
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
import { create } from "@bufbuild/protobuf";
import { groupBy } from "lodash-es";
import { PlusIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed } from "vue";
import { useRiskStore } from "@/store";
import { PresetRiskLevelList, useSupportedSourceList } from "@/types";
import { RiskSchema } from "@/types/proto-es/v1/risk_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import { orderByLevelDesc, RiskFilter, useRiskFilter } from "../common";
import { useRiskCenterContext } from "./context";
import RiskSection from "./RiskSection.vue";

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
  const search = filter.search.value.trim();
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
  return groups;
});

const addRisk = () => {
  const source = supportedSourceList.value[0];
  const risk = create(RiskSchema, {
    level: PresetRiskLevelList[0].level,
    source: source,
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
</script>
