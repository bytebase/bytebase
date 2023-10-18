<template>
  <div class="w-full mt-4 space-y-4">
    <FeatureAttentionForInstanceLicense
      v-if="hasSensitiveDataFeature"
      feature="bb.feature.sensitive-data"
    />
    <FeatureAttention v-else feature="bb.feature.sensitive-data" />
    <NTabs v-model:value="state.selectedTab" type="line">
      <NTabPane
        name="sensitive-column-list"
        :tab="$t('settings.sensitive-data.sensitive-column-list')"
      >
        <SensitiveColumnView />
      </NTabPane>
      <NTabPane
        name="global-masking-rule"
        :tab="$t('settings.sensitive-data.global-rules.self')"
      >
        <GlobalMaskingRulesView />
      </NTabPane>
      <NTabPane
        v-if="isDev()"
        name="semantic-types"
        :tab="$t('settings.sensitive-data.semantic-types.self')"
      >
        <SemanticTypesView />
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
import { isDev } from "@/utils";

interface LocalState {
  selectedTab:
    | "sensitive-column-list"
    | "global-masking-rule"
    | "semantic-types";
}

const state = reactive<LocalState>({
  selectedTab: "sensitive-column-list",
});
const hasSensitiveDataFeature = featureToRef("bb.feature.sensitive-data");
const router = useRouter();
const route = useRoute();

watch(
  () => route.hash,
  (hash) => {
    switch (hash) {
      case "#semantic-types":
        state.selectedTab = "semantic-types";
        break;
      case "#global-masking-rule":
        state.selectedTab = "global-masking-rule";
        break;
      default:
        state.selectedTab = "sensitive-column-list";
        break;
    }
  },
  {
    immediate: true,
  }
);

watch(
  () => state.selectedTab,
  (tab) => {
    switch (tab) {
      case "global-masking-rule":
        router.push({ hash: "#global-masking-rule" });
        break;
      case "semantic-types":
        router.push({ hash: "#semantic-types" });
        break;
      default:
        router.push({ hash: "" });
        break;
    }
  }
);
</script>
