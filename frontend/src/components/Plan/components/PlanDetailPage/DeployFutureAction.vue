<template>
  <p v-if="showCreateRollout" class="text-sm text-control-placeholder mt-0.5">
    <i18n-t keypath="plan.phase.deploy-description-with-action" tag="span">
      <template #action>
        <span
          class="text-accent cursor-pointer hover:underline"
          @click="showRolloutCreatePanel = true"
        >{{ $t("plan.phase.create-rollout-action") }}</span>
      </template>
    </i18n-t>
    <RolloutCreatePanel
      :show="showRolloutCreatePanel"
      :context="actionContext"
      @close="showRolloutCreatePanel = false"
      @confirm="showRolloutCreatePanel = false"
    />
  </p>
  <p v-else class="text-sm text-control-placeholder mt-0.5">
    {{ $t("plan.phase.deploy-description") }}
  </p>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { State } from "@/types/proto-es/v1/common_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import { usePlanContext } from "../../logic";
import RolloutCreatePanel from "../HeaderSection/Actions/create/RolloutCreatePanel.vue";
import { useActionRegistry } from "../HeaderSection/Actions/registry/useActionRegistry";

const { plan, issue } = usePlanContext();
const { context: actionContext } = useActionRegistry();

const showRolloutCreatePanel = ref(false);

const showCreateRollout = computed(() => {
  if (!issue.value) return false;
  if (plan.value.hasRollout) return false;
  if (plan.value.state !== State.ACTIVE) return false;
  if (issue.value.status !== IssueStatus.OPEN) return false;
  if (!actionContext.value.permissions.createRollout) return false;
  return true;
});
</script>
