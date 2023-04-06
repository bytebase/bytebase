<template>
  <div class="w-full mt-4 space-y-4 text-sm">
    <FeatureAttention
      v-if="!hasSlowQueryFeature"
      feature="bb.feature.slow-query"
      :description="$t('subscription.features.bb-feature-slow-query.desc')"
    />

    <SlowQuerySettings @show-feature-modal="showFeatureModal = true" />
  </div>

  <FeatureModal
    v-if="showFeatureModal"
    feature="bb.feature.slow-query"
    @cancel="showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { ref } from "vue";

import { featureToRef } from "@/store";
import { SlowQuerySettings } from "@/components/SlowQuery";

const showFeatureModal = ref(false);

const hasSlowQueryFeature = featureToRef("bb.feature.slow-query");
</script>
