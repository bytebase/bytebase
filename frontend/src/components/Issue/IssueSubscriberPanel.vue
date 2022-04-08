<template>
  <div
    class="mt-6 border-t border-block-border pt-6 grid gap-y-4 gap-x-6 grid-cols-3"
  >
    <h2
      class="textlabel flex items-center col-span-1 col-start-1 whitespace-nowrap"
    >
      {{ $t("issue.subscriber", subscriberList.length) }}
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
        <heroicons-outline:ban
          v-if="isCurrentUserSubscribed"
          class="h-5 w-5 text-control inline -mt-0.5 mr-1"
        />
        <heroicons-solid:bell
          v-else
          class="h-5 w-5 text-control inline -mt-0.5 mr-1"
        />
        {{
          isCurrentUserSubscribed
            ? $t("issue.unsubscribe")
            : $t("issue.subscribe")
        }}
      </span>
    </button>
  </div>
</template>

<script lang="ts">
import {
  reactive,
  PropType,
  computed,
  watchEffect,
  defineComponent,
} from "vue";
import PrincipalAvatar from "../PrincipalAvatar.vue";
import { Issue, IssueSubscriber } from "../../types";
import { useCurrentUser, useIssueSubscriberStore } from "@/store";

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface LocalState {}

export default defineComponent({
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
    const state = reactive<LocalState>({});
    const issueSubscriberStore = useIssueSubscriberStore();

    const currentUser = useCurrentUser();

    const prepareSubscriberList = () => {
      issueSubscriberStore.fetchSubscriberListByIssue(props.issue.id);
    };

    watchEffect(prepareSubscriberList);

    const subscriberList = computed((): IssueSubscriber[] => {
      return issueSubscriberStore.subscriberListByIssue(props.issue.id);
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
});
</script>
