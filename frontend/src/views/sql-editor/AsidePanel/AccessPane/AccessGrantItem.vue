<template>
  <div
    class="w-full p-2 gap-y-2 border-b flex flex-col justify-start items-start cursor-pointer hover:bg-gray-50"
    :class="highlight ? 'highlight-pulse' : 'transition-colors duration-1000'"
  >
    <div class="w-full flex flex-row justify-between items-center gap-x-2">
      <div class="flex items-center gap-x-1">
        <NTag :type="statusTagType" size="tiny" :bordered="false" round>
          {{ statusLabel }}
        </NTag>
        <NTag v-if="grant.unmask" size="tiny" :bordered="false" round>
          {{ $t("sql-editor.grant-type-unmask") }}
        </NTag>
      </div>
      <span class="text-xs text-gray-500 shrink-0">
        {{ expirationText }}
      </span>
    </div>

    <p
      class="max-w-full text-xs wrap-break-word font-mono line-clamp-2"
      :class="{ 'line-through text-gray-400': isExpired || isRevoked }"
    >
      {{ grant.query }}
    </p>

    <div class="w-full flex flex-col gap-y-1">
      <NTooltip :disabled="allDatabaseNames.length <= 2" placement="right">
        <template #trigger>
          <span class="text-xs text-gray-400 truncate">
            {{ databaseNamesDisplay }}
          </span>
        </template>
        <div class="flex flex-col">
          <span v-for="name in allDatabaseNames" :key="name">{{ name }}</span>
        </div>
      </NTooltip>
      <div class="flex items-center gap-x-1">
        <NButton
          v-if="isActive && !isExpired"
          quaternary
          size="tiny"
          type="primary"
          @click.stop="$emit('run', grant)"
        >
          {{ $t("common.run") }}
        </NButton>
        <NButton
          v-if="grant.issue"
          quaternary
          size="tiny"
          tag="a"
          :href="issueLink"
          target="_blank"
          @click.stop
        >
          {{ $t("sql-editor.view-issue") }}
        </NButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { NButton, NTag, NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { getTimeForPbTimestampProtoEs } from "@/types";
import {
  type AccessGrant,
  AccessGrant_Status,
} from "@/types/proto-es/v1/access_grant_service_pb";

const props = defineProps<{
  grant: AccessGrant;
  highlight?: boolean;
}>();

defineEmits<{
  (e: "run", grant: AccessGrant): void;
}>();

const { t } = useI18n();

const isActive = computed(
  () => props.grant.status === AccessGrant_Status.ACTIVE
);

const isRevoked = computed(
  () => props.grant.status === AccessGrant_Status.REVOKED
);

const expireTimeMs = computed(() => {
  if (props.grant.expiration.case === "expireTime") {
    return getTimeForPbTimestampProtoEs(props.grant.expiration.value);
  }
  return undefined;
});

const isExpired = computed(() => {
  if (!isActive.value || expireTimeMs.value === undefined) return false;
  return expireTimeMs.value < Date.now();
});

const displayStatus = computed(() => {
  if (isActive.value && isExpired.value) return "EXPIRED";
  switch (props.grant.status) {
    case AccessGrant_Status.PENDING:
      return "PENDING";
    case AccessGrant_Status.ACTIVE:
      return "ACTIVE";
    case AccessGrant_Status.REVOKED:
      return "REVOKED";
    default:
      return "UNKNOWN";
  }
});

const statusTagType = computed(() => {
  switch (displayStatus.value) {
    case "ACTIVE":
      return "success";
    case "PENDING":
      return "warning";
    case "EXPIRED":
      return "default";
    case "REVOKED":
      return "error";
    default:
      return "default";
  }
});

const statusLabel = computed(() => {
  return displayStatus.value;
});

const expirationText = computed(() => {
  if (expireTimeMs.value === undefined) return "";
  if (isActive.value && !isExpired.value) {
    const diff = expireTimeMs.value - Date.now();
    const hours = Math.floor(diff / (1000 * 60 * 60));
    if (hours >= 24) {
      return new Date(expireTimeMs.value).toLocaleString();
    }
    const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
    if (hours > 0) {
      return t("sql-editor.time-left", { time: `${hours}h${minutes}m` });
    }
    return t("sql-editor.time-left", { time: `${minutes}m` });
  }
  return new Date(expireTimeMs.value).toLocaleString();
});

const allDatabaseNames = computed(() => {
  return props.grant.targets.map((t) => {
    const match = t.match(/databases\/(.+)$/);
    return match ? match[1] : t;
  });
});

const databaseNamesDisplay = computed(() => {
  const names = allDatabaseNames.value;
  if (names.length <= 2) {
    return names.join(", ");
  }
  return `${names.slice(0, 2).join(", ")} ${t("sql-editor.and-n-more-databases", { n: names.length - 2 })}`;
});

const issueLink = computed(() => {
  if (!props.grant.issue) return "";
  let path = props.grant.issue;
  if (!path.startsWith("/")) {
    path = `/${path}`;
  }
  return path;
});
</script>

<style scoped>
.highlight-pulse {
  animation: highlight-fade 3s ease-in-out;
}

@keyframes highlight-fade {
  0% {
    background-color: rgb(219 234 254); /* bg-blue-100 */
  }
  60% {
    background-color: rgb(219 234 254);
  }
  100% {
    background-color: transparent;
  }
}
</style>
