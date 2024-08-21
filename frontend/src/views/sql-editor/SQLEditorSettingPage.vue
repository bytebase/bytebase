<template>
  <div class="flex flex-row items-stretch w-full flex-1 overflow-hidden">
    <div v-if="windowWidth >= 800" class="border-r">
      <Sidebar class="w-52" />
    </div>
    <template v-else>
      <teleport to="body">
        <div
          id="fff"
          class="fixed rounded-full border border-control-border shadow-lg w-10 h-10 bottom-[4rem] flex items-center justify-center bg-white hover:bg-control-bg cursor-pointer z-[99999999] transition-all"
          :class="[
            state.sidebarExpanded ? 'left-[80%] -translate-x-5' : 'left-[1rem]',
          ]"
          style="
            transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1);
            transition-duration: 300ms;
          "
          @click="state.sidebarExpanded = !state.sidebarExpanded"
        >
          <ChevronLeftIcon
            class="w-6 h-6 transition-transform"
            :class="[state.sidebarExpanded ? '' : '-scale-100']"
          />
        </div>
        <Drawer
          v-model:show="state.sidebarExpanded"
          width="80vw"
          placement="left"
        >
          <Sidebar />
        </Drawer>
      </teleport>
    </template>

    <div class="flex-1">
      <router-view />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useWindowSize } from "@vueuse/core";
import { ChevronLeftIcon } from "lucide-vue-next";
import { reactive } from "vue";
import { Drawer } from "@/components/v2";
import { Sidebar } from "./Setting";

type LocalState = {
  sidebarExpanded: boolean;
};

const state = reactive<LocalState>({
  sidebarExpanded: false,
});

const { width: windowWidth } = useWindowSize();
</script>
