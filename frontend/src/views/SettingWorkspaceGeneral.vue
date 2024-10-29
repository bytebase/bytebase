<template>
  <div class="space-y-0 divide-y divide-block-border pt-2">
    <DatabaseChangeModeSetting :allow-edit="allowEdit" />
    <NetworkSetting v-if="!isSaaSMode" :allow-edit="allowEdit" />
    <BrandingSetting :allow-edit="allowEdit" />
    <AccountSetting :allow-edit="allowEdit" />
    <SecuritySetting :allow-edit="allowEdit" />
    <AIAugmentationSetting :allow-edit="allowEdit" />
    <AnnouncementSetting :allow-edit="allowEdit" />
  </div>
</template>

<script lang="ts" setup>
import { storeToRefs } from "pinia";
import { onMounted } from "vue";
import { useRoute } from "vue-router";
import {
  BrandingSetting,
  SecuritySetting,
  AccountSetting,
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
