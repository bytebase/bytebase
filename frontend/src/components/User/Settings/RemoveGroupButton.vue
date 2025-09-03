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

  <ResourceOccupiedModal
    ref="resourceOccupiedModalRef"
    :target="group.name"
    :resources="resourcesOccupied"
    :show-positive-button="true"
    @on-submit="onGroupRemove"
  />
</template>

<script lang="tsx" setup>
import { computedAsync } from "@vueuse/core";
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import ResourceOccupiedModal from "@/components/v2/ResourceOccupiedModal/ResourceOccupiedModal.vue";
import {
  useCurrentUserV1,
  useGroupStore,
  useProjectV1Store,
  useProjectIamPolicyStore,
  pushNotification,
  extractGroupEmail,
  usePolicyV1Store,
} from "@/store";
import { extractUserId } from "@/store/modules/v1/common";
import { getGroupEmailInBinding } from "@/types";
import {
  type Group,
  GroupMember_Role,
} from "@/types/proto-es/v1/group_service_pb";
import { PolicyType } from "@/types/proto-es/v1/org_policy_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

const props = defineProps<{
  group: Group;
}>();

const emit = defineEmits<{
  (event: "removed"): void;
}>();

const { t } = useI18n();
const groupStore = useGroupStore();
const currentUserV1 = useCurrentUserV1();
const projectStore = useProjectV1Store();
const projectIamPolicyStore = useProjectIamPolicyStore();
const policyStore = usePolicyV1Store();
const resourceOccupiedModalRef =
  ref<InstanceType<typeof ResourceOccupiedModal>>();

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

const resourcesOccupied = computedAsync(async () => {
  const member = getGroupEmailInBinding(extractGroupEmail(props.group.name));
  const resources: Set<string> = new Set();

  // Don't need to be so strict, it's okay to keep this way.
  for (const project of projectStore.getProjectList()) {
    const iamPolicy = projectIamPolicyStore.getProjectIamPolicy(project.name);
    for (const binding of iamPolicy.bindings) {
      if (binding.members.includes(member)) {
        resources.add(project.name);
        break;
      }
    }
    if (resources.has(project.name)) {
      continue;
    }

    const policy = await policyStore.getOrFetchPolicyByParentAndType({
      parentPath: project.name,
      policyType: PolicyType.MASKING_EXCEPTION,
    });

    const exceptions =
      policy?.policy?.case === "maskingExceptionPolicy"
        ? policy.policy.value.maskingExceptions
        : [];
    for (const exception of exceptions) {
      if (exception.member === member) {
        resources.add(project.name);
        break;
      }
    }
  }
  return [...resources];
}, []);

const handleDeleteGroup = async () => {
  resourceOccupiedModalRef.value?.open();
};

const onGroupRemove = () => {
  groupStore.deleteGroup(props.group.name).then(() => {
    emit("removed");
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.deleted"),
    });
  });
};
</script>
