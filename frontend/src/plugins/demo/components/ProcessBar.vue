<template>
  <div class="process-bar-container">
    <div
      v-if="currentProcessData"
      class="w-full flex flex-col justify-start items-start"
    >
      <p class="text-gray-800 w-full leading-7">
        {{ currentProcessData.description }}
      </p>
    </div>
    <div v-else class="w-full flex flex-col justify-start items-start">
      <p class="text-gray-800 w-full leading-7">
        Looks like you've got out of the demo process 🙁
      </p>
      <span
        class="border px-3 py-1 mt-2 rounded-lg text-indigo-600 bg-indigo-50 cursor-pointer hover:opacity-80"
        @click="handleBackToDemoButtonClick"
        >👉 Get back to Demo</span
      >
    </div>
    <div
      class="my-4 bg-gray-200 w-full flex flex-row justify-start items-center rounded-full overflow-hidden"
    >
      <div
        class="h-2 bg-indigo-600 rounded-full transition-all"
        :style="{
          width: processBarWidth,
        }"
      ></div>
    </div>
    <div
      class="w-full grid text-base text-gray-400"
      :class="`grid-cols-${processDataList.length}`"
    >
      <div
        v-for="(processData, index) in processDataList"
        :key="index"
        class="select-none text-center first:text-left last:text-right"
        :class="`${
          index <= currentProcessIndex
            ? 'text-indigo-600 cursor-pointer hover:opacity-80'
            : ''
        } ${
          index <= currentProcessIndex + 1
            ? 'cursor-pointer hover:opacity-80'
            : ''
        }`"
        @click="handleProcessItemClick(processData)"
      >
        {{ processData.title }}
      </div>
    </div>
    <div
      v-show="
        currentProcessIndex >= 0 && currentProcessIndex < processDataList.length
      "
      class="w-full flex flex-row justify-between items-center mt-2"
    >
      <span
        class="border px-3 py-1 mt-2 rounded-lg text-indigo-600 bg-indigo-50 cursor-pointer hover:opacity-80"
        :class="
          currentProcessIndex === 0 ? '!opacity-40 !cursor-not-allowed' : ''
        "
        @click="handlePrevButtonClick"
        >👈 Prev</span
      >
      <span
        class="border px-3 py-1 mt-2 rounded-lg text-indigo-600 bg-indigo-50 cursor-pointer hover:opacity-80"
        :class="
          currentProcessIndex === processDataList.length - 1
            ? '!opacity-40 !cursor-not-allowed'
            : ''
        "
        @click="handleNextButtonClick"
        >Next 👉</span
      >
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { first, indexOf } from "lodash-es";
import { ProcessData } from "../types";
import useAppStore from "../store";

const emit = defineEmits(["finish"]);

const route = useRoute();
const router = useRouter();

const store = useAppStore();
const processDataList = computed(() => store.processDataList);

const currentProcessIndex = ref<number>(-1);
const currentProcessData = computed(() => {
  return currentProcessIndex.value >= 0
    ? processDataList.value[currentProcessIndex.value]
    : undefined;
});
const processBarWidth = computed(() => {
  return (
    Math.max(
      (currentProcessIndex.value / (processDataList.value.length - 1)) * 100,
      2
    ) + "%"
  );
});

watch(
  route,
  () => {
    let matchedProcessDataIndex = -1;
    for (let i = 0; i < processDataList.value.length; i++) {
      const processData = processDataList.value[i];
      if (window.location.href.includes(processData.url)) {
        matchedProcessDataIndex = i;
        break;
      }
    }
    currentProcessIndex.value = matchedProcessDataIndex;
    if (
      matchedProcessDataIndex >= 0 &&
      matchedProcessDataIndex === processDataList.value.length - 1
    ) {
      emit("finish");
    }
  },
  {
    immediate: true,
  }
);

const handleProcessItemClick = (processData: ProcessData) => {
  const index = indexOf(processDataList.value, processData);
  if (currentProcessIndex.value + 1 >= index) {
    router.push(processData.url);
  }
};

const handlePrevButtonClick = () => {
  const process = processDataList.value[currentProcessIndex.value - 1];
  if (process) {
    router.push(process.url);
  }
};

const handleNextButtonClick = () => {
  const process = processDataList.value[currentProcessIndex.value + 1];
  if (process) {
    router.push(process.url);
  }
};

const handleBackToDemoButtonClick = () => {
  const process = first(processDataList.value);
  if (process) {
    router.push(process.url);
  }
};
</script>

<style scoped>
.process-bar-container {
  @apply max-w-full fixed bottom-6 left-1/2 -translate-x-1/2 translate-y-0 transition-transform duration-500 flex flex-col justify-start items-center bg-white px-8 py-6 rounded-lg;
  min-width: 600px;
  z-index: 10001;
  box-shadow: 0 0 24px 8px rgb(0 0 0 / 20%);
}
</style>
