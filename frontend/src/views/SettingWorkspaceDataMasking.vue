<template>
  <div class="w-full space-y-4">
    <FeatureAttention
      v-if="!hasSensitiveDataFeature"
      :feature="PlanFeature.FEATURE_DATA_MASKING"
    />
    <GlobalMaskingRulesView :embedded="embedded" />
  </div>
</template>

<script lang="ts" setup>
import { FeatureAttention } from "@/components/FeatureGuard";
import { GlobalMaskingRulesView } from "@/components/SensitiveData";
import { featureToRef } from "@/store";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

defineProps<{
  embedded?: boolean;
}>();

const hasSensitiveDataFeature = featureToRef(PlanFeature.FEATURE_DATA_MASKING);
</script>
