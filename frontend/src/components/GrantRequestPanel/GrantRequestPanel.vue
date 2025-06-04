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
import dayjs from "dayjs";
import { uniq } from "lodash-es";
import { NButton } from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import AddProjectMemberForm from "@/components/ProjectMember/AddProjectMember/AddProjectMemberForm.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { issueServiceClient } from "@/grpcweb";
import { useCurrentUserV1, useProjectV1Store } from "@/store";
import type { DatabaseResource } from "@/types";
import { getUserEmailInBinding } from "@/types";
import { Duration } from "@/types/proto/google/protobuf/duration";
import { Binding } from "@/types/proto/v1/iam_policy";
import {
  GrantRequest,
  Issue,
  Issue_Type,
} from "@/types/proto/v1/issue_service";
import { generateIssueTitle, displayRoleTitle } from "@/utils";

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
  binding: Binding.fromPartial({
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
    newIssue.grantRequest.expiration = Duration.fromPartial({
      seconds:
        dayjs(formRef.value.expirationTimestampInMS).unix() - dayjs().unix(),
    });
  }

  const createdIssue = await issueServiceClient.createIssue({
    parent: props.projectName,
    issue: newIssue,
  });

  // TODO(ed): handle no permission
  const path = `/${createdIssue.name}`;

  window.open(path, "_blank");

  emit("close");
};
</script>
