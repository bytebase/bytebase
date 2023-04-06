<template>
  <div class="w-full mt-4 space-y-4 text-sm">
    <FeatureAttention
      v-if="!hasSlowQueryFeature"
      feature="bb.feature.slow-query"
      :description="$t('subscription.features.bb-feature-slow-query.desc')"
    />

    <SlowQuerySettings @show-feature-modal="state.showFeatureModal = true" />
  </div>

  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.slow-query"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { reactive } from "vue";

import { featureToRef } from "@/store";
import { SlowQuerySettings } from "@/components/SlowQuery";

interface LocalState {
  showFeatureModal: boolean;
}

const state = reactive<LocalState>({
  showFeatureModal: false,
});
const hasSlowQueryFeature = featureToRef("bb.feature.slow-query");
</script>
