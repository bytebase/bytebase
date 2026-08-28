import { beforeEach, describe, expect, test, vi } from "vitest";

type StoredDocument = Record<string, unknown> & {
  _id: string;
  _rev?: string;
};

const pouch = vi.hoisted(() => ({
  databases: new Map<string, Map<string, StoredDocument>>(),
}));

vi.mock("pouchdb", () => {
  class MockPouchDB {
    static plugin() {}

    constructor(private readonly name: string) {
      if (!pouch.databases.has(name)) {
        pouch.databases.set(name, new Map());
      }
    }

    private get documents() {
      return pouch.databases.get(this.name)!;
    }

    async createIndex() {}

    async find({ selector }: { selector: Record<string, unknown> }) {
      const docs = [...this.documents.values()].filter((document) =>
        Object.entries(selector).every(([field, condition]) => {
          const value = document[field];
          const matcher = condition as { $eq?: unknown; $in?: unknown[] };
          if ("$eq" in matcher) return value === matcher.$eq;
          if ("$in" in matcher) return matcher.$in?.includes(value) ?? false;
          return value === condition;
        })
      );
      return { docs };
    }

    async get(id: string) {
      const document = this.documents.get(id);
      if (!document) throw new Error("missing");
      return { ...document };
    }

    async put(document: StoredDocument) {
      const existing = this.documents.get(document._id);
      if (existing && document._rev !== existing._rev) {
        const error = new Error("Document update conflict") as Error & {
          status: number;
        };
        error.status = 409;
        throw error;
      }
      const generation = existing
        ? Number(existing._rev?.split("-")[0]) + 1
        : 1;
      const rev = `${generation}-test`;
      this.documents.set(document._id, { ...document, _rev: rev });
      return { id: document._id, ok: true, rev };
    }

    async bulkDocs(documents: StoredDocument[]) {
      return Promise.all(documents.map((document) => this.put(document)));
    }

    async destroy() {
      this.documents.clear();
    }
  }

  return { default: MockPouchDB };
});

vi.mock("pouchdb-find", () => ({ default: {} }));

import { useConversationStore } from "./conversation";

const connection = {
  instance: "instances/test",
  database: "instances/test/databases/demo",
};

beforeEach(() => {
  for (const documents of pouch.databases.values()) {
    documents.clear();
  }
  useConversationStore.setState({
    conversationsById: {},
    readyByConnection: {},
  });
});

describe("AI conversation persistence", () => {
  test("removing an empty conversation during reload does not conflict", async () => {
    await useConversationStore.getState().createConversation({
      ...connection,
      name: "",
    });
    useConversationStore.setState({
      conversationsById: {},
      readyByConnection: {},
    });

    const conversations = await useConversationStore
      .getState()
      .fetchConversationListByConnection(connection as never);

    expect(conversations).toEqual([]);
    expect(useConversationStore.getState().conversationsById).toEqual({});
  });

  test("reload marks an interrupted response as failed", async () => {
    const conversation = await useConversationStore
      .getState()
      .createConversation({ ...connection, name: "" });
    await useConversationStore.getState().createMessage({
      author: "AI",
      content: "",
      prompt: "",
      error: "",
      status: "LOADING",
      conversation_id: conversation.id,
    });
    useConversationStore.setState({
      conversationsById: {},
      readyByConnection: {},
    });

    const conversations = await useConversationStore
      .getState()
      .fetchConversationListByConnection(connection as never);

    expect(conversations[0]?.messageList[0]).toMatchObject({
      status: "FAILED",
      error: "Request timeout",
    });
    expect(
      useConversationStore.getState().conversationsById[conversation.id]
        ?.messageList[0]
    ).toMatchObject({ status: "FAILED", error: "Request timeout" });
  });
});
