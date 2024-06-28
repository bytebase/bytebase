<template>
  <div class="space-y-4 divide-y divide-block-border">
    <NetworkSetting v-if="!isSaaSMode" :allow-edit="allowEdit" />
    <BrandingSetting :allow-edit="allowEdit" />
    <SecuritySetting :allow-edit="allowEdit" />
    <AIAugmentationSetting :allow-edit="allowEdit" />
    <AnnouncementSetting :allow-edit="allowEdit" />
    <DatabaseChangeModeSetting :allow-edit="allowEdit" />
  </div>
</template>

<script lang="ts" setup>
import { storeToRefs } from "pinia";
import { onMounted } from "vue";
import { useRoute } from "vue-router";
import {
  BrandingSetting,
  SecuritySetting,
  NetworkSetting,
  AIAugmentationSetting,
  AnnouncementSetting,
  DatabaseChangeModeSetting,
} from "@/components/GeneralSetting";
import { useActuatorV1Store } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";

defineProps<{
  allowEdit: boolean;
}>();

const route = useRoute();
const actuatorStore = useActuatorV1Store();

onMounted(async () => {
  await useSettingV1Store().fetchSettingList();
  // If the route has a hash, try to scroll to the element with the value.
  if (route.hash) {
    document.body.querySelector(route.hash)?.scrollIntoView();
  }
});
const { isSaaSMode } = storeToRefs(actuatorStore);
</script>
