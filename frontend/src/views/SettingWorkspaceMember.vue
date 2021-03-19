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
import { useStore } from "vuex";
import MemberInvite from "../components/MemberInvite.vue";
import MemberTable from "../components/MemberTable.vue";

export default {
  name: "SettingWorkspaceMember",
  components: { MemberInvite, MemberTable },
  props: {},
  setup(props, ctx) {
    const store = useStore();
    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const allowInvite = computed(() => {
      return currentUser.value.role == "OWNER";
    });

    return {
      allowInvite,
    };
  },
};
</script>
