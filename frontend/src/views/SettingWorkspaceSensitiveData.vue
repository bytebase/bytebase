<template>
  <div class="w-full mt-4 space-y-4">
    <FeatureAttentionForInstanceLicense
      v-if="hasSensitiveDataFeature"
      feature="bb.feature.sensitive-data"
    />
    <FeatureAttention v-else feature="bb.feature.sensitive-data" />

    <BBTab
      :tab-item-list="tabItemList"
      :selected-index="state.selectedIndex"
      reorder-model="NEVER"
      @select-index="(index: number) => state.selectedIndex = index"
    >
      <div class="mt-5">
        <BBTabPanel :active="state.selectedIndex === 0">
          <SensitiveColumnList />
        </BBTabPanel>
      </div>
    </BBTab>
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";
// import { useI18n } from "vue-i18n";
import type { BBTabItem } from "@/bbkit/types";
import { featureToRef } from "@/store";

interface LocalState {
  selectedIndex: number;
}

// const { t } = useI18n();
const state = reactive<LocalState>({
  selectedIndex: 0,
});
const hasSensitiveDataFeature = featureToRef("bb.feature.sensitive-data");

const tabItemList = computed((): BBTabItem[] => {
  return [
    {
      title: "Sensitive Column List",
      id: "sensitive-column-list",
    },
    {
      title: "Global Masking Rule",
      id: "global-masking-rule",
    },
    {
      title: "Data Feature",
      id: "data-feature",
    },
  ];
});
</script>
