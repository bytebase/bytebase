<template>
  <div class="w-full h-full grid grid-cols-[212px_1fr] gap-0">
    <div
      class="w-full h-auto flex flex-col justify-start items-start px-4 pl-6 py-2 border-r"
    >
      <router-link
        v-for="nav in navigationList"
        :key="nav.path"
        :to="nav.path"
        class="text-base w-full p-2 px-3 rounded-lg mt-1 select-none hover:bg-gray-100"
        active-class="active-link"
        exact-active-class="active-link"
        >{{ nav.label }}</router-link
      >
    </div>
    <div
      class="w-full h-full flex flex-col justify-start items-start overflow-y-auto px-4 py-4"
    >
      <div class="w-full px-4 mb-2 flex flex-row justify-between items-center">
        <div class="flex flex-row justify-start items-center">
          <div class="grow flex flex-row justify-start items-center">
            <span class="text-sm mr-2 whitespace-nowrap"
              >{{ $t("common.project") }}:
            </span>
            <n-select
              v-model:value="projectSelectorValue"
              :consistent-menu-width="false"
              :options="projectSelectOptions"
            />
          </div>
          <div class="ml-4">
            <n-input
              v-model:value="sheetSearchValue"
              type="text"
              :placeholder="t('common.search')"
            >
              <template #prefix>
                <heroicons-outline:search class="w-4 h-auto text-gray-300" />
              </template>
            </n-input>
          </div>
          <div class="ml-4">
            <n-button
              v-show="shouldShowClearSearchBtn"
              text
              @click="handleClearSearchBtnClick"
            >
              {{ $t("common.clear-search") }}
            </n-button>
          </div>
        </div>
        <div>
          <n-button
            v-if="selectedProject?.workflow === Workflow.VCS"
            @click="handleSyncSheetFromVCS"
          >
            <heroicons-outline:refresh
              v-if="hasFeature('bb.feature.vcs-sheet-sync')"
              class="w-4 h-auto mr-1"
            />
            <FeatureBadge v-else feature="bb.feature.vcs-sheet-sync" />
            {{ $t("sheet.actions.sync-from-vcs") }}
          </n-button>
        </div>
      </div>
      <div class="w-full">
        <SheetTable
          :view="currentSheetViewMode"
          :sheet-list="shownSheetList"
          :loading="state.isLoading"
          @refresh="fetchSheetData"
        />
      </div>
    </div>
  </div>

  <FeatureModal
    feature="bb.feature.vcs-sheet-sync"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { last } from "lodash-es";
import { useDialog } from "naive-ui";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { t } from "@/plugins/i18n";
import {
  hasFeature,
  useUserStore,
  useSheetV1Store,
  useProjectV1ListByCurrentUser,
  useProjectV1Store,
  useEnvironmentV1Store,
} from "@/store";
import { Workflow } from "@/types/proto/v1/project_service";
import { Sheet } from "@/types/proto/v1/sheet_service";
import {
  type SheetViewMode,
  SheetTable,
  SheetViewModeList,
} from "./sql-editor/Sheet";

interface LocalState {
  isLoading: boolean;
  sheetList: Sheet[];
  showFeatureModal: boolean;
}

const route = useRoute();
const dialog = useDialog();
const state = reactive<LocalState>({
  isLoading: true,
  sheetList: [],
  showFeatureModal: false,
});
const sheetV1Store = useSheetV1Store();

const projectSelectorValue = ref("");
const sheetSearchValue = ref("");

const navigationList = computed(() => {
  const list = [
    {
      path: "/sheets/my",
      label: t("sheet.my-sheets"),
    },
    {
      path: "/sheets/shared",
      label: t("sheet.shared-with-me"),
    },
    {
      path: "/sheets/starred",
      label: t("sheet.starred"),
    },
  ];

  return list;
});

const shownSheetList = computed(() => {
  let list = [...state.sheetList];
  const projectName = projectSelectorValue.value;
  if (projectName !== "") {
    list = list.filter((sheet) => sheet.name.startsWith(projectName));
  }

  const keyword = sheetSearchValue.value.trim().toLowerCase();
  if (keyword) {
    list = list.filter((sheet) => {
      return (
        sheet.name.toLowerCase().includes(keyword) ||
        new TextDecoder().decode(sheet.content).toLowerCase().includes(keyword)
      );
    });
  }

  return list.sort(
    (a, b) =>
      (b.updateTime ?? new Date()).getTime() -
      (a.updateTime ?? new Date()).getTime()
  );
});

const { projectList } = useProjectV1ListByCurrentUser(false /* !showDeleted */);

const selectedProject = computed(() => {
  for (const project of projectList.value) {
    if (project.uid === projectSelectorValue.value) {
      return project;
    }
  }

  return null;
});

const projectSelectOptions = computed(() => {
  return [
    {
      label: t("common.all"),
      value: "",
    },
  ].concat(
    projectList.value.map((project) => {
      return {
        label: project.title,
        value: project.name,
      };
    })
  );
});

const shouldShowClearSearchBtn = computed(() => {
  return projectSelectorValue.value !== "" || sheetSearchValue.value !== "";
});

const currentSheetViewMode = computed((): SheetViewMode => {
  const { path } = route;
  const subPath = (last(path.split("/")) || "my") as SheetViewMode;
  if (SheetViewModeList.includes(subPath)) {
    return subPath;
  }
  return "my";
});

const fetchSheetData = async () => {
  await useUserStore().fetchUserList();

  // TODO: switching view mode very quickly will cause some
  // race condition problems.
  let sheetList: Sheet[] = [];
  if (currentSheetViewMode.value === "my") {
    sheetList = await sheetV1Store.fetchMySheetList();
  } else if (currentSheetViewMode.value === "starred") {
    sheetList = await sheetV1Store.fetchStarredSheetList();
  } else if (currentSheetViewMode.value === "shared") {
    sheetList = await sheetV1Store.fetchSharedSheetList();
  }

  state.sheetList = sheetList;
};

onMounted(async () => {
  // Initialize project list state for iam policy and `project` fields of sheets.
  await useProjectV1Store().fetchProjectList(true /* include archived */);
  // Initialize environment list for composing.
  await useEnvironmentV1Store().fetchEnvironments(true /* include archived */);
  await fetchSheetData();
  state.isLoading = false;
});

watch(
  () => route.path,
  async () => {
    await fetchSheetData();
  }
);

const handleClearSearchBtnClick = () => {
  projectSelectorValue.value = "";
  sheetSearchValue.value = "";
};

const handleSyncSheetFromVCS = () => {
  if (!hasFeature("bb.feature.vcs-sheet-sync")) {
    state.showFeatureModal = true;
    return;
  }

  if (
    selectedProject.value === null ||
    selectedProject.value.workflow !== Workflow.VCS
  ) {
    return;
  }

  const selectedProjectName = selectedProject.value.name;
  const dialogInstance = dialog.create({
    title: t("sheet.hint-tips.confirm-to-sync-sheet"),
    type: "info",
    autoFocus: false,
    closable: false,
    maskClosable: false,
    closeOnEsc: false,
    async onPositiveClick() {
      dialogInstance.closable = false;
      dialogInstance.loading = true;
      await sheetV1Store.syncSheetFromVCS(selectedProjectName);
      await fetchSheetData();
      dialogInstance.destroy();
    },
    positiveText: t("common.confirm"),
    showIcon: true,
  });
};
</script>

<style scoped>
.active-link {
  @apply bg-gray-100 text-accent;
}
</style>
