<template>
  <UserNameCell
    :allow-edit="false"
    :show-mfa-enabled="false"
    :show-source="false"
    :user="user"
    :on-click-user="onClickUser"
  >
    <template #suffix>
      <NTag
        v-if="role"
        size="small"
        round
        class="mx-1"
        :type="role === GroupMember_Role.OWNER ? 'primary' : 'default'"
      >
        {{
          (() => {
            switch (role) {
              case GroupMember_Role.OWNER:
                return $t("settings.members.groups.form.role.owner");
              case GroupMember_Role.MEMBER:
                return $t("settings.members.groups.form.role.member");
              default:
                return "ROLE UNRECOGNIZED";
            }
          })()
        }}
      </NTag>
    </template>
  </UserNameCell>
</template>

<script lang="ts" setup>
import { NTag } from "naive-ui";
import { UserNameCell } from "@/components/v2/Model/cells";
import { GroupMember_Role } from "@/types/proto-es/v1/group_service_pb";
import { type User } from "@/types/proto-es/v1/user_service_pb";

withDefaults(
  defineProps<{
    user: User;
    role?: GroupMember_Role;
    onClickUser?: (user: User, event: MouseEvent) => void;
  }>(),
  {
    role: undefined,
    onClickUser: undefined,
  }
);
</script>
