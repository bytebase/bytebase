<template>
  <Transition name="drawer">
    <DemoDrawer v-if="state.showDemoDrawer" @close="handleDemoDrawerClose" />
  </Transition>
  <Transition name="process-bar">
    <ProcessBar
      v-if="state.showProcessBar"
      @finish.once="handleProcessFinished"
    />
  </Transition>
</template>

<script lang="ts" setup>
import { onMounted, reactive } from "vue";
import { fetchDemoDataWithName } from "../data";
import useAppStore from "../store";
import DemoDrawer from "./DemoDrawer.vue";
import ProcessBar from "./ProcessBar.vue";

interface LocalState {
  showDemoDrawer: boolean;
  showProcessBar: boolean;
}

const props = defineProps<{
  demoName: string;
}>();

const state = reactive<LocalState>({
  showDemoDrawer: false,
  showProcessBar: false,
});

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
});

const handleDemoDrawerClose = () => {
  state.showDemoDrawer = false;
};

const handleProcessFinished = () => {
  state.showDemoDrawer = true;
};
</script>

<style scoped>
.drawer-enter-active,
.drawer-leave-active,
.process-bar-leave-active,
.process-bar-leave-active {
  @apply transition-all duration-300;
}

.drawer-enter-from,
.drawer-leave-to {
  @apply translate-x-full;
}

.process-bar-enter-from,
.process-bar-leave-to {
  @apply translate-y-full;
}
</style>
