<template>
  <div class="py-2">
    <ArchiveBanner v-if="project.rowStatus == 'ARCHIVED'" />
  </div>
  <h1 class="px-4 pb-4 text-xl font-bold leading-6 text-main truncate">
    {{ project.name }}
  </h1>
  <BBTableTabFilter
    class="px-1 pb-2 border-b border-block-border"
    :responsive="false"
    :tabList="['Overview', 'Repository', 'Settings']"
    :selectedIndex="state.selectedIndex"
    @select-index="
      (index) => {
        state.selectedIndex = index;
      }
    "
  />
  <div
    class="max-w-4xl mx-auto py-6 px-4 divide-y divide-block-border space-y-6 sm:px-6 lg:px-8"
  >
    <template v-if="state.selectedIndex == OVERVIEW_TAB"> </template>
    <template v-else-if="state.selectedIndex == REPO_TAB"> </template>
    <template v-else-if="state.selectedIndex == SETTING_TAB">
      <ProjectGeneralSettingPanel :project="project" />
      <ProjectMemberPanel class="pt-4" :project="project" />
    </template>
  </div>
</template>

<script lang="ts">
import { computed, reactive } from "vue";
import { useStore } from "vuex";
import { idFromSlug } from "../utils";
import ArchiveBanner from "../components/ArchiveBanner.vue";
import DatabaseTable from "../components/DatabaseTable.vue";
import ProjectGeneralSettingPanel from "../components/ProjectGeneralSettingPanel.vue";
import ProjectMemberPanel from "../components/ProjectMemberPanel.vue";

const OVERVIEW_TAB = 0;
const REPO_TAB = 1;
const SETTING_TAB = 2;

interface LocalState {
  selectedIndex: number;
}

export default {
  name: "ProjectDetail",
  components: {
    ArchiveBanner,
    DatabaseTable,
    ProjectGeneralSettingPanel,
    ProjectMemberPanel,
  },
  props: {
    projectSlug: {
      required: true,
      type: String,
    },
  },
  setup(props, { emit }) {
    const store = useStore();
    const state = reactive<LocalState>({
      selectedIndex: SETTING_TAB,
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const project = computed(() => {
      return store.getters["project/projectById"](
        idFromSlug(props.projectSlug)
      );
    });

    return {
      OVERVIEW_TAB,
      REPO_TAB,
      SETTING_TAB,
      state,
      project,
    };
  },
};
</script>
