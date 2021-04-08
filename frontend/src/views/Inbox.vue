<template>
  <!-- This example requires Tailwind CSS v2.0+ -->
  <div class="mx-4">
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
        <!-- Heroicon name: solid/pencil -->
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
        v-for="(message, index) in effectiveMessageList"
        :key="index"
        class="p-4 hover:bg-control-bg-hover cursor-pointer"
        @click.prevent="clickItem(message)"
      >
        <div class="flex items-top space-x-3">
          <BBAvatar :username="message.creator.name" />
          <div class="flex-1 space-y-1">
            <div class="flex items-center justify-between">
              <h3 class="text-sm text-control flex flex-row">
                {{ message.creator.name }}
                <span class="text-sm font-medium text-main ml-1">{{
                  message.name
                }}</span>
                <span
                  v-if="message.status == 'DELIVERED'"
                  class="ml-2 mt-1 h-3 w-3 rounded-full bg-accent"
                ></span>
              </h3>
              <p class="text-sm text-control">
                {{ humanizeTs(message.createdTs) }}
              </p>
            </div>
            <div class="text-sm text-control">
              {{ message.description }}
            </div>
          </div>
        </div>
      </li>
      <!-- More items... -->
    </ul>
  </div>
</template>

<script lang="ts">
import { computed, onMounted, reactive, watchEffect } from "vue";
import { useStore } from "vuex";
import { Message } from "../types";

interface LocalState {
  showAll: boolean;
  messageList: Message[];
  // To maintain a stable view when user clicks an item.
  // When user clicks an item, we will set the item as "CONSUMED".
  // Without this logic, if the user stays on the display unread item view,
  // that item will suddenly be removed from the list, which is not ideal for UX.
  whitelist: Message[];
}

export default {
  name: "Inbox",
  components: {},
  setup(props, ctx) {
    const store = useStore();

    const state = reactive<LocalState>({
      showAll: false,
      messageList: [],
      whitelist: [],
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const prepareMessageList = () => {
      store
        .dispatch("message/fetchMessageListByUser", currentUser.value.id)
        .then((list: Message[]) => {
          state.messageList = list;
        })
        .catch((error) => {
          console.log(error);
        });
    };

    watchEffect(prepareMessageList);

    onMounted(() => {
      state.whitelist = [];
    });

    const effectiveMessageList = computed(() => {
      return state.showAll
        ? state.messageList
        : state.messageList.filter((message: Message) => {
            return (
              message.status == "DELIVERED" ||
              state.whitelist.find((item: Message) => {
                return item.id == message.id;
              })
            );
          });
    });

    const clickItem = (item: Message) => {
      if (item.status == "DELIVERED") {
        state.whitelist.push(item);
        store.dispatch("message/updateStatus", {
          messageId: item.id,
          updatedStatus: "CONSUMED",
        });
      }
    };

    const showAll = (show: boolean) => {
      state.whitelist = [];
      state.showAll = show;
    };

    const markAllAsRead = () => {
      state.messageList.forEach((item: Message) => {
        if (item.status == "DELIVERED") {
          state.whitelist.push(item);
          store.dispatch("message/updateStatus", {
            messageId: item.id,
            updatedStatus: "CONSUMED",
          });
        }
      });
    };

    return {
      state,
      effectiveMessageList,
      clickItem,
      showAll,
      markAllAsRead,
    };
  },
};
</script>
