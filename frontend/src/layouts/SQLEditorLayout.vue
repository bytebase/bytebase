<template>
  <RoutePermissionGuard class="m-6" :routes="sqlEditorRoutes">
    <div class="relative h-screen overflow-hidden flex flex-col">
      <ul
        id="sql-editor-debug"
        class="hidden text-xs font-mono max-h-[33vh] max-w-[40vw] overflow-auto fixed bottom-0 right-0 p-2 bg-white/50 border border-gray-400 z-999999"
      ></ul>

      <BannersWrapper />
      <template v-if="ready">
        <ProvideSQLEditorContext />
      </template>
    </div>
  </RoutePermissionGuard>
</template>

<script lang="ts" setup>
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import BannersWrapper from "@/components/BannersWrapper.vue";
import RoutePermissionGuard from "@/components/Permission/RoutePermissionGuard.vue";
import ProvideSQLEditorContext from "@/components/ProvideSQLEditorContext.vue";
import sqlEditorRoutes from "@/router/sqlEditor";
import {
  useEnvironmentV1Store,
  usePolicyV1Store,
  useSettingV1Store,
} from "@/store";
import { PolicyResourceType } from "@/types/proto-es/v1/org_policy_service_pb";
import { provideSQLEditorContext } from "@/views/sql-editor/context";
import { provideSheetContext } from "@/views/sql-editor/Sheet";
import { provideTabListContext } from "@/views/sql-editor/TabList/context";

const router = useRouter();

// provide context for SQL Editor
provideSQLEditorContext();
// provide context for sheets
provideSheetContext();
// provide context for tabs
provideTabListContext();

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
