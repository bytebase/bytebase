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
import { useSQLEditorV2Store } from "@/store";
import ExecuteHint from "./ExecuteHint.vue";

const sqlEditorStore = useSQLEditorV2Store();

const handleClose = () => {
  sqlEditorStore.isShowExecutingHint = false;
};
</script>
