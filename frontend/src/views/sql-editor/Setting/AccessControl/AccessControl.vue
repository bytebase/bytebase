<template>
  <div class="w-full flex flex-col gap-4 py-4 px-2 overflow-y-auto">
    <FeatureAttention
      v-if="!hasSensitiveDataFeature"
      feature="bb.feature.sensitive-data"
    />
    <SensitiveColumnView />
  </div>
</template>

<script setup lang="ts">
import { FeatureAttention } from "@/components/FeatureGuard";
import { SensitiveColumnView } from "@/components/SensitiveData";
import { featureToRef } from "@/store";

const hasSensitiveDataFeature = featureToRef("bb.feature.sensitive-data");
</script>
