<template>
  <NButton :style="buttonStyle" v-bind="buttonProps" @click="show">
    <template #icon>
      <SettingsIcon />
    </template>
  </NButton>
</template>

<script setup lang="ts">
import { SettingsIcon } from "lucide-vue-next";
import { useButton } from "../common";
import { useSettingsContext } from "./context";

const { size, showPanel } = useSettingsContext();

const { props: buttonProps, style: buttonStyle } = useButton({
  size,
  active: showPanel,
  disabled: false,
});

const show = () => {
  showPanel.value = true;
};
</script>
