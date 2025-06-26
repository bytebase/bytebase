<template>
  <CommonDrawer
    :show="action !== undefined"
    :title="title"
    :loading="state.loading"
    @show="resetState"
    @close="$emit('close')"
  >
    <template #default>
      <div v-if="action" class="flex flex-col gap-y-4 px-1">
        <div class="flex flex-col gap-y-1">
          <div class="font-medium text-control">Plan</div>
          <div class="textinfolabel">
            {{ plan.title }}
          </div>
        </div>

        <div class="flex flex-col gap-y-1">
          <div class="font-medium text-control">
            {{ $t("common.description") }}
          </div>
          <div class="textinfolabel">
            {{ plan.description || "No description" }}
          </div>
        </div>

        <div class="flex flex-col gap-y-1">
          <p class="font-medium text-control">
            {{ $t("common.comment") }}
          </p>
          <NInput
            v-model:value="comment"
            type="textarea"
            :placeholder="$t('issue.leave-a-comment')"
            :autosize="{
              minRows: 3,
              maxRows: 10,
            }"
          />
        </div>
      </div>
    </template>
    <template #footer>
      <div
        v-if="action"
        class="w-full flex flex-row justify-end items-center gap-2"
      >
        <div class="flex justify-end gap-x-3">
          <NButton @click="$emit('close')">
            {{ $t("common.cancel") }}
          </NButton>

          <NTooltip :disabled="confirmErrors.length === 0" placement="top">
            <template #trigger>
              <NButton
                type="primary"
                :disabled="confirmErrors.length > 0"
                @click="handleConfirm"
              >
                {{ $t("common.create") }}
              </NButton>
            </template>
            <template #default>
              <ErrorList :errors="confirmErrors" />
            </template>
          </NTooltip>
        </div>
      </div>
    </template>
  </CommonDrawer>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { NButton, NInput, NTooltip } from "naive-ui";
import { computed, nextTick, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import CommonDrawer from "@/components/IssueV1/components/Panel/CommonDrawer.vue";
import { ErrorList } from "@/components/Plan/components/common";
import { usePlanContext, useIssueReviewContext } from "@/components/Plan/logic";
import { rolloutServiceClientConnect } from "@/grpcweb";
import { PROJECT_V1_ROUTE_ROLLOUT_DETAIL } from "@/router/dashboard/projectV1";
import { useCurrentProjectV1 } from "@/store";
import { CreateRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractProjectResourceName,
  extractRolloutUID,
  hasProjectPermissionV2,
} from "@/utils";

type RolloutAction = "CREATE";

type LocalState = {
  loading: boolean;
};

const props = defineProps<{
  action?: RolloutAction;
}>();
const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const router = useRouter();
const state = reactive<LocalState>({
  loading: false,
});
const { project } = useCurrentProjectV1();
const { plan } = usePlanContext();
const reviewContext = useIssueReviewContext();
const comment = ref("");

const title = computed(() => {
  switch (props.action) {
    case "CREATE":
      return t("common.create") + " " + t("common.rollout");
  }
  return ""; // Make linter happy
});

const confirmErrors = computed(() => {
  const errors: string[] = [];

  if (!hasProjectPermissionV2(project.value, "bb.rollouts.create")) {
    errors.push(t("common.missing-required-permission"));
  }

  if (plan.value.rollout) {
    errors.push("Rollout already exists for this plan");
  }

  if (!reviewContext.done.value) {
    errors.push("Issue must pass approval review before creating rollout");
  }

  return errors;
});

const handleConfirm = async () => {
  const { action } = props;
  if (!action) return;
  state.loading = true;

  try {
    if (action === "CREATE") {
      const request = create(CreateRolloutRequestSchema, {
        parent: project.value.name,
        rollout: {
          plan: plan.value.name,
          title: plan.value.title,
        },
      });

      const createdRollout =
        await rolloutServiceClientConnect.createRollout(request);

      nextTick(() => {
        router.push({
          name: PROJECT_V1_ROUTE_ROLLOUT_DETAIL,
          params: {
            projectId: extractProjectResourceName(plan.value.name),
            rolloutId: extractRolloutUID(createdRollout.name),
          },
        });
      });
    }
  } finally {
    state.loading = false;
    emit("close");
  }
};

const resetState = () => {
  comment.value = "";
};
</script>
