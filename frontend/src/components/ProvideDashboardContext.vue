<template>
  <slot v-if="!isInitializing" />
  <MaskSpinner v-else class="!bg-white" />

  <div
    v-if="!isInitializing && isSwitchingProject"
    class="fixed inset-0 z-[1000000] bg-white/50 flex flex-col items-center justify-center"
  >
    <NSpin />
  </div>
</template>

<script lang="ts" setup>
import { NSpin } from "naive-ui";
import { ref, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  useEnvironmentV1Store,
  useInstanceV1Store,
  usePolicyV1Store,
  useProjectV1Store,
  useRoleStore,
  useSettingV1Store,
  useUserStore,
  useUIStateStore,
  useDatabaseV1Store,
  useUserGroupStore,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { PolicyResourceType } from "@/types/proto/v1/org_policy_service";
import MaskSpinner from "./misc/MaskSpinner.vue";

const route = useRoute();
const router = useRouter();
const isInitializing = ref<boolean>(true);
const isSwitchingProject = ref(false);

const policyStore = usePolicyV1Store();
const databaseStore = useDatabaseV1Store();

const fetchInstances = async (project: string) => {
  const parent = project ? `${projectNamePrefix}${project}` : undefined;
  await useInstanceV1Store().fetchInstanceList(
    /* !showDeleted */ false,
    parent
  );
};

const fetchDatabases = async (project: string) => {
  const filters = [`instance = "instances/-"`];
  // If `projectId` is provided in the route, filter the database list by the project.
  if (project) {
    filters.push(`project = "${projectNamePrefix}${project}"`);
  }
  await databaseStore.searchDatabases({
    filter: filters.join(" && "),
  });
};

const instanceAndDatabaseInitialized = new Set<string /* project */>();
const fetchInstancesAndDatabases = async (project: string) => {
  if (instanceAndDatabaseInitialized.has(project || "")) return;
  await fetchInstances(project);
  await fetchDatabases(project);
  instanceAndDatabaseInitialized.add(project || "");
};

onMounted(async () => {
  await router.isReady();

  // Prepare roles, workspace policies and settings first.
  await Promise.all([
    useRoleStore().fetchRoleList(),
    policyStore.fetchPolicies({
      resourceType: PolicyResourceType.WORKSPACE,
    }),
    useSettingV1Store().fetchSettingList(),
  ]);

  // Then prepare the other resources.
  await Promise.all([
    useUserStore().fetchUserList(),
    useUserGroupStore().fetchGroupList(),
    useEnvironmentV1Store().fetchEnvironments(),
    useProjectV1Store().fetchProjectList(true /* showDeleted */),
  ]);
  await Promise.all([useUIStateStore().restoreState()]);
  await fetchInstancesAndDatabases(route.params.projectId as string);

  isInitializing.value = false;

  router.beforeEach((to, from, next) => {
    const fromProject = from.params.projectId as string;
    const toProject = to.params.projectId as string;
    if (fromProject !== toProject) {
      console.debug(
        `[ProvideDashboardContext] project switched ${fromProject} -> ${toProject}`
      );
      isSwitchingProject.value = true;
      fetchInstancesAndDatabases(toProject).finally(() => {
        isSwitchingProject.value = false;

        next();
      });
      return;
    }

    next();
  });
});
</script>
