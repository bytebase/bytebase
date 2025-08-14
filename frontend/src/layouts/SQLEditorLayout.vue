<template>
  <div class="relative h-screen overflow-hidden flex flex-col">
    <ul
      id="sql-editor-debug"
      class="hidden text-xs font-mono max-h-[33vh] max-w-[40vw] overflow-auto fixed bottom-0 right-0 p-2 bg-white/50 border border-gray-400 z-[999999]"
    ></ul>

    <BannersWrapper />
    <template v-if="ready">
      <ProvideSQLEditorSettingContext v-if="isSettingPage" />
      <ProvideSQLEditorContext v-else />
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { onMounted } from "vue";
import { useRouter } from "vue-router";
import BannersWrapper from "@/components/BannersWrapper.vue";
import ProvideSQLEditorContext from "@/components/ProvideSQLEditorContext.vue";
import ProvideSQLEditorSettingContext from "@/components/ProvideSQLEditorSettingContext.vue";
import { SQL_EDITOR_SETTING_MODULE } from "@/router/sqlEditor";
import {
  useEnvironmentV1Store,
  usePolicyV1Store,
  useSettingV1Store,
} from "@/store";
import { PolicyResourceType } from "@/types/proto-es/v1/org_policy_service_pb";
import { provideSheetContext } from "@/views/sql-editor/Sheet";
import { provideSQLEditorContext } from "@/views/sql-editor/context";

const router = useRouter();

// provide context for SQL Editor
provideSQLEditorContext();
// provide context for sheets
provideSheetContext();

const isSettingPage = computed(() => {
  return !!router.currentRoute.value.matched.find(
    (r) => r.name === SQL_EDITOR_SETTING_MODULE
  );
});

const ready = ref(false);

// We may tweak the resource loading here, and pre-load some frontend component
// resources, and make a loading screen here in the future.
const prepare = async () => {
  await router.isReady();

  const policyV1Store = usePolicyV1Store();
  // Prepare roles, workspace policies and settings.
  await Promise.all([
    useSettingV1Store().fetchSettingList(),
    useEnvironmentV1Store().fetchEnvironments(),
    policyV1Store.fetchPolicies({
      resourceType: PolicyResourceType.WORKSPACE,
    }),
    policyV1Store.fetchPolicies({
      resourceType: PolicyResourceType.ENVIRONMENT,
    }),
    policyV1Store.fetchPolicies({
      resourceType: PolicyResourceType.PROJECT,
    }),
  ]);
  ready.value = true;
};

onMounted(prepare);
</script>
