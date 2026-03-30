import { mount } from "@vue/test-utils";
import { NInput, NPopconfirm } from "naive-ui";
import { createPinia, setActivePinia } from "pinia";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
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
        "new-chat": "New chat",
        "rename-chat-placeholder": "Enter a chat name",
        "archive-chat": "Archive",
        "unarchive-chat": "Unarchive",
        "delete-chat": "Delete",
        "active-only-chats": "Active only",
        "archived-only-chats": "Archived only",
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

afterEach(() => {
  vi.useRealTimers();
});

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

  test("shows a plain default title and humanized updated time for untitled chats", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-02T12:00:00Z"));

    const { store, wrapper } = mountAgentWindow();
    const chatId = store.currentChatId!;

    store.getChat(chatId)!.updatedTs = Date.now() - 60_000;
    await nextTick();

    const currentRow = wrapper.find(`[data-agent-chat-row='${chatId}']`);
    expect(currentRow.find("[data-agent-chat-title]").text()).toBe("New chat");
    expect(currentRow.find("[data-agent-chat-title]").text()).not.toContain(
      "·"
    );
    expect(currentRow.find("[data-agent-chat-updated-ts]").text()).toBe(
      "1 minute ago"
    );
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

  test("shows mode-specific sidebar actions for active and archived chats", async () => {
    const { store, wrapper } = mountAgentWindow();
    const originalChatId = store.currentChatId!;
    const secondChat = store.createChat({ title: "Disposable thread" });
    store.setCurrentChat(secondChat.id);
    await nextTick();

    const actionArea = () => wrapper.find("[data-agent-chat-sidebar-actions]");
    const modeToggle = () => wrapper.find("[data-agent-chat-list-mode]");
    const buttonLabels = () =>
      actionArea()
        .findAll("button")
        .map((button) => button.text());

    expect(modeToggle().text()).toBe("");
    expect(modeToggle().attributes("aria-label")).toBe("Active only");
    expect(buttonLabels()).toEqual(["Archive", ""]);
    expect(
      actionArea()
        .findAll("button")
        .at(-1)
        ?.attributes("data-agent-chat-list-mode")
    ).toBe("");
    expect(modeToggle().classes()).toContain("ml-auto");
    expect(modeToggle().find("svg").exists()).toBe(true);
    expect(wrapper.findAllComponents(NPopconfirm)).toHaveLength(1);

    await wrapper.findComponent(NPopconfirm).vm.$emit("positive-click");
    await nextTick();

    expect(store.getChat(secondChat.id)?.archived).toBe(true);
    expect(store.currentChatId).toBe(originalChatId);

    await modeToggle().trigger("click");
    await nextTick();

    expect(modeToggle().text()).toBe("");
    expect(modeToggle().attributes("aria-label")).toBe("Archived only");
    expect(store.currentChatId).toBe(secondChat.id);
    expect(buttonLabels()).toEqual(["Unarchive", "Delete", ""]);
    expect(
      actionArea()
        .findAll("button")
        .at(-1)
        ?.attributes("data-agent-chat-list-mode")
    ).toBe("");
    expect(modeToggle().classes()).toContain("ml-auto");
    expect(modeToggle().find("svg").exists()).toBe(true);
    expect(wrapper.findAllComponents(NPopconfirm)).toHaveLength(1);
  });

  test("unarchives and deletes chats from archived-only mode", async () => {
    const { store, wrapper } = mountAgentWindow();
    const originalChatId = store.currentChatId!;
    const archivedChat = store.createChat({ title: "Disposable thread" });
    store.setCurrentChat(archivedChat.id);
    await nextTick();

    await wrapper.findComponent(NPopconfirm).vm.$emit("positive-click");
    await nextTick();
    await wrapper.find("[data-agent-chat-list-mode]").trigger("click");
    await nextTick();

    await wrapper.find("[data-agent-unarchive-chat]").trigger("click");
    await nextTick();

    expect(store.getChat(archivedChat.id)?.archived).toBe(false);
    expect(
      wrapper.find("[data-agent-chat-list-mode]").attributes("aria-label")
    ).toBe("Active only");
    expect(wrapper.find("[data-agent-chat-list-mode]").text()).toBe("");
    expect(store.currentChatId).toBe(archivedChat.id);

    await wrapper.findComponent(NPopconfirm).vm.$emit("positive-click");
    await nextTick();
    await wrapper.find("[data-agent-chat-list-mode]").trigger("click");
    await nextTick();

    expect(store.currentChatId).toBe(archivedChat.id);

    await wrapper.findComponent(NPopconfirm).vm.$emit("positive-click");
    await nextTick();

    expect(store.getChat(archivedChat.id)).toBeNull();
    expect(
      wrapper.find("[data-agent-chat-list-mode]").attributes("aria-label")
    ).toBe("Active only");
    expect(wrapper.find("[data-agent-chat-list-mode]").text()).toBe("");
    expect(store.currentChatId).toBe(originalChatId);
  });
});
