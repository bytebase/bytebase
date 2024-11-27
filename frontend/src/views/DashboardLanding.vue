<template>
  <div class="px-4 flex flex-col h-full items-center relative">
    <div class="flex-1" />
    <div class="flex-[60%] space-y-6">
      <div class="flex items-baseline gap-x-4">
        <div class="flex items-start gap-x-1">
          <div class="font-semibold text-2xl">
            {{ $t("landing.quick-link.self") }}
          </div>
          <NButton
            quaternary
            size="small"
            @click="state.showConfigDrawer = true"
          >
            <template #icon>
              <SettingsIcon class="w-4" />
            </template>
          </NButton>
        </div>
        <router-link
          v-if="lastProject"
          class="underline normal-link"
          :to="{ path: lastVisitProjectPath }"
        >
          {{ $t("landing.last-visit") }}:
          {{ lastProject.title }}
        </router-link>
      </div>
      <div
        class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4"
      >
        <component
          :is="access.route ? 'router-link' : 'div'"
          v-for="(access, i) in quickLinkList"
          :key="i"
          :to="{
            name: access.route,
          }"
          class="flex justify-center items-center gap-x-2 cursor-pointer border rounded px-4 py-5 bg-white hover:bg-gray-100"
          @click="handleClick(access)"
        >
          <component :is="access.icon" class="w-5 h-5 text-gray-500" />
          {{ access.title }}
        </component>
      </div>
      <div class="space-y-2">
        <a
          v-if="actuatorStore.changelogURL"
          class="underline normal-link"
          target="_blank"
          :href="actuatorStore.changelogURL"
        >
          {{
            $t("landing.changelog-for-version", {
              version: actuatorStore.version,
            })
          }}
        </a>
        <a
          v-if="actuatorStore.hasNewRelease"
          class="underline normal-link flex items-center gap-x-1"
          target="_blank"
          :href="actuatorStore.releaseInfo.latest?.html_url"
        >
          <Volume2Icon class="h-4 w-4" />
          {{
            $t("settings.release.new-version-available-with-tag", {
              tag: actuatorStore.releaseInfo.latest?.tag_name,
            })
          }}
        </a>
      </div>
    </div>
  </div>

  <ProjectSwitchModal
    :show="state.showProjectModal"
    @dismiss="state.showProjectModal = false"
  />

  <Drawer
    :show="state.showConfigDrawer"
    :close-on-esc="true"
    @close="state.showConfigDrawer = false"
  >
    <DrawerContent
      :title="$t('landing.quick-link.manage')"
      class="!w-96 max-w-full"
      style="max-width: calc(100vw - 8rem)"
    >
      <div>
        <Draggable v-model="quickLinkList" item-key="id" animation="300">
          <template #item="{ element }: { element: QuickLink }">
            <div
              class="flex items-center justify-between p-2 hover:bg-gray-100 rounded-sm cursor-grab"
            >
              <div :key="element.id" class="flex items-center gap-x-2">
                <NCheckbox
                  :disabled="quickLinkList.length <= 1"
                  :checked="true"
                  @update:checked="() => uncheckAccessItem(element)"
                />
                <component :is="element.icon" class="w-5 h-5 text-gray-500" />
                {{ element.title }}
              </div>
              <ArrowDownUpIcon class="w-5 h-5 text-gray-500" />
            </div>
          </template>
        </Draggable>
        <NDivider />

        <div
          v-for="access in unSelectedAccessList"
          :key="access.id"
          class="flex items-center gap-x-2 p-2 hover:bg-gray-100 rounded-sm cursor-pointer"
          @click.prevent.stop="() => checkAccessItem(access)"
        >
          <NCheckbox
            :checked="false"
            @update:checked="() => checkAccessItem(access)"
          />
          <component :is="access.icon" class="w-5 h-5 text-gray-500" />
          {{ access.title }}
        </div>
      </div>
    </DrawerContent>
  </Drawer>
</template>

<script lang="tsx" setup>
import { Volume2Icon } from "lucide-vue-next";
import { SettingsIcon, ArrowDownUpIcon } from "lucide-vue-next";
import { NButton, NCheckbox, NDivider } from "naive-ui";
import { computed, reactive } from "vue";
import { useRouter } from "vue-router";
import Draggable from "vuedraggable";
import ProjectSwitchModal from "@/components/Project/ProjectSwitch/ProjectSwitchModal.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { WORKSPACE_ROUTE_MY_ISSUES } from "@/router/dashboard/workspaceRoutes";
import { useRecentVisit } from "@/router/useRecentVisit";
import {
  useProjectV1Store,
  useActuatorV1Store,
  useQuickLink,
  type QuickLink,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { UNKNOWN_PROJECT_NAME } from "@/types";
import { extractProjectResourceName } from "@/utils";

interface LocalState {
  showProjectModal: boolean;
  showConfigDrawer: boolean;
}

const state = reactive<LocalState>({
  showProjectModal: false,
  showConfigDrawer: false,
});
const projectStore = useProjectV1Store();
const actuatorStore = useActuatorV1Store();
const { lastVisitProjectPath } = useRecentVisit();
const router = useRouter();

const lastProject = computed(() => {
  if (!lastVisitProjectPath.value) {
    return;
  }
  const projectName = `${projectNamePrefix}${extractProjectResourceName(lastVisitProjectPath.value)}`;
  const project = projectStore.getProjectByName(projectName);
  if (project.name === UNKNOWN_PROJECT_NAME) {
    return;
  }
  return project;
});

const { quickLinkList, quickAccessConfig, fullQuickLinkList } = useQuickLink();

const handleClick = (access: QuickLink) => {
  if (access.route) {
    return;
  }

  switch (access.id) {
    case "visit-projects":
      state.showProjectModal = true;
      break;
    case "visit-issues":
      router.push({
        name: WORKSPACE_ROUTE_MY_ISSUES,
      });
      return;
    default:
      return;
  }
};

const unSelectedAccessList = computed(() =>
  fullQuickLinkList.value.filter(
    (item) => !quickAccessConfig.value.includes(item.id)
  )
);

const checkAccessItem = (access: QuickLink) => {
  const index = quickAccessConfig.value.findIndex((id) => id === access.id);
  if (index < 0) {
    quickAccessConfig.value.push(access.id);
  }
};

const uncheckAccessItem = (access: QuickLink) => {
  const index = quickAccessConfig.value.findIndex((id) => id === access.id);
  if (index >= 0) {
    quickAccessConfig.value.splice(index, 1);
  }
};
</script>
