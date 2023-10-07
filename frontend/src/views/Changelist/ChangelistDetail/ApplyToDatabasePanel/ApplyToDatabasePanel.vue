<template>
  <Drawer :show="show" :close-on-esc="true" @close="show = false">
    <DrawerContent> </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { useChangelistDetailContext } from "../context";

const { showApplyToDatabasePanel: show } = useChangelistDetailContext();
</script>
