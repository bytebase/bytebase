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
    <InstanceTable
      v-else-if="state.selectedIndex == INSTANCE_TAB"
      :instance-list="filteredInstanceList(instanceList)"
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
import InstanceTable from "../components/InstanceTable.vue";
import { EnvironmentV1Table, ProjectV1Table } from "../components/v2";
import { Instance } from "../types";
import { filterProjectV1ListByKeyword, hasWorkspacePermission } from "../utils";
import { BBTabFilterItem } from "../bbkit/types";
import { useI18n } from "vue-i18n";
import {
  useCurrentUser,
  useEnvironmentV1Store,
  useIdentityProviderStore,
  useInstanceStore,
  useProjectV1ListByCurrentUser,
} from "@/store";
import { IdentityProvider } from "@/types/proto/v1/idp_service";
import IdentityProviderTable from "@/components/IdentityProviderTable.vue";
import { State } from "@/types/proto/v1/common";

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
    InstanceTable,
    ProjectV1Table,
    IdentityProviderTable,
  },
  setup() {
    const { t } = useI18n();
    const instanceStore = useInstanceStore();

    const state = reactive<LocalState>({
      selectedIndex: PROJECT_TAB,
      searchText: "",
    });

    const currentUser = useCurrentUser();

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
        hasWorkspacePermission(
          "bb.permission.workspace.manage-instance",
          currentUser.value.role
        )
      ) {
        instanceStore.fetchInstanceList(["ARCHIVED"]);

        useEnvironmentV1Store().fetchEnvironments(true);
      }
    };

    watchEffect(prepareList);

    const instanceList = computed((): Instance[] => {
      return instanceStore.getInstanceList(["ARCHIVED"]);
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
        hasWorkspacePermission(
          "bb.permission.workspace.manage-instance",
          currentUser.value.role
        )
      ) {
        list.push({ title: t("common.instance"), alert: false });
      }

      if (
        hasWorkspacePermission(
          "bb.permission.workspace.manage-environment",
          currentUser.value.role
        )
      ) {
        list.push({ title: t("common.environment"), alert: false });
      }

      if (
        hasWorkspacePermission(
          "bb.permission.workspace.manage-sso",
          currentUser.value.role
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

    const filteredInstanceList = (list: Instance[]) => {
      if (!state.searchText) {
        return list;
      }
      return list.filter((instance) => {
        return instance.name
          .toLowerCase()
          .includes(state.searchText.toLowerCase());
      });
    };

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
