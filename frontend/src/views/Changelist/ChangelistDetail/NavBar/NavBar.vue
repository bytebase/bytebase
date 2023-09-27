<template>
  <div class="flex flex-row items-center justify-between py-0.5">
    <div class="flex items-center justify-start">
      <ProjectV1Name :project="project" />
      <span class="ml-3 mr-0.5">/</span>
      <TitleEditor />
    </div>
    <div class="flex items-center justify-end gap-x-3">
      <template v-if="!reorderMode">
        <NButton
          icon
          style="--n-padding: 0 10px"
          :disabled="!allowEdit"
          @click="beginReorder"
        >
          <template #icon>
            <heroicons:arrows-up-down />
          </template>
        </NButton>
        <NButton icon style="--n-padding: 0 10px" :disabled="!allowEdit">
          <template #icon>
            <heroicons:arrow-down-tray />
          </template>
        </NButton>
        <NButton icon style="--n-padding: 0 10px" :disabled="!allowEdit">
          <template #icon>
            <heroicons:plus />
          </template>
        </NButton>
        <NButton type="primary">
          {{ $t("changelist.apply-to-database") }}
        </NButton>
      </template>

      <template v-if="reorderMode">
        <NButton :disabled="isReorderUpdating" @click="cancelReorder">
          {{ $t("common.cancel") }}
        </NButton>
        <NButton
          type="primary"
          :loading="isReorderUpdating"
          @click="confirmReorder"
        >
          {{ $t("common.confirm") }}
        </NButton>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { ProjectV1Name } from "@/components/v2";
import { useChangelistDetailContext } from "../context";
import TitleEditor from "./TitleEditor.vue";
import { useReorderChangelist } from "./reorder";

const { allowEdit, project, reorderMode } = useChangelistDetailContext();

const {
  updating: isReorderUpdating,
  begin: beginReorder,
  cancel: cancelReorder,
  confirm: confirmReorder,
} = useReorderChangelist();
</script>
