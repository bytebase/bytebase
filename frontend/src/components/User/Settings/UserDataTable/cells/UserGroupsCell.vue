<template>
  <div class="flex flex-row items-center flex-wrap gap-2">
    <NTag
      v-for="group in userGroups"
      :key="group.name"
      class="!cursor-pointer hover:bg-gray-200"
      @click="$emit('select-group', group)"
    >
      {{ group.title }}
    </NTag>
    <span v-if="userGroups.length === 0">-</span>
  </div>
</template>

<script lang="ts" setup>
import { NTag } from "naive-ui";
import { computed } from "vue";
import { useUserGroupStore } from "@/store";
import { getUserEmailFromIdentifier } from "@/store/modules/v1/common";
import type { ComposedUser } from "@/types";
import type { UserGroup } from "@/types/proto/v1/user_group";

const props = defineProps<{
  user: ComposedUser;
}>();

defineEmits<{
  (event: "select-group", group: UserGroup): void;
}>();

const groupStore = useUserGroupStore();

const userGroups = computed(() => {
  const groups = [];
  for (const group of groupStore.groupList) {
    for (const member of group.members) {
      if (getUserEmailFromIdentifier(member.member) === props.user.email) {
        groups.push(group);
        break;
      }
    }
  }
  return groups;
});
</script>
