<template>
  <slot v-if="!isLoading" />
</template>

<script lang="ts" setup>
import { ref, onMounted, watch } from "vue";
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

const route = useRoute();
const router = useRouter();
const isLoading = ref<boolean>(true);

const policyStore = usePolicyV1Store();
const databaseStore = useDatabaseV1Store();

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
    useInstanceV1Store().fetchInstanceList(),
    useProjectV1Store().fetchProjectList(true /* showDeleted */),
  ]);
  await Promise.all([useUIStateStore().restoreState()]);
  await fetchDatabases(route.params.projectId as string);

  watch(
    () => route.params.projectId,
    (project) => {
      fetchDatabases(project as string);
    },
    { immediate: false }
  );

  isLoading.value = false;
});
</script>
