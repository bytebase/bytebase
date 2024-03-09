<template>
  <div class="relative h-screen overflow-hidden flex flex-col">
    <BannersWrapper v-if="showBanners" />
    <!-- Suspense is experimental, be aware of the potential change -->
    <Suspense>
      <div v-if="ready" class="text-xs font-mono">
        <ProvideSQLEditorContextV2>
          <!-- <router-view /> -->
          haha
        </ProvideSQLEditorContextV2>
        <div>ready: {{ ready }}</div>
        <div>fullPath: {{ $route.fullPath }}</div>
        <div>match: {{ $route.matched }}</div>
      </div>
    </Suspense>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { onMounted } from "vue";
import BannersWrapper from "@/components/BannersWrapper.vue";
import ProvideSQLEditorContextV2 from "@/components/ProvideSQLEditorContextV2.vue";
import {
  useEnvironmentV1Store,
  usePageMode,
  usePolicyV1Store,
  useRoleStore,
  useSettingV1Store,
} from "@/store";
import {
  PolicyResourceType,
  PolicyType,
} from "@/types/proto/v1/org_policy_service";

const pageMode = usePageMode();

const showBanners = computed(() => {
  return pageMode.value === "BUNDLED";
});

const ready = ref(false);

// We may tweak the resource loading here, and pre-load some frontend component
// resources, and make a loading screen here in the future.
const prepare = async () => {
  const policyV1Store = usePolicyV1Store();
  // Prepare roles, workspace policies and settings.
  await Promise.all([
    useSettingV1Store().fetchSettingList(),
    useRoleStore().fetchRoleList(),
    useEnvironmentV1Store().fetchEnvironments(),
    policyV1Store.fetchPolicies({
      resourceType: PolicyResourceType.WORKSPACE,
    }),
    policyV1Store.fetchPolicies({
      resourceType: PolicyResourceType.ENVIRONMENT,
      policyType: PolicyType.DISABLE_COPY_DATA,
    }),
  ]);
  ready.value = true;
};

onMounted(prepare);
</script>
