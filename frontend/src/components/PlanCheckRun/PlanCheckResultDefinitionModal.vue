<template>
  <BBModal :title="$t('sql-review.view-definition')" @close="emit('close')">
    <div class="flex flex-col gap-y-4 w-[calc(100vw-5rem)] sm:w-[40rem] pb-1">
      <div class="flex flex-col gap-y-1">
        <div class="flex flex-col gap-x-2 whitespace-pre-wrap">
          {{ definition }}
        </div>
      </div>
      <div class="mt-4 pt-2 border-t flex justify-end">
        <NButton @click.prevent="emit('close')">
          {{ $t("common.close") }}
        </NButton>
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { BBModal } from "@/bbkit";

defineProps<{ definition: string }>();

const emit = defineEmits<{
  (event: "close"): void;
}>();
</script>
