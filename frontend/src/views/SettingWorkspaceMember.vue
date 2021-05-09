<template>
  <div v-if="allowInvite" class="w-full flex justify-center mb-6">
    <MemberInvite />
  </div>
  <div>
    <MemberTable />
  </div>
</template>

<script lang="ts">
import { computed } from "vue";
import MemberInvite from "../components/MemberInvite.vue";
import MemberTable from "../components/MemberTable.vue";

export default {
  name: "SettingWorkspaceMember",
  components: { MemberInvite, MemberTable },
  props: {},
  setup(props, ctx) {
    // TODO: Disable invite for now as it requires mail server
    const allowInvite = computed(() => {
      return false;
    });

    return {
      allowInvite,
    };
  },
};
</script>
