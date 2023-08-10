<template>
  <div class="flex flex-col">
    <div class="px-4 py-2 flex justify-between items-center">
      <BBTabFilter
        :tab-item-list="tabItemList"
        :selected-index="state.selectedIndex"
        @select-index="
          (index: number) => {
            state.selectedIndex = index;
          }
        "
      />
      <BBTableSearch
        ref="searchField"
        class="w-56"
        :placeholder="searchFieldPlaceholder"
        @change-text="(text: string) => changeSearchText(text)"
      />
    </div>
    <ProjectV1Table
      v-if="state.selectedIndex == PROJECT_TAB"
      :project-list="filteredProjectList"
      class="border-x-0"
    />
    <InstanceV1Table
      v-else-if="state.selectedIndex == INSTANCE_TAB"
      :instance-list="filteredInstanceList"
    />
    <EnvironmentV1Table
      v-else-if="state.selectedIndex == ENVIRONMENT_TAB"
      :environment-list="filteredEnvironmentList"
    />
    <IdentityProviderTable
      v-else-if="state.selectedIndex == SSO_TAB"
      :identity-provider-list="filteredSSOList(deletedSSOList)"
    />
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, reactive, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { BBTabFilterItem } from "@/bbkit/types";
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

const PROJECT_TAB = 0;
const INSTANCE_TAB = 1;
const ENVIRONMENT_TAB = 2;
const SSO_TAB = 3;

interface LocalState {
  selectedIndex: number;
  searchText: string;
}

export default defineComponent({
  name: "Archive",
  components: {
    EnvironmentV1Table,
    InstanceV1Table,
    ProjectV1Table,
    IdentityProviderTable,
  },
  setup() {
    const { t } = useI18n();
    const instanceStore = useInstanceV1Store();

    const state = reactive<LocalState>({
      selectedIndex: PROJECT_TAB,
      searchText: "",
    });

    const currentUserV1 = useCurrentUserV1();

    const searchFieldPlaceholder = computed(() => {
      if (state.selectedIndex == PROJECT_TAB) {
        return t("archive.project-search-bar-placeholder");
      } else if (state.selectedIndex == INSTANCE_TAB) {
        return t("archive.instance-search-bar-placeholder");
      } else if (state.selectedIndex == ENVIRONMENT_TAB) {
        return t("archive.environment-search-bar-placeholder");
      } else if (state.selectedIndex == SSO_TAB) {
        return t("archive.sso-search-bar-placeholder");
      } else {
        return "";
      }
    });

    const { projectList } = useProjectV1ListByCurrentUser(
      true /* showDeleted */
    );

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

    const tabItemList = computed((): BBTabFilterItem[] => {
      const list: BBTabFilterItem[] = [
        { title: t("common.project"), alert: false },
      ];

      if (
        hasWorkspacePermissionV1(
          "bb.permission.workspace.manage-instance",
          currentUserV1.value.userRole
        )
      ) {
        list.push({ title: t("common.instance"), alert: false });
      }

      if (
        hasWorkspacePermissionV1(
          "bb.permission.workspace.manage-environment",
          currentUserV1.value.userRole
        )
      ) {
        list.push({ title: t("common.environment"), alert: false });
      }

      if (
        hasWorkspacePermissionV1(
          "bb.permission.workspace.manage-sso",
          currentUserV1.value.userRole
        )
      ) {
        list.push({ title: t("settings.sidebar.sso"), alert: false });
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

    const changeSearchText = (searchText: string) => {
      state.searchText = searchText;
    };

    return {
      PROJECT_TAB,
      INSTANCE_TAB,
      ENVIRONMENT_TAB,
      SSO_TAB,
      state,
      instanceList,
      environmentList,
      deletedSSOList,
      tabItemList,
      searchFieldPlaceholder,
      filteredProjectList,
      filteredInstanceList,
      filteredEnvironmentList,
      filteredSSOList,
      changeSearchText,
    };
  },
});
</script>
