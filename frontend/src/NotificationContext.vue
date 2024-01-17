<template>
  <slot></slot>
</template>

<script setup lang="ts">
import { useNotification } from "naive-ui";
import { watchEffect, h } from "vue";
import { useNotificationStore } from "@/store";
import { BBNotificationStyle } from "@/types";

const NOTIFICATION_DURATION = 6000;
const CRITICAL_NOTIFICATION_DURATION = 10000;

const notificationStore = useNotificationStore();
const notification = useNotification();

const getNotificationType = (style: BBNotificationStyle) => {
  switch (style) {
    case "CRITICAL":
      return "error";
    case "INFO":
      return "info";
    case "WARN":
      return "warning";
    default:
      return "success";
  }
};

const watchNotification = () => {
  const item = notificationStore.tryPopNotification({
    module: "bytebase",
  });
  if (!item) {
    return;
  }

  const duration = item.manualHide
    ? undefined
    : item.style === "CRITICAL"
    ? CRITICAL_NOTIFICATION_DURATION
    : NOTIFICATION_DURATION;

  notification.create({
    type: getNotificationType(item.style),
    keepAliveOnHover: true,
    duration,
    title: () =>
      h(
        "div",
        {
          class: "text-sm font-medium text-gray-900 whitespace-pre-wrap mt-1",
        },
        item.title
      ),
    content:
      typeof item.description !== "undefined"
        ? () => {
            if (typeof item.description === "function") {
              return item.description();
            }
            if (typeof item.description === "string") {
              return h(
                "div",
                { class: "text-sm text-gray-500 whitespace-pre-wrap" },
                item.description
              );
            }
          }
        : undefined,
    meta:
      item.link && item.linkTitle
        ? () => {
            return h(
              "a",
              { href: item.link, class: "normal-link", target: "_blank" },
              item.linkTitle
            );
          }
        : undefined,
  });
};

watchEffect(watchNotification);
</script>
