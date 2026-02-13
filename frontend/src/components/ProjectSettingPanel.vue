<template>
  <div class="w-full flex flex-col gap-y-0 pt-4">
    <div class="divide-y divide-block-border">
      <!-- General Settings Section -->
      <div class="pb-6 lg:flex">
        <div class="text-left lg:w-1/4">
          <h1 class="text-2xl font-bold">
            {{ $t("common.general") }}
          </h1>
        </div>
        <div class="flex-1 mt-4 lg:px-4 lg:mt-0">
          <ProjectGeneralSettingPanel
            ref="projectGeneralSettingPanelRef"
            :project="project"
          />
        </div>
      </div>

      <!-- Security Settings Section -->
      <div class="py-6 lg:flex">
        <div class="text-left lg:w-1/4">
          <h1 class="text-2xl font-bold">
            {{ $t("settings.sidebar.security-and-policy") }}
          </h1>
        </div>
        <div class="flex-1 mt-4 lg:px-4 lg:mt-0">
          <ProjectSecuritySettingPanel
            ref="projectSecuritySettingPanelRef"
            :project="project"
          />
        </div>
      </div>

      <div id="issue-related" class="py-6 lg:flex">
        <div class="text-left lg:w-1/4">
          <h1 class="text-2xl font-bold">
            {{ $t("project.settings.issue-related.self") }}
          </h1>
        </div>
        <div class="flex-1 mt-4 lg:px-4 lg:mt-0">
          <PermissionGuardWrapper
            v-slot="slotProps"
            :project="project"
            :permissions="[
              'bb.projects.update'
            ]"
          >
            <ProjectIssueRelatedSettingPanel
              ref="projectIssueRelatedSettingPanelRef"
              :project="project"
              :allow-edit="!slotProps.disabled"
            />
          </PermissionGuardWrapper>
        </div>
      </div>

      <div class="py-6 lg:flex">
        <div class="text-left lg:w-1/4">
          <div class="flex items-center gap-x-2">
            <h1 class="text-2xl font-bold">
              {{ $t("common.danger-zone") }}
            </h1>
          </div>
        </div>
        <div class="flex-1 mt-4 lg:px-4 lg:mt-0">
          <div class="border border-error-alpha bg-error-alpha rounded-lg divide-y divide-error-alpha">
            <!-- Archive/Restore Section -->
            <div class="p-6 flex items-start justify-between gap-x-6">
              <template v-if="project.state === State.ACTIVE">
                <div class="flex-1">
                  <h4 class="font-medium text-main">
                    {{ $t('common.archive-resource', { type: $t('common.project') }) }}
                  </h4>
                  <p class="text-sm text-control-light mt-1">
                    {{ $t('common.archive-description', { name: project.title || project.name }) }}
                  </p>
                </div>
                <BBButtonConfirm
                  type="ARCHIVE"
                  :button-text="$t('common.archive')"
                  :confirm-title="$t('common.confirm-archive')"
                  :confirm-description="
                    $t('project.settings.confirm-archive-project', { name: project.title || project.name })
                  "
                  :require-confirm="true"
                  :disabled="dangerZoneState.isExecuting"
                  @confirm="handleArchive"
                />
              </template>
              <template v-else-if="project.state === State.DELETED">
                <div class="flex-1">
                  <h4 class="font-medium text-main">
                    {{ $t('project.settings.restore.title') }}
                  </h4>
                  <p class="text-sm text-control-light mt-1">
                    {{ $t('project.settings.restore.btn-text') }}
                  </p>
                </div>
                <BBButtonConfirm
                  type="RESTORE"
                  :button-text="$t('common.restore')"
                  :confirm-title="$t('project.settings.restore.title')"
                  :confirm-description="
                    $t('project.settings.restore.title') + ` '${project.title || project.name}'?`
                  "
                  :require-confirm="true"
                  :disabled="dangerZoneState.isExecuting"
                  @confirm="handleRestore"
                />
              </template>
            </div>

            <!-- Delete Section -->
            <div class="p-6 flex items-start justify-between gap-x-6">
              <div class="flex-1">
                <h4 class="font-medium text-error">
                  {{ $t('common.delete-resource', { type: $t('common.project') }) }}
                </h4>
                <p class="text-sm text-control-light mt-1">
                  {{ $t('common.delete-resource-description', { name: project.title || project.name }) }}
                </p>
              </div>
              <BBButtonConfirm
                type="DELETE"
                :button-text="$t('common.delete')"
                :confirm-title="$t('common.confirm-delete')"
                :confirm-description="
                  $t('project.settings.confirm-delete-project', { name: project.title || project.name })
                "
                :require-confirm="true"
                :disabled="dangerZoneState.isExecuting"
                @confirm="handleDelete"
              />
            </div>
          </div>
        </div>
      </div>

      <!-- Save/Cancel buttons -->
      <div v-if="allowEdit && isDirty" class="sticky bottom-0 z-10">
        <div
          class="flex justify-between w-full py-4 border-t border-block-border bg-white"
        >
          <NButton @click.prevent="onRevert">
            {{ $t("common.cancel") }}
          </NButton>
          <NButton type="primary" @click.prevent="onUpdate">
            {{ $t("common.confirm-and-update") }}
          </NButton>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBButtonConfirm } from "@/bbkit";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import { useRouteChangeGuard } from "@/composables/useRouteChangeGuard";
