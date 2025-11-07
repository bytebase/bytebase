<template>
  <div
    class="group grow-0 w-full flex flex-row justify-start items-center gap-2"
  >
    <p class="line-clamp-2">
      <code class="text-sm break-all">{{ jsonString }}</code>
    </p>
    <div class="hidden group-hover:block shrink-0 h-[22px]">
      <NButton :size="'tiny'" @click="showModal = true">
        <template #icon>
          <Maximize2Icon />
        </template>
      </NButton>
    </div>
  </div>

  <BBModal
    :show="showModal"
    :title="t('common.view-details')"
    @close="showModal = false"
  >
    <div style="width: calc(100vw - 12rem); height: calc(100vh - 12rem)">
      <MonacoEditor
        :content="formattedJSONString"
        :readonly="true"
        language="json"
        class="border w-full h-full grow"
      />
    </div>
  </BBModal>
</template>

<script setup lang="ts">
import { Maximize2Icon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { BBModal } from "@/bbkit";
import { MonacoEditor } from "@/components/MonacoEditor";

const { t } = useI18n();

const props = defineProps<{
  jsonString: string;
}>();

const showModal = ref(false);

const formattedJSONString = computed(() => {
  try {
    return JSON.stringify(JSON.parse(props.jsonString), null, 2);
  } catch {
    return "-";
  }
});
</script>
