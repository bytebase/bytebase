<template>
  <!-- eslint-disable-next-line vue/no-v-html -->
  <span :id="id" class="truncate" v-html="html" />
</template>

<script lang="ts" setup>
import { escape } from "lodash-es";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { ConnectionAtom, DEFAULT_PROJECT_ID } from "@/types";
import { getHighlightHTMLByRegExp } from "@/utils";

const props = defineProps<{
  atom: ConnectionAtom;
  keyword: string;
}>();
const { t } = useI18n();

// render an unique id for every node
// for auto scroll to the node when tab switches
const id = computed(() => {
  const { atom } = props;
  return `tree-node-label-${atom.type}-${atom.id}`;
});

const text = computed(() => {
  const { atom } = props;
  if (atom.type === "project" && atom.id === String(DEFAULT_PROJECT_ID)) {
    return t("database.unassigned-databases");
  }
  return atom.label;
});

const html = computed(() => {
  return getHighlightHTMLByRegExp(
    escape(text.value),
    escape(props.keyword.trim()),
    false /* !caseSensitive */
  );
});
</script>
