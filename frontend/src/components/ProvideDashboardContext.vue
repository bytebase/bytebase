<template>
  <slot v-if="!isInitializing" />
  <MaskSpinner v-else class="bg-white!" />
</template>

<script lang="ts" setup>
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import {
  useEnvironmentV1Store,
  useSettingV1Store,
  useUIStateStore,
} from "@/store";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import MaskSpinner from "./misc/MaskSpinner.vue";

const router = useRouter();
const isInitializing = ref<boolean>(true);

onMounted(async () => {
  await router.isReady();

  await Promise.all([
    useEnvironmentV1Store().fetchEnvironments(),
    useSettingV1Store().getOrFetchSettingByName(
      Setting_SettingName.WORKSPACE_PROFILE
    ),
  ]);

  useUIStateStore().restoreState();

  isInitializing.value = false;
});
</script>
