<template>
  <!-- This example requires Tailwind CSS v2.0+ -->
  <div class="my-2 space-y-2 divide-y divide-block-border">
    <BBTabFilter
      v-if="isCurrentUserDBAOrOwner"
      class="mx-2"
      :tabList="['General', 'Membership']"
      :selectedIndex="state.selectedIndex"
      @select-index="
        (index) => {
          state.selectedIndex = index;
        }
      "
    />
    <div>
      <div class="px-4 py-2 flex justify-between">
        <BBSwitch
          :label="'Display all messages'"
          :value="state.showAll"
          @toggle="
            (on) => {
              showAll(on);
            }
          "
        />
        <button type="button" class="btn-normal" @click.prevent="markAllAsRead">
          <svg
            class="-ml-1 mr-2 h-5 w-5 text-control-light"
            fill="currentColor"
            viewBox="0 0 20 20"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z"
            ></path>
          </svg>
          <span>Mark all as read</span>
        </button>
      </div>
      <ul class="divide-y divide-block-border">
        <li
          v-for="(inbox, index) in effectiveInboxList"
          :key="index"
          class="p-3 hover:bg-control-bg-hover cursor-default"
          :class="
            actionLink(inbox.activity) ? 'cursor-pointer' : 'cursor-default'
          "
          @click.prevent="clickInbox(inbox)"
        >
          <div class="flex space-x-3">
            <PrincipalAvatar
              :principal="inbox.activity.creator"
              :size="'SMALL'"
              :class="inbox.activity.comment ? '' : '-mt-0.5'"
            />
            <div class="flex-1 space-y-1">
              <div class="flex w-full items-center justify-between space-x-2">
                <h3
                  class="
                    text-sm
                    font-base
                    text-control-light
                    flex flex-row
                    items-center
                    whitespace-nowrap
                  "
                >
                  <template v-if="showCreator(inbox.activity)">
                    <router-link
                      :to="`/u/${inbox.activity.creator.id}`"
                      class="mr-1 font-medium text-main hover:underline"
                      >{{ inbox.activity.creator.name }}</router-link
                    >
                  </template>
                  <span> {{ actionSentence(inbox.activity) }}</span>
                  <span
                    v-if="inbox.status == 'UNREAD'"
                    class="ml-2 h-3 w-3 rounded-full bg-accent"
                  ></span>
                </h3>
                <p class="text-sm text-control">
                  {{ humanizeTs(inbox.activity.createdTs) }}
                </p>
              </div>
              <div v-if="inbox.activity.comment" class="text-sm text-control">
                {{ inbox.activity.comment }}
              </div>
            </div>
          </div>
        </li>
        <!-- More items... -->
      </ul>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, onMounted, reactive, watchEffect } from "vue";
import { useStore } from "vuex";
import PrincipalAvatar from "../components/PrincipalAvatar.vue";
import {
  ActionIssueCommentCreatePayload,
  ActionIssueCreatePayload,
  ActionIssueFieldUpdatePayload,
  ActionIssueStatusUpdatePayload,
  ActionTaskStatusUpdatePayload,
  Activity,
  Inbox,
  UNKNOWN_ID,
} from "../types";
import { isDBAOrOwner, issueActivityActionSentence, issueSlug } from "../utils";
import { isEmpty } from "lodash";
import { useRouter } from "vue-router";
import activity from "../store/modules/activity";

const GENERAL_TAB = 0;
const MEMBERSHIP_TAB = 1;

interface LocalState {
  selectedIndex: number;
  showAll: boolean;
  inboxList: Inbox[];
  // To maintain a stable view when user clicks an item.
  // When user clicks an item, we will set the item as "CONSUMED".
  // Without this logic, if the user stays on the display unread item view,
  // that item will suddenly be removed from the list, which is not ideal for UX.
  whitelist: Inbox[];
}

