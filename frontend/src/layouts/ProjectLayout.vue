<template>
  <div class="py-2">
    <ArchiveBanner v-if="project.rowStatus == 'ARCHIVED'" />
  </div>
  <h1 class="px-6 pb-4 text-xl font-bold leading-6 text-main truncate">
    {{ project.name }}
    <span
      v-if="project.tenantMode === 'TENANT'"
      class="text-sm font-normal px-2 ml-2 rounded whitespace-nowrap inline-flex items-center bg-gray-200"
    >
      {{ $t("project.mode.tenant") }}
    </span>
  </h1>
  <BBTabFilter
    class="px-3 pb-2 border-b border-block-border"
    :responsive="false"
    :tab-item-list="tabItemList"
    :selected-index="state.selectedIndex"
    @select-index="
      (index: number) => {
        selectTab(index);
      }
    "
  />

  <div class="py-6 px-6">
    <router-view
      :project-slug="projectSlug"
      :project-webhook-slug="projectWebhookSlug"
      :selected-tab="state.selectedIndex"
      :allow-edit="allowEdit"
    />
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, onMounted, reactive, watch } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { idFromSlug, isProjectOwner } from "../utils";
import ArchiveBanner from "../components/ArchiveBanner.vue";
import { BBTabFilterItem } from "../bbkit/types";
import { useI18n } from "vue-i18n";
import { Project } from "../types";

const OVERVIEW_TAB = 0;
const WEBHOOK_TAB = 4;

type ProjectTabItem = {
  name: string;
  hash: string;
};

interface LocalState {
  selectedIndex: number;
}

export default defineComponent({
  name: "ProjectLayout",
  components: {
    ArchiveBanner,
  },
  props: {
    projectSlug: {
      required: true,
      type: String,
    },
    projectWebhookSlug: {
      type: String,
      default: undefined,
    },
  },
  setup(props) {
    const store = useStore();
    const router = useRouter();
    const { t } = useI18n();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const project = computed((): Project => {
      return store.getters["project/projectById"](
        idFromSlug(props.projectSlug)
      );
    });

    const projectTabItemList = computed((): ProjectTabItem[] => {
      const list: ProjectTabItem[] = [
        { name: t("common.overview"), hash: "overview" },
        { name: t("common.migration-history"), hash: "migration-history" },
        { name: t("common.activities"), hash: "activity" },
        { name: t("common.version-control"), hash: "version-control" },
        { name: t("common.webhooks"), hash: "webhook" },
        { name: t("common.settings"), hash: "setting" },
      ];
      // TODO: we can't put DeploymentConfig before Settings for now
      // because BBTabFilter works on numeric index
      // making indices dynamic needs a refactor
      if (project.value.tenantMode === "TENANT") {
        list.push({
          name: t("common.deployment-config"),
          hash: "deployment-config",
        });
      }

      return list;
    });

    const state = reactive<LocalState>({
      selectedIndex: OVERVIEW_TAB,
    });

    // Only the project owner can edit the project general info and configure version control.
    // This means even the workspace owner won't be able to edit it.
    // On the other hand, we allow workspace owner to change project membership in case
    // project is locked somehow. See the relevant method in ProjectMemberTable for more info.
    const allowEdit = computed(() => {
      if (project.value.rowStatus == "ARCHIVED") {
        return false;
      }

      for (const member of project.value.memberList) {
        if (member.principal.id == currentUser.value.id) {
          if (isProjectOwner(member.role)) {
            return true;
          }
        }
      }
      return false;
    });

    const tabItemList = computed((): BBTabFilterItem[] => {
      return projectTabItemList.value.map((item) => {
        return {
          title: item.name,
          alert: false,
        };
      });
    });

    const selectProjectTabOnHash = () => {
      if (router.currentRoute.value.name == "workspace.project.detail") {
        if (router.currentRoute.value.hash) {
          for (let i = 0; i < projectTabItemList.value.length; i++) {
            if (
              projectTabItemList.value[i].hash ==
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
        hash: "#" + projectTabItemList.value[index].hash,
      });
    };

    return {
      state,
      project,
      allowEdit,
      tabItemList,
      selectTab,
    };
  },
});
</script>
