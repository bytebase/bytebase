<template>
  <div class="flex justify-end">
    <RemoveGroupButton v-if="allowDeleteGroup" :group="group!">
      <template #icon>
        <Trash2Icon class="w-4 h-auto" />
      </template>
    </RemoveGroupButton>

    <NButton
      v-if="allowEditGroup"
      quaternary
      size="small"
      @click="$emit('update-group')"
    >
      <template #icon>
        <PencilIcon class="w-4 h-auto" />
      </template>
    </NButton>
  </div>
</template>

<script lang="ts" setup>
import { PencilIcon, Trash2Icon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed } from "vue";
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
