<template>
  <!-- eslint-disable-next-line vue/no-v-html -->
  <span :id="id" class="truncate" :class="classes" v-html="html" />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { escape } from "lodash-es";

import { ConnectionAtom, DEFAULT_PROJECT_ID } from "@/types";
import { useConnectionTreeStore } from "@/store";
import { getHighlightHTMLByKeyWords } from "@/utils";

const props = defineProps<{
  atom: ConnectionAtom;
  keyword: string;
}>();
const { t } = useI18n();
const connectionTreeStore = useConnectionTreeStore();

const classes = computed(() => {
  const classes = ["truncate"];
  const { atom } = props;
  if (atom.type === "table") {
    if (atom === connectionTreeStore.selectedTableAtom) {
      classes.push("font-bold");
    }
  }
  return classes;
});

// render an unique id for every node
// for auto scroll to the node when tab switches
const id = computed(() => {
  const { atom } = props;
  return `tree-node-label-${atom.type}-${atom.id}`;
});

const text = computed(() => {
  const { atom } = props;
  if (atom.type === "project" && atom.id === DEFAULT_PROJECT_ID) {
    return t("database.unassigned-databases");
  }
  return atom.label;
});

const html = computed(() => {
  return getHighlightHTMLByKeyWords(escape(text.value), escape(props.keyword));
});
</script>
