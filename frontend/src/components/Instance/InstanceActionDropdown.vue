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
import { NButton, NCheckbox, NDropdown, useDialog } from "naive-ui";
import { computed, h, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { INSTANCE_ROUTE_DASHBOARD } from "@/router/dashboard/workspaceRoutes";
import { pushNotification, useInstanceV1Store } from "@/store";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Instance } from "@/types/proto-es/v1/instance_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

const props = defineProps<{
  instance: Instance;
}>();

const emit = defineEmits<{
  (e: "deleted"): void;
}>();

const { t } = useI18n();
const dialog = useDialog();
const instanceStore = useInstanceV1Store();
const router = useRouter();

const force = ref(false);

const canArchive = computed(() =>
  hasWorkspacePermissionV2("bb.instances.delete")
);
const canRestore = computed(() =>
  hasWorkspacePermissionV2("bb.instances.undelete")
);

const dropdownOptions = computed((): DropdownOption[] => {
  const options: DropdownOption[] = [];

  if (props.instance.state === State.ACTIVE && canArchive.value) {
    options.push({
      key: "archive",
      label: t("common.archive"),
    });
  } else if (props.instance.state === State.DELETED && canRestore.value) {
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
  force.value = false;
  dialog.warning({
    title: t("instance.archive-instance-instance-name", [props.instance.title]),
    content: () =>
      h("div", [
        h(
          "div",
          { class: "mb-3" },
          t("instance.archived-instances-will-not-be-displayed")
        ),
        h(
          NCheckbox,
          {
            checked: force.value,
            "onUpdate:checked": (value: boolean) => {
              force.value = value;
            },
          },
          {
            default: () =>
              h(
                "div",
                { class: "text-sm font-normal text-control-light" },
                t("instance.force-archive-description")
              ),
          }
        ),
      ]),
    negativeText: t("common.cancel"),
    positiveText: t("common.archive"),
    onPositiveClick: async () => {
      await instanceStore.archiveInstance(props.instance, force.value);
      pushNotification({
        module: "bytebase",
        style: "INFO",
        title: t("instance.successfully-archived-instance", [
          props.instance.title,
        ]),
      });
      router.replace({
        name: INSTANCE_ROUTE_DASHBOARD,
      });
    },
  });
};

const handleRestore = () => {
  dialog.info({
    title: t("instance.restore-instance-instance-name-to-normal-state", [
      props.instance.title,
    ]),
    negativeText: t("common.cancel"),
    positiveText: t("instance.restore"),
    onPositiveClick: async () => {
      await instanceStore.restoreInstance(props.instance);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("instance.successfully-restored-instance", [
          props.instance.title,
        ]),
      });
    },
  });
};

const handleDelete = () => {
  dialog.warning({
    title: t("common.delete-resource", {
      resource: props.instance.title,
    }),
    content: t("common.cannot-undo-this-action"),
    negativeText: t("common.cancel"),
    positiveText: t("common.delete"),
    onPositiveClick: async () => {
      await instanceStore.deleteInstance(props.instance.name);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.deleted"),
      });
      emit("deleted");
      router.replace({
        name: INSTANCE_ROUTE_DASHBOARD,
        query: { q: "state:DELETED" },
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
