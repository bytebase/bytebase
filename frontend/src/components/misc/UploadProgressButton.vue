<template>
  <NButton
    :disabled="disabled"
    tag="div"
    style="--n-icon-margin: 3px"
    @click="handleClick"
  >
    <template #icon>
      <slot v-if="!uploading" name="icon">
        <heroicons-outline:arrow-up-tray class="w-4 h-4" />
      </slot>
      <template v-else>
        <BBProgressPie
          v-if="percent > 0"
          :percent="percent"
          class="w-4 h-4 text-info"
        >
          <template #default="{ percent: displayPercent }">
            <span class="scale-[66%]">{{ displayPercent }}</span>
          </template>
        </BBProgressPie>
        <BBSpin v-else class="w-4 h-4" />
      </template>
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
  </NButton>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
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
  } catch {
    // nothing
  } finally {
    uploading.value = false;
    percent.value = -1;
    if (inputRef.value) {
      // Clear the selected file.
      // Otherwise selecting the same file again will not trigger
      // change event
      inputRef.value.value = "";
    }
  }
};
</script>
