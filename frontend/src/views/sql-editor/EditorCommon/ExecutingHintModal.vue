<template>
  <BBModal
    v-if="sqlEditorStore.isShowExecutingHint"
    :title="$t('common.tips')"
    @close="handleClose"
  >
    <ExecuteHint @close="handleClose" />
  </BBModal>
</template>

<script lang="ts" setup>
import { useSQLEditorStore } from "@/store";
import ExecuteHint from "./ExecuteHint.vue";

const sqlEditorStore = useSQLEditorStore();

const handleClose = () => {
  sqlEditorStore.setSQLEditorState({
    isShowExecutingHint: false,
  });
};
</script>
