<template>
  <div class="w-full flex flex-col gap-4 py-4 overflow-y-auto">
    <div class="flex items-center space-x-2 px-4">
      <SearchBox
        v-model:value="state.keyword"
        style="max-width: 100%"
        :placeholder="$t('common.filter-by-name')"
        :autofocus="true"
      />
      <NButton type="primary" @click="handleClickNewProject">
        <template #icon>
          <PlusIcon class="h-4 w-4" />
        </template>
        <NEllipsis>
          {{ $t("quick-action.new-project") }}
        </NEllipsis>
      </NButton>
    </div>

    <PagedProjectTable
      session-key="bb.sql-editor.project-table"
      :filter="{
        query: state.keyword,
        excludeDefault: true,
      }"
      :footer-class="'mx-4'"
      :bordered="false"
      :prevent-default="true"
      @row-click="showProjectDetail"
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
        class="project-detail-drawer"
        body-content-class="flex flex-col gap-2 overflow-hidden"
      >
        <Detail :project="state.detail.project" />
      </DrawerContent>
      <ProjectCreatePanel
        v-else
        :simple="true"
        :on-created="handleCreated"
        style="width: calc(100vw - 8rem); max-width: 50rem"
        @dismiss="hideDrawer"
      />
    </Drawer>
  </div>
</template>

<script setup lang="ts">
import { PlusIcon } from "lucide-vue-next";
import { NButton, NEllipsis } from "naive-ui";
import { reactive, watch } from "vue";
import { useRouter } from "vue-router";
import ProjectCreatePanel from "@/components/Project/ProjectCreatePanel.vue";
import {
  Drawer,
  DrawerContent,
  PagedProjectTable,
  SearchBox,
} from "@/components/v2";
import type { ComposedProject } from "@/types";
import Detail from "./Detail.vue";

interface LocalState {
  keyword: string;
  detail: {
    show: boolean;
    project: ComposedProject | undefined;
  };
}

const router = useRouter();
const state = reactive<LocalState>({
  keyword: "",
  detail: {
    show: false,
    project: undefined,
  },
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

const handleCreated = async (project: ComposedProject) => {
  state.detail.project = project;
};

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
</script>

<style scoped lang="postcss">
.project-detail-drawer :deep(.n-drawer-header__main) {
  @apply flex-1 flex items-center justify-between;
}
</style>
