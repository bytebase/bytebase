<template>
  <div
    class="group grow-0 w-full flex flex-row justify-start items-center gap-2"
  >
    <p
      class="overflow-hidden"
      style="display: -webkit-box; -webkit-box-orient: vertical"
      :style="[
        {
          '-webkit-line-clamp': lines,
        },
        contentStyle,
      ]"
      :class="contentClass"
    >
      <code class="text-sm break-all">{{ sql }}</code>
    </p>
    <div class="hidden group-hover:block shrink-0">
      <NButton :size="'tiny'" @click="showModal = true">
        <template #icon>
          <Maximize2Icon />
        </template>
      </NButton>
    </div>
  </div>

  <BBModal
    :show="showModal"
    :title="$t('common.view-details')"
    @close="showModal = false"
  >
    <div style="width: calc(100vw - 12rem); height: calc(100vh - 12rem)">
      <MonacoEditor
        :content="sql"
        :readonly="true"
        language="sql"
        class="border w-full h-full grow"
      />
    </div>
  </BBModal>
</template>

<script setup lang="ts">
import { Maximize2Icon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { ref } from "vue";
import { BBModal } from "@/bbkit";
import { MonacoEditor } from "@/components/MonacoEditor";
import type { VueClass, VueStyle } from "@/utils";

withDefaults(
  defineProps<{
    sql: string;
    lines?: number;
    contentClass?: VueClass;
    contentStyle?: VueStyle;
  }>(),
  {
    lines: 2,
    contentClass: undefined,
    contentStyle: undefined,
  }
);

const showModal = ref(false);
</script>
