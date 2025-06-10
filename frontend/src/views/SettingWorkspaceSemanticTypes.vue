<template>
  <div class="w-full space-y-4">
    <FeatureAttention :feature="PlanLimitConfig_Feature.DATA_MASKING" />
    <SemanticTypesView />
  </div>
</template>

<script lang="tsx" setup>
import { FeatureAttention } from "@/components/FeatureGuard";
import { SemanticTypesView } from "@/components/SensitiveData";
import { PlanLimitConfig_Feature } from "@/types/proto/v1/subscription_service";
</script>
