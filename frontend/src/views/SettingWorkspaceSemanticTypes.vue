<template>
  <div class="w-full space-y-4">
    <FeatureAttention :feature="PlanFeature.FEATURE_DATA_MASKING" />
    <SemanticTypesView />
  </div>
</template>

<script lang="tsx" setup>
import { FeatureAttention } from "@/components/FeatureGuard";
import { SemanticTypesView } from "@/components/SensitiveData";
import { PlanFeature } from "@/types/proto/v1/subscription_service";
</script>
