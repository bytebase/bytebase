<template>
  <div v-if="allowAddOrInvite" class="w-full flex justify-center mb-6">
    <MemberAddOrInvite />
  </div>
  <div>
    <p class="text-lg font-medium leading-7 text-main">
      {{ $t("settings.members.active") }}
    </p>
    <MemberTable :member-list="activeMemberList" />
    <div v-if="inactiveMemberList.length > 0" class="mt-8">
      <p class="text-lg font-medium leading-7 text-control-light">
        {{ $t("settings.members.inactive") }}
      </p>
      <MemberTable :member-list="inactiveMemberList" />
    </div>
  </div>
</template>

<script lang="ts">
import { computed, watchEffect } from "vue";
import { useStore } from "vuex";
import MemberAddOrInvite from "../components/MemberAddOrInvite.vue";
import MemberTable from "../components/MemberTable.vue";
import { isOwner } from "../utils";
import { Member } from "../types";

export default {
  name: "SettingWorkspaceMember",
  components: { MemberAddOrInvite, MemberTable },
  setup() {
    const store = useStore();
    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const prepareMemberList = () => {
      store.dispatch("member/fetchMemberList");
    };

    watchEffect(prepareMemberList);

    const activeMemberList = computed(() => {
      return store.getters["member/memberList"]().filter(
        (member: Member) => member.rowStatus == "NORMAL"
      );
    });

    const inactiveMemberList = computed(() => {
      return store.getters["member/memberList"]().filter(
        (member: Member) => member.rowStatus == "ARCHIVED"
      );
    });

    const allowAddOrInvite = computed(() => {
      // TODO(tianzhou): Implement invite mode for DBA and developer
      // If current user is owner, MemberAddOrInvite is in Add mode.
      // If current user is DBA or developer, MemberAddOrInvite is in Invite mode.
      // For now, we only enable Add mode for owner
      return isOwner(currentUser.value.role);
    });

    return {
      activeMemberList,
      inactiveMemberList,
      allowAddOrInvite,
    };
  },
};
</script>
