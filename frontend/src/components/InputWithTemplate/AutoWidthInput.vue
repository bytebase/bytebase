<template>
  <div>
    <div ref="divRef" class="fixed invisible text-base leading-6">
      {{ state.data }}
    </div>
    <input
      ref="inputRef"
      v-model="state.data"
      :style="`width: ${state.width}px;`"
      :class="className"
      type="text"
      @keyup="(e) => $emit('keyup', e)"
    />
  </div>
</template>

<script lang="ts" setup>
import { watch, reactive, ref, watchEffect, nextTick } from "vue";

const props = defineProps({
  value: {
    default: "",
    type: String,
  },
  maxWidth: {
    default: 0,
    type: Number,
  },
  className: {
    default: "",
    type: String,
  },
});

const emit = defineEmits(["change", "keyup"]);

const state = reactive<{ data: string; width: number }>({
  data: props.value,
  width: 0,
});

const inputRef = ref<HTMLInputElement>();
const divRef = ref<HTMLDivElement>();

watch(
  () => state.data,
  (val) => {
    nextTick(updateWidth);
    emit("change", val);
  }
);

watch(
  () => props.value,
  (val) => {
    state.data = val;
    nextTick(updateWidth);
  }
);

const updateWidth = () => {
  const width = divRef.value?.offsetWidth ?? state.width;
  state.width = Math.min(props.maxWidth ?? width, width);
};

watchEffect(updateWidth);
</script>

<style scoped>
.cleared-input,
.cleared-input:focus {
  @apply shadow-none ring-0 border-0 border-none;
}
</style>
