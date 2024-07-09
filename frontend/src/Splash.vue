<template>
  <NConfigProvider
    :locale="generalLang"
    :date-locale="dateLang"
    :theme-overrides="themeOverrides"
  >
    <NNotificationProvider
      :max="MAX_NOTIFICATION_DISPLAY_COUNT"
      placement="bottom-right"
    >
      <NDialogProvider>
        <NotificationContext>
          <div class="fixed inset-0 bg-white flex flex-col items-center justify-center">
            <NSpin />
          </div>
        </NotificationContext>
      </NDialogProvider>
    </NNotificationProvider>
  </NConfigProvider>
</template>

<script lang="ts" setup>
import {
  NConfigProvider,
  NDialogProvider,
  NNotificationProvider,
  NSpin,
} from "naive-ui";
import { themeOverrides, dateLang, generalLang } from "../naive-ui.config";
import NotificationContext from "./NotificationContext.vue";

const MAX_NOTIFICATION_DISPLAY_COUNT = 3;
</script>
