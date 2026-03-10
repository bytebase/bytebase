<template>
  <div class="flex flex-col gap-y-4">
    <h3 class="text-base font-medium">
      {{ $t("issue.access-grant.details") }}
    </h3>

    <div v-if="isLoading" class="flex items-center justify-center py-8">
      <BBSpin />
    </div>

    <div v-else-if="accessGrant" class="p-4 border rounded-sm flex flex-col gap-y-4">
      <!-- Target Databases -->
      <div v-if="accessGrant.targets.length > 0" class="flex flex-col gap-y-2">
        <span class="text-sm text-control-light">
          {{ $t("common.databases") }}
        </span>
        <div class="flex flex-wrap gap-2">
          <div
            v-for="target in accessGrant.targets"
            :key="target"
            class="inline-flex items-center gap-2 px-2 py-1.5 border rounded-sm min-w-0"
          >
            <DatabaseDisplay
              :database="target"
              :show-environment="true"
              size="medium"
              class="flex-1 min-w-0"
            />
          </div>
        </div>
      </div>

      <!-- Query -->
      <div v-if="accessGrant.query" class="flex flex-col gap-y-2">
        <span class="text-sm text-control-light">
          {{ $t("common.statement") }}
        </span>
        <BBAttention
          v-if="accessGrant.unmask"
          :type="'warning'"
        >
          {{ $t("sql-editor.unmask-warning") }}
        </BBAttention>
        <div class="max-h-[30em] overflow-auto rounded-xs p-4 bg-gray-50">
          <NConfigProvider :hljs="hljs">
            <NCode language="sql" :code="accessGrant.query" />
          </NConfigProvider>
        </div>
        <NCheckbox :checked="accessGrant.unmask" :disabled="true">
          <span class="text-base">
            {{ $t("sql-editor.access-type-unmask") }}
          </span>
        </NCheckbox>
      </div>

      <!-- Expiration -->
      <div class="flex flex-col gap-y-1">
        <span class="text-sm text-control-light">
          <template v-if="expirationInfo.type === 'duration'">
            {{ $t("common.duration") }}
          </template>
          <template v-else>
            {{ $t("issue.grant-request.expired-at") }}
          </template>
        </span>
        <div class="text-base">
          <template v-if="expirationInfo.type === 'datetime'">
            {{ expirationInfo.value }}
          </template>
          <template v-else-if="expirationInfo.type === 'duration'">
            {{ expirationInfo.value }}
          </template>
          <template v-else>
            {{ $t("project.members.never-expires") }}
          </template>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import hljs from "highlight.js/lib/core";
import { NCheckbox, NCode, NConfigProvider } from "naive-ui";
import { computed, ref, watchEffect } from "vue";
import { BBAttention, BBSpin } from "@/bbkit";
import DatabaseDisplay from "@/components/Plan/components/common/DatabaseDisplay.vue";
import { useAccessGrantStore, useDatabaseV1Store } from "@/store";
import { isValidDatabaseName } from "@/types";
import type { AccessGrant } from "@/types/proto-es/v1/access_grant_service_pb";
import { extractProjectResourceName, hasProjectPermissionV2 } from "@/utils";
import { getAccessGrantExpirationText } from "@/utils/accessGrant";
import { usePlanContextWithIssue } from "../../../logic";

const { issue, project } = usePlanContextWithIssue();
const dbStore = useDatabaseV1Store();
const accessGrantStore = useAccessGrantStore();

const isLoading = ref(true);
const accessGrant = ref<AccessGrant>();

const expirationInfo = computed(() => {
  if (!accessGrant.value) return { type: "never" as const };
  return getAccessGrantExpirationText(accessGrant.value);
});

watchEffect(async () => {
  const name = issue.value.accessGrant;
  if (!name) {
    isLoading.value = false;
    return;
  }
  try {
    let grant: AccessGrant | undefined;
    if (hasProjectPermissionV2(project.value, "bb.accessGrants.get")) {
      grant = await accessGrantStore.getAccessGrant(name);
    } else {
      const parent = `projects/${extractProjectResourceName(issue.value.name)}`;
      const response = await accessGrantStore.searchMyAccessGrants({
        parent,
        filter: { name },
      });
      grant = response.accessGrants[0];
    }
    accessGrant.value = grant;
    // Pre-fetch databases for display
    if (accessGrant.value) {
      for (const target of accessGrant.value.targets) {
        if (isValidDatabaseName(target)) {
          dbStore.getOrFetchDatabaseByName(target);
        }
      }
    }
  } finally {
    isLoading.value = false;
  }
});
</script>
