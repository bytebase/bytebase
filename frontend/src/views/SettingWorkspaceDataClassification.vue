<template>
  <div class="w-full space-y-4">
    <FeatureAttention
      v-if="!hasSensitiveDataFeature"
      :feature="PlanLimitConfig_Feature.DATA_CLASSIFICATION"
    />
    <ClassificationView />
  </div>
</template>

<script lang="ts" setup>
import { FeatureAttention } from "@/components/FeatureGuard";
import { ClassificationView } from "@/components/SensitiveData";
import { featureToRef } from "@/store";
import { PlanLimitConfig_Feature } from "@/types/proto/v1/subscription_service";

const hasSensitiveDataFeature = featureToRef(PlanLimitConfig_Feature.DATA_CLASSIFICATION);
</script>
