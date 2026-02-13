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

    <NTooltip placement="right">
      <template #trigger>
        <p
          class="max-w-full text-xs wrap-break-word whitespace-pre-wrap font-mono line-clamp-2"
          :class="{ 'line-through text-gray-400': isExpired || isRevoked }"
        >
          {{ grant.query }}
        </p>
      </template>
      <pre class="max-w-lg whitespace-pre-wrap text-xs">{{ grant.query }}</pre>
    </NTooltip>

    <div class="w-full flex flex-col gap-y-2">
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
          tertiary
          size="tiny"
          type="primary"
          @click.stop="$emit('run', grant)"
        >
          {{ $t("common.run") }}
        </NButton>
        <NButton
          v-if="grant.issue"
          tertiary
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
import {
  type AccessGrant,
  AccessGrant_Status,
} from "@/types/proto-es/v1/access_grant_service_pb";
import {
  getAccessGrantDisplayStatus,
  getAccessGrantExpirationText,
  getAccessGrantExpireTimeMs,
  getAccessGrantStatusTagType,
} from "@/utils/accessGrant";

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

const expireTimeMs = computed(() => getAccessGrantExpireTimeMs(props.grant));

const isExpired = computed(() => {
  if (!isActive.value || expireTimeMs.value === undefined) return false;
  return expireTimeMs.value < Date.now();
});

const displayStatus = computed(() => getAccessGrantDisplayStatus(props.grant));

const statusTagType = computed(() =>
  getAccessGrantStatusTagType(displayStatus.value)
);

const statusLabel = computed(() => {
  return displayStatus.value;
});

const expirationInfo = computed(() =>
  getAccessGrantExpirationText(props.grant)
);

const expirationText = computed(() => {
  const info = expirationInfo.value;
  if (info.type === "never") return t("project.members.never-expires");
  if (info.type === "duration") return info.value;
  if (isActive.value && !isExpired.value && expireTimeMs.value !== undefined) {
    const diff = expireTimeMs.value - Date.now();
    const hours = Math.floor(diff / (1000 * 60 * 60));
    if (hours >= 24) {
      return t("sql-editor.expire-at", { time: info.value });
    }
    const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
    const dur = hours > 0 ? `${hours}h${minutes}m` : `${minutes}m`;
    return t("sql-editor.expire-in", { time: dur });
  }
  return t("sql-editor.expire-at", { time: info.value });
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
