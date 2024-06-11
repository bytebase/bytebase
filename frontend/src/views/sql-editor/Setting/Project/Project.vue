<template>
  <div class="w-full flex flex-col gap-4 py-4 px-2 overflow-y-auto">
    <div class="flex flex-col items-start gap-2 sm:flex-row sm:justify-between sm:items-center">
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
      :show="state.detail.show && !!state.detail.project"
      :close-on-esc="true"
      :mask-closable="true"
      @update:show="hideDrawer"
    >
      <DrawerContent
        :title="detailTitle"
        body-content-class="flex flex-col gap-2 overflow-hidden"
      >
        <Detail v-if="state.detail.project" :project="state.detail.project" />
      </DrawerContent>
    </Drawer>
  </div>
</template>

<script setup lang="ts">
import { PlusIcon } from "lucide-vue-next";
import { NButton, NEllipsis } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import {
  Drawer,
  DrawerContent,
  ProjectV1Table,
  SearchBox,
} from "@/components/v2";
import { useProjectV1List } from "@/store";
import { DEFAULT_PROJECT_ID, type ComposedProject } from "@/types";
import { filterProjectV1ListByKeyword } from "@/utils";
import Detail from "./Detail.vue";

interface LocalState {
  keyword: string;
  detail: {
    show: boolean;
    project: ComposedProject | undefined;
  };
}

const route = useRoute();
const router = useRouter();
const { t } = useI18n();

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

const detailTitle = computed(() => {
  return state.detail.project
    ? `${t("common.project")} - ${state.detail.project.title}`
    : t("quick-action.new-project");
});

const handleClickNewProject = () => {
  state.detail.show = true;
  state.detail.project = undefined;
};

const showProjectDetail = (project: ComposedProject) => {
  state.detail.show = true;
  state.detail.project = project;
};

const hideDrawer = () => {
  state.detail.show = false;
};
</script>
