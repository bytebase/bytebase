<template>
  <div class="space-y-4 divide-y divide-block-border">
    <NetworkSetting v-if="!isSaaSMode" :allow-edit="allowEdit" />
    <BrandingSetting :allow-edit="allowEdit" />
    <SecuritySetting :allow-edit="allowEdit" />
    <AIAugmentationSetting :allow-edit="allowEdit" />
    <AnnouncementSetting :allow-edit="allowEdit" />
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
  AnnouncementSetting,
} from "@/components/GeneralSetting";
import { useActuatorV1Store } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";

defineProps<{
  allowEdit: boolean;
}>();

const actuatorStore = useActuatorV1Store();

onMounted(async () => {
  await useSettingV1Store().fetchSettingList();
});
const { isSaaSMode } = storeToRefs(actuatorStore);
</script>
