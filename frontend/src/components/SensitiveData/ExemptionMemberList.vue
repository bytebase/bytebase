<template>
  <div class="overflow-y-auto">
    <template v-if="members.length > 0">
      <div class="divide-y divide-gray-100">
      <div v-for="member in members" :key="member.member">
        <ExemptionMemberItem
          :member="member"
          :selected="!expandable && effectiveSelectedKey === member.member"
          :expandable="expandable"
          :expanded="expandable && expandedMemberKey === member.member"
          @select="handleSelect(member)"
          @toggle="handleToggle(member)"
        />
        <!-- Inline detail panel in narrow/expandable mode -->
        <div
          v-if="expandable && expandedMemberKey === member.member"
          class="border-t border-b border-gray-200"
        >
          <ExemptionDetailPanel
            :member="member"
            :disabled="disabled"
            :show-database-link="showDatabaseLink"
            :database-filter="databaseFilter"
            @revoke="(grant) => $emit('revoke', member, grant)"
          />
        </div>
      </div>
      </div>
    </template>
    <div
      v-else-if="loading"
      class="flex items-center justify-center py-12"
    >
      <NSpin size="small" />
    </div>
    <div
      v-else
      class="flex items-center justify-center py-12 text-control-placeholder text-sm"
    >
      {{ $t("project.masking-exemption.no-exemptions") }}
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NSpin } from "naive-ui";
import { computed, ref, watch } from "vue";
import ExemptionDetailPanel from "./ExemptionDetailPanel.vue";
import ExemptionMemberItem from "./ExemptionMemberItem.vue";
import type { ExemptionGrant, ExemptionMember } from "./types";

const props = withDefaults(
  defineProps<{
    members: ExemptionMember[];
    disabled: boolean;
    loading?: boolean;
    expandable?: boolean;
    showDatabaseLink?: boolean;
    databaseFilter?: string;
    selectedMemberKey?: string;
  }>(),
  {
    loading: false,
    expandable: false,
    showDatabaseLink: true,
    databaseFilter: undefined,
    selectedMemberKey: "",
  }
);

const emit = defineEmits<{
  (e: "select", memberKey: string): void;
  (e: "revoke", member: ExemptionMember, grant: ExemptionGrant): void;
}>();

const expandedMemberKey = ref("");

// In wide mode, use the parent-controlled prop as the source of truth.
const effectiveSelectedKey = computed(() => props.selectedMemberKey);

const handleSelect = (member: ExemptionMember) => {
  emit("select", member.member);
};

const handleToggle = (member: ExemptionMember) => {
  expandedMemberKey.value =
    expandedMemberKey.value === member.member ? "" : member.member;
};

// Auto-select first member in wide mode
watch(
  () => props.members,
  (members) => {
    if (!props.expandable && members.length > 0) {
      if (
        !props.selectedMemberKey ||
        !members.some((m) => m.member === props.selectedMemberKey)
      ) {
        emit("select", members[0].member);
      }
    }
  },
  { immediate: true }
);
</script>
