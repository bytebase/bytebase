<template>
  <div
    class="
      mt-6
      border-t border-block-border
      pt-6
      grid
      gap-y-4 gap-x-6
      grid-cols-3
    "
  >
    <h2
      class="
        textlabel
        flex
        items-center
        col-span-1 col-start-1
        whitespace-nowrap
      "
    >
      {{
        subscriberList.length +
        (subscriberList.length > 1 ? " subscribers" : " subscriber")
      }}
    </h2>
    <div v-if="subscriberList.length > 0" class="col-span-3 col-start-1">
      <div class="flex space-x-1">
        <template v-for="(subscriber, index) in subscriberList" :key="index">
          <router-link
            :to="`/u/${subscriber.subscriber.id}`"
            class="hover:opacity-75"
          >
            <PrincipalAvatar
              :principal="subscriber.subscriber"
              :size="'SMALL'"
            />
          </router-link>
        </template>
      </div>
    </div>
    <button
      type="button"
      class="btn-normal items-center col-span-3 col-start-1"
      @click.prevent="toggleSubscription"
    >
      <span class="w-full">
        <svg
          class="h-5 w-5 text-control inline -mt-0.5 mr-1"
          fill="currentColor"
          viewBox="0 0 20 20"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path
            v-if="isCurrentUserSubscribed"
            fill-rule="evenodd"
            d="M13.477 14.89A6 6 0 015.11 6.524l8.367 8.368zm1.414-1.414L6.524 5.11a6 6 0 018.367 8.367zM18 10a8 8 0 11-16 0 8 8 0 0116 0z"
            clip-rule="evenodd"
          ></path>
          <path
            v-else
            d="M10 2a6 6 0 00-6 6v3.586l-.707.707A1 1 0 004 14h12a1 1 0 00.707-1.707L16 11.586V8a6 6 0 00-6-6zM10 18a3 3 0 01-3-3h6a3 3 0 01-3 3z"
          ></path></svg
        >{{ isCurrentUserSubscribed ? "Unsubscribe" : "Subscribe" }}</span
      >
    </button>
  </div>
</template>

<script lang="ts">
import { reactive, PropType, computed, watchEffect } from "vue";
import PrincipalAvatar from "../components/PrincipalAvatar.vue";
import { Issue, IssueSubscriber } from "../types";
import { useStore } from "vuex";

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface LocalState {}

export default {
  name: "IssueSubscriberPanel",
  components: { PrincipalAvatar },
  props: {
    issue: {
      required: true,
      type: Object as PropType<Issue>,
    },
  },
  emits: ["add-subscriber-id", "remove-subscriber-id"],
  setup(props, { emit }) {
    const store = useStore();
    const state = reactive<LocalState>({});

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const prepareSubscriberList = () => {
      store.dispatch(
        "issueSubscriber/fetchSubscriberListByIssue",
        props.issue.id
      );
    };

    watchEffect(prepareSubscriberList);

    const subscriberList = computed((): IssueSubscriber[] => {
      return store.getters["issueSubscriber/subscriberListByIssue"](
        props.issue.id
      );
    });

    const isCurrentUserSubscribed = computed((): boolean => {
      for (const subscriber of subscriberList.value) {
        if (currentUser.value.id == subscriber.subscriber.id) {
          return true;
        }
      }
      return false;
    });

    const toggleSubscription = () => {
      if (isCurrentUserSubscribed.value) {
        emit("remove-subscriber-id", currentUser.value.id);
      } else {
        emit("add-subscriber-id", currentUser.value.id);
      }
    };

    return {
      state,
      subscriberList,
      isCurrentUserSubscribed,
      toggleSubscription,
    };
  },
};
</script>
