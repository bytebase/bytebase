<template>
  <div class="w-full px-4 py-4 flex flex-col gap-y-4">
    <FeatureAttention :feature="PlanFeature.FEATURE_DATA_MASKING" />
    <SemanticTypesView />
  </div>
</template>

<script lang="tsx" setup>
import { FeatureAttention } from "@/components/FeatureGuard";
import { SemanticTypesView } from "@/components/SensitiveData";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
</script>
