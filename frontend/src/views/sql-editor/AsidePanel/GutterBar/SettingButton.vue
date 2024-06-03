<template>
  <NButton :style="buttonStyle" v-bind="buttonProps" @click="goSetting">
    <template #icon>
      <SettingsIcon />
    </template>
  </NButton>
</template>

<script setup lang="ts">
import { SettingsIcon } from "lucide-vue-next";
import { toRef } from "vue";
import { useRouter } from "vue-router";
import { SQL_EDITOR_SETTING_MODULE } from "@/router/sqlEditor";
import { type Size, useButton } from "./common";

const props = defineProps<{
  size: Size;
}>();

const router = useRouter();

const { props: buttonProps, style: buttonStyle } = useButton({
  size: toRef(props, "size"),
  active: false,
  disabled: false,
});

const goSetting = () => {
  router.push({
    name: SQL_EDITOR_SETTING_MODULE,
  });
};
</script>
