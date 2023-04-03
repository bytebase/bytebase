<template>
  <button
    class="select-none inline-flex items-center space-x-1 border border-control-border rounded-md text-control bg-white hover:bg-control-bg-hover disabled:bg-white disabled:opacity-50 disabled:cursor-not-allowed px-4 py-2 text-sm leading-5 font-medium focus:ring-control focus:outline-none focus-visible:ring-2 focus:ring-offset-2"
    :disabled="disabled"
    @click="handleClick"
  >
    <slot v-if="!uploading" name="icon">
      <heroicons-outline:arrow-up-tray class="w-4 h-auto mr-1" />
    </slot>
    <template v-else>
      <BBProgressPie
        v-if="percent > 0"
        :percent="percent"
        class="w-5 h-5 text-info"
      >
        <template #default="{ percent: displayPercent }">
          <span class="scale-[66%]">{{ displayPercent }}</span>
        </template>
      </BBProgressPie>
      <BBSpin v-else class="w-5 h-5" />
    </template>

    <span>
      <slot v-if="!uploading" />
      <slot v-else name="uploading">
        {{ $t("common.uploading") }}
      </slot>
    </span>

    <input
      ref="inputRef"
      type="file"
      accept=".sql,.txt,application/sql,text/plain"
      class="hidden"
      @change="handleUpload"
    />
  </button>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import { BBProgressPie } from "@/bbkit";

type Tick = (p: number) => void;

const props = defineProps<{
  upload: (e: Event, tick: Tick) => Promise<any>;
  disabled?: boolean;
}>();

const inputRef = ref<HTMLInputElement>();
const uploading = ref(false);
const percent = ref(-1); // -1 to show a simple spinner instead of progress

const disabled = computed(() => {
  return props.disabled || uploading.value;
});

const updatePercent = (p: number) => {
  percent.value = p;
};

const handleClick = () => {
  inputRef.value?.click();
};

const handleUpload = async (e: Event) => {
  uploading.value = true;
  percent.value = -1;
  try {
    await props.upload(e, updatePercent);
  } finally {
    uploading.value = false;
    percent.value = -1;
  }
};
</script>
