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
import { onMounted } from "vue";
import {
  BrandingSetting,
  SecuritySetting,
  NetworkSetting,
  AIAugmentationSetting,
} from "@/components/GeneralSetting";
import { useActuatorV1Store } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";

const actuatorStore = useActuatorV1Store();

onMounted(async () => {
  await useSettingV1Store().fetchSettingList();
});
const { isSaaSMode } = storeToRefs(actuatorStore);
</script>
