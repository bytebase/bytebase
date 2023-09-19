<template>
  <NTooltip :disabled="!showTooltip">
    <template #trigger>
      <div
        class="flex items-center text-sm"
        :class="classes"
        v-bind="$attrs"
        @click.stop.prevent="gotoSQLEditor"
      >
        <span v-if="label" class="mr-1">{{ $t("sql-editor.self") }}</span>
        <heroicons-solid:terminal class="w-5 h-5" />
      </div>
    </template>

    <div class="whitespace-nowrap">
      {{ $t("sql-editor.self") }}
    </div>
  </NTooltip>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { useConnectionTreeStore, useCurrentUserV1 } from "@/store";
import {
  ComposedDatabase,
  ConnectionTreeMode,
  DEFAULT_PROJECT_V1_NAME,
  UNKNOWN_PROJECT_NAME,
} from "@/types";
import { connectionV1Slug, hasWorkspacePermissionV1 } from "@/utils";

const props = withDefaults(
  defineProps<{
    database?: ComposedDatabase;
    label?: boolean;
    disabled?: boolean;
    tooltip?: boolean;
  }>(),
  {
    database: undefined,
    label: false,
    disabled: false,
    tooltip: false,
  }
);

const emit = defineEmits<{
  (name: "failed", database: ComposedDatabase): void;
}>();

const currentUserV1 = useCurrentUserV1();

const disabled = computed(() => props.disabled || !props.database);

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
  if (disabled.value) {
    return;
  }

  const database = props.database as ComposedDatabase;
  if (
    database.project === DEFAULT_PROJECT_V1_NAME ||
    database.project === UNKNOWN_PROJECT_NAME
  ) {
    if (
      !hasWorkspacePermissionV1(
        "bb.permission.workspace.manage-database",
        currentUserV1.value.userRole
      )
    ) {
      // For unassigned databases, only high-privileged users
      // are accessible via SQL Editor.
      emit("failed", database);
      return;
    }
    // Set the default sidebar view of SQL Editor to "INSTANCE"
    // since unassigned databases won't be listed in "PROJECT" view.
    useConnectionTreeStore().tree.mode = ConnectionTreeMode.INSTANCE;
  }
  const url = `/sql-editor/${connectionV1Slug(
    database.instanceEntity,
    database
  )}`;
  window.open(url);
};
</script>
