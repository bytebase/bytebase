<template>
  <div class="py-2">
    <ArchiveBanner v-if="project.rowStatus == 'ARCHIVED'" />
  </div>
  <h1 class="px-6 pb-4 text-xl font-bold leading-6 text-main truncate">
    {{ project.name }}
  </h1>
  <BBTabFilter
    class="px-3 pb-2 border-b border-block-border"
    :responsive="false"
    :tabList="projectTabItemList.map((item) => item.name)"
    :selectedIndex="state.selectedIndex"
    @select-index="
      (index) => {
        selectTab(index);
      }
    "
  />

  <div class="max-w-7xl mx-auto py-6 px-6">
    <router-view
      :selectedTab="state.selectedIndex"
      :projectSlug="projectSlug"
      :projectHookSlug="projectHookSlug"
    />
  </div>
</template>

<script lang="ts">
import { computed, onMounted, reactive, watch } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { idFromSlug } from "../utils";
import ArchiveBanner from "../components/ArchiveBanner.vue";

const OVERVIEW_TAB = 0;
const WEBHOOK_TAB = 3;

type ProjectTabItem = {
  name: string;
  hash: string;
};

const projectTabItemList: ProjectTabItem[] = [
  { name: "Overview", hash: "overview" },
  { name: "Migration History", hash: "migration-history" },
  { name: "Version Control", hash: "version-control" },
  { name: "Webhooks", hash: "hook" },
  { name: "Settings", hash: "setting" },
];

interface LocalState {
  selectedIndex: number;
}

export default {
  name: "ProjectLayout",
  components: {
    ArchiveBanner,
  },
  props: {
    projectSlug: {
      required: true,
      type: String,
    },
    projectHookSlug: {
      type: String,
    },
  },
  setup(props, { emit }) {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      selectedIndex: OVERVIEW_TAB,
    });

    const project = computed(() => {
      return store.getters["project/projectById"](
        idFromSlug(props.projectSlug)
      );
    });

    const selectProjectTabOnHash = () => {
      if (router.currentRoute.value.name == "workspace.project.detail") {
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
      } else if (
        router.currentRoute.value.name == "workspace.project.hook.create" ||
        router.currentRoute.value.name == "workspace.project.hook.detail"
      ) {
        state.selectedIndex = WEBHOOK_TAB;
      }
    };

    onMounted(() => {
      selectProjectTabOnHash();
    });

    watch(
      () => router.currentRoute.value.hash,
      () => {
        selectProjectTabOnHash();
      }
    );

    const selectTab = (index: number) => {
      state.selectedIndex = index;
      router.replace({
        name: "workspace.project.detail",
        hash: "#" + projectTabItemList[index].hash,
      });
    };

    return {
      state,
      project,
      selectTab,
      projectTabItemList,
    };
  },
};
</script>
