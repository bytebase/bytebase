<template>
  <BBDialog
    ref="resourceOccupiedModalRef"
    :title="$t('common.warning')"
    :closable="true"
    :show-positive-btn="resources.length === 0"
    :positive-text="resources.length === 0 ? $t('common.continue-anyway') : ''"
    :type="'warning'"
    @on-positive-click="() => $emit('on-submit')"
    @on-negative-click="() => $emit('on-close')"
  >
    <template #default>
      <div
        class="w-[30rem] max-w-full pt-2 pb-4 text-control break-words text-sm"
      >
        <div v-if="resources.length === 0">
          {{ $t("resource.delete-warning", { name: target }) }}
        </div>
        <div v-else class="space-y-2">
          <p>
            {{
              $t("resource.delete-warning-with-resources", {
                name: target,
              })
            }}
          </p>
          <ul class="list-disc">
            <Resource
              v-for="(resource, i) in resources"
              :key="i"
              :show-prefix="true"
              :link="true"
              :resource="resource"
            />
          </ul>
          <p>{{ $t("resource.delete-warning-retry") }}</p>
        </div>
      </div>
    </template>
  </BBDialog>
</template>

<script lang="tsx" setup>
import { ref } from "vue";
import { BBDialog } from "@/bbkit";
import Resource from "./Resource.vue";

defineProps<{
  target: string;
  resources: string[];
}>();

defineEmits<{
  (event: "on-submit"): void;
  (event: "on-close"): void;
}>();

const resourceOccupiedModalRef = ref<InstanceType<typeof BBDialog>>();

defineExpose({ open: () => resourceOccupiedModalRef.value?.open() });
</script>
