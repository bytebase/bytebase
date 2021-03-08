<template>
  <div class="mx-4 flex justify-center">
    <MemberInvite />
  </div>
</template>

<script lang="ts">
import MemberInvite from "../components/MemberInvite.vue";

export default {
  name: "SettingWorkspaceMember",
  components: { MemberInvite },
  props: {},
  setup(props, ctx) {},
};
</script>
