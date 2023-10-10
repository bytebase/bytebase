<template>
  <div class="flex flex-row items-center justify-between gap-x-4">
    <div class="flex-1 p-0.5 overflow-hidden truncate">
      <TitleEditor />
    </div>
    <div class="flex items-center justify-end gap-x-3">
      <template v-if="!reorderMode">
        <ReorderButton @click="beginReorder" />
        <ExportButton />
        <AddChangeButton />
        <ApplyToDatabaseButton />
      </template>

      <template v-if="reorderMode">
        <NButton @click="cancelReorder">
          {{ $t("common.cancel") }}
        </NButton>
        <NButton type="primary" @click="confirmReorder">
          {{ $t("common.confirm") }}
        </NButton>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { useChangelistDetailContext } from "../context";
import AddChangeButton from "./AddChangeButton.vue";
import ApplyToDatabaseButton from "./ApplyToDatabaseButton.vue";
import ExportButton from "./ExportButton.vue";
import ReorderButton from "./ReorderButton.vue";
import TitleEditor from "./TitleEditor.vue";
import { useReorderChangelist } from "./reorder";

const { reorderMode } = useChangelistDetailContext();

const {
  begin: beginReorder,
  cancel: cancelReorder,
  confirm: confirmReorder,
} = useReorderChangelist();
</script>
