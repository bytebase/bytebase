<template>
  <div class="flex flex-col gap-y-1">
    <div class="flex items-center justify-between">
      <h3 class="textlabel">
        {{ $t("plan.navigator.checks") }}
      </h3>
      <NButton
        v-if="allowRunChecks"
        size="tiny"
        :loading="isRunningChecks"
        @click="runChecks"
      >
        <template #icon>
          <PlayIcon class="w-4 h-4" />
        </template>
        {{ $t("plan.run") }}
      </NButton>
    </div>
    <div class="flex items-center gap-2">
      <PlanCheckStatusCount
        :plan="plan"
        clickable
        @click="selectedResultStatus = $event"
      />
      <span v-if="!hasAnyChecks" class="text-sm text-control-placeholder">
        {{ $t("plan.overview.no-checks") }}
      </span>
    </div>

    <ChecksDrawer
      v-if="selectedResultStatus"
      :status="selectedResultStatus"
      @close="selectedResultStatus = undefined"
    />
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import type { ConnectError } from "@connectrpc/connect";
import { PlayIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import { planServiceClientConnect } from "@/connect";
import {
  extractUserId,
  pushNotification,
  useCurrentProjectV1,
  useCurrentUserV1,
} from "@/store";
import { RunPlanChecksRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import { usePlanCheckStatus, usePlanContext } from "../../../logic";
import { useResourcePoller } from "../../../logic/poller";
import ChecksDrawer from "../../ChecksView/ChecksDrawer.vue";
import PlanCheckStatusCount from "../../PlanCheckStatusCount.vue";

const currentUser = useCurrentUserV1();
const { plan } = usePlanContext();
const { project } = useCurrentProjectV1();
const { refreshResources } = useResourcePoller();
const { hasAnyStatus: hasAnyChecks } = usePlanCheckStatus(plan);

const isRunningChecks = ref(false);
const selectedResultStatus = ref<Advice_Level | undefined>(undefined);

const allowRunChecks = computed(() => {
  const me = currentUser.value;
  if (extractUserId(plan.value.creator) === me.email) {
    return true;
  }
  return hasProjectPermissionV2(project.value, "bb.planCheckRuns.run");
});

const runChecks = async () => {
  if (!plan.value.name) return;

  isRunningChecks.value = true;
  try {
    const request = create(RunPlanChecksRequestSchema, {
      name: plan.value.name,
    });
    await planServiceClientConnect.runPlanChecks(request);

    refreshResources(["plan", "planCheckRuns"], true);

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: "Plan checks started",
    });
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Failed to run plan checks",
      description: (error as ConnectError).message,
    });
  } finally {
    isRunningChecks.value = false;
  }
};
</script>
