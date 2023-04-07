<template>
  <div class="space-y-4 divide-y divide-block-border">
    <NetworkSetting v-if="!isSaaSMode" />
    <BrandingSetting />
    <SecuritySetting />
    <AIAugmentationSetting />
  </div>
</template>

<script lang="ts" setup>
import { storeToRefs } from "pinia";
import {
  BrandingSetting,
  SecuritySetting,
  NetworkSetting,
  AIAugmentationSetting,
} from "@/components/GeneralSetting";
import { useActuatorStore } from "@/store";

const actuatorStore = useActuatorStore();

const { isSaaSMode } = storeToRefs(actuatorStore);
</script>
