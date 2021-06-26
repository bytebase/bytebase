<template>
  <div v-if="allowAddOrInvite" class="w-full flex justify-center mb-6">
    <MemberAddOrInvite />
  </div>
  <div>
    <MemberTable />
  </div>
</template>

<script lang="ts">
import { computed } from "vue";
import { useStore } from "vuex";
import MemberAddOrInvite from "../components/MemberAddOrInvite.vue";
import MemberTable from "../components/MemberTable.vue";
import { isOwner } from "../utils";

export default {
  name: "SettingWorkspaceMember",
  components: { MemberAddOrInvite, MemberTable },
  props: {},
  setup(props, ctx) {
    const store = useStore();
    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const allowAddOrInvite = computed(() => {
      // TODO: Enable for DBA and developer
      // If current user is owner, MemberAddOrInvite is in Add mode.
      // If current user is DBA or developer, MemberAddOrInvite is in Invite mode.
      // For now, we only enable Add mode for owner
      return isOwner(currentUser.value.role);
    });

    return {
      allowAddOrInvite,
    };
  },
};
</script>
