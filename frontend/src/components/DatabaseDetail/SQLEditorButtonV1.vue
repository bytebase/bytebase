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
import { useRouter } from "vue-router";
import { SQL_EDITOR_DATABASE_MODULE } from "@/router/sqlEditor";
import type { ComposedDatabase } from "@/types";
import { DEFAULT_PROJECT_NAME, defaultProject } from "@/types";
import type { VueClass } from "@/utils";
import {
  extractInstanceResourceName,
  extractProjectResourceName,
  hasProjectPermissionV2,
  isSQLEditorRoute,
} from "@/utils";

const props = withDefaults(
  defineProps<{
    database: ComposedDatabase;
    table?: string;
    schema?: string;
    label?: boolean;
    disabled?: boolean;
    tooltip?: boolean;
    class?: VueClass;
  }>(),
  {
    label: false,
    disabled: false,
    tooltip: false,
    class: undefined,
    table: undefined,
    schema: undefined,
  }
);

const emit = defineEmits<{
  (name: "failed", database: ComposedDatabase): void;
}>();

const router = useRouter();

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
    return;
  }

  const database = props.database;
  if (database.project === DEFAULT_PROJECT_NAME) {
    if (!hasProjectPermissionV2(defaultProject(), "bb.sql.select")) {
      // For unassigned databases, only high-privileged users
      // are accessible via SQL Editor.
      emit("failed", database);
      return;
    }
  }

  const route = router.resolve({
    name: SQL_EDITOR_DATABASE_MODULE,
    params: {
      project: extractProjectResourceName(database.project),
      instance: extractInstanceResourceName(database.instance),
      database: database.databaseName,
    },
    query: {
      table: props.table ? props.table : undefined,
      schema: props.schema ? props.schema : undefined,
    },
  });
  if (isSQLEditorRoute(router)) {
    router.push(route);
  } else {
    window.open(route.fullPath);
  }
};
</script>
