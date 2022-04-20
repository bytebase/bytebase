<template>
  <div
    class="divide-y divide-block-border border border-block-border rounded-sm"
  >
    <div class="flex py-2 px-4 justify-between">
      <div class="flex flex-row space-x-2 items-center">
        <h3 class="text-lg leading-6 font-medium text-main">
          {{ review.name }}
        </h3>
      </div>
      <button
        type="button"
        class="btn-normal py-2 px-4"
        @click.prevent="emit('click', review)"
      >
        {{ $t("common.view") }}
      </button>
    </div>
    <div class="border-t border-block-border">
      <dl class="divide-y divide-block-border">
        <div class="grid grid-cols-4 gap-4 px-4 py-2 items-center">
          <dt class="text-sm font-medium text-control-light">
            {{ $t("common.environment") }}
          </dt>
          <dd class="mt-1 flex gap-x-2 text-sm text-main col-span-2">
            <BBBadge
              v-if="review.environmentId"
              :text="envStore.getEnvironmentNameById(review.environmentId)"
              :can-remove="false"
            />
          </dd>
        </div>
        <div class="grid grid-cols-4 gap-4 px-4 py-2 items-center">
          <dt class="text-sm font-medium text-control-light">
            {{ $t("common.created-at") }}
          </dt>
          <dd class="mt-1 flex text-sm text-main col-span-2">
            {{ humanizeTs(review.createdTs) }}
          </dd>
        </div>
        <div class="grid grid-cols-4 gap-4 px-4 py-2 items-center">
          <dt class="text-sm font-medium text-control-light">
            {{ $t("common.updated-at") }}
          </dt>
          <dd class="mt-1 flex text-sm text-main col-span-2">
            {{ humanizeTs(review.updatedTs) }}
          </dd>
        </div>
        <div class="grid grid-cols-4 gap-4 px-4 py-2 items-center">
          <dt class="text-sm font-medium text-control-light">
            {{ $t("common.creator") }}
          </dt>
          <dd class="mt-1 flex text-sm text-main col-span-2">
            {{ review.creator.name }}
          </dd>
        </div>
      </dl>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { PropType } from "vue";
import { useRouter } from "vue-router";
import { DatabaseSchemaReviewPolicy } from "../../../types/schemaSystem";
import { useEnvironmentStore } from "@/store";

const props = defineProps({
  review: {
    required: true,
    type: Object as PropType<DatabaseSchemaReviewPolicy>,
  },
});
const emit = defineEmits(["click"]);

const router = useRouter();
const envStore = useEnvironmentStore();
</script>
