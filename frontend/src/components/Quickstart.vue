<template>
  <div
    v-if="showQuickstart"
    class="py-2 px-4 w-full shrink-0 border-t border-block-border hidden lg:block bg-yellow-50"
  >
    <p
      class="text-sm font-medium text-gray-900 flex items-center justify-between"
    >
      <span
        >🎈 {{ $t("quick-start.self") }} - {{ $t("quick-start.guide") }}</span
      >

      <button class="btn-icon" @click.prevent="() => hideQuickstart()">
        <XIcon class="w-4 h-4" />
      </button>
    </p>
    <div class="mt-2" aria-hidden="true">
      <div class="overflow-hidden rounded-full bg-gray-200">
        <div
          class="h-2 rounded-full bg-indigo-600"
          :style="{ width: percent }"
        />
      </div>
      <div
        class="mt-2 hidden grid-cols-4 text-sm font-medium text-gray-600 sm:grid"
        :style="{
          gridTemplateColumns: `repeat(${introList.length}, minmax(0, 1fr))`,
        }"
      >
        <component
          :is="intro.link ? 'router-link' : 'div'"
          v-for="(intro, index) in introList"
          :key="index"
          :to="intro.link"
          active-class=""
          exact-active-class=""
          class="group cursor-pointer flex items-center gap-x-1 text-sm font-medium"
          :class="[
            index === 0 && 'justify-start',
            index > 0 && index < introList.length - 1 && 'justify-center',
            index === introList.length - 1 && 'justify-end',
            isTaskActive(index)
              ? 'text-indigo-600'
              : 'text-control-light group-hover:text-control-light-hover',
            unref(intro.done) && 'line-through',
          ]"
          @click="intro.click"
        >
          <span
            class="relative h-5 w-5 inline-flex items-center justify-center"
          >
            <template v-if="intro.done.value">
              <CheckCircleIcon
                class="w-4 h-auto text-success group-hover:text-success-hover"
              />
            </template>
            <template v-else-if="isTaskActive(index)">
              <span class="relative flex h-3 w-3">
                <span
                  class="absolute inline-flex h-full w-full rounded-full bg-indigo-400 opacity-75"
                  style="animation: ping 2s cubic-bezier(0, 0, 0.2, 1) infinite"
                ></span>
                <span
                  class="relative inline-flex rounded-full h-3 w-3 bg-indigo-500"
                ></span>
              </span>
            </template>
            <template v-else>
              <div
                class="h-2 w-2 bg-gray-300 rounded-full group-hover:bg-gray-400"
              ></div>
            </template>
          </span>
          <span class="inline-flex">
            {{ unref(intro.name) }}
          </span>
        </component>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computedAsync } from "@vueuse/core";
