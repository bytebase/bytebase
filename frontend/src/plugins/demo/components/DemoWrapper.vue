<template>
  <ProcessBar v-if="state.showProcessBar" @close.once="handleProcessClose" />
</template>

<script lang="ts" setup>
import { onMounted, reactive, watchEffect } from "vue";
import { useRoute } from "vue-router";
import { removeDemo } from "..";
import { fetchDemoDataWithName } from "../data";
import useAppStore from "../store";
import ProcessBar from "./ProcessBar.vue";

interface LocalState {
  completed: boolean;
  showProcessBar: boolean;
}

const props = defineProps<{
  demoName: string;
}>();

const state = reactive<LocalState>({
  completed: false,
  showProcessBar: false,
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
      });

      if (demoData.process.length > 0) {
        state.showProcessBar = true;
      }
    }
  } catch (error) {
    // do nth
  }

  watchEffect(() => {
    if (state.completed) {
      return;
    }

    if (String(route.name).startsWith("auth")) {
      state.showProcessBar = false;
    } else {
      state.showProcessBar = true;
    }
  });
});

const handleProcessClose = () => {
  state.showProcessBar = false;
  state.completed = true;
  removeDemo();
};
</script>
