<template>
  <template v-if="!state.isDemoCompleted && state.showDemo">
    <ProcessBar @close.once="handleProcessClose" />
  </template>
</template>

<script lang="ts" setup>
import { onMounted, reactive, watchEffect } from "vue";
import { useRoute } from "vue-router";
import { removeDemo } from "..";
import { fetchDemoDataWithName } from "../data";
import useAppStore from "../store";
import ProcessBar from "./ProcessBar.vue";

interface LocalState {
  isDemoCompleted: boolean;
  showDemo: boolean;
}

const props = defineProps<{
  demoName: string;
}>();

const state = reactive<LocalState>({
  isDemoCompleted: false,
  showDemo: false,
});

const route = useRoute();
const store = useAppStore();

onMounted(async () => {
  try {
    const demoData = await fetchDemoDataWithName(props.demoName);
    if (demoData) {
      store.setState({
        demoName: props.demoName,
        processDataList: demoData.process,
        hintDataList: demoData.hint,
      });
      state.showDemo = true;
    }
  } catch (error) {
    // do nth
  }

  watchEffect(() => {
    if (state.isDemoCompleted) {
      return;
    }

    if (String(route.name).startsWith("auth")) {
      state.showDemo = false;
    } else {
      state.showDemo = true;
    }
  });
});

const handleProcessClose = () => {
  state.showDemo = false;
  state.isDemoCompleted = true;
  removeDemo();
};
</script>
