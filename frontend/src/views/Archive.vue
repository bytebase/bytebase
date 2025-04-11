<template>
  <div class="flex flex-col space-y-4">
    <div class="flex justify-between items-end">
      <TabFilter
        :value="state.selectedTab"
        :items="tabItemList"
        @update:value="(val) => (state.selectedTab = val as LocalTabType)"
      />

      <SearchBox
        v-model:value="state.searchText"
        :placeholder="$t('common.filter-by-name')"
      />
    </div>
    <div class="">
      <PagedProjectTable
        v-if="state.selectedTab == 'PROJECT'"
        session-key="bb.project-table.archived"
        :filter="{
          query: state.searchText,
          state: State.DELETED,
          excludeDefault: true,
        }"
        :bordered="true"
      />
      <PagedInstanceTable
        v-else-if="state.selectedTab == 'INSTANCE'"
        session-key="bb.instance-table.archived"
        :bordered="true"
        :show-selection="false"
        :filter="{
          query: state.searchText,
          state: State.DELETED,
        }"
      />
      <EnvironmentV1Table
        v-else-if="state.selectedTab == 'ENVIRONMENT'"
        key="archived-environment-table"
        :environment-list="filteredEnvironmentList"
      />
      <IdentityProviderTable
        v-else-if="state.selectedTab == 'SSO'"
        key="archived-sso-table"
        :identity-provider-list="filteredSSOList(deletedSSOList)"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import IdentityProviderTable from "@/components/SSO/IdentityProviderTable.vue";
import {
  EnvironmentV1Table,
  SearchBox,
  TabFilter,
  PagedProjectTable,
  PagedInstanceTable,
} from "@/components/v2";
import { useEnvironmentV1Store, useIdentityProviderStore } from "@/store";
import { State } from "@/types/proto/v1/common";
import type { IdentityProvider } from "@/types/proto/v1/idp_service";
import { hasWorkspacePermissionV2 } from "@/utils";

type LocalTabType = "PROJECT" | "INSTANCE" | "ENVIRONMENT" | "SSO";

interface LocalState {
  selectedTab: LocalTabType;
  searchText: string;
}

const { t } = useI18n();
const environmentStore = useEnvironmentV1Store();
const identityProviderStore = useIdentityProviderStore();
const state = reactive<LocalState>({
  selectedTab: "PROJECT",
  searchText: "",
});

const prepareList = async () => {
  const [_1, _2] = await Promise.all([
    environmentStore.fetchEnvironments(true /* showDeleted */),
    identityProviderStore.fetchIdentityProviderList(true /* showDeleted */),
  ]);
};

watchEffect(prepareList);

const environmentList = computed(() => {
  return environmentStore.environmentList;
});

const deletedSSOList = computed(() => {
  return useIdentityProviderStore().deletedIdentityProviderList;
});

const tabItemList = computed(() => {
  const list: { value: LocalTabType; label: string }[] = [
    { value: "PROJECT", label: t("common.project") },
  ];

  if (hasWorkspacePermissionV2("bb.instances.undelete")) {
    list.push({ value: "INSTANCE", label: t("common.instance") });
  }

  if (hasWorkspacePermissionV2("bb.environments.undelete")) {
    list.push({ value: "ENVIRONMENT", label: t("common.environment") });
  }

  if (hasWorkspacePermissionV2("bb.identityProviders.undelete")) {
    list.push({ value: "SSO", label: t("settings.sidebar.sso") });
  }

  return list;
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
