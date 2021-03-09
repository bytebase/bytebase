<template>
  <div class="w-full flex justify-center">
    <MemberInvite />
  </div>
  <div class="mt-6">
    <p class="text-xl font-bold leading-7 text-main">Current Members</p>
    <MemberTable class="mt-2" />
  </div>
</template>

<script lang="ts">
import MemberInvite from "../components/MemberInvite.vue";
import MemberTable from "../components/MemberTable.vue";

export default {
  name: "SettingWorkspaceMember",
  components: { MemberInvite, MemberTable },
  props: {},
  setup(props, ctx) {},
};
</script>
