<template>
  <div class="flex flex-col">
    <div class="px-4 py-2 flex justify-between items-center">
      <BBTabFilter
        :tab-item-list="tabItemList"
        :selected-index="state.selectedIndex"
        @select-index="
          (index) => {
            state.selectedIndex = index;
          }
        "
      />
      <BBTableSearch
        ref="searchField"
        class="w-56"
        :placeholder="
          state.selectedIndex == PROJECT_TAB
            ? $t('archive.project-search-bar-placeholder')
            : state.selectedIndex == INSTANCE_TAB
            ? $t('archive.instance-search-bar-placeholder')
            : $t('archive.environment-search-bar-placeholder')
        "
        @change-text="(text) => changeSearchText(text)"
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
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, reactive, watchEffect } from "vue";
import { useStore } from "vuex";
import EnvironmentTable from "../components/EnvironmentTable.vue";
import InstanceTable from "../components/InstanceTable.vue";
import ProjectTable from "../components/ProjectTable.vue";
import { Environment, Instance, Project, UNKNOWN_ID } from "../types";
import { isDBAOrOwner } from "../utils";
import { BBTabFilterItem } from "../bbkit/types";
import { useI18n } from "vue-i18n";
import {
  useCurrentUser,
  useEnvironmentList,
  useEnvironmentStore,
  useInstanceStore,
} from "@/store";

const PROJECT_TAB = 0;
const INSTANCE_TAB = 1;
const ENVIRONMENT_TAB = 2;

interface LocalState {
  selectedIndex: number;
  searchText: string;
}

export default defineComponent({
  name: "Archive",
  components: { EnvironmentTable, InstanceTable, ProjectTable },
  setup() {
    const { t } = useI18n();
    const instanceStore = useInstanceStore();

    const state = reactive<LocalState>({
      selectedIndex: PROJECT_TAB,
      searchText: "",
    });

    const currentUser = useCurrentUser();

    const store = useStore();

    const prepareList = () => {
      // It will also be called when user logout
      if (currentUser.value.id != UNKNOWN_ID) {
        store.dispatch("project/fetchProjectListByUser", {
          userId: currentUser.value.id,
          rowStatusList: ["ARCHIVED"],
        });
      }

      if (isDBAOrOwner(currentUser.value.role)) {
        instanceStore.fetchInstanceList(["ARCHIVED"]);

        useEnvironmentStore().fetchEnvironmentList(["ARCHIVED"]);
      }
    };

    watchEffect(prepareList);

    const projectList = computed((): Project[] => {
      return store.getters["project/projectListByUser"](currentUser.value.id, [
        "ARCHIVED",
      ]);
    });

    const instanceList = computed((): Instance[] => {
      return instanceStore.getInstanceList(["ARCHIVED"]);
    });

    const environmentList = useEnvironmentList(["ARCHIVED"]);

    const tabItemList = computed((): BBTabFilterItem[] => {
      return isDBAOrOwner(currentUser.value.role)
        ? [
            { title: t("common.project"), alert: false },
            { title: t("common.instance"), alert: false },
            { title: t("common.environment"), alert: false },
          ]
        : [{ title: t("common.instance"), alert: false }];
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

    const changeSearchText = (searchText: string) => {
      state.searchText = searchText;
    };

    return {
      PROJECT_TAB,
      INSTANCE_TAB,
      ENVIRONMENT_TAB,
      state,
      projectList,
      instanceList,
      environmentList,
      tabItemList,
      filteredProjectList,
      filteredInstanceList,
      filteredEnvironmentList,
      changeSearchText,
    };
  },
});
</script>
