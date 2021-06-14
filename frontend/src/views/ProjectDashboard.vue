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
</template>

<script lang="ts">
import { watchEffect, computed, onMounted, reactive, ref } from "vue";
import { useStore } from "vuex";
import ProjectTable from "../components/ProjectTable.vue";
import { Project } from "../types";

interface LocalState {
  projectList: Project[];
  searchText: string;
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
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    onMounted(() => {
      // Focus on the internal search field when mounted
      searchField.value.$el.querySelector("#search").focus();
    });

    const prepareProjectList = () => {
      store
        .dispatch("project/fetchProjectListByUser", {
          userId: currentUser.value.id,
        })
        .then((projectList: Project[]) => {
          state.projectList = projectList;
        });
    };

    watchEffect(prepareProjectList);

    const changeSearchText = (searchText: string) => {
      state.searchText = searchText;
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
      changeSearchText,
    };
  },
};
</script>
