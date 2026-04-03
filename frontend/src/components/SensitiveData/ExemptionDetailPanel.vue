<template>
  <div class="flex flex-col">
    <!-- Header -->
    <div class="px-4 pt-3 pb-1">
      <div class="flex items-center gap-x-2">
        <template v-if="member.type === 'group'">
          <GroupNameCell
            v-if="group"
            :group="group"
            :show-icon="false"
            :show-name="false"
            :show-member="false"
            :link="false"
          />
          <span v-else class="font-medium">{{ member.member }}</span>
        </template>
        <template v-else>
          <UserLink :title="userEmail" :email="userEmail" />
        </template>
      </div>
      <div class="mt-1 text-sm textinfolabel">
        {{ member.grants.length }}
        {{ $t("project.masking-exemption.self").toLowerCase() }}
      </div>
    </div>

    <!-- Grants as cards -->
    <div class="flex flex-col gap-y-3 px-4 pb-4">
      <div
        v-for="(grant, idx) in member.grants"
        :key="grant.id"
        class="border border-gray-200 rounded-lg overflow-hidden pt-4"
      >
        <ExemptionGrantSection
          :grant="grant"
          :disabled="disabled"
          :show-database-link="showDatabaseLink"
          :default-expanded="shouldExpand(idx)"
          @revoke="$emit('revoke', grant)"
        />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import GroupNameCell from "@/components/User/Settings/UserDataTableByGroup/cells/GroupNameCell.vue";
import { UserLink } from "@/components/v2/Model/cells";
import { extractUserEmail, useGroupStore } from "@/store";
import { groupBindingPrefix } from "@/types";
import ExemptionGrantSection from "./ExemptionGrantSection.vue";
import type { ExemptionGrant, ExemptionMember } from "./types";

const props = withDefaults(
  defineProps<{
    member: ExemptionMember;
    disabled: boolean;
    showDatabaseLink?: boolean;
    databaseFilter?: string;
  }>(),
  {
    showDatabaseLink: true,
    databaseFilter: undefined,
  }
);

defineEmits<{
  (e: "revoke", grant: ExemptionGrant): void;
}>();

const groupStore = useGroupStore(); // NOSONAR

const userEmail = computed(() => extractUserEmail(props.member.member));

const group = computed(() => {
  if (!props.member.member.startsWith(groupBindingPrefix)) return undefined;
  return groupStore.getGroupByIdentifier(props.member.member);
});

const grantMatchesFilter = (grant: ExemptionGrant): boolean => {
  if (!props.databaseFilter) return false;
  return (
    grant.databaseResources?.some(
      (r) => r.databaseFullName === props.databaseFilter
    ) ?? false
  );
};

const shouldExpand = (idx: number): boolean => {
  const grant = props.member.grants[idx];
  if (props.databaseFilter && grantMatchesFilter(grant)) {
    return true;
  }
  if (props.member.grants.length >= 3) {
    return idx === 0;
  }
  return true;
};
</script>
