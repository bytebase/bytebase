<template>
  <div class="space-y-4 divide-y divide-block-border">
    <NetworkSetting v-if="!isSaaSMode" />
    <BrandingSetting />
    <SecuritySetting />
    <AIAugmentationSetting />
  </div>
</template>

<script lang="ts" setup>
import { onMounted } from "vue";
import { storeToRefs } from "pinia";
import {
  BrandingSetting,
  SecuritySetting,
  NetworkSetting,
  AIAugmentationSetting,
} from "@/components/GeneralSetting";
import { useActuatorStore } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";

const actuatorStore = useActuatorStore();

onMounted(async () => {
  await useSettingV1Store().fetchSettingList();
});
const { isSaaSMode } = storeToRefs(actuatorStore);
</script>
