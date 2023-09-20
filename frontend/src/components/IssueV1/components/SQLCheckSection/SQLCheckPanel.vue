<template>
  <BBModal
    :title="$t('task.check-result.title-general')"
    class="!w-[56rem]"
    header-class="whitespace-pre-wrap break-all gap-x-1"
    container-class="!pt-0 -mt-px"
    @close="$emit('close')"
  >
    <SQLCheckDetail :advices="advices" />
  </BBModal>
</template>

<script setup lang="ts">
import { Advice } from "@/types/proto/v1/sql_service";
import SQLCheckDetail from "./SQLCheckDetail.vue";

defineProps<{
  advices: Advice[];
}>();

defineEmits<{
  (event: "close"): void;
}>();
</script>
