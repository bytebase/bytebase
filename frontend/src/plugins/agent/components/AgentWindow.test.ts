import { mount } from "@vue/test-utils";
import { NInput, NPopconfirm } from "naive-ui";
import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { nextTick } from "vue";
import { createI18n } from "vue-i18n";
import { useAgentStore } from "../store/agent";

vi.mock("./AgentChat.vue", () => ({
  default: {
    name: "AgentChat",
    template: "<div data-agent-chat />",
  },
}));

vi.mock("./AgentInput.vue", () => ({
  default: {
    name: "AgentInput",
    template: "<div data-agent-input />",
  },
}));

import AgentWindow from "./AgentWindow.vue";

const mockRoute = vi.hoisted(() => ({ fullPath: "/projects/demo" }));

vi.mock("vue-router", () => ({
  useRoute: () => mockRoute,
}));

class ResizeObserverMock {
  observe() {}
  disconnect() {}
  unobserve() {}
}

const i18n = createI18n({
  legacy: false,
  locale: "en-US",
  messages: {
    "en-US": {
      agent: {
        "assistant-title": "Bytebase Assistant",
        minimize: "Minimize",
        close: "Close",
        resize: "Resize",
        "chat-list-label": "Chats",
        "chat-switch-locked":
          "Finish the running chat before switching to another one.",
        "new-chat": "New chat",
        "rename-chat-placeholder": "Enter a chat name",
        "archive-chat": "Archive",
        "unarchive-chat": "Unarchive",
        "delete-chat": "Delete",
        "show-archived-chats": "Show archived chats",
        "hide-archived-chats": "Hide archived chats",
        "archive-chat-confirmation": "Archive this chat?",
        "delete-chat-confirmation": "Delete this chat?",
        "chat-default-title": "New chat",
        "chat-archived-label": "Archived",
        "chat-status-idle": "Idle",
        "chat-status-running": "Running",
        "chat-status-awaiting-user": "Awaiting input",
        "chat-status-error": "Error",
        "chat-status-interrupted": "Interrupted",
        "chat-total-tokens": "Tokens used: {count}",
      },
    },
  },
});

const mountAgentWindow = () => {
  const pinia = createPinia();
  setActivePinia(pinia);
  const store = useAgentStore();
  store.visible = true;
  return {
    store,
    wrapper: mount(AgentWindow, {
      global: {
        plugins: [pinia, i18n],
        components: {
          NInput,
          NPopconfirm,
        },
        stubs: {
          Teleport: true,
        },
      },
    }),
  };
};

