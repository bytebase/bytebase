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
import { watch, onMounted, toRef } from "vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useSettingV1Store } from "@/store";
import { Engine } from "@/types/proto/v1/common";
import { Instance } from "@/types/proto/v1/instance_service";
import { isDev } from "@/utils";
import { FeatureModal } from "../FeatureGuard";
import { defaultPortForEngine } from "./constants";
import { provideInstanceFormContext } from "./context";

const props = defineProps<{
  instance?: Instance;
}>();

const emit = defineEmits<{
  (event: "dismiss"): void;
}>();

const settingV1Store = useSettingV1Store();

const instance = toRef(props, "instance");
const context = provideInstanceFormContext({ instance });
const { events, isCreating, basicInfo, adminDataSource, missingFeature } =
  context;
onMounted(async () => {
  if (isCreating.value) {
    adminDataSource.value.host = isDev() ? "127.0.0.1" : "host.docker.internal";
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
