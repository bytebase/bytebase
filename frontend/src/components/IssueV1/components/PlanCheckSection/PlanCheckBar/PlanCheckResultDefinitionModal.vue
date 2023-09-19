<template>
  <BBModal :title="$t('sql-review.view-definition')" @close="emit('cancel')">
    <div class="space-y-4 w-[calc(100vw-5rem)] sm:w-[40rem] pb-1">
      <div class="space-y-1">
        <div class="flex flex-col gap-x-2 whitespace-pre-wrap">
          {{ definition }}
        </div>
      </div>
      <div class="mt-4 pt-2 border-t flex justify-end">
        <button
          type="button"
          class="btn-normal py-2 px-4"
          @click.prevent="emit('cancel')"
        >
          {{ $t("common.cancel") }}
        </button>
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
defineProps({
  definition: {
    required: true,
    type: String,
  },
});

const emit = defineEmits<{
  (event: "cancel"): void;
}>();
</script>
