<template>
  <div
    class="flex items-center gap-x-3 px-3 py-2.5 cursor-pointer rounded transition-colors"
    :class="selected ? 'bg-blue-50' : 'hover:bg-gray-50'"
    @click="handleClick"
  >
    <!-- Chevron for expandable mode -->
    <ChevronRightIcon
      v-if="expandable"
      class="w-4 h-4 shrink-0 text-control-placeholder transition-transform"
      :class="expanded ? 'rotate-90' : ''"
    />

    <!-- Avatar -->
    <BBAvatar
      :username="displayName"
      :email="member.type === 'user' ? userEmail : ''"
      size="SMALL"
    />

    <!-- Content -->
    <div class="flex-1 min-w-0">
      <!-- Line 1: Name -->
      <div class="flex items-center gap-x-1.5">
        <span class="font-medium text-sm truncate">{{ displayName }}</span>
        <NTag
          v-if="member.type === 'group'"
          size="small"
          round
        >
          {{ $t("common.groups") }}
        </NTag>
        <NTag
          v-else-if="isServiceAccount"
          size="small"
          round
          type="info"
        >
          {{ $t("settings.members.service-account") }}
        </NTag>
        <NTag
          v-else-if="isWorkloadIdentity"
          size="small"
          round
          type="info"
        >
          {{ $t("settings.members.workload-identity") }}
        </NTag>
      </div>

      <!-- Line 2: Scope summary -->
      <div class="text-xs text-control-light truncate mt-0.5">
        {{ scopeSummary }}
      </div>

    </div>
  </div>
</template>

<script lang="ts" setup>
import { ChevronRightIcon } from "lucide-vue-next";
import { NTag } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBAvatar } from "@/bbkit";
import { extractUserEmail } from "@/store";
import {
  serviceAccountBindingPrefix,
  workloadIdentityBindingPrefix,
} from "@/types";
import type { ExemptionMember } from "./types";

const { t: $t } = useI18n(); // NOSONAR

const props = withDefaults(
  defineProps<{
    member: ExemptionMember;
    selected?: boolean;
    expanded?: boolean;
    expandable?: boolean;
  }>(),
  {
    selected: false,
    expanded: false,
    expandable: false,
  }
);

const emit = defineEmits<{
  (e: "select"): void;
  (e: "toggle"): void;
}>();

const userEmail = computed(() => extractUserEmail(props.member.member));

const isServiceAccount = computed(() =>
  props.member.member.startsWith(serviceAccountBindingPrefix)
);

const isWorkloadIdentity = computed(() =>
  props.member.member.startsWith(workloadIdentityBindingPrefix)
);

const displayName = computed(() => {
  // Strip prefix (user:, group:, serviceAccount:, workloadIdentity:) to show just the email/name
  const raw = props.member.member;
  const idx = raw.indexOf(":");
  return idx >= 0 ? raw.substring(idx + 1) : raw;
});

const scopeSummary = computed(() => {
  const n = props.member.grants.length;
  const exemptionWord = $t("project.masking-exemption.n-exemptions", { n }, n);

  const realDbs = props.member.databaseNames.filter(
    (name) => name !== "-1" && name !== ""
  );
  const hasAllDbs = props.member.databaseNames.some(
    (name) => name === "-1" || name === ""
  );

  const parts: string[] = [];
  if (realDbs.length > 0) {
    parts.push(realDbs.join(", "));
  }
  if (hasAllDbs) {
    parts.push($t("database.all"));
  }

  const scope = parts.join(", ");
  return scope ? `${exemptionWord} · ${scope}` : exemptionWord;
});

const handleClick = () => {
  if (props.expandable) {
    emit("toggle");
  } else {
    emit("select");
  }
};
</script>