import { PROJECT_V1_ROUTE_DASHBOARD } from "@/router/dashboard/workspaceRoutes";
import {
  pushNotification,
  useGracefulRequest,
  useProjectV1Store,
} from "@/store";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  ProjectGeneralSettingPanel,
  ProjectIssueRelatedSettingPanel,
  ProjectSecuritySettingPanel,
} from "./Project/Settings/";

const props = defineProps<{
  project: Project;
  allowEdit: boolean;
}>();

const { t } = useI18n();
const router = useRouter();
const projectStore = useProjectV1Store();

const projectSecuritySettingPanelRef =
  ref<InstanceType<typeof ProjectSecuritySettingPanel>>();
const projectGeneralSettingPanelRef =
  ref<InstanceType<typeof ProjectGeneralSettingPanel>>();
const projectIssueRelatedSettingPanelRef =
  ref<InstanceType<typeof ProjectIssueRelatedSettingPanel>>();

const settingRefList = computed(() => {
  return [
    projectGeneralSettingPanelRef,
    projectSecuritySettingPanelRef,
    projectIssueRelatedSettingPanelRef,
  ];
});

const isDirty = computed(() => {
  return settingRefList.value.some((settingRef) => settingRef.value?.isDirty);
});

useRouteChangeGuard(isDirty);

const onUpdate = async () => {
  for (const settingRef of settingRefList.value) {
    if (!settingRef.value?.isDirty) {
      continue;
    }
    try {
      await settingRef.value.update();
    } catch {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("project.settings.update-failed"),
      });
      return;
    }
  }
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("project.settings.success-updated"),
  });
};

const onRevert = () => {
  for (const settingRef of settingRefList.value) {
    settingRef.value?.revert();
  }
};

const dangerZoneState = reactive({
  isExecuting: false,
});

const handleArchive = () => {
  dangerZoneState.isExecuting = true;
  useGracefulRequest(async () => {
    try {
      await projectStore.archiveProject(props.project);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: `${props.project.title || props.project.name} ${t("common.archived")}`,
      });
      router.push({ name: PROJECT_V1_ROUTE_DASHBOARD });
    } finally {
      dangerZoneState.isExecuting = false;
    }
  });
};

const handleRestore = () => {
  dangerZoneState.isExecuting = true;
  useGracefulRequest(async () => {
    try {
      await projectStore.restoreProject(props.project);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: `${props.project.title || props.project.name} ${t("common.restored")}`,
      });
      router.push({ name: PROJECT_V1_ROUTE_DASHBOARD });
    } finally {
      dangerZoneState.isExecuting = false;
    }
  });
};

const handleDelete = () => {
  dangerZoneState.isExecuting = true;
  useGracefulRequest(async () => {
    try {
      await projectStore.deleteProject(props.project.name);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: `${props.project.title || props.project.name} ${t("common.deleted")}`,
      });
      router.push({ name: PROJECT_V1_ROUTE_DASHBOARD });
    } finally {
      dangerZoneState.isExecuting = false;
    }
  });
};
</script>
