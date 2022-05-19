<template>
  <div class="flex flex-col">
    <BBAttention style="INFO" :title="$t('setting.project.description')" />
    <div class="px-2 py-2 flex justify-end items-center">
      <BBTableSearch
        ref="searchField"
        :placeholder="$t('project.dashboard.search-bar-placeholder')"
        @change-text="(text: string) => changeSearchText(text)"
      />
    </div>
    <ProjectTable :project-list="filteredList(state.projectList)" />
  </div>
</template>

<script lang="ts">
import { useProjectStore } from "@/store";
import { watchEffect, onMounted, reactive, ref, defineComponent } from "vue";
import ProjectTable from "../components/ProjectTable.vue";
import { Project } from "../types";

interface LocalState {
  projectList: Project[];
  searchText: string;
}

export default defineComponent({
  name: "SettingWorkspaceMember",
  components: {
    ProjectTable,
  },
  setup() {
    const searchField = ref();

    const projectStore = useProjectStore();

    const state = reactive<LocalState>({
      projectList: [],
      searchText: "",
    });

    onMounted(() => {
      // Focus on the internal search field when mounted
      searchField.value.$el.querySelector("#search").focus();
    });

    const prepareProjectList = () => {
      projectStore.fetchAllProjectList().then((projectList: Project[]) => {
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
});
</script>
