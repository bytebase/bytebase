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
    <ProjectTable
      v-if="state.selectedIndex == PROJECT_TAB"
      :project-list="filteredProjectList(projectList)"
    />
    <InstanceTable
      v-else-if="state.selectedIndex == INSTANCE_TAB"
      :instance-list="filteredInstanceList(instanceList)"
    />
    <EnvironmentTable
      v-else-if="state.selectedIndex == ENVIRONMENT_TAB"
      :environment-list="filteredEnvironmentList(environmentList)"
    />
    <IdentityProviderTable
      v-else-if="state.selectedIndex == SSO_TAB"
      :identity-provider-list="filteredSSOList(deletedSSOList)"
    />
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, reactive, watchEffect } from "vue";
import EnvironmentTable from "../components/EnvironmentTable.vue";
import InstanceTable from "../components/InstanceTable.vue";
import ProjectTable from "../components/ProjectTable.vue";
import { Environment, Instance, Project, UNKNOWN_ID } from "../types";
import { hasWorkspacePermission } from "../utils";
import { BBTabFilterItem } from "../bbkit/types";
import { useI18n } from "vue-i18n";
import {
  useCurrentUser,
  useEnvironmentList,
  useEnvironmentStore,
  useIdentityProviderStore,
  useInstanceStore,
  useProjectStore,
} from "@/store";
import { IdentityProvider } from "@/types/proto/v1/idp_service";
import IdentityProviderTable from "@/components/IdentityProviderTable.vue";

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
    EnvironmentTable,
    InstanceTable,
    ProjectTable,
    IdentityProviderTable,
  },
  setup() {
    const { t } = useI18n();
    const instanceStore = useInstanceStore();
    const projectStore = useProjectStore();

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

    const prepareList = () => {
      // It will also be called when user logout
      if (currentUser.value.id != UNKNOWN_ID) {
        projectStore.fetchProjectListByUser({
          userId: currentUser.value.id,
          rowStatusList: ["ARCHIVED"],
        });
      }

      if (
        hasWorkspacePermission(
          "bb.permission.workspace.manage-instance",
          currentUser.value.role
        )
      ) {
        instanceStore.fetchInstanceList(["ARCHIVED"]);

        useEnvironmentStore().fetchEnvironmentList(["ARCHIVED"]);
      }
    };

    watchEffect(prepareList);

    const projectList = computed((): Project[] => {
      return projectStore.getProjectListByUser(currentUser.value.id, [
        "ARCHIVED",
      ]);
    });

    const instanceList = computed((): Instance[] => {
      return instanceStore.getInstanceList(["ARCHIVED"]);
    });

    const environmentList = useEnvironmentList(["ARCHIVED"]);

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

    const filteredProjectList = (list: Project[]) => {
      if (!state.searchText) {
        return list;
      }
      return list.filter((project) => {
        return project.name
          .toLowerCase()
          .includes(state.searchText.toLowerCase());
      });
    };

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

    const filteredEnvironmentList = (list: Environment[]) => {
      if (!state.searchText) {
        return list;
      }
      return list.filter((environment) => {
        return environment.name
          .toLowerCase()
          .includes(state.searchText.toLowerCase());
      });
    };

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
      projectList,
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
