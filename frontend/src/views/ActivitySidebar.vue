<template>
  <div class="pl-6 pt-6 pb-2">
    <h2 class="text-sm leading-5 font-semibold">Activity</h2>
  </div>
  <div>
    <ul class="divide-y divide-block-border pr-4">
      <li class="py-4" v-for="item in activityList" :key="item.id">
        <div class="flex space-x-3">
          <img
            class="h-6 w-6 rounded-full"
            src="../assets/avatar.jpeg"
            alt=""
          />
          <div class="flex-1 space-y-1">
            <div class="flex items-center justify-between">
              <h3 class="text-sm font-medium leading-5">
                {{ item.attributes.creator }}
              </h3>
              <p class="text-sm leading-5 text-gray-500">
                {{ humanize(item.attributes.endTs) }}
              </p>
            </div>
            <p class="text-sm leading-5 text-gray-500">
              {{ item.attributes.description }}
            </p>
          </div>
        </div>
      </li>

      <!-- More items... -->
    </ul>
    <div class="pl-6 py-4 text-sm leading-5 border-t border-block-border">
      <a href="#" class="link">View all activity &rarr;</a>
    </div>
  </div>
</template>

<script lang="ts">
import moment from "moment";
import { watchEffect, computed, inject } from "vue";
import { useStore } from "vuex";
import { UserStateSymbol } from "../components/ProvideUser.vue";
import { User } from "../types";

export default {
  name: "ActivitySidebar",
  props: {},
  components: {},
  setup(props, ctx) {
    const store = useStore();

    const currentUser = inject<User>(UserStateSymbol);
    const prepareActivityList = () => {
      store
        .dispatch("activity/fetchActivityListForUser", currentUser!.id)
        .catch((error) => {
          console.log(error);
        });
    };

    const activityList = computed(() =>
      store.getters["activity/activityListByUser"](currentUser!.id)
    );

    watchEffect(prepareActivityList);

    return {
      currentUser,
      activityList,
    };
  },
  methods: {
    humanize(ts: number) {
      return moment().from(ts);
    },
  },
};
</script>
