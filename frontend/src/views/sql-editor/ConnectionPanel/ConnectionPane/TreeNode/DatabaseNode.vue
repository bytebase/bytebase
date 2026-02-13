<template>
  <div class="flex items-center max-w-full overflow-hidden gap-x-1">
    <NTooltip v-if="tabStore.supportBatchMode" :disabled="!checkTooltip" :placement="'bottom-start'">
      <template #trigger>
        <NCheckbox
          class="mr-2"
          :checked="checked"
          :disabled="checkDisabled"
          @click.stop.prevent=""
          @update:checked="$emit('update:checked', $event)"
        />
      </template>
      {{ checkTooltip }}
    </NTooltip>

    <RichDatabaseName
      class="cursor-pointer tree-node-database"
      :database="database"
      :show-instance="true"
      :show-engine-icon="true"
      :show-environment="false"
      :show-arrow="true"
      :keyword="keyword"
      @click.stop.prevent="$emit('click')"
    />

    <RequestQueryButton
      v-if="!canQuery"
      class="ml-auto"
      :text="true"
      :prefer-jit="false"
      :permission-denied-detail="create(PermissionDeniedDetailSchema, {
        resources: [database.name],
        requiredPermissions: ['bb.sql.select']
      })"
      :size="'tiny'"
    />
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { NCheckbox, NTooltip } from "naive-ui";
import { computed } from "vue";
import { RichDatabaseName } from "@/components/v2";
import { useSQLEditorTabStore } from "@/store";
import type { SQLEditorTreeNode as TreeNode } from "@/types";
import { PermissionDeniedDetailSchema } from "@/types/proto-es/v1/common_pb";
import { isDatabaseV1Queryable } from "@/utils";
import RequestQueryButton from "../../../EditorCommon/ResultView/RequestQueryButton.vue";

const props = defineProps<{
  node: TreeNode;
  keyword: string;
  checked?: boolean;
  checkDisabled?: boolean;
  checkTooltip?: string;
}>();

defineEmits<{
  (event: "click"): void;
  (event: "update:checked", checked: boolean): void;
}>();

const tabStore = useSQLEditorTabStore();

const database = computed(
  () => (props.node as TreeNode<"database">).meta.target
);

const canQuery = computed(() => isDatabaseV1Queryable(database.value));
</script>
