<template>
  <div class="flex flex-col">
    <div class="px-2 py-2 flex justify-end items-center">
      <BBTableSearch
        ref="searchField"
        :placeholder="'Search project name'"
        @change-text="(text) => changeSearchText(text)"
      />
    </div>
    <ProjectTable :projectList="filteredList(state.projectList)" />
  </div>

  <BBAlert
    v-if="state.showGuide"
    :style="'INFO'"
    :okText="'Do not show again'"
    :cancelText="'Dismiss'"
    :title="'How to setup \'Project\' ?'"
    :description="'Bytebase project is similar to the project concept in other common dev tools.\n\nA project has its own members, and every issue and database always belongs to a single project.\n\nA project can also be configured to link to a repository to enable version control workflow.'"
    @ok="
      () => {
        doDismissGuide();
      }
    "
    @cancel="state.showGuide = false"
  >
  </BBAlert>
</template>

<script lang="ts">
import { watchEffect, computed, onMounted, reactive, ref } from "vue";
import { useStore } from "vuex";
import ProjectTable from "../components/ProjectTable.vue";
import { Project, UNKNOWN_ID } from "../types";

interface LocalState {
  projectList: Project[];
  searchText: string;
  showGuide: boolean;
}

export default {
  name: "Home",
  components: {
    ProjectTable,
  },
  props: {},
  setup(props, ctx) {
    const searchField = ref();

    const store = useStore();

    const state = reactive<LocalState>({
      projectList: [],
      searchText: "",
      showGuide: false,
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    onMounted(() => {
      // Focus on the internal search field when mounted
      searchField.value.$el.querySelector("#search").focus();

      if (!store.getters["uistate/introStateByKey"]("guide.project")) {
        setTimeout(() => {
          state.showGuide = true;
          store.dispatch("uistate/saveIntroStateByKey", {
            key: "project.visit",
            newState: true,
          });
        }, 1000);
      }
    });

    const prepareProjectList = () => {
      // It will also be called when user logout
      if (currentUser.value.id != UNKNOWN_ID) {
        store
          .dispatch("project/fetchProjectListByUser", {
            userID: currentUser.value.id,
          })
          .then((projectList: Project[]) => {
            state.projectList = projectList;
          });
      }
    };

    watchEffect(prepareProjectList);

    const changeSearchText = (searchText: string) => {
      state.searchText = searchText;
    };

    const doDismissGuide = () => {
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "guide.project",
        newState: true,
      });
      state.showGuide = false;
    };

    const filteredList = (list: Project[]) => {
      if (!state.searchText) {
        // Select "All"
        return list;
      }
      return list.filter((issue) => {
        return (
          !state.searchText ||
          issue.name.toLowerCase().includes(state.searchText.toLowerCase())
        );
      });
    };

    return {
      searchField,
      state,
      filteredList,
      doDismissGuide,
      changeSearchText,
    };
  },
};
</script>
