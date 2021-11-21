<template>
  <div class="flex flex-col">
    <div class="px-4 py-2 flex justify-between items-center">
      <BBTabFilter
        :tabItemList="tabItemList"
        :selectedIndex="state.selectedIndex"
        @select-index="
          (index) => {
            state.selectedIndex = index;
          }
        "
      />
      <BBTableSearch
        class="w-56"
        ref="searchField"
        :placeholder="
          state.selectedIndex == PROJECT_TAB
            ? 'Search project name'
            : state.selectedIndex == INSTANCE_TAB
            ? 'Search instance name'
            : 'Search environment name'
        "
        @change-text="(text) => changeSearchText(text)"
      />
    </div>
    <ProjectTable
      v-if="state.selectedIndex == PROJECT_TAB"
      :projectList="filteredProjectList(projectList)"
    />
    <InstanceTable
      v-else-if="state.selectedIndex == INSTANCE_TAB"
      :instanceList="filteredInstanceList(instanceList)"
    />
    <EnvironmentTable
      v-else-if="state.selectedIndex == ENVIRONMENT_TAB"
      :environmentList="filteredEnvironmentList(environmentList)"
    />
  </div>
</template>

<script lang="ts">
import { computed, ComputedRef, reactive, watchEffect } from "vue";
import { useStore } from "vuex";
import EnvironmentTable from "../components/EnvironmentTable.vue";
import InstanceTable from "../components/InstanceTable.vue";
import ProjectTable from "../components/ProjectTable.vue";
import {
  Environment,
  Instance,
  Principal,
  Project,
  UNKNOWN_ID,
} from "../types";
import { isDBAOrOwner } from "../utils";
import { BBTabFilterItem } from "../bbkit/types";

const PROJECT_TAB = 0;
const INSTANCE_TAB = 1;
const ENVIRONMENT_TAB = 2;

interface LocalState {
  selectedIndex: number;
  searchText: string;
}

export default {
  name: "Archive",
  components: { EnvironmentTable, InstanceTable, ProjectTable },
  setup(props, ctx) {
    const state = reactive<LocalState>({
      selectedIndex: PROJECT_TAB,
      searchText: "",
    });

    const currentUser: ComputedRef<Principal> = computed(() =>
      store.getters["auth/currentUser"]()
    );

    const store = useStore();

    const prepareList = () => {
      // It will also be called when user logout
      if (currentUser.value.id != UNKNOWN_ID) {
        store.dispatch("project/fetchProjectListByUser", {
          userID: currentUser.value.id,
          rowStatusList: ["ARCHIVED"],
        });
      }

      if (isDBAOrOwner(currentUser.value.role)) {
        store.dispatch("instance/fetchInstanceList", ["ARCHIVED"]);

        store.dispatch("environment/fetchEnvironmentList", ["ARCHIVED"]);
      }
    };

    watchEffect(prepareList);

    const projectList = computed((): Project[] => {
      return store.getters["project/projectListByUser"](currentUser.value.id, [
        "ARCHIVED",
      ]);
    });

    const instanceList = computed((): Instance[] => {
      return store.getters["instance/instanceList"](["ARCHIVED"]);
    });

    const environmentList = computed(() => {
      return store.getters["environment/environmentList"](["ARCHIVED"]);
    });

    const tabItemList = computed((): BBTabFilterItem[] => {
      return isDBAOrOwner(currentUser.value.role)
        ? [
            { title: "Project", alert: false },
            { title: "Instance", alert: false },
            { title: "Environment", alert: false },
          ]
        : [{ title: "Project", alert: false }];
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
};
</script>
