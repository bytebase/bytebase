<template>
  <Drawer v-model:show="showPanel" :close-on-esc="false" :mask-closable="false">
    <DrawerContent
      :title="$t('common.settings')"
      style="width: calc(100vw - 3rem)"
    >
      <InstanceForm :drawer="false" :instance="undefined" />
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import InstanceForm from "@/components/InstanceForm";
import { Drawer, DrawerContent } from "@/components/v2";
import { useSettingsContext } from "./context";

const { showPanel } = useSettingsContext();
</script>
