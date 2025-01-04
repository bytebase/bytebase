<template>
  <div class="w-full space-y-4">
    <FeatureAttention
      v-if="!hasSensitiveDataFeature"
      feature="bb.feature.sensitive-data"
    />
    <SemanticTypesView />
  </div>
</template>

<script lang="tsx" setup>
import { FeatureAttention } from "@/components/FeatureGuard";
import { SemanticTypesView } from "@/components/SensitiveData";
import { featureToRef } from "@/store";

const hasSensitiveDataFeature = featureToRef("bb.feature.sensitive-data");
</script>
