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
        <div
          class="max-h-[10em] overflow-auto border rounded-sm p-2 font-mono text-sm whitespace-pre-wrap"
        >
          {{ accessGrant.query }}
        </div>
      </div>

      <NCheckbox :checked="accessGrant.unmask" :disabled="true">
      {{ $t("sql-editor.access-type-unmask") }}
      </NCheckbox>

      <!-- Expiration Time -->
      <div class="flex flex-col gap-y-2">
        <span class="text-sm text-control-light">
          {{ $t("issue.grant-request.expired-at") }}
        </span>
        <div class="text-base">
          {{
            expireTime
              ? dayjs(expireTime).format("LLL")
              : $t("project.members.never-expires")
          }}
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { NCheckbox } from "naive-ui";
import { computed, ref, watchEffect } from "vue";
import { BBSpin } from "@/bbkit";
import DatabaseDisplay from "@/components/Plan/components/common/DatabaseDisplay.vue";
import { useAccessGrantStore, useDatabaseV1Store } from "@/store";
import { getDateForPbTimestampProtoEs, isValidDatabaseName } from "@/types";
import type { AccessGrant } from "@/types/proto-es/v1/access_grant_service_pb";
import { extractProjectResourceName } from "@/utils";
import { usePlanContextWithIssue } from "../../../logic";

const { issue } = usePlanContextWithIssue();
const dbStore = useDatabaseV1Store();
const accessGrantStore = useAccessGrantStore();

const isLoading = ref(true);
const accessGrant = ref<AccessGrant>();

const expireTime = computed(() => {
  if (accessGrant.value?.expiration.case === "expireTime") {
    return getDateForPbTimestampProtoEs(accessGrant.value.expiration.value);
  }
  return undefined;
});

watchEffect(async () => {
  const name = issue.value.accessGrant;
  if (!name) {
    isLoading.value = false;
    return;
  }
  try {
    const project = `projects/${extractProjectResourceName(issue.value.name)}`;
    const response = await accessGrantStore.searchMyAccessGrants(project, {
      name,
    });
    accessGrant.value = response.accessGrants[0];
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
