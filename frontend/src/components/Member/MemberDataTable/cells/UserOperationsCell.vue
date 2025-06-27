<template>
  <div v-if="allowEdit" class="flex justify-end">
    <NPopconfirm
      v-if="
        allowRevoke &&
        (scope === 'workspace' || binding.projectRoleBindings.length > 0)
      "
      @positive-click="$emit('revoke-binding')"
    >
      <template #trigger>
        <NButton quaternary circle @click.stop>
          <template #icon>
            <Trash2Icon class="w-4 h-auto" />
          </template>
        </NButton>
      </template>

      <template #default>
        <div>
          {{ $t("settings.members.revoke-access-alert") }}
        </div>
      </template>
    </NPopconfirm>

    <NButton
      v-if="allowUpdate"
      quaternary
      circle
      @click="$emit('update-binding')"
    >
      <template #icon>
        <PencilIcon class="w-4 h-auto" />
      </template>
    </NButton>
  </div>
</template>

<script lang="ts" setup>
import { PencilIcon, Trash2Icon } from "lucide-vue-next";
import { NButton, NPopconfirm } from "naive-ui";
import { computed } from "vue";
import { unknownUser, SYSTEM_BOT_USER_NAME } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import { convertStateToNew } from "@/utils/v1/common-conversions";
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
  return convertStateToNew(user.state) === State.ACTIVE;
});
</script>
