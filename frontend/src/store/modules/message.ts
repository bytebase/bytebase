import axios from "axios";
import {
  PrincipalId,
  MessageId,
  Message,
  MessageState,
  ResourceObject,
  MessagePatch,
} from "../../types";

function convert(message: ResourceObject, rootGetters: any): Message {
  const creator = rootGetters["principal/principalById"](
    message.attributes.creatorId
  );
  const updater = rootGetters["principal/principalById"](
    message.attributes.updaterId
  );
  const receiver = rootGetters["principal/principalById"](
    message.attributes.receiverId
  );

  return {
    ...(message.attributes as Omit<
      Message,
      "id" | "creator" | "updater" | "receiver"
    >),
    id: parseInt(message.id),
    creator,
    updater,
    receiver,
  };
}

const state: () => MessageState = () => ({
  messageListByUser: new Map(),
});

const getters = {
  messageListByUser:
    (state: MessageState) =>
    (userId: PrincipalId): Message[] => {
      return state.messageListByUser.get(userId) || [];
    },
};

const actions = {
  async fetchMessageListByUser(
    { commit, rootGetters }: any,
    userId: PrincipalId
  ) {
    const messageList = (
      await axios.get(`/api/message?user=${userId}`)
    ).data.data.map((message: ResourceObject) => {
      return convert(message, rootGetters);
    });

    commit("setMessageListByUser", { userId, messageList });
    return messageList;
  },

  async patchMessage(
    { commit, rootGetters }: any,
    {
      messageId,
      messagePatch,
    }: { messageId: MessageId; messagePatch: MessagePatch }
  ) {
    const updatedMessage = convert(
      (
        await axios.patch(`/api/message/${messageId}`, {
          data: {
            type: "messagePatch",
            attributes: messagePatch,
          },
        })
      ).data.data,
      rootGetters
    );

    commit("updateMessageById", { messageId, message: updatedMessage });

    return updatedMessage;
  },
};

const mutations = {
  setMessageListByUser(
    state: MessageState,
    {
      userId,
      messageList,
    }: {
      userId: PrincipalId;
      messageList: Message[];
    }
  ) {
    state.messageListByUser.set(userId, messageList);
  },

  updateMessageById(
    state: MessageState,
    {
      messageId,
      message,
    }: {
      messageId: MessageId;
      message: Message;
    }
  ) {
    for (let [_, messageList] of state.messageListByUser) {
      const i = messageList.findIndex((item: Message) => item.id == messageId);
      if (i >= 0) {
        messageList[i] = message;
      }
    }
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
