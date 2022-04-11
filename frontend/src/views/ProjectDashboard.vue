<template>
  <div class="flex flex-col">
    <div class="px-2 py-2 flex justify-end items-center">
      <BBTableSearch
        ref="searchField"
        :placeholder="$t('project.dashboard.search-bar-placeholder')"
        @change-text="(text: string) => changeSearchText(text)"
      />
    </div>
    <ProjectTable :project-list="filteredList(state.projectList)" />
  </div>

  <BBAlert
    v-if="state.showGuide"
    :style="'INFO'"
    :ok-text="$t('project.dashboard.modal.confirm')"
    :cancel-text="$t('project.dashboard.modal.cancel')"
    :title="$t('project.dashboard.modal.title')"
    :description="$t('project.dashboard.modal.content')"
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
import { useCurrentUser, useUIStateStore } from "@/store";
import { watchEffect, onMounted, reactive, ref, defineComponent } from "vue";
import { useStore } from "vuex";
import ProjectTable from "../components/ProjectTable.vue";
import { Project, UNKNOWN_ID } from "../types";

interface LocalState {
  projectList: Project[];
  searchText: string;
  showGuide: boolean;
}

export default defineComponent({
  name: "ProjectDashboard",
  components: {
    ProjectTable,
  },
  setup() {
    const searchField = ref();

    const store = useStore();
    const uiStateStore = useUIStateStore();

    const state = reactive<LocalState>({
      projectList: [],
      searchText: "",
      showGuide: false,
    });

    const currentUser = useCurrentUser();

    onMounted(() => {
      // Focus on the internal search field when mounted
      searchField.value.$el.querySelector("#search").focus();

      if (!uiStateStore.getIntroStateByKey("guide.project")) {
        setTimeout(() => {
          state.showGuide = true;
          uiStateStore.saveIntroStateByKey({
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
            userId: currentUser.value.id,
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
      uiStateStore.saveIntroStateByKey({
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
});
</script>
