<template>
  <!-- This example requires Tailwind CSS v2.0+ -->
  <div class="space-y-4">
    <div class="mx-6 space-y-2">
      <div
        class="flex items-center justify-between pb-2 border-b border-block-border"
      >
        <div class="text-lg leading-6 font-medium text-main">
          {{ $t("common.unread") }}
        </div>
        <button type="button" class="btn-normal" @click.prevent="markAllAsRead">
          <heroicons-outline:check
            class="-ml-1 mr-2 h-5 w-5 text-control-light"
          />
          <span>{{ $t("inbox.mark-all-as-read") }}</span>
        </button>
      </div>
      <InboxList :inbox-list="state.unreadList" />
    </div>
    <div class="mt-6 mx-6 space-y-2">
      <div
        class="text-lg leading-6 font-medium text-main pb-2 border-b border-block-border"
      >
        {{ $t("common.read") }}
      </div>
      <InboxList class="opacity-70" :inbox-list="state.readList" />
      <div class="mt-2 flex justify-end">
        <button type="button" class="normal-link" @click.prevent="viewOlder">
          {{ $t("inbox.view-older") }}
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { reactive, watchEffect } from "vue";
import { useRouter } from "vue-router";
import { useCurrentUser, useInboxV1Store } from "@/store";
import { UNKNOWN_ID } from "@/types";
import {
  InboxMessage,
  InboxMessage_Status,
} from "@/types/proto/v1/inbox_service";
import InboxList from "../components/InboxList.vue";

// We alway fetch all "UNREAD" items. But for "READ" items, by default, we only fetch the most recent 7 days.
// And each time clicking "View older" will extend 7 days further.
const READ_INBOX_DURATION_STEP = 3600 * 24 * 7;

interface LocalState {
  readList: InboxMessage[];
  unreadList: InboxMessage[];
  readCreatedAfterTs: number;
}

export default {
  name: "Inbox",
  components: { InboxList },
  setup() {
    const inboxV1Store = useInboxV1Store();
    const router = useRouter();

    const state = reactive<LocalState>({
      readList: [],
      unreadList: [],
      readCreatedAfterTs:
        parseInt(router.currentRoute.value.query.created as string) ||
        Math.ceil(Date.now() / 1000) - READ_INBOX_DURATION_STEP,
    });

    const currentUser = useCurrentUser();

    const prepareInboxList = () => {
      // It will also be called when user logout
      if (currentUser.value.id != UNKNOWN_ID) {
        inboxV1Store.fetchInboxList(state.readCreatedAfterTs).then((list) => {
          state.readList = [];
          state.unreadList = [];

          for (const item of list) {
            if (!item.activity) {
              continue;
            }
            if (item.status == InboxMessage_Status.STATUS_READ) {
              state.readList.push(item);
            } else if (item.status == InboxMessage_Status.STATUS_UNREAD) {
              state.unreadList.push(item);
            }
          }
        });
      }
    };

    watchEffect(prepareInboxList);

    const markAllAsRead = async () => {
      const list = await Promise.all(
        state.unreadList.map(async (item) => {
          item.status = InboxMessage_Status.STATUS_READ;
          // TODO: use batch instead.
          await inboxV1Store.patchInbox(item);
          return item;
        })
      );
      state.readList.push(...list);
      state.unreadList = [];
      await inboxV1Store.fetchInboxSummary();
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
