<template>
  <Drawer :show="show" :mask-closable="false" @close="$emit('close')">
    <DrawerContent
      :title="title"
      class="relative overflow-hidden !w-[30rem] !max-w-[30rem]"
    >
      <template #default>
        <slot />

        <div
          v-if="loading"
          v-zindexable="{ enabled: true }"
          class="absolute inset-0 flex items-center justify-center bg-white/50"
        >
          <BBSpin />
        </div>
      </template>

      <template v-if="$slots.footer" #footer>
        <slot name="footer" />
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { zindexable as vZindexable } from "vdirs";
import { BBSpin } from "@/bbkit";
import { Drawer, DrawerContent } from "@/components/v2";

defineProps<{
  title: string;
  show: boolean;
  loading: boolean;
}>();

defineEmits<{
  (event: "close"): void;
}>();
</script>
