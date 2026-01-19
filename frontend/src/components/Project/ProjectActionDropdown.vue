<template>
  <NDropdown
    trigger="click"
    :options="dropdownOptions"
    @select="handleSelect"
  >
    <NButton size="small" quaternary class="px-1!">
      <template #icon>
        <EllipsisVerticalIcon class="w-4 h-4" />
      </template>
    </NButton>
  </NDropdown>
</template>

<script setup lang="ts">
import { EllipsisVerticalIcon } from "lucide-vue-next";
import type { DropdownOption } from "naive-ui";
import { NButton, NDropdown, useDialog } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { PROJECT_V1_ROUTE_DASHBOARD } from "@/router/dashboard/workspaceRoutes";
import {
  pushNotification,
  useGracefulRequest,
  useProjectV1Store,
} from "@/store";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { hasProjectPermissionV2 } from "@/utils";

const props = defineProps<{
  project: Project;
}>();

const emit = defineEmits<{
  (e: "deleted"): void;
}>();

const { t } = useI18n();
const dialog = useDialog();
const projectStore = useProjectV1Store();
const router = useRouter();

const canArchive = computed(() =>
  hasProjectPermissionV2(props.project, "bb.projects.delete")
);
const canRestore = computed(() =>
  hasProjectPermissionV2(props.project, "bb.projects.undelete")
);

const dropdownOptions = computed((): DropdownOption[] => {
  const options: DropdownOption[] = [];

  if (props.project.state === State.ACTIVE && canArchive.value) {
    options.push({
      key: "archive",
      label: t("common.archive"),
    });
  } else if (props.project.state === State.DELETED && canRestore.value) {
    options.push({
      key: "restore",
      label: t("common.restore"),
    });
  }

  if (canArchive.value || canRestore.value) {
    options.push({
      key: "delete",
      label: t("common.delete"),
    });
  }

  return options;
});

const handleArchive = () => {
  dialog.warning({
    title: t("common.confirm-archive"),
    content: t("project.settings.confirm-archive-project", {
      name: props.project.title || props.project.name,
    }),
    negativeText: t("common.cancel"),
    positiveText: t("common.archive"),
    onPositiveClick: () => {
      useGracefulRequest(async () => {
        await projectStore.archiveProject(props.project);
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: `${props.project.title || props.project.name} ${t("common.archived")}`,
        });
        router.push({ name: PROJECT_V1_ROUTE_DASHBOARD });
      });
    },
  });
};

const handleRestore = () => {
  dialog.info({
    title: t("project.settings.restore.title"),
    content:
      t("project.settings.restore.title") +
      ` '${props.project.title || props.project.name}'?`,
    negativeText: t("common.cancel"),
    positiveText: t("common.restore"),
    onPositiveClick: () => {
      useGracefulRequest(async () => {
        await projectStore.restoreProject(props.project);
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: `${props.project.title || props.project.name} ${t("common.restored")}`,
        });
      });
    },
  });
};

const handleDelete = () => {
  dialog.warning({
    title: t("common.confirm-delete"),
    content:
      t("project.settings.confirm-delete-project", {
        name: props.project.title || props.project.name,
      }) +
      "\n\n" +
      t("common.cannot-undo-this-action"),
    negativeText: t("common.cancel"),
    positiveText: t("common.delete"),
    onPositiveClick: () => {
      useGracefulRequest(async () => {
        await projectStore.deleteProject(props.project.name);
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: `${props.project.title || props.project.name} ${t("common.deleted")}`,
        });
        emit("deleted");
        router.push({ name: PROJECT_V1_ROUTE_DASHBOARD });
      });
    },
  });
};

const handleSelect = (key: string) => {
  if (key === "archive") {
    handleArchive();
  } else if (key === "restore") {
    handleRestore();
  } else if (key === "delete") {
    handleDelete();
  }
};
</script>
