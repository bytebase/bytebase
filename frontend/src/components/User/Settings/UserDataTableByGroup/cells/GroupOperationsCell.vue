<template>
  <div class="flex justify-end gap-x-2">
    <RemoveGroupButton v-if="allowDeleteGroup" :group="group" @removed="$emit('remove-group')">
      <Trash2Icon class="w-4 h-auto" />
    </RemoveGroupButton>

    <MiniActionButton
      v-if="allowEditGroup"
      @click="$emit('update-group')"
    >
      <PencilIcon />
    </MiniActionButton>
  </div>
</template>

<script lang="ts" setup>
import { PencilIcon, Trash2Icon } from "lucide-vue-next";
import { computed } from "vue";
import { MiniActionButton } from "@/components/v2";
import { useCurrentUserV1 } from "@/store";
import { extractUserId } from "@/store/modules/v1/common";
import type { Group } from "@/types/proto-es/v1/group_service_pb";
import { GroupMember_Role } from "@/types/proto-es/v1/group_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import RemoveGroupButton from "../../RemoveGroupButton.vue";

const props = defineProps<{
  group: Group;
}>();

defineEmits<{
  (event: "update-group"): void;
  (event: "remove-group"): void;
}>();

const currentUser = useCurrentUserV1();

const isGroupOwner = computed(() => {
  return (
    props.group.members.find(
      (m) => extractUserId(m.member) === currentUser.value.email
    )?.role === GroupMember_Role.OWNER
  );
});

const allowEditGroup = computed(() => {
  return isGroupOwner.value || hasWorkspacePermissionV2("bb.groups.update");
});

const allowDeleteGroup = computed(() => {
  return isGroupOwner.value || hasWorkspacePermissionV2("bb.groups.update");
});
</script>
