<template>
  <MiniActionButton
    v-if="allowDelete"
    type="error"
    v-bind="$attrs"
    @click="handleDeleteGroup"
  >
    <template #default>
      <slot name="icon" />
    </template>
    <template #text>
      <slot name="default" />
    </template>
  </MiniActionButton>

  <ResourceOccupiedModal
    ref="resourceOccupiedModalRef"
    :target="group.name"
    :resources="resourcesOccupied"
    :show-positive-button="true"
    @on-submit="onGroupRemove"
  />
</template>

<script lang="tsx" setup>
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { MiniActionButton } from "@/components/v2";
import ResourceOccupiedModal from "@/components/v2/ResourceOccupiedModal/ResourceOccupiedModal.vue";
import { pushNotification, useCurrentUserV1, useGroupStore } from "@/store";
import { extractUserId } from "@/store/modules/v1/common";
import {
  type Group,
  GroupMember_Role,
} from "@/types/proto-es/v1/group_service_pb";
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

// We don't check resourcesOccupied because:
// 1. projectStore.getProjectList() only returns cached projects (pagination)
// 2. getProjectIamPolicy() may return empty for projects not visited
// The check would be unreliable, so we skip it.
const resourcesOccupied: string[] = [];

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