export default {
  name: "Inbox",
  components: { PrincipalAvatar },
  setup(props, ctx) {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      selectedIndex: 0,
      showAll: false,
      inboxList: [],
      whitelist: [],
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const prepareInboxList = () => {
      // It will also be called when user logout
      if (currentUser.value.id != UNKNOWN_ID) {
        store
          .dispatch("inbox/fetchInboxListByUser", currentUser.value.id)
          .then((list: Inbox[]) => {
            state.inboxList = list;
          });
      }
    };

    watchEffect(prepareInboxList);

    onMounted(() => {
      state.whitelist = [];
    });

    const isCurrentUserDBAOrOwner = computed((): boolean => {
      return isDBAOrOwner(currentUser.value.role);
    });

    const effectiveInboxList = computed(() => {
      return state.inboxList.filter((inbox: Inbox) => {
        if (
          (state.selectedIndex == GENERAL_TAB &&
            inbox.activity.actionType.startsWith("bb.inbox.member.")) ||
          (state.selectedIndex == MEMBERSHIP_TAB &&
            !inbox.activity.actionType.startsWith("bb.inbox.member."))
        ) {
          return false;
        }

        return (
          state.showAll ||
          inbox.status == "UNREAD" ||
          state.whitelist.find((item: Inbox) => {
            return item.id == inbox.id;
          })
        );
      });
    });

    const showCreator = (activity: Activity): boolean => {
      return activity.actionType.startsWith("bb.issue.");
    };

    const actionSentence = (activity: Activity): string => {
      if (activity.actionType.startsWith("bb.issue.")) {
        const actionStr = issueActivityActionSentence(activity);
        switch (activity.actionType) {
          case "bb.issue.create": {
            const payload = activity.payload as ActionIssueCreatePayload;
            return `${actionStr} - '${payload?.issueName || ""}'`;
          }
          case "bb.issue.comment.create": {
            const payload = activity.payload as ActionIssueCommentCreatePayload;
            return `${actionStr} - '${payload?.issueName || ""}'`;
          }
          case "bb.issue.field.update": {
            const payload = activity.payload as ActionIssueFieldUpdatePayload;
            return `${actionStr} - '${payload?.issueName || ""}'`;
          }
          case "bb.issue.status.update": {
            const payload = activity.payload as ActionIssueStatusUpdatePayload;
            return `${actionStr} - '${payload?.issueName || ""}'`;
          }
        }
        return actionStr;
      } else if (activity.actionType == "bb.pipeline.task.status.update") {
        const payload = activity.payload as ActionTaskStatusUpdatePayload;
        var actionStr = `changed`;
        switch (payload.newStatus) {
          case "PENDING": {
            if (payload.oldStatus == "RUNNING") {
              actionStr = `canceled`;
            } else if (payload.oldStatus == "PENDING_APPROVAL") {
              actionStr = `approved`;
            }
            break;
          }
          case "RUNNING": {
            actionStr = `started`;
            break;
          }
          case "DONE": {
            actionStr = `completed`;
            break;
          }
          case "FAILED": {
            actionStr = `failed`;
            break;
          }
        }
        return `Task '${payload.taskName}' ${actionStr} - '${
          payload?.issueName || ""
        }'`;
      }

      return "";
    };

    const actionLink = (activity: Activity): string => {
      if (activity.actionType.startsWith("bb.issue.")) {
        return `/issue/${activity.containerId}`;
      } else if (activity.actionType == "bb.pipeline.task.status.update") {
        const payload = activity.payload as ActionTaskStatusUpdatePayload;
        return `/issue/${activity.containerId}?task=${payload.taskId}`;
      }

      return "";
    };

    const clickInbox = (inbox: Inbox) => {
      if (inbox.status == "UNREAD") {
        state.whitelist.push(inbox);
        store.dispatch("inbox/patchInbox", {
          inboxId: inbox.id,
          inboxPatch: {
            status: "READ",
          },
        });
      }
      const link = actionLink(inbox.activity);
      if (!isEmpty(link)) {
        router.push(link);
      }
    };

    const showAll = (show: boolean) => {
      state.whitelist = [];
      state.showAll = show;
    };

    const markAllAsRead = () => {
      state.inboxList.forEach((item: Inbox) => {
        if (item.status == "UNREAD") {
          state.whitelist.push(item);
          store.dispatch("inbox/patchInbox", {
            inboxId: item.id,
            inboxPatch: {
              status: "READ",
            },
          });
        }
      });
    };

    return {
      state,
      currentUser,
      isCurrentUserDBAOrOwner,
      effectiveInboxList,
      showCreator,
      actionSentence,
      actionLink,
      clickInbox,
      showAll,
      markAllAsRead,
    };
  },
};
</script>