describe("AgentWindow", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.stubGlobal("ResizeObserver", ResizeObserverMock);
    document.title = "Demo Page";
  });

  test("renders chats in a left sidebar instead of a select picker", async () => {
    const { store, wrapper } = mountAgentWindow();
    const firstChatId = store.currentChatId!;
    store.addMessage({
      chatId: firstChatId,
      role: "user",
      content: "Summarize the production incident timeline",
    });
    const secondChat = store.createChat({
      title: "Renamed thread",
      select: false,
    });

    await wrapper.vm.$nextTick();

    expect(wrapper.find("select").exists()).toBe(false);
    const chatRows = wrapper.findAll("[data-agent-chat-list] button");
    expect(chatRows).toHaveLength(2);
    expect(chatRows[0].text()).toContain("Renamed thread");
    expect(chatRows[1].text()).toContain(
      "Summarize the production incident timeline"
    );
    await chatRows[0].trigger("click");
    expect(store.currentChatId).toBe(secondChat.id);
    expect(
      wrapper.findAll("button").some((button) => button.text() === "Rename")
    ).toBe(false);
  });

  test("disables switching chats while a chat is running", async () => {
    const { store, wrapper } = mountAgentWindow();
    const firstChatId = store.currentChatId!;
    const secondChat = store.createChat({
      title: "Second thread",
      select: false,
    });

    store.startChatRun(firstChatId, {
      path: "/projects/demo",
      title: "Demo Page",
    });
    await wrapper.vm.$nextTick();

    const chatRows = wrapper.findAll("[data-agent-chat-list] button");
    const currentRow = chatRows.find(
      (row) => row.attributes("aria-current") === "true"
    );
    const otherRow = chatRows.find((row) =>
      row.text().includes("Second thread")
    );
    const newChatButton = wrapper
      .findAll("button")
      .find((button) => button.text() === "New chat");

    expect(wrapper.find("[data-agent-chat-lock-message]").text()).toContain(
      "Finish the running chat before switching to another one."
    );
    expect(currentRow?.attributes("disabled")).toBeUndefined();
    expect(otherRow?.attributes("disabled")).toBe("");
    expect(newChatButton?.attributes("disabled")).toBe("");

    await otherRow?.trigger("click");
    expect(store.currentChatId).toBe(firstChatId);
    expect(store.canSelectChat(secondChat.id)).toBe(false);
  });

  test("renames the current chat with inline editing on the selected row", async () => {
    const { store, wrapper } = mountAgentWindow();
    const currentRowButton = wrapper.find(
      "[data-agent-chat-list] button[aria-current='true']"
    );
    const currentChatId = store.currentChatId!;

    await currentRowButton.trigger("click");
    await nextTick();

    const currentRow = wrapper.find(`[data-agent-chat-row='${currentChatId}']`);
    const input = currentRow.find("input");
    expect(input.exists()).toBe(true);
    expect(currentRow.findComponent(NInput).exists()).toBe(true);
    await input.setValue("  Renamed thread in app  ");
    await input.trigger("keydown", { key: "Enter" });
    await nextTick();

    expect(store.currentChat?.title).toBe("Renamed thread in app");
    expect(wrapper.find("input").exists()).toBe(false);
  });

  test("cancels inline rename on escape", async () => {
    const { store, wrapper } = mountAgentWindow();
    const originalTitle = store.currentChat?.title;
    const currentRow = wrapper.find(
      "[data-agent-chat-list] button[aria-current='true']"
    );

    await currentRow.trigger("click");
    await nextTick();

    const input = wrapper.find("input");
    await input.setValue("Should not save");
    await input.trigger("keydown", { key: "Escape" });
    await nextTick();

    expect(store.currentChat?.title).toBe(originalTitle);
    expect(wrapper.find("input").exists()).toBe(false);
  });

  test("resizes and persists the sidebar width", async () => {
    const { store, wrapper } = mountAgentWindow();
    store.size.width = 760;
    store.sidebarWidth = 240;

    await nextTick();

    const sidebar = wrapper.find("aside");
    expect(sidebar.attributes("style")).toContain("width: 240px");

    await wrapper
      .find("[data-agent-sidebar-resize]")
      .trigger("mousedown", { clientX: 240 });
    document.dispatchEvent(new MouseEvent("mousemove", { clientX: 320 }));
    document.dispatchEvent(new MouseEvent("mouseup", { clientX: 320 }));
    await nextTick();

    expect(store.sidebarWidth).toBe(320);
    expect(wrapper.find("aside").attributes("style")).toContain("width: 320px");
    expect(localStorage.getItem("bb-agent-window")).toContain(
      '"sidebarWidth":320'
    );
  });

  test("clamps the sidebar width when the window narrows", async () => {
    const { store, wrapper } = mountAgentWindow();
    store.size.width = 760;
    store.sidebarWidth = 420;

    await nextTick();
    expect(wrapper.find("aside").attributes("style")).toContain("width: 420px");

    store.size.width = 360;
    await nextTick();

    expect(store.sidebarWidth).toBe(180);
    expect(wrapper.find("aside").attributes("style")).toContain("width: 180px");
  });

  test("archives and deletes chats through popconfirm", async () => {
    const { store, wrapper } = mountAgentWindow();
    const originalChatId = store.currentChatId!;
    const secondChat = store.createChat({ title: "Disposable thread" });
    store.setCurrentChat(secondChat.id);
    await nextTick();

    const popconfirms = wrapper.findAllComponents(NPopconfirm);
    expect(popconfirms).toHaveLength(2);

    await popconfirms[0].vm.$emit("positive-click");
    await nextTick();
    expect(store.getChat(secondChat.id)?.archived).toBe(true);

    store.setCurrentChat(originalChatId);
    await nextTick();
    const deletePopconfirm = wrapper.findAllComponents(NPopconfirm)[1];
    await deletePopconfirm.vm.$emit("positive-click");
    await nextTick();

    expect(store.getChat(originalChatId)).toBeNull();
    expect(store.currentChatId).toBe(secondChat.id);
  });
});
