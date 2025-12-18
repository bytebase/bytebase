<template>
  <div class="w-full flex flex-col gap-y-4">
    <FeatureAttention
      :feature="PlanFeature.FEATURE_DATA_MASKING"
    />
    <BBAttention v-if="hasSensitiveDataFeature" type="info" :description="$t('custom-approval.rule.first-match-wins')" />
    <GlobalMaskingRulesView />
  </div>
</template>

<script lang="ts" setup>
import { BBAttention } from "@/bbkit";
import { FeatureAttention } from "@/components/FeatureGuard";
import { GlobalMaskingRulesView } from "@/components/SensitiveData";
import { featureToRef } from "@/store";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

const hasSensitiveDataFeature = featureToRef(PlanFeature.FEATURE_DATA_MASKING);
</script>
