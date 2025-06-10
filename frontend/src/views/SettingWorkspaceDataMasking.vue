<template>
  <div class="w-full space-y-4">
    <FeatureAttention
      v-if="!hasSensitiveDataFeature"
      :feature="PlanLimitConfig_Feature.DATA_MASKING"
    />
    <GlobalMaskingRulesView :embedded="embedded" />
  </div>
</template>

<script lang="ts" setup>
import { FeatureAttention } from "@/components/FeatureGuard";
import { GlobalMaskingRulesView } from "@/components/SensitiveData";
import { featureToRef } from "@/store";
import { PlanLimitConfig_Feature } from "@/types/proto/v1/subscription_service";

defineProps<{
  embedded?: boolean;
}>();

const hasSensitiveDataFeature = featureToRef(PlanLimitConfig_Feature.DATA_MASKING);
</script>
