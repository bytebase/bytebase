<template>
  <div class="w-full space-y-4">
    <FeatureAttention
      v-if="!hasSensitiveDataFeature"
      feature="bb.feature.sensitive-data"
    />
    <NTabs v-model:value="state.selectedTab" type="line">
      <NTabPane
        name="global-masking-rule"
        :tab="$t('settings.sensitive-data.global-rules.self')"
      >
        <GlobalMaskingRulesView />
      </NTabPane>
      <NTabPane
        name="sensitive-column-list"
        :tab="$t('settings.sensitive-data.sensitive-column-list')"
      >
        <SensitiveColumnView />
      </NTabPane>
      <NTabPane
        name="semantic-types"
        :tab="$t('settings.sensitive-data.semantic-types.self')"
      >
        <SemanticTypesView />
      </NTabPane>
      <NTabPane
        name="masking-algorithms"
        :tab="$t('settings.sensitive-data.algorithms.self')"
      >
        <MaskingAlgorithmsView />
      </NTabPane>
    </NTabs>
  </div>
</template>

<script lang="ts" setup>
import { NTabs, NTabPane } from "naive-ui";
import { reactive, watch } from "vue";
import { useRouter, useRoute } from "vue-router";
import {
  SensitiveColumnView,
  GlobalMaskingRulesView,
  SemanticTypesView,
} from "@/components/SensitiveData";
import { featureToRef } from "@/store";

const dataMaskingTabList = [
  "sensitive-column-list",
  "global-masking-rule",
  "semantic-types",
  "masking-algorithms",
] as const;
type DataMaskingTab = typeof dataMaskingTabList[number];
const isDataMaskingTab = (tab: any): tab is DataMaskingTab =>
  dataMaskingTabList.includes(tab);

interface LocalState {
  selectedTab: DataMaskingTab;
}

const state = reactive<LocalState>({
  selectedTab: "global-masking-rule",
});
const hasSensitiveDataFeature = featureToRef("bb.feature.sensitive-data");
const router = useRouter();
const route = useRoute();

watch(
  () => route.hash,
  (hash) => {
    const tab = hash.slice(1);
    if (isDataMaskingTab(tab)) {
      state.selectedTab = tab;
    } else {
      state.selectedTab = "global-masking-rule";
    }
  },
  {
    immediate: true,
  }
);

watch(
  () => state.selectedTab,
  (tab) => {
    router.push({ hash: `#${tab}` });
  }
);
</script>
