<template>
  <slot v-if="!isInitializing" />
  <MaskSpinner v-else class="bg-white!" />
</template>

<script lang="ts" setup>
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import {
  useEnvironmentV1Store,
  usePolicyV1Store,
  useSettingV1Store,
  useUIStateStore,
} from "@/store";
import { PolicyResourceType } from "@/types/proto-es/v1/org_policy_service_pb";
import MaskSpinner from "./misc/MaskSpinner.vue";

const router = useRouter();
const isInitializing = ref<boolean>(true);

const policyStore = usePolicyV1Store();

onMounted(async () => {
  await router.isReady();

  // Prepare roles, workspace policies and settings first.
  await Promise.all([
    policyStore.fetchPolicies({
      resourceType: PolicyResourceType.WORKSPACE,
    }),
    useSettingV1Store().fetchSettingList(),
  ]);

  // Then prepare the other resources.
  await Promise.all([useEnvironmentV1Store().fetchEnvironments()]);

  useUIStateStore().restoreState();

  isInitializing.value = false;
});
</script>
