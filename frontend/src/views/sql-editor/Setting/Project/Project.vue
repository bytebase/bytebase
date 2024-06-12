<template>
  <div class="w-full flex flex-col gap-4 py-4 px-2 overflow-y-auto">
    <div
      class="flex flex-col items-start gap-2 sm:flex-row sm:justify-between sm:items-center"
    >
      <div class="flex justify-start items-center">
        <NButton @click="handleClickNewProject">
          <template #icon>
            <PlusIcon class="h-4 w-4" />
          </template>
          <NEllipsis>
            {{ $t("quick-action.new-project") }}
          </NEllipsis>
        </NButton>
      </div>

      <div class="flex justify-end items-center">
        <SearchBox
          v-model:value="state.keyword"
          class="!max-w-full md:!max-w-[18rem]"
          :placeholder="$t('common.filter-by-name')"
          :autofocus="true"
        />
      </div>
    </div>

    <ProjectV1Table
      :project-list="filteredProjectList"
      :on-click="showProjectDetail"
    />

    <Drawer
      :show="state.detail.show"
      :close-on-esc="!!state.detail.project"
      :mask-closable="!!state.detail.project"
      @update:show="hideDrawer"
    >
      <DrawerContent
        v-if="state.detail.project"
        :title="`${$t('common.project')} - ${state.detail.project.title}`"
        body-content-class="flex flex-col gap-2 overflow-hidden"
      >
        <Detail :project="state.detail.project" />
      </DrawerContent>
      <ProjectCreatePanel
        v-else
        :simple="true"
        :on-created="(project: Project) => (state.detail.project = project)"
        style="width: calc(100vw - 8rem); max-width: 50rem"
        @dismiss="hideDrawer"
      />
    </Drawer>
  </div>
</template>

<script setup lang="ts">
import { PlusIcon } from "lucide-vue-next";
import { NButton, NEllipsis } from "naive-ui";
import { computed, onMounted, reactive, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  Drawer,
  DrawerContent,
  ProjectV1Table,
  SearchBox,
} from "@/components/v2";
import { useDatabaseV1Store, useProjectV1List } from "@/store";
import type { Project } from "@/types/proto/v1/project_service";
import { filterProjectV1ListByKeyword, wrapRefAsPromise } from "@/utils";
import Detail from "./Detail.vue";

interface LocalState {
  keyword: string;
  detail: {
    show: boolean;
    project: Project | undefined;
  };
}

const route = useRoute();
const router = useRouter();

const state = reactive<LocalState>({
  keyword: "",
  detail: {
    show: false,
    project: undefined,
  },
});
const { projectList, ready } = useProjectV1List(
  /* showDeleted */ false,
  /* forceUpdate */ true
);

const filteredProjectList = computed(() => {
  return filterProjectV1ListByKeyword(projectList.value, state.keyword);
});

const handleClickNewProject = () => {
  state.detail.show = true;
  state.detail.project = undefined;
};

const showProjectDetail = (project: Project) => {
  state.detail.show = true;
  state.detail.project = project;
};

const hideDrawer = () => {
  state.detail.show = false;
};

onMounted(() => {
  // prepare for transferring databases
  useDatabaseV1Store().searchDatabases({});

  if (route.hash === "#add") {
    state.detail.show = true;
    state.detail.project = undefined;
  }
  wrapRefAsPromise(ready, true).then(() => {
    const maybeProjectName = route.hash.replace(/^#*/g, "");
    if (maybeProjectName) {
      const project = projectList.value.find(
        (proj) => proj.name === maybeProjectName
      );
      if (project) {
        state.detail.show = true;
        state.detail.project = project;
      }
    }

    watch(
      [() => state.detail.show, () => state.detail.project?.name],
      ([show, projectName]) => {
        if (show) {
          router.replace({ hash: projectName ? `#${projectName}` : "#add" });
        } else {
          router.replace({ hash: "" });
        }
      }
    );
  });
});
</script>
