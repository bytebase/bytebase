<template>
  <NButton
    v-if="allowDelete"
    quaternary
    size="small"
    @click="handleDeleteGroup"
  >
    <template #icon>
      <slot name="icon" />
    </template>
    <template #default>
      <slot name="default" />
    </template>
  </NButton>
</template>

<script lang="tsx" setup>
import { NButton, useDialog, type DialogReactive } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink, type RouteLocationRaw } from "vue-router";
import ProjectV1Name from "@/components/v2/Model/ProjectV1Name.vue";
import { PROJECT_V1_ROUTE_MEMBERS } from "@/router/dashboard/projectV1";
import { WORKSPACE_ROUTE_GLOBAL_MASKING } from "@/router/dashboard/workspaceRoutes";
import {
  useCurrentUserV1,
  useGroupStore,
  useProjectV1Store,
  pushNotification,
  extractGroupEmail,
  usePolicyV1Store,
} from "@/store";
import { getProjectName } from "@/store/modules/v1/common";
import { extractUserId } from "@/store/modules/v1/common";
import { getGroupEmailInBinding } from "@/types";
import { type Group, GroupMember_Role } from "@/types/proto/v1/group_service";
import { PolicyType } from "@/types/proto/v1/org_policy_service";
import type { Project } from "@/types/proto/v1/project_service";
import { hasWorkspacePermissionV2 } from "@/utils";

const props = defineProps<{
  group: Group;
}>();

const emit = defineEmits<{
  (event: "removed"): void;
}>();

const { t } = useI18n();
const dialog = useDialog();
const groupStore = useGroupStore();
const currentUserV1 = useCurrentUserV1();
const projectStore = useProjectV1Store();
const policyStore = usePolicyV1Store();

const selfMemberInGroup = computed(() => {
  return props.group?.members.find(
    (member) => extractUserId(member.member) === currentUserV1.value.email
  );
});

const allowDelete = computed(() => {
  if (selfMemberInGroup.value?.role === GroupMember_Role.OWNER) {
    return true;
  }
  return hasWorkspacePermissionV2("bb.groups.delete");
});

const getProjectsBindingGroup = async (group: Group) => {
  const member = getGroupEmailInBinding(extractGroupEmail(group.name));
  interface ProjectGroupResource {
    member: boolean;
    policy: boolean;
    project: Project;
  }
  const response: ProjectGroupResource[] = [];

  // TODO(ed): Do we need a API to get IAM permission by user?
  // Or we can just don't need to be so strict, it's okay to keep this way.
  for (const project of projectStore.getProjectList()) {
    let resource: ProjectGroupResource | undefined;
    for (const binding of project.iamPolicy.bindings) {
      if (binding.members.includes(member)) {
        resource = {
          project,
          member: true,
          policy: false,
        };
        break;
      }
    }

    const policy = await policyStore.getOrFetchPolicyByParentAndType({
      parentPath: project.name,
      policyType: PolicyType.MASKING_EXCEPTION,
    });

    for (const exception of policy?.maskingExceptionPolicy?.maskingExceptions ??
      []) {
      if (exception.member === member) {
        if (!resource) {
          resource = {
            project,
            member: false,
            policy: false,
          };
        }
        resource.policy = true;
        break;
      }
    }

    if (resource) {
      response.push(resource);
    }
  }

  return response;
};

const renderProjectResources = (
  title: string,
  projects: Project[],
  $dialog: DialogReactive,
  getRoute: (project: Project) => RouteLocationRaw
) => {
  if (projects.length === 0) {
    return;
  }

  return (
    <div>
      {title}
      <ul class="list-disc ml-4">
        {projects.map((project) => (
          <li onClick={() => $dialog.destroy()}>
            <RouterLink to={getRoute(project)} class="normal-link">
              <ProjectV1Name project={project} link={false} />
            </RouterLink>
          </li>
        ))}
      </ul>
    </div>
  );
};

const handleDeleteGroup = async () => {
  const resources = await getProjectsBindingGroup(props.group);
  // TODO(ed): use ResourceOccupiedModal
  const $dialog = dialog.create({
    type: "warning",
    title: t("common.warning"),
    content: () => {
      if (resources.length === 0) {
        return t("settings.members.groups.delete-warning", {
          name: props.group.title,
        });
      }
      return (
        <div class="space-y-2">
          <p>
            {t("settings.members.groups.delete-warning-with-resources", {
              name: props.group.title,
            })}
          </p>
          {renderProjectResources(
            t("common.projects"),
            resources.filter((r) => r.member).map((r) => r.project),
            $dialog,
            (project) => ({
              name: PROJECT_V1_ROUTE_MEMBERS,
              params: {
                projectId: getProjectName(project.name),
              },
            })
          )}

          {renderProjectResources(
            t("settings.sidebar.global-masking"),
            resources.filter((r) => r.policy).map((r) => r.project),
            $dialog,
            (_) => ({
              name: WORKSPACE_ROUTE_GLOBAL_MASKING,
            })
          )}
          <p>{t("bbkit.confirm-button.sure-to-delete")}</p>
        </div>
      );
    },
    style: "z-index: 100000",
    negativeText: t("common.cancel"),
    positiveText: t("common.continue-anyway"),
    onPositiveClick: () => {
      groupStore.deleteGroup(props.group.name).then(() => {
        emit("removed");
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.deleted"),
        });
      });
    },
  });
};
</script>
