<template>
  <slot />

  <FeatureModal
    v-if="missingFeature"
    :feature="missingFeature"
    :open="true"
    :instance="instance"
    @cancel="missingFeature = undefined"
  />
</template>

<script lang="ts" setup>
import { useEventListener } from "@vueuse/core";
import { watch, onMounted, toRef } from "vue";
import { useI18n } from "vue-i18n";
import { onBeforeRouteLeave } from "vue-router";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useActuatorV1Store, useSettingV1Store } from "@/store";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";
import { FeatureModal } from "../FeatureGuard";
import { defaultPortForEngine } from "./constants";
import { provideInstanceFormContext } from "./context";

const props = defineProps<{
  instance?: Instance;
  hideAdvancedFeatures?: boolean;
}>();

const emit = defineEmits<{
  (event: "dismiss"): void;
}>();

const settingV1Store = useSettingV1Store();
const actuatorStore = useActuatorV1Store();
const { t } = useI18n();

const instance = toRef(props, "instance");
const hideAdvancedFeatures = toRef(props, "hideAdvancedFeatures");
const context = provideInstanceFormContext({ instance, hideAdvancedFeatures });
const {
  events,
  isCreating,
  valueChanged,
  basicInfo,
  adminDataSource,
  missingFeature,
} = context;

useEventListener("beforeunload", (e) => {
  if (isCreating.value || !valueChanged.value) {
    return;
  }
  e.returnValue = t("common.leave-without-saving");
  return e.returnValue;
});

onBeforeRouteLeave((to, from, next) => {
  if (!isCreating.value && valueChanged.value) {
    if (!window.confirm(t("common.leave-without-saving"))) {
      return;
    }
  }
  next();
});

onMounted(async () => {
  if (isCreating.value) {
    adminDataSource.value.host = actuatorStore.isDocker
      ? "host.docker.internal"
      : "127.0.0.1";
    if (basicInfo.value.engine === Engine.DYNAMODB) {
      adminDataSource.value.host = "";
    }
    adminDataSource.value.srv = false;
    adminDataSource.value.authenticationDatabase = "";
  }
  await settingV1Store.fetchSettingList();
});

watch(
  () => basicInfo.value.engine,
  () => {
    if (isCreating.value) {
      adminDataSource.value.port = defaultPortForEngine(basicInfo.value.engine);
    }
  },
  {
    immediate: true,
  }
);

watch(
  () => props.instance?.activation,
  (val) => {
    if (val !== undefined) {
      basicInfo.value.activation = val;
    }
  }
);

useEmitteryEventListener(events, "dismiss", () => {
  emit("dismiss");
});
</script>
