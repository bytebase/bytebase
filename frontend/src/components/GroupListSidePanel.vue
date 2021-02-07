<template>
  <h3
    class="px-3 text-xs leading-4 font-semibold text-control-light uppercase tracking-wider"
    id="group-headline"
  >
    Groups
  </h3>
  <GroupSidePanel v-for="item in groupList" :key="item.id" :group="item" />
</template>

<script lang="ts">
import { watchEffect, computed, inject } from "vue";
import { useStore } from "vuex";
import { UserStateSymbol } from "./ProvideUser.vue";
import GroupSidePanel from "./GroupSidePanel.vue";
import { User } from "../types";

export default {
  name: "GroupListSidePanel",
  props: {},
  components: {
    GroupSidePanel,
  },
  setup(props, ctx) {
    const store = useStore();

    const currentUser = inject<User>(UserStateSymbol);

    const prepareGroupList = () => {
      store
        .dispatch("group/fetchGroupListForUser", currentUser!.id)
        .catch((error) => {
          console.log(error);
        });
    };

    const groupList = computed(() =>
      store.getters["group/groupListByUser"](currentUser!.id)
    );

    watchEffect(prepareGroupList);

    return {
      groupList,
    };
  },
};
</script>
