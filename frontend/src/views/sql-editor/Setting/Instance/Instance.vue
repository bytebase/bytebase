<template>
  <div class="w-full flex flex-col gap-4 py-4 overflow-y-auto">
    <InstanceDashboard :on-row-click="handleClick" />
  </div>

  <Drawer
    v-if="state.instance"
    :show="state.instance !== undefined"
    :close-on-esc="true"
    :mask-closable="true"
    @update:show="() => (state.instance = undefined)"
  >
    <DrawerContent
      header-style="--n-header-padding: 0 24px;"
      body-style="--n-body-padding: 16px 24px 0;"
    >
      <template #header>
        <div class="h-[50px] flex">
          <div class="flex items-center gap-x-2 h-[50px]">
            <EngineIcon :engine="state.instance.engine" custom-class="!h-6" />
            <span class="font-medium">{{
              instanceV1Name(state.instance)
            }}</span>
          </div>
        </div>
      </template>
      <InstanceDetail
        :instance-id="extractInstanceResourceName(state.instance.name)"
        :embedded="true"
        :hide-archive-restore="true"
        class="!px-0 !mb-0 w-[850px]"
      />
    </DrawerContent>
  </Drawer>
</template>

<script lang="tsx" setup>
import { reactive } from "vue";
import { EngineIcon } from "@/components/Icon";
import { Drawer, DrawerContent } from "@/components/v2";
import type { Instance } from "@/types/proto/v1/instance_service";
import { extractInstanceResourceName, instanceV1Name } from "@/utils";
import InstanceDashboard from "@/views/InstanceDashboard.vue";
import InstanceDetail from "@/views/InstanceDetail.vue";

interface LocalState {
  instance?: Instance;
}

const state = reactive<LocalState>({});

const handleClick = (instance: Instance) => {
  state.instance = instance;
};
</script>
