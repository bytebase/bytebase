<template>
  <div class="w-full flex flex-col gap-y-4">
    <FeatureAttention
      v-if="!hasSensitiveDataFeature"
      :feature="PlanFeature.FEATURE_DATA_CLASSIFICATION"
    />
    <ClassificationView />
  </div>
</template>

<script lang="ts" setup>
import { FeatureAttention } from "@/components/FeatureGuard";
import { ClassificationView } from "@/components/SensitiveData";
import { featureToRef } from "@/store";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

const hasSensitiveDataFeature = featureToRef(
  PlanFeature.FEATURE_DATA_CLASSIFICATION
);
</script>
