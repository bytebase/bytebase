<template>
  <div class="flex flex-col space-y-4">
    <div class="flex justify-between items-end">
      <TabFilter v-model:value="state.selectedTab" :items="tabItemList" />

      <SearchBox
        v-model:value="state.searchText"
        :placeholder="$t('common.filter-by-name')"
      />
    </div>
    <div class="border-x">
      <ProjectV1Table
        v-if="state.selectedTab == 'PROJECT'"
        :project-list="filteredProjectList"
        class="border-x-0"
      />
      <InstanceV1Table
        v-else-if="state.selectedTab == 'INSTANCE'"
        :allow-selection="false"
        :can-assign-license="false"
        :instance-list="filteredInstanceList"
      />
      <EnvironmentV1Table
        v-else-if="state.selectedTab == 'ENVIRONMENT'"
        :environment-list="filteredEnvironmentList"
      />
      <IdentityProviderTable
        v-else-if="state.selectedTab == 'SSO'"
        :identity-provider-list="filteredSSOList(deletedSSOList)"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import IdentityProviderTable from "@/components/IdentityProviderTable.vue";
import {
  EnvironmentV1Table,
  InstanceV1Table,
  ProjectV1Table,
} from "@/components/v2";
import {
  useCurrentUserV1,
  useEnvironmentV1Store,
  useIdentityProviderStore,
  useInstanceV1Store,
  useProjectV1ListByCurrentUser,
} from "@/store";
import { State } from "@/types/proto/v1/common";
import { IdentityProvider } from "@/types/proto/v1/idp_service";
import {
  filterProjectV1ListByKeyword,
  hasWorkspacePermissionV1,
} from "@/utils";

interface LocalState {
  selectedTab: "PROJECT" | "INSTANCE" | "ENVIRONMENT" | "SSO";
  searchText: string;
}

const { t } = useI18n();
const instanceStore = useInstanceV1Store();

const state = reactive<LocalState>({
  selectedTab: "PROJECT",
  searchText: "",
});

const currentUserV1 = useCurrentUserV1();

const { projectList } = useProjectV1ListByCurrentUser(true /* showDeleted */);

const prepareList = () => {
  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-instance",
      currentUserV1.value.userRole
    )
  ) {
    instanceStore.fetchInstanceList(true /* showDeleted */);

    useEnvironmentV1Store().fetchEnvironments(true);
  }
};

watchEffect(prepareList);

const instanceList = computed(() => {
  return instanceStore.instanceList.filter(
    (instance) => instance.state === State.DELETED
  );
});

const environmentList = computed(() => {
  return useEnvironmentV1Store().environmentList.filter(
    (env) => env.state === State.DELETED
  );
});

const deletedSSOList = computed(() => {
  return useIdentityProviderStore().deletedIdentityProviderList;
});

const tabItemList = computed(() => {
  const list = [{ value: "PROJECT", label: t("common.project") }];

  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-instance",
      currentUserV1.value.userRole
    )
  ) {
    list.push({ value: "INSTANCE", label: t("common.instance") });
  }

  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-environment",
      currentUserV1.value.userRole
    )
  ) {
    list.push({ value: "ENVIRONMENT", label: t("common.environment") });
  }

  if (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-sso",
      currentUserV1.value.userRole
    )
  ) {
    list.push({ value: "SSO", label: t("settings.sidebar.sso") });
  }

  return list;
});

const filteredProjectList = computed(() => {
  const list = projectList.value.filter(
    (project) => project.state === State.DELETED
  );
  return filterProjectV1ListByKeyword(list, state.searchText);
});

const filteredInstanceList = computed(() => {
  const keyword = state.searchText.trim();
  if (!keyword) {
    return instanceList.value;
  }
  return instanceList.value.filter((instance) => {
    return instance.title
      .toLowerCase()
      .includes(state.searchText.toLowerCase());
  });
});

const filteredEnvironmentList = computed(() => {
  const list = environmentList.value;
  const keyword = state.searchText.trim().toLowerCase();
  if (!keyword) {
    return list;
  }
  return list.filter((environment) => {
    environment.title.toLowerCase().includes(keyword);
  });
});

const filteredSSOList = (list: IdentityProvider[]) => {
  if (!state.searchText) {
    return list;
  }
  return list.filter((identityProvider) => {
    return identityProvider.name
      .toLowerCase()
      .includes(state.searchText.toLowerCase());
  });
};
</script>
