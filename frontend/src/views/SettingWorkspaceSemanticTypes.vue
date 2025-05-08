<template>
  <div class="w-full space-y-4">
    <FeatureAttention feature="bb.feature.sensitive-data" />
    <SemanticTypesView />
  </div>
</template>

<script lang="tsx" setup>
import { FeatureAttention } from "@/components/FeatureGuard";
import { SemanticTypesView } from "@/components/SensitiveData";
</script>
