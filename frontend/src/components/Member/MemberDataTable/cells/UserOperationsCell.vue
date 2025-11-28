<template>
  <div v-if="allowEdit" class="flex justify-end gap-x-2">
    <NPopconfirm
      v-if="
        allowRevoke &&
        (scope === 'workspace' || binding.projectRoleBindings.length > 0)
      "
      :positive-button-props="{
        type: 'error',
      }"
      @positive-click="$emit('revoke-binding')"
    >
      <template #trigger>
        <MiniActionButton @click.stop type="error">
          <Trash2Icon />
        </MiniActionButton>
      </template>

      <template #default>
        <div>
          {{ $t("settings.members.revoke-access-alert") }}
        </div>
      </template>
    </NPopconfirm>

    <MiniActionButton
      v-if="allowUpdate"
      @click="$emit('update-binding')"
    >
      <PencilIcon />
    </MiniActionButton>
  </div>
</template>

<script lang="ts" setup>
import { PencilIcon, Trash2Icon } from "lucide-vue-next";
import { NPopconfirm } from "naive-ui";
import { computed } from "vue";
import { MiniActionButton } from "@/components/v2";
import { SYSTEM_BOT_USER_NAME, unknownUser } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import type { MemberBinding } from "../../types";

const props = defineProps<{
  scope: "workspace" | "project";
  allowEdit: boolean;
  binding: MemberBinding;
}>();

defineEmits<{
  (event: "update-binding"): void;
  (event: "revoke-binding"): void;
}>();

const allowRevoke = computed(() => {
  if (props.binding.type === "groups") {
    return true;
  }
  const user = props.binding.user ?? unknownUser();
  return user.name !== SYSTEM_BOT_USER_NAME;
});

const allowUpdate = computed(() => {
  if (props.binding.type === "groups") {
    return props.allowEdit && !props.binding.group?.deleted;
  }

  const user = props.binding.user ?? unknownUser();
  if (user.name === SYSTEM_BOT_USER_NAME) {
    // Cannot edit the member binding for support@bytebase.com, but can edit allUsers
    return false;
  }
  return user.state === State.ACTIVE;
});
</script>
