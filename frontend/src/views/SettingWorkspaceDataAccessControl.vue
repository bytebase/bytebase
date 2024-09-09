<template>
  <div class="w-full space-y-4">
    <FeatureAttention
      v-if="!hasSensitiveDataFeature"
      feature="bb.feature.sensitive-data"
    />
    <SensitiveColumnView />
  </div>
</template>

<script lang="ts" setup>
import { FeatureAttention } from "@/components/FeatureGuard";
import { SensitiveColumnView } from "@/components/SensitiveData";
import { featureToRef } from "@/store";

const hasSensitiveDataFeature = featureToRef("bb.feature.sensitive-data");
</script>
