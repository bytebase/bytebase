<template>
  <div class="w-full mt-4 space-y-4">
    <FeatureAttentionForInstanceLicense
      v-if="hasSensitiveDataFeature"
      feature="bb.feature.sensitive-data"
    />
    <FeatureAttention v-else feature="bb.feature.sensitive-data" />

    <BBTab
      v-if="isDev()"
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
    <SensitiveColumnList v-else />
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import type { BBTabItem } from "@/bbkit/types";
import { SensitiveColumnList } from "@/components/SensitiveData";
import { featureToRef } from "@/store";
import { isDev } from "@/utils";

interface LocalState {
  selectedIndex: number;
}

const { t } = useI18n();
const state = reactive<LocalState>({
  selectedIndex: 0,
});
const hasSensitiveDataFeature = featureToRef("bb.feature.sensitive-data");
const router = useRouter();

const tabItemList = computed((): BBTabItem[] => {
  return [
    {
      title: t("settings.sensitive-data.sensitive-column-list"),
      id: "sensitive-column-list",
    },
    {
      title: t("settings.sensitive-data.global-masking-rule"),
      id: "global-masking-rule",
    },
    {
      title: t("settings.sensitive-data.data-feature"),
      id: "data-feature",
    },
  ];
});

onMounted(() => {
  const hash = router.currentRoute.value.hash.slice(1);
  const index = tabItemList.value.findIndex((tab) => tab.id === hash);
  if (index >= 0) {
    state.selectedIndex = index;
  }
});
</script>