import { CheckCircleIcon, XIcon } from "lucide-vue-next";
import type { Ref } from "vue";
import { computed, unref } from "vue";
import { useI18n } from "vue-i18n";
import type { RouteLocationRaw } from "vue-router";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import {
  DATABASE_ROUTE_DASHBOARD,
  ENVIRONMENT_V1_ROUTE_DASHBOARD,
  INSTANCE_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DASHBOARD,
  WORKSPACE_ROUTE_USERS,
} from "@/router/dashboard/workspaceRoutes";
import { SQL_EDITOR_WORKSHEET_MODULE } from "@/router/sqlEditor";
import {
  pushNotification,
  useActuatorV1Store,
  useIssueV1Store,
  useProjectIamPolicyStore,
  useProjectV1Store,
  useUIStateStore,
  useWorkSheetStore,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import type { Permission } from "@/types";
import { isValidProjectName, UNKNOWN_PROJECT_NAME } from "@/types";
import {
  extractProjectResourceName,
  hasProjectPermissionV2,
  hasWorkspacePermissionV2,
} from "@/utils";

// The name of the sample project.
const SAMPLE_PROJECT_NAME = "project-sample";
const SAMPLE_ISSUE_ID = "101";
const SAMPLE_SHEET_ID = "101";

type IntroItem = {
  name: string | Ref<string>;
  link?: RouteLocationRaw;
  done: Ref<boolean>;
  click?: () => void;
  hide?: boolean;
  requiredPermissions?: Permission[];
};

const { t } = useI18n();
const projectStore = useProjectV1Store();
const uiStateStore = useUIStateStore();
const actuatorStore = useActuatorV1Store();
const issueStore = useIssueV1Store();
const worksheetStore = useWorkSheetStore();
const projectIamPolicyStore = useProjectIamPolicyStore();

const sampleProject = computedAsync(async () => {
  if (!actuatorStore.quickStartEnabled) {
    return;
  }
  const project = await projectStore.getOrFetchProjectByName(
    `${projectNamePrefix}${SAMPLE_PROJECT_NAME}`,
    true /* silent */
  );
  if (!isValidProjectName(project.name)) {
    return;
  }
  await projectIamPolicyStore.getOrFetchProjectIamPolicy(project.name);
  return project;
});

const sampleIssue = computedAsync(async () => {
  if (!sampleProject.value) {
    return;
  }
  if (!hasProjectPermissionV2(sampleProject.value, "bb.issues.get")) {
    return;
  }
  const issue = await issueStore.fetchIssueByName(
    `${sampleProject.value.name}/issues/${SAMPLE_ISSUE_ID}`,
    {
      // Don't need to fetch the plan and rollout.
      withPlan: false,
      withRollout: false,
    },
    true /* silent */
  );
  return issue;
});

const sampleWorksheet = computedAsync(async () => {
  if (!sampleProject.value) {
    return;
  }
  if (!hasProjectPermissionV2(sampleProject.value, "bb.worksheets.get")) {
    return;
  }
  if (!hasProjectPermissionV2(sampleProject.value, "bb.sql.select")) {
    return;
  }
  const sheet = await worksheetStore.getOrFetchWorksheetByName(
    `${sampleProject.value.name}/sheets/${SAMPLE_SHEET_ID}`,
    true /* silent */
  );
  return sheet;
});

const introList = computed(() => {
  const introList: IntroItem[] = [
    {
      name: computed(() => t("quick-start.view-an-issue")),
      link: {
        name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
        params: {
          projectId: extractProjectResourceName(
            sampleProject.value?.name ?? UNKNOWN_PROJECT_NAME
          ),
          issueSlug: SAMPLE_ISSUE_ID,
        },
      },
      done: computed(() => uiStateStore.getIntroStateByKey("issue.visit")),
      hide: !sampleIssue.value,
    },
    {
      name: computed(() => t("quick-start.query-data")),
      link: {
        name: SQL_EDITOR_WORKSHEET_MODULE,
        params: {
          project: SAMPLE_PROJECT_NAME,
          sheet: SAMPLE_SHEET_ID,
        },
      },
      done: computed(() => uiStateStore.getIntroStateByKey("data.query")),
      hide: !sampleWorksheet.value,
    },
    {
      name: computed(() => t("quick-start.visit-project")),
      link: {
        name: PROJECT_V1_ROUTE_DASHBOARD,
      },
      done: computed(() => uiStateStore.getIntroStateByKey("project.visit")),
      requiredPermissions: ["bb.projects.list"],
    },
    {
      name: computed(() => t("quick-start.visit-environment")),
      link: {
        name: ENVIRONMENT_V1_ROUTE_DASHBOARD,
      },
      done: computed(() =>
        uiStateStore.getIntroStateByKey("environment.visit")
      ),
      requiredPermissions: ["bb.settings.get"],
    },
    {
      name: computed(() => t("quick-start.visit-instance")),
      link: {
        name: INSTANCE_ROUTE_DASHBOARD,
      },
      done: computed(() => uiStateStore.getIntroStateByKey("instance.visit")),
      requiredPermissions: ["bb.instances.list"],
    },
    {
      name: computed(() => t("quick-start.visit-database")),
      link: {
        name: DATABASE_ROUTE_DASHBOARD,
      },
      done: computed(() => uiStateStore.getIntroStateByKey("database.visit")),
      requiredPermissions: ["bb.databases.list"],
    },
    {
      name: computed(() => t("quick-start.visit-member")),
      link: {
        name: WORKSPACE_ROUTE_USERS,
      },
      done: computed(() => uiStateStore.getIntroStateByKey("member.visit")),
    },
  ];

  return introList.filter(
    (item) =>
      !item.hide &&
      (item.requiredPermissions ?? []).every((permission) =>
        hasWorkspacePermissionV2(permission)
      )
  );
});

const showQuickstart = computed(() => {
  if (!actuatorStore.quickStartEnabled) {
    return false;
  }
  return !uiStateStore.getIntroStateByKey("hidden");
});

const currentStep = computed(() => {
  let i = 0;
  const list = introList.value;
  while (i < list.length && list[i].done.value) {
    i++;
  }
  return i;
});

const isTaskActive = (index: number): boolean => {
  for (let i = index - 1; i >= 0; i--) {
    if (!introList.value[i].done.value) {
      return false;
    }
  }
  return !introList.value[index].done.value;
};

const progress = computed(() => {
  return {
    current: currentStep.value,
    total: introList.value.length,
  };
});

const percent = computed(() => {
  const { current, total } = progress.value;
  if (current === 0) {
    return "3rem";
  }
  if (current === total - 1) {
    return "calc(100% - 3rem)";
  }

  const offset = 0.5;
  const unit = 100 / total;
  const percent = Math.min((current + offset) * unit, 100);
  return `${percent}%`;
});

const hideQuickstart = (silent = false) => {
  uiStateStore
    .saveIntroStateByKey({
      key: "hidden",
      newState: true,
    })
    .then(() => {
      if (!silent) {
        pushNotification({
          module: "bytebase",
          style: "INFO",
          title: t("quick-start.notice.title"),
          description: t("quick-start.notice.desc"),
          manualHide: true,
        });
      }
    });
};
</script>
