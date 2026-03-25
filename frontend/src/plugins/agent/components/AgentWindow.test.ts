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
        "thread-list-label": "Chats",
        "thread-switch-locked":
          "Finish the running chat before switching to another one.",
        "new-thread": "New chat",
        "rename-thread": "Rename",
        "rename-thread-placeholder": "Enter a chat name",
        "archive-thread": "Archive",
        "unarchive-thread": "Unarchive",
        "delete-thread": "Delete",
        "show-archived-threads": "Show archived chats",
        "hide-archived-threads": "Hide archived chats",
        "archive-thread-confirmation": "Archive this chat?",
        "delete-thread-confirmation": "Delete this chat?",
        "thread-default-title": "New chat",
        "thread-archived-label": "Archived",
        "thread-status-idle": "Idle",
        "thread-status-running": "Running",
        "thread-status-awaiting-user": "Awaiting input",
        "thread-status-error": "Error",
        "thread-status-interrupted": "Interrupted",
        "thread-total-tokens": "Tokens used: {count}",
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

  test("renders threads in a left sidebar instead of a select picker", async () => {
    const { store, wrapper } = mountAgentWindow();
    const firstThreadId = store.currentThreadId!;
    store.addMessage({
      threadId: firstThreadId,
      role: "user",
      content: "Summarize the production incident timeline",
    });
    const secondThread = store.createThread({
      title: "Renamed thread",
      select: false,
    });

    await wrapper.vm.$nextTick();

    expect(wrapper.find("select").exists()).toBe(false);
    const threadRows = wrapper.findAll("[data-agent-thread-list] button");
    expect(threadRows).toHaveLength(2);
    expect(threadRows[0].text()).toContain("Renamed thread");
    expect(threadRows[1].text()).toContain(
      "Summarize the production incident timeline"
    );
    await threadRows[0].trigger("click");
    expect(store.currentThreadId).toBe(secondThread.id);
  });

  test("disables switching threads while a thread is running", async () => {
    const { store, wrapper } = mountAgentWindow();
    const firstThreadId = store.currentThreadId!;
    const secondThread = store.createThread({
      title: "Second thread",
      select: false,
    });

    store.startRun(firstThreadId, {
      path: "/projects/demo",
      title: "Demo Page",
    });
    await wrapper.vm.$nextTick();

    const threadRows = wrapper.findAll("[data-agent-thread-list] button");
    const currentRow = threadRows.find(
      (row) => row.attributes("aria-current") === "true"
    );
    const otherRow = threadRows.find((row) =>
      row.text().includes("Second thread")
    );
    const newThreadButton = wrapper
      .findAll("button")
      .find((button) => button.text() === "New chat");

    expect(wrapper.find("[data-agent-thread-lock-message]").text()).toContain(
      "Finish the running chat before switching to another one."
    );
    expect(currentRow?.attributes("disabled")).toBeUndefined();
    expect(otherRow?.attributes("disabled")).toBe("");
    expect(newThreadButton?.attributes("disabled")).toBe("");

    await otherRow?.trigger("click");
    expect(store.currentThreadId).toBe(firstThreadId);
    expect(store.canSelectThread(secondThread.id)).toBe(false);
  });

  test("renames the current thread with inline editing", async () => {
    const { store, wrapper } = mountAgentWindow();
    const renameButton = wrapper
      .findAll("button")
      .find((button) => button.text() === "Rename");

    await renameButton?.trigger("click");
    await nextTick();

    const input = wrapper.find("input");
    expect(input.exists()).toBe(true);
    await input.setValue("  Renamed thread in app  ");
    await input.trigger("keydown", { key: "Enter" });
    await nextTick();

    expect(store.currentThread?.title).toBe("Renamed thread in app");
    expect(wrapper.find("input").exists()).toBe(false);
  });

  test("cancels inline rename on escape", async () => {
    const { store, wrapper } = mountAgentWindow();
    const originalTitle = store.currentThread?.title;
    const renameButton = wrapper
      .findAll("button")
      .find((button) => button.text() === "Rename");

    await renameButton?.trigger("click");
    await nextTick();

    const input = wrapper.find("input");
    await input.setValue("Should not save");
    await input.trigger("keydown", { key: "Escape" });
    await nextTick();

    expect(store.currentThread?.title).toBe(originalTitle);
    expect(wrapper.find("input").exists()).toBe(false);
  });

  test("archives and deletes threads through popconfirm", async () => {
    const { store, wrapper } = mountAgentWindow();
    const originalThreadId = store.currentThreadId!;
    const secondThread = store.createThread({ title: "Disposable thread" });
    store.setCurrentThread(secondThread.id);
    await nextTick();

    const popconfirms = wrapper.findAllComponents(NPopconfirm);
    expect(popconfirms).toHaveLength(2);

    await popconfirms[0].vm.$emit("positive-click");
    await nextTick();
    expect(store.getThread(secondThread.id)?.archived).toBe(true);

    store.setCurrentThread(originalThreadId);
    await nextTick();
    const deletePopconfirm = wrapper.findAllComponents(NPopconfirm)[1];
    await deletePopconfirm.vm.$emit("positive-click");
    await nextTick();

    expect(store.getThread(originalThreadId)).toBeNull();
    expect(store.currentThreadId).toBe(secondThread.id);
  });
});
