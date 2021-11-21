<template>
  <!-- This example requires Tailwind CSS v2.0+ -->
  <div class="space-y-4">
    <div class="mx-6 space-y-2">
      <div
        class="
          flex
          items-center
          justify-between
          pb-2
          border-b border-block-border
        "
      >
        <div class="text-lg leading-6 font-medium text-main">Unread</div>
        <button type="button" class="btn-normal" @click.prevent="markAllAsRead">
          <svg
            class="-ml-1 mr-2 h-5 w-5 text-control-light"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M5 13l4 4L19 7"
            ></path>
          </svg>
          <span>Mark all as read</span>
        </button>
      </div>
      <InboxList :inboxList="state.unreadList" />
    </div>
    <div class="mt-6 mx-6 space-y-2">
      <div
        class="
          text-lg
          leading-6
          font-medium
          text-main
          pb-2
          border-b border-block-border
        "
      >
        Read
      </div>
      <InboxList class="opacity-70" :inboxList="state.readList" />
      <div class="mt-2 flex justify-end">
        <button type="button" class="normal-link" @click.prevent="viewOlder">
          View older
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, reactive, watchEffect } from "vue";
import { useStore } from "vuex";
import InboxList from "../components/InboxList.vue";
import { Inbox, UNKNOWN_ID } from "../types";
import { isDBAOrOwner } from "../utils";
import { useRouter } from "vue-router";

// We alway fetch all "UNREAD" items. But for "READ" items, by default, we only fetch the most recent 7 days.
// And each time clicking "View older" will extend 7 days further.
const READ_INBOX_DURATION_STEP = 3600 * 24 * 7;

interface LocalState {
  readList: Inbox[];
  unreadList: Inbox[];
  readCreatedAfterTs: number;
}

export default {
  name: "Inbox",
  components: { InboxList },
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      readList: [],
      unreadList: [],
      readCreatedAfterTs:
        parseInt(router.currentRoute.value.query.created as string) ||
        Math.ceil(Date.now() / 1000) - READ_INBOX_DURATION_STEP,
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const prepareInboxList = () => {
      // It will also be called when user logout
      if (currentUser.value.id != UNKNOWN_ID) {
        store
          .dispatch("inbox/fetchInboxListByUser", {
            userID: currentUser.value.id,
            readCreatedAfterTs: state.readCreatedAfterTs,
          })
          .then((list: Inbox[]) => {
            state.readList = [];
            state.unreadList = [];

            for (const item of list) {
              if (item.status == "READ") {
                state.readList.push(item);
              } else if (item.status == "UNREAD") {
                state.unreadList.push(item);
              }
            }
          });
      }
    };

    watchEffect(prepareInboxList);

    const markAllAsRead = () => {
      var count = state.unreadList.length;
      const inboxList = state.unreadList.map((item) => item);

      inboxList.forEach((item: Inbox) => {
        store
          .dispatch("inbox/patchInbox", {
            inboxID: item.id,
            inboxPatch: {
              status: "READ",
            },
          })
          .then(() => {
            const i = state.unreadList.findIndex(
              (unreadItem) => unreadItem.id == item.id
            );
            if (i >= 0) {
              state.unreadList.splice(i, 1);
            }
            count--;
            if (count == 0) {
              store.dispatch(
                "inbox/fetchInboxSummaryByUser",
                currentUser.value.id
              );
            }
            state.readList.push(item);
          });
      });
    };

    const viewOlder = () => {
      state.readCreatedAfterTs -= READ_INBOX_DURATION_STEP;
      router
        .replace({
          name: "workspace.inbox",
          query: {
            ...router.currentRoute.value.query,
            created: state.readCreatedAfterTs,
          },
        })
        .then(() => {
          prepareInboxList();
        });
    };

    return {
      state,
      markAllAsRead,
      viewOlder,
    };
  },
};
</script>
