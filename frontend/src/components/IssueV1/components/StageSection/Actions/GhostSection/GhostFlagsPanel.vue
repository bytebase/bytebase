<template>
  <Drawer v-model:show="showFlagsPanel">
    <DrawerContent> hha </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { Drawer, DrawerContent } from "@/components/v2";
import { useIssueGhostContext } from "./common";

const { showFlagsPanel } = useIssueGhostContext();
</script>
