<template>
  <slot v-if="!isLoading" />
</template>

<script lang="ts" setup>
import { ref, onMounted } from "vue";
import { useRoute } from "vue-router";
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
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";

const route = useRoute();
const isLoading = ref<boolean>(true);

const databaseStore = useDatabaseV1Store();

const prepareDatabases = async () => {
  let filter = "";
  // If `projectId` is provided in the route, filter the database list by the project.
  if (route.params.projectId) {
    filter = `project == "${projectNamePrefix}${route.params.projectId}"`;
  }

  await databaseStore.fetchDatabaseList({
    parent: "instances/-",
    filter: filter,
  });
};

onMounted(async () => {
  await Promise.all([
    useUserStore().fetchUserList(),
    useSettingV1Store().fetchSettingList(),
    useRoleStore().fetchRoleList(),
    useEnvironmentV1Store().fetchEnvironments(),
    useInstanceV1Store().fetchInstanceList(),
    useProjectV1Store().fetchProjectList(true),
    usePolicyV1Store().getOrFetchPolicyByName("policies/WORKSPACE_IAM"),
  ]);

  await Promise.all([prepareDatabases(), useUIStateStore().restoreState()]);

  isLoading.value = false;
});
</script>
