<template>
  <div class="space-y-2">
    <div class="flex items-center justify-between gap-2">
      <h3 class="text-base font-medium">
        {{ $t("common.description") }}
      </h3>
      <NButton
        v-if="!isEditing && allowEdit"
        size="small"
        @click="startEditing"
      >
        <template #icon>
          <EditIcon class="w-4 h-4" />
        </template>
        {{ $t("common.edit") }}
      </NButton>
      <div v-if="isEditing" class="flex items-center gap-2">
        <NButton size="small" @click="cancelEditing">
          {{ $t("common.cancel") }}
        </NButton>
        <NButton size="small" type="primary" @click="saveChanges">
          {{ $t("common.save") }}
        </NButton>
      </div>
    </div>
    <div v-if="!isEditing">
      <MarkdownEditor
        v-if="plan.description"
        :content="plan.description"
        mode="preview"
        :project="project"
        :issue-list="[]"
      />
      <div v-else class="text-control-placeholder text-sm italic">
        {{ $t("issue.add-some-description") }}
      </div>
    </div>
    <div v-else>
      <MarkdownEditor
        :content="editingDescription"
        mode="editor"
        :project="project"
        :placeholder="$t('issue.add-some-description')"
        :issue-list="[]"
        @change="onDescriptionChange"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { EditIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import MarkdownEditor from "@/components/MarkdownEditor";
import { useResourcePoller } from "@/components/Plan/logic/poller";
import { planServiceClientConnect } from "@/grpcweb";
import {
  extractUserId,
  pushNotification,
  useCurrentProjectV1,
  useCurrentUserV1,
} from "@/store";
import {
  PlanSchema,
  UpdatePlanRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import { usePlanContext } from "../../..";

const { t } = useI18n();
const currentUser = useCurrentUserV1();
const { project } = useCurrentProjectV1();
const { plan, readonly, isCreating } = usePlanContext();
const { refreshResources } = useResourcePoller();

const isEditing = ref(false);
const editingDescription = ref("");

const allowEdit = computed(() => {
  if (readonly.value) {
    return false;
  }
  if (isCreating.value) {
    return true;
  }
  // Allowed if current user is the creator.
  if (extractUserId(plan.value.creator) === currentUser.value.email) {
    return true;
  }
  // Allowed if current user has related permission.
  if (hasProjectPermissionV2(project.value, "bb.plans.update")) {
    return true;
  }
  return false;
});

const startEditing = () => {
  editingDescription.value = plan.value.description;
  isEditing.value = true;
};

const cancelEditing = () => {
  isEditing.value = false;
  editingDescription.value = "";
};

const onDescriptionChange = (value: string) => {
  editingDescription.value = value;
};

const saveChanges = async () => {
  // Only update if description actually changed
  if (editingDescription.value === plan.value.description) {
    isEditing.value = false;
    return;
  }

  const request = create(UpdatePlanRequestSchema, {
    plan: create(PlanSchema, {
      name: plan.value.name,
      description: editingDescription.value,
    }),
    updateMask: { paths: ["description"] },
  });
  await planServiceClientConnect.updatePlan(request);
  refreshResources(["plan"], true /** force */);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
  isEditing.value = false;
};
</script>
