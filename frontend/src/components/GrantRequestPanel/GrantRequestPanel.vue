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
      class="w-[50rem] max-w-[100vw] relative"
    >
      <div class="w-full mx-auto space-y-4">
        <AddProjectMemberForm
          ref="formRef"
          class="w-full"
          :project-name="projectName"
          :binding="state.binding"
          :allow-remove="false"
          :disable-member-change="true"
          :require-reason="project.enforceIssueTitle"
          :support-roles="supportRoles"
          :database-resource="databaseResource"
        />
      </div>
      <template #footer>
        <div class="flex items-center justify-end gap-x-2">
          <NButton @click="$emit('close')">{{ $t("common.cancel") }}</NButton>
          <NButton
            type="primary"
            :disabled="!allowCreate"
            @click="doCreateIssue"
          >
            <div class="flex items-center gap-1">
              {{ $t("common.submit") }}
            </div>
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
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import AddProjectMemberForm from "@/components/ProjectMember/AddProjectMember/AddProjectMemberForm.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { issueServiceClientConnect } from "@/grpcweb";
import { useCurrentUserV1, useProjectV1Store } from "@/store";
import type { DatabaseResource } from "@/types";
import { getUserEmailInBinding } from "@/types";
import {
  CreateIssueRequestSchema,
  IssueSchema,
  Issue_Type as NewIssue_Type,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";
import { BindingSchema } from "@/types/proto-es/v1/iam_policy_pb";
import {
  GrantRequest,
  Issue,
  Issue_Type,
} from "@/types/proto/v1/issue_service";
import { generateIssueTitle, displayRoleTitle } from "@/utils";
import { convertDurationToOld } from "@/utils/v1/common-conversions";
import { convertNewIssueToOld } from "@/utils/v1/issue-conversions";

interface LocalState {
  binding: Binding;
}

const props = withDefaults(
  defineProps<{
    projectName: string;
    role?: string;
    databaseResource?: DatabaseResource;
    placement?: "left" | "right";
    supportRoles?: string[];
  }>(),
  {
    databaseResource: undefined,
    role: undefined,
    placement: "right",
    supportRoles: () => [],
  }
);

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const currentUser = useCurrentUserV1();
const projectStore = useProjectV1Store();

const state = reactive<LocalState>({
  binding: create(BindingSchema, {
    role: props.role,
    members: [getUserEmailInBinding(currentUser.value.email)],
  }),
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

  const newIssue = Issue.fromPartial({
    title: project.value.enforceIssueTitle
      ? `[${t("issue.title.request-role")}] ${formRef.value?.reason}`
      : generateIssueTitle(
          "bb.issue.grant.request",
          uniq(
            formRef.value?.databaseResources?.map(
              (databaseResource) => databaseResource.databaseFullName
            )
          ),
          t("issue.title.request-specific-role", {
            role: displayRoleTitle(state.binding.role),
          })
        ),
    description: state.binding.condition?.description,
    type: Issue_Type.GRANT_REQUEST,
    grantRequest: {},
  });

  newIssue.grantRequest = GrantRequest.fromPartial({
    role: state.binding.role,
    user: `users/${currentUser.value.email}`,
  });
  if (state.binding.condition) {
    newIssue.grantRequest.condition = state.binding.condition;
  }
  if (formRef.value?.expirationTimestampInMS) {
    newIssue.grantRequest.expiration = convertDurationToOld(
      create(DurationSchema, {
        seconds: BigInt(
          dayjs(formRef.value.expirationTimestampInMS).unix() - dayjs().unix()
        ),
      })
    );
  }

  const request = create(CreateIssueRequestSchema, {
    parent: props.projectName,
    issue: create(IssueSchema, {
      title: newIssue.title,
      description: newIssue.description,
      type: NewIssue_Type.DATABASE_CHANGE,
      grantRequest: newIssue.grantRequest
        ? {
            role: newIssue.grantRequest.role,
            user: newIssue.grantRequest.user,
            condition: newIssue.grantRequest.condition,
            expiration: newIssue.grantRequest.expiration
              ? {
                  seconds: BigInt(
                    newIssue.grantRequest.expiration.seconds.toString()
                  ),
                  nanos: newIssue.grantRequest.expiration.nanos,
                }
              : undefined,
          }
        : undefined,
    }),
  });
  const response = await issueServiceClientConnect.createIssue(request);
  const createdIssue = convertNewIssueToOld(response);

  // TODO(ed): handle no permission
  const path = `/${createdIssue.name}`;

  window.open(path, "_blank");

  emit("close");
};
</script>
