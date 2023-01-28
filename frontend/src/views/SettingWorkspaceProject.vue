<template>
  <div class="flex flex-col">
    <div class="py-2 flex justify-between items-center">
      <div class="flex justify-start items-center gap-x-2">
        <BBAttention
          class="px-0"
          style="INFO"
          :title="$t('setting.project.description')"
        />
      </div>

      <div class="flex justify-end items-center px-2 gap-x-2">
        <BBTableSearch
          ref="searchField"
          :placeholder="$t('project.dashboard.search-bar-placeholder')"
          @change-text="(text: string) => changeSearchText(text)"
        />
      </div>
    </div>

    <ProjectTable :project-list="filteredList(state.projectList)" />

    <BBModal
      v-if="state.showCreateModal"
      class="relative overflow-hidden"
      :title="$t('quick-action.create-project')"
      @close="state.showCreateModal = false"
    >
      <ProjectCreate @dismiss="state.showCreateModal = false" />
    </BBModal>
  </div>
</template>

<script lang="ts">
import { useProjectStore } from "@/store";
import { watchEffect, onMounted, reactive, ref, defineComponent } from "vue";
import ProjectTable from "../components/ProjectTable.vue";
import ProjectCreate from "../components/ProjectCreate.vue";
import { Project } from "../types";

interface LocalState {
  projectList: Project[];
  searchText: string;
  showCreateModal: boolean;
}

export default defineComponent({
  name: "SettingWorkspaceProject",
  components: {
    ProjectCreate,
    ProjectTable,
  },
  setup() {
    const searchField = ref();

    const projectStore = useProjectStore();

    const state = reactive<LocalState>({
      projectList: [],
      searchText: "",
      showCreateModal: false,
    });

    onMounted(() => {
      // Focus on the internal search field when mounted
      searchField.value.$el.querySelector("#search").focus();
    });

    const prepareProjectList = async () => {
      const projectList = await projectStore.fetchAllProjectList();
      state.projectList = projectList;
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
