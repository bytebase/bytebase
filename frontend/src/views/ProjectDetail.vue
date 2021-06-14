<template>
  <div class="py-2">
    <ArchiveBanner v-if="project.rowStatus == 'ARCHIVED'" />
  </div>
  <h1 class="px-4 pb-4 text-xl font-bold leading-6 text-main truncate">
    {{ project.name }}
  </h1>
  <BBTabFilter
    class="px-1 pb-2 border-b border-block-border"
    :responsive="false"
    :tabList="projectTabItemList.map((item) => item.name)"
    :selectedIndex="state.selectedIndex"
    @select-index="
      (index) => {
        selectTab(index);
      }
    "
  />
  <div class="max-w-7xl mx-auto py-6 px-4 sm:px-6 lg:px-8">
    <template v-if="state.selectedIndex == OVERVIEW_TAB">
      <ProjectOverviewPanel :project="project" id="overview" />
    </template>
    <template v-else-if="state.selectedIndex == VERSION_CONTROL_TAB">
      <ProjectVersionControlPanel :project="project" id="version-control" />
    </template>
    <template v-else-if="state.selectedIndex == SETTING_TAB">
      <ProjectSettingPanel :project="project" id="setting" />
    </template>
  </div>
</template>

<script lang="ts">
import { computed, onMounted, reactive, watch } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { idFromSlug } from "../utils";
import ArchiveBanner from "../components/ArchiveBanner.vue";
import ProjectOverviewPanel from "../components/ProjectOverviewPanel.vue";
import ProjectVersionControlPanel from "../components/ProjectVersionControlPanel.vue";
import ProjectSettingPanel from "../components/ProjectSettingPanel.vue";

const OVERVIEW_TAB = 0;
const VERSION_CONTROL_TAB = 1;
const SETTING_TAB = 2;

type ProjectTabItem = {
  name: string;
  hash: string;
};

const projectTabItemList: ProjectTabItem[] = [
  { name: "Overview", hash: "overview" },
  { name: "Version Control", hash: "version-control" },
  { name: "Settings", hash: "setting" },
];

interface LocalState {
  selectedIndex: number;
}

export default {
  name: "ProjectDetail",
  components: {
    ArchiveBanner,
    ProjectOverviewPanel,
    ProjectVersionControlPanel,
    ProjectSettingPanel,
  },
  props: {
    projectSlug: {
      required: true,
      type: String,
    },
  },
  setup(props, { emit }) {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      selectedIndex: OVERVIEW_TAB,
    });

    const selectProjectTabOnHash = () => {
      if (router.currentRoute.value.hash) {
        for (let i = 0; i < projectTabItemList.length; i++) {
          if (
            projectTabItemList[i].hash ==
            router.currentRoute.value.hash.slice(1)
          ) {
            selectTab(i);
            break;
          }
        }
      } else {
        selectTab(OVERVIEW_TAB);
      }
    };

    onMounted(() => {
      selectProjectTabOnHash();
    });

    watch(
      () => router.currentRoute.value.hash,
      () => {
        if (router.currentRoute.value.name == "workspace.project.detail") {
          selectProjectTabOnHash();
        }
      }
    );

    const project = computed(() => {
      return store.getters["project/projectById"](
        idFromSlug(props.projectSlug)
      );
    });

    const selectTab = (index: number) => {
      state.selectedIndex = index;
      router.replace({
        name: "workspace.project.detail",
        hash: "#" + projectTabItemList[index].hash,
      });
    };

    return {
      OVERVIEW_TAB,
      VERSION_CONTROL_TAB,
      SETTING_TAB,
      state,
      project,
      selectTab,
      projectTabItemList,
    };
  },
};
</script>
