<template>
  <div class="flex flex-col gap-y-4">
    <h3 class="text-base font-medium">
      {{ $t("issue.grant-request.details") }}
    </h3>

    <div class="p-4 border rounded-sm flex flex-col gap-y-4">
      <!-- Requested Role -->
      <div v-if="requestRole" class="flex flex-col gap-y-2">
        <span class="text-sm text-control-light">
          {{ $t("role.self") }}
        </span>
        <div class="text-base">
          {{ displayRoleTitle(requestRole) }}
        </div>
      </div>

      <!-- Database Resources -->
      <div v-if="condition?.databaseResources" class="flex flex-col gap-y-2">
        <span class="text-sm text-control-light">
          {{ $t("common.database") }}
        </span>
        <div>
          <span v-if="condition.databaseResources.length === 0">
            {{ $t("issue.grant-request.all-databases") }}
          </span>
          <DatabaseResourceTable
            v-else
            class="w-full"
            :database-resource-list="condition.databaseResources"
          />
        </div>
      </div>

      <!-- Expiration Time -->
      <div class="flex flex-col gap-y-2">
        <span class="text-sm text-control-light">
          {{ $t("issue.grant-request.expired-at") }}
        </span>
        <div class="text-base">
          {{
            condition?.expiredTime
              ? dayjs(new Date(condition.expiredTime)).format("LLL")
              : $t("project.members.never-expires")
          }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computedAsync } from "@vueuse/core";
import dayjs from "dayjs";
import { computed } from "vue";
import { DatabaseResourceTable } from "@/components/IssueV1/components";
import { displayRoleTitle } from "@/utils";
import { convertFromCELString } from "@/utils/issue/cel";
import { usePlanContextWithIssue } from "../../../logic";

const { issue } = usePlanContextWithIssue();

const requestRole = computed(() => {
  return issue.value.grantRequest?.role;
});

const condition = computedAsync(
  async () => {
    try {
      const conditionExpression = await convertFromCELString(
        issue.value.grantRequest?.condition?.expression ?? ""
      );
      return conditionExpression;
    } catch (error) {
      console.error("Failed to parse CEL expression:", error);
      return undefined;
    }
  },
  undefined // default value while loading or on error
);
</script>
