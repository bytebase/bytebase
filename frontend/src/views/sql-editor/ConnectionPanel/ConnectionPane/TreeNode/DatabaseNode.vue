<template>
  <div class="flex items-center max-w-full overflow-hidden gap-x-1">
    <template v-if="tabStore.supportBatchMode">
      <LinkIcon
        v-if="tabStore.currentTab?.connection.database === database.name"
        class="w-4 textinfolabel"
      />
      <NCheckbox
        v-else-if="canQuery"
        :checked="checked"
        @click.stop.prevent=""
        @update:checked="$emit('update:checked', $event)"
      />
    </template>

    <RichDatabaseName
      :database="database"
      :show-instance="true"
      :show-engine-icon="true"
      :show-environment="false"
      :show-arrow="true"
      :keyword="keyword"
    />

    <span v-if="connected" class="truncate textinfolabel">
      ({{ $t("sql-editor.connected") }})
    </span>
    <RequestQueryButton
      v-if="showRequestQueryButton"
      :database-resource="{
        databaseFullName: database.name,
      }"
      :size="'tiny'"
    />
  </div>
</template>

<script setup lang="ts">
import { LinkIcon } from "lucide-vue-next";
import { NCheckbox } from "naive-ui";
import { computed } from "vue";
import { RichDatabaseName } from "@/components/v2";
import { useSQLEditorTabStore } from "@/store";
import type { SQLEditorTreeNode as TreeNode } from "@/types";
import { isDatabaseV1Queryable } from "@/utils";
import RequestQueryButton from "../../../EditorCommon/ResultView/RequestQueryButton.vue";

const props = defineProps<{
  node: TreeNode;
  keyword: string;
  connected?: boolean;
  checked?: boolean;
}>();

defineEmits<{
  (event: "update:checked", checked: boolean): void;
}>();

const tabStore = useSQLEditorTabStore();

const database = computed(
  () => (props.node as TreeNode<"database">).meta.target
);

const canQuery = computed(() => isDatabaseV1Queryable(database.value));

const showRequestQueryButton = computed(() => {
  return !canQuery.value;
});
</script>
