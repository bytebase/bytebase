<template>
  <dd
    class="flex items-center text-sm"
    :class="classes"
    @click.stop.prevent="gotoSQLEditor"
  >
    <span v-if="label" class="mr-1">{{ $t("sql-editor.self") }}</span>
    <heroicons-solid:terminal class="w-5 h-5" />

    <div v-if="showTooltip" class="tooltip whitespace-nowrap">
      {{ $t("sql-editor.self") }}
    </div>
  </dd>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useConnectionTreeStore, useCurrentUserV1 } from "@/store";
import type { Database } from "@/types";
import { ConnectionTreeMode, DEFAULT_PROJECT_ID, UNKNOWN_ID } from "@/types";
import { connectionSlug, hasWorkspacePermissionV1 } from "@/utils";

const props = withDefaults(
  defineProps<{
    database: Database;
    label?: boolean;
    disabled?: boolean;
    tooltip?: boolean;
  }>(),
  {
    label: false,
    disabled: false,
    tooltip: false,
  }
);

const emit = defineEmits<{
  (name: "failed", database: Database): void;
}>();

const currentUserV1 = useCurrentUserV1();

const showTooltip = computed((): boolean => {
  return !props.disabled && props.tooltip;
});

const classes = computed((): string[] => {
  const classes: string[] = [];
  if (showTooltip.value) {
    classes.push("tooltip-wrapper");
  }
  if (props.disabled) {
    classes.push("text-gray-400", "cursor-not-allowed");
  } else {
    classes.push("textlabel", "cursor-pointer", "hover:text-accent");
  }
  return classes;
});

const gotoSQLEditor = () => {
  const { disabled, database } = props;
  if (disabled) {
    return;
  }
  if (
    database.projectId === UNKNOWN_ID ||
    database.projectId === DEFAULT_PROJECT_ID
  ) {
    if (
      !hasWorkspacePermissionV1(
        "bb.permission.workspace.manage-database",
        currentUserV1.value.userRole
      )
    ) {
      // For unassigned databases, only high-privileged users
      // are accessible via SQL Editor.
      emit("failed", props.database);
      return;
    }
    // Set the default sidebar view of SQL Editor to "INSTANCE"
    // since unassigned databases won't be listed in "PROJECT" view.
    useConnectionTreeStore().tree.mode = ConnectionTreeMode.INSTANCE;
  }
  const url = `/sql-editor/${connectionSlug(database.instance, database)}`;
  window.open(url);
};
</script>
