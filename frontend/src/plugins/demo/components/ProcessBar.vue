<template>
  <div
    v-show="!showRequestDemoDialog"
    class="process-bar-container"
    :class="isCompleted ? 'completed' : ''"
  >
    <span
      v-if="isCompleted"
      class="absolute right-3 top-3 p-px rounded cursor-pointer hover:bg-gray-100 hover:shadow"
      @click="handleCloseButtonClick"
    >
      <heroicons-outline:x class="w-5 h-auto" />
    </span>
    <div
      v-if="isCompleted"
      class="w-full flex flex-col justify-start items-start my-2"
    >
      <p class="text-gray-800 w-full text-2xl leading-10">
        <span class="text-3xl mr-2">🎉</span> Demo completed
      </p>
    </div>
    <div
      v-else-if="currentProcessData"
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
        :class="actionButtonFlag ? 'cursor-not-allowed !opacity-80' : ''"
        @click="handleBackToDemoButtonClick"
        >👉 Get back to Demo</span
      >
    </div>

    <div
      class="my-4 bg-gray-200 w-full flex flex-row justify-start items-center rounded-full overflow-hidden"
    >
      <div
        class="h-1 bg-indigo-600 rounded-full transition-all duration-500"
        :style="{
          width: processBarWidth,
        }"
      ></div>
    </div>
    <div
      class="relative w-full grid text-base text-gray-400"
      :class="`grid-cols-${processDataList.length}`"
    >
      <div
        v-for="(processData, index) in processDataList"
        :key="index"
        class="select-none flex flex-col justify-start items-center first:items-start last:items-end"
        :class="`${index <= currentProcessIndex ? 'text-indigo-600' : ''} ${
          index <= currentProcessIndex + 1 && !isCompleted
            ? 'cursor-pointer'
            : ''
        }`"
        @click="handleProcessItemClick(processData)"
      >
        <span
          class="absolute -top-6 block w-3 h-3 rounded-full bg-gray-200"
          :class="`${index <= currentProcessIndex ? 'bg-indigo-600' : ''}`"
        ></span>
        <p>{{ processData.title }}</p>
      </div>
    </div>

    <div
      v-if="isCompleted"
      class="w-full mt-6 flex flex-col justify-start items-start"
    >
      <div class="w-full flex flex-row justify-between items-center">
        <button
          class="w-auto flex flex-row justify-center items-center border px-3 leading-10 select-none rounded-md hover:opacity-60"
          :class="actionButtonFlag ? 'cursor-wait !opacity-80' : ''"
          @click="handleRestartButtonClick"
        >
          <heroicons-outline:refresh class="w-5 h-auto mr-2" /> Replay
        </button>
        <button
          class="w-auto flex flex-row justify-center items-center border px-3 leading-10 select-none rounded-md hover:opacity-60"
          @click="handleDeployNowButtonClick"
        >
          <img src="../../../assets/deploy.svg" class="w-5 h-auto mr-2" />
          Deploy now
        </button>
        <button
          class="w-auto flex flex-row justify-center items-center px-3 leading-10 select-none rounded-md font-medium bg-indigo-600 text-white shadow hover:opacity-80"
          @click="showRequestDemoDialog = true"
        >
          <heroicons-outline:chat class="w-5 h-auto mr-2" /> Request full demo
        </button>
      </div>

      <hr class="w-full bg-gray-50 mt-6" />

      <div class="mt-6 w-full grid grid-cols-3 leading-9">
        <p class="col-span-1">More quick demos:</p>
        <div class="w-full col-span-2 grid grid-cols-2 leading-9">
          <a
            class="text-indigo-600 w-full flex flex-row justify-start items-center hover:underline"
            target="_blank"
            href="#placeholder"
            >VCS Integration<heroicons-outline:arrow-right
              class="w-4 h-auto ml-2"
          /></a>
          <a
            class="text-indigo-600 w-full flex flex-row justify-start items-center hover:underline"
            target="_blank"
            href="#placeholder"
            >SQL Editor<heroicons-outline:arrow-right class="w-4 h-auto ml-2"
          /></a>
          <a
            class="text-indigo-600 w-full flex flex-row justify-start items-center hover:underline"
            target="_blank"
            href="#placeholder"
            >SQL Review<heroicons-outline:arrow-right class="w-4 h-auto ml-2"
          /></a>
        </div>
      </div>
    </div>
    <div
      v-else
      v-show="
        currentProcessIndex >= 0 && currentProcessIndex < processDataList.length
      "
      class="w-full flex flex-row justify-between items-center mt-2"
    >
      <span
        class="border px-3 py-1 mt-2 rounded-lg text-indigo-600 bg-indigo-50 cursor-pointer select-none hover:opacity-80"
        :class="`${
          currentProcessIndex === 0 ? '!opacity-40 !cursor-not-allowed' : ''
        } ${actionButtonFlag ? '!cursor-wait !opacity-80' : ''}`"
        @click="handlePrevButtonClick"
        >👈 Prev</span
      >
      <span
        class="border px-3 py-1 mt-2 rounded-lg text-indigo-600 bg-indigo-50 cursor-pointer select-none hover:opacity-80"
        :class="actionButtonFlag ? '!cursor-wait !opacity-80' : ''"
        @click="handleNextButtonClick"
        >{{
          currentProcessIndex < processDataList.length - 1
            ? "Next 👉"
            : "Complete"
        }}</span
      >
    </div>
  </div>

  <RequestDemoDialog
    v-show="showRequestDemoDialog"
    @close="handleRequestDemoDialogClose"
  />
</template>

<script lang="ts" setup>
import { first, indexOf } from "lodash-es";
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import useAppStore from "../store";
import { ProcessData } from "../types";
import RequestDemoDialog from "./RequestDemoDialog.vue";

const emit = defineEmits(["close"]);

const route = useRoute();
const router = useRouter();

const store = useAppStore();
const processDataList = computed(() => store.processDataList);

const isCompleted = ref<boolean>(false);
const actionButtonFlag = ref(false);
const currentProcessIndex = ref<number>(-1);
const showRequestDemoDialog = ref(false);

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
    actionButtonFlag.value = false;
    let matchedProcessDataIndex = -1;
    for (let i = 0; i < processDataList.value.length; i++) {
      const processData = processDataList.value[i];
      if (window.location.href.includes(processData.url)) {
        matchedProcessDataIndex = i;
        break;
      }
    }
    currentProcessIndex.value = matchedProcessDataIndex;
  },
  {
    immediate: true,
  }
);

const handleProcessItemClick = (processData: ProcessData) => {
  if (isCompleted.value) {
    return;
  }

  actionButtonFlag.value = true;
  const index = indexOf(processDataList.value, processData);
  if (currentProcessIndex.value + 1 >= index) {
    router.push(processData.url);
  }
};

const handlePrevButtonClick = () => {
  const process = processDataList.value[currentProcessIndex.value - 1];
  if (process) {
    actionButtonFlag.value = true;
    router.push(process.url);
  }
};

const handleNextButtonClick = async () => {
  if (currentProcessIndex.value === processDataList.value.length - 1) {
    isCompleted.value = true;
    return;
  }

  const process = processDataList.value[currentProcessIndex.value + 1];
  if (process) {
    actionButtonFlag.value = true;
    await router.push(process.url);
  }
};

const handleCloseButtonClick = () => {
  emit("close");
};

const handleRestartButtonClick = async () => {
  const process = first(processDataList.value);
  if (process) {
    actionButtonFlag.value = true;
    await router.push(process.url);
  }
};

const handleDeployNowButtonClick = () => {
  window.open(
    "https://www.bytebase.com/docs/get-started/install/overview",
    "_blank"
  );
};

const handleBackToDemoButtonClick = async () => {
  const process = first(processDataList.value);
  if (process) {
    actionButtonFlag.value = true;
    await router.push(process.url);
  }
};

const handleRequestDemoDialogClose = () => {
  showRequestDemoDialog.value = false;
};

watch(currentProcessIndex, () => {
  if (currentProcessIndex.value === processDataList.value.length - 1) {
    // do nth
  } else {
    isCompleted.value = false;
  }
});
</script>

<style scoped>
.process-bar-container {
  @apply max-w-full fixed bottom-4 left-1/2 -translate-x-1/2 translate-y-0 flex flex-col justify-start items-center bg-white px-8 py-6 rounded-lg;
  min-width: 600px;
  z-index: 10001;
  box-shadow: 0 0 24px 8px rgb(0 0 0 / 20%);
  transition: bottom ease 0.4s;
}

.completed {
  @apply bottom-1/2 translate-y-1/2;
}
</style>
