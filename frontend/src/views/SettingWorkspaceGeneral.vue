<template>
  <div class="space-y-0 divide-y divide-block-border pt-2">
    <DatabaseChangeModeSetting
      ref="databaseChangeModeSettingRef"
      :allow-edit="allowEdit"
    />
    <NetworkSetting
      ref="networkSettingRef"
      v-if="!isSaaSMode"
      :allow-edit="allowEdit"
    />
    <BrandingSetting ref="brandingSettingRef" :allow-edit="allowEdit" />
    <AccountSetting ref="accountSettingRef" :allow-edit="allowEdit" />
    <SecuritySetting ref="securitySettingRef" :allow-edit="allowEdit" />
    <AIAugmentationSetting
      ref="aiAugmentationSettingRef"
      :allow-edit="allowEdit"
    />
    <AnnouncementSetting ref="announcementSettingRef" :allow-edit="allowEdit" />
  </div>
</template>

<script lang="ts" setup>
import { useEventListener } from "@vueuse/core";
import { storeToRefs } from "pinia";
import { onMounted, ref, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, onBeforeRouteLeave } from "vue-router";
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
const { t } = useI18n();

const accountSettingRef = ref<InstanceType<typeof AccountSetting>>();
const databaseChangeModeSettingRef =
  ref<InstanceType<typeof DatabaseChangeModeSetting>>();
const networkSettingRef = ref<InstanceType<typeof NetworkSetting>>();
const brandingSettingRef = ref<InstanceType<typeof BrandingSetting>>();
const securitySettingRef = ref<InstanceType<typeof SecuritySetting>>();
const aiAugmentationSettingRef =
  ref<InstanceType<typeof AIAugmentationSetting>>();
const announcementSettingRef = ref<InstanceType<typeof AnnouncementSetting>>();

onMounted(async () => {
  await useSettingV1Store().fetchSettingList();
  // If the route has a hash, try to scroll to the element with the value.
  if (route.hash) {
    document.body.querySelector(route.hash)?.scrollIntoView();
  }
});
const { isSaaSMode } = storeToRefs(actuatorStore);

const isDirty = computed(() => {
  return (
    accountSettingRef.value?.isDirty ||
    databaseChangeModeSettingRef.value?.isDirty ||
    networkSettingRef.value?.isDirty ||
    brandingSettingRef.value?.isDirty ||
    aiAugmentationSettingRef.value?.isDirty ||
    announcementSettingRef.value?.isDirty
  );
});

useEventListener("beforeunload", (e) => {
  if (!isDirty.value) {
    return;
  }
  e.returnValue = t("common.leave-without-saving");
  return e.returnValue;
});

onBeforeRouteLeave((to, from, next) => {
  if (isDirty.value) {
    if (!window.confirm(t("common.leave-without-saving"))) {
      return;
    }
  }
  next();
});
</script>
