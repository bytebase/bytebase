<template>
  <Drawer
    :show="true"
    width="auto"
    :placement="placement"
    @update:show="(show: boolean) => !show && $emit('close')"
  >
    <DrawerContent
      :title="$t('issue.title.request-role')"
      :closable="true"
      class="w-200 max-w-[100vw] relative"
    >
      <div class="w-full mx-auto flex flex-col gap-y-4">
        <AddProjectMemberForm
          ref="formRef"
          class="w-full"
          :project-name="projectName"
          :binding="binding"
          :allow-remove="false"
          :disable-member-change="true"
          :require-reason="project.enforceIssueTitle"
          :support-roles="supportRoles"
          :database-resources="databaseResources"
        />
      </div>
      <template #footer>
        <div class="flex items-center justify-end gap-x-2">
          <NButton quaternary @click="$emit('close')">{{
            $t("common.cancel")
          }}</NButton>
          <NButton
            type="primary"
            :disabled="!allowCreate"
            @click="doCreateIssue"
          >
            {{ $t("common.submit") }}
          </NButton>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { DurationSchema } from "@bufbuild/protobuf/wkt";
import dayjs from "dayjs";
import { uniq } from "lodash-es";
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import AddProjectMemberForm from "@/components/ProjectMember/AddProjectMember/AddProjectMemberForm.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { issueServiceClientConnect } from "@/connect";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL_V1 } from "@/router/dashboard/projectV1";
import { useCurrentUserV1, useProjectV1Store } from "@/store";
import type { DatabaseResource } from "@/types";
import { getUserEmailInBinding } from "@/types";
import { BindingSchema } from "@/types/proto-es/v1/iam_policy_pb";
import {
  CreateIssueRequestSchema,
  GrantRequestSchema,
  IssueSchema,
  Issue_Type as NewIssue_Type,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  displayRoleTitle,
  extractIssueUID,
  extractProjectResourceName,
  generateIssueTitle,
} from "@/utils";

const props = withDefaults(
  defineProps<{
    projectName: string;
    role?: string;
    databaseResources?: DatabaseResource[];
    placement?: "left" | "right";
    supportRoles?: string[];
  }>(),
  {
    databaseResources: () => [],
    role: undefined,
    placement: "right",
  }
);

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const currentUser = useCurrentUserV1();
const projectStore = useProjectV1Store();
const router = useRouter();

const binding = computed(() => {
  return create(BindingSchema, {
    role: props.role,
    members: [getUserEmailInBinding(currentUser.value.email)],
  });
});

const formRef = ref<InstanceType<typeof AddProjectMemberForm>>();

const project = computed(() =>
  projectStore.getProjectByName(props.projectName)
);

const allowCreate = computed(() => {
  return formRef.value?.allowConfirm;
});

const doCreateIssue = async () => {
  if (!allowCreate.value) {
    return;
  }

  const binding = await formRef.value?.getBinding();
  if (!binding) {
    return;
  }

  const grantRequest = create(GrantRequestSchema, {
    role: binding.role,
    user: `users/${currentUser.value.email}`,
    condition: binding.condition,
    expiration: formRef.value?.expirationTimestampInMS
      ? create(DurationSchema, {
          seconds: BigInt(
            dayjs(formRef.value.expirationTimestampInMS).unix() - dayjs().unix()
          ),
        })
      : undefined,
  });

  const databaseResources = await formRef.value?.getDatabaseResources();

  const newIssue = create(IssueSchema, {
    title: project.value.enforceIssueTitle
      ? `[${t("issue.title.request-role")}] ${formRef.value?.reason}`
      : generateIssueTitle(
          "bb.issue.grant.request",
          uniq(
            databaseResources?.map(
              (databaseResource) => databaseResource.databaseFullName
            )
          ),
          t("issue.title.request-specific-role", {
            role: displayRoleTitle(binding.role),
          })
        ),
    description: binding.condition?.description,
    type: NewIssue_Type.GRANT_REQUEST,
    grantRequest,
  });

  const request = create(CreateIssueRequestSchema, {
    parent: props.projectName,
    issue: newIssue,
  });
  const response = await issueServiceClientConnect.createIssue(request);

  const route = router.resolve({
    name: PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
    params: {
      projectId: extractProjectResourceName(response.name),
      issueId: extractIssueUID(response.name),
    },
  });

  window.open(route.fullPath, "_blank");

  emit("close");
};
</script>
