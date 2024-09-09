<template>
  <div class="w-full space-y-4">
    <FeatureAttention
      v-if="!hasSensitiveDataFeature"
      feature="bb.feature.sensitive-data"
    />
    <ClassificationView />
  </div>
</template>

<script lang="ts" setup>
import { FeatureAttention } from "@/components/FeatureGuard";
import { ClassificationView } from "@/components/SensitiveData";
import { featureToRef } from "@/store";

const hasSensitiveDataFeature = featureToRef("bb.feature.sensitive-data");
</script>
