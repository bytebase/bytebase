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

  <RequestQueryPanel
    v-if="state.showRequestQueryPanel"
    :project-id="database?.projectEntity.uid"
    :database="database"
    :redirect-to-issue-page="pageMode === 'BUNDLED'"
    @close="state.showRequestQueryPanel = false"
  />
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { reactive } from "vue";
import RequestQueryPanel from "@/components/Issue/panel/RequestQueryPanel/index.vue";
import { useCurrentUserV1, usePageMode, useSQLEditorTreeStore } from "@/store";
import {
  ComposedDatabase,
  DEFAULT_PROJECT_V1_NAME,
  UNKNOWN_PROJECT_NAME,
} from "@/types";
import { VueClass, connectionV1Slug, hasWorkspacePermissionV1 } from "@/utils";

interface LocalState {
  showRequestQueryPanel: boolean;
}

const props = withDefaults(
  defineProps<{
    database?: ComposedDatabase;
    label?: boolean;
    disabled?: boolean;
    tooltip?: boolean;
    class?: VueClass;
  }>(),
  {
    database: undefined,
    label: false,
    disabled: false,
    tooltip: false,
    class: undefined,
  }
);

const emit = defineEmits<{
  (name: "failed", database: ComposedDatabase): void;
}>();

const currentUserV1 = useCurrentUserV1();
const pageMode = usePageMode();
const state = reactive<LocalState>({
  showRequestQueryPanel: false,
});

const disabled = computed(() => props.disabled || !props.database);

const showTooltip = computed((): boolean => {
  return !props.disabled && props.tooltip;
});

const classes = computed(() => {
  const classes: string[] = [];
  if (props.disabled) {
    classes.push("text-gray-400");
  } else {
    classes.push("textlabel", "cursor-pointer", "hover:text-accent");
  }
  return [...classes, props.class];
});

const gotoSQLEditor = () => {
  if (disabled.value) {
    state.showRequestQueryPanel = true;
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
    useSQLEditorTreeStore().factorList = [
      {
        factor: "instance",
        disabled: false,
      },
    ];
  }
  const url = `/sql-editor/${connectionV1Slug(
    database.instanceEntity,
    database
  )}`;
  window.open(url);
};
</script>
