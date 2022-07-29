<template>
  <DemoDrawer v-if="state.showDemoDrawer" @close="handleDemoDrawerClose" />
  <ProcessBar
    v-if="state.showProcessBar"
    @finish.once="handleProcessFinished"
  />
</template>

<script lang="ts" setup>
import { onMounted, reactive, ref } from "vue";
import * as storage from "../storage";
import { DemoData } from "../types";
import { fetchDemoDataWithName } from "../data";
import DemoDrawer from "./DemoDrawer.vue";
import ProcessBar from "./ProcessBar.vue";

interface LocalState {
  showDemoDrawer: boolean;
  showProcessBar: boolean;
}

const state = reactive<LocalState>({
  showDemoDrawer: false,
  showProcessBar: false,
});

const demoDataRef = ref<DemoData>();

onMounted(async () => {
  const { demo } = storage.get(["demo"]);
  if (demo) {
    const demoData = await fetchDemoDataWithName(demo.name);
    if (demoData) {
      demoDataRef.value = demoData;
      state.showProcessBar = true;
    }
  }
});

const handleDemoDrawerClose = () => {
  state.showDemoDrawer = false;
};

const handleProcessFinished = () => {
  state.showDemoDrawer = true;
};
</script>

<style scoped></style>
