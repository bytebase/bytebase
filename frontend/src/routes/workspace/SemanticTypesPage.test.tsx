import type { ReactNode } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  lazyExtractDomRefSuggestions,
  lazyExtractDomTree,
} from "@/modules/agent/dom";
import { SemanticTypesPage } from "./SemanticTypesPage";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  getOrFetchSettingByName: vi.fn(async () => undefined),
  getSettingByName: vi.fn<() => unknown>(() => undefined),
  upsertSetting: vi.fn(async (_request: unknown): Promise<void> => undefined),
  useUnsavedChangesGuard: vi.fn(),
  getSetting: vi.fn(async () => undefined as unknown),
  setSettingByName: vi.fn(),
  validateDraft: vi.fn(async (draft: unknown, _context: unknown) => draft),
  recordHandoff: vi.fn(),
  hasPermission: true,
  hasFeature: true,
  userName: "users/member@example.com",
  workspaceName: "workspaces/test",
  currentChatId: "chat-test",
  chats: [
    { id: "chat-test", workspaceName: "workspaces/test", archived: false },
  ],
  target: undefined as
    | {
        getSnapshot: () => Promise<
          { sessionId: string; revision: string; busy: boolean } | undefined
        >;
        open: (
          draft: unknown,
          chatId: string,
          origin?: unknown
        ) => Promise<void>;
      }
    | undefined,
}));

vi.mock("@/api", () => ({
  settingServiceClientConnect: { getSetting: mocks.getSetting },
}));

vi.mock("@/stores", () => ({ pushNotification: vi.fn() }));

vi.mock("@/modules/agent/components/AgentPageHelpButton", () => ({
  AgentPageHelpButton: () => <button type="button">Ask Agent</button>,
}));

vi.mock("@/modules/agent/store/agent", () => ({
  useAgentStore: {
    getState: () => ({
      recordSemanticTypeDraftHandoff: mocks.recordHandoff,
      currentChatId: mocks.currentChatId,
      chats: mocks.chats,
    }),
  },
}));

vi.mock("@/modules/agent/logic/semanticTypeDraft", () => ({
  validateSemanticTypeDraft: mocks.validateDraft,
}));

vi.mock("@/modules/agent/logic/semanticTypeDraftTarget", () => ({
  registerSemanticTypeDraftTarget: (
    _context: unknown,
    target: NonNullable<typeof mocks.target>
  ) => {
    mocks.target = target;
    return () => {
      if (mocks.target === target) mocks.target = undefined;
    };
  },
}));

vi.mock("react-i18next", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-i18next")>()),
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/components/FeatureAttention", () => ({
  FeatureAttention: () => null,
}));

vi.mock("@/hooks/useUnsavedChangesGuard", () => ({
  useUnsavedChangesGuard: mocks.useUnsavedChangesGuard,
}));

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({
    children,
    content,
  }: {
    children: ReactNode;
    content: ReactNode;
  }) => (
    <span
      data-tooltip-content={typeof content === "string" ? content : undefined}
    >
      {children}
    </span>
  ),
}));

vi.mock("@/components/WorkspacePageLayout", () => ({
  WorkspacePageInfo: ({ description }: { description: ReactNode }) => (
    <div>{description}</div>
  ),
  WorkspacePageLayout: ({ children }: { children: ReactNode }) => (
    <main>{children}</main>
  ),
  WorkspacePageToolbar: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
}));

vi.mock("@/stores/app", () => {
  type MockAppState = {
    getSettingByName: () => unknown;
    hasInstanceFeature: () => boolean;
    settingsByName: Map<unknown, unknown>;
    currentUser: { name: string };
    workspaceResourceName: () => string;
  };
  const state: MockAppState = {
    getSettingByName: mocks.getSettingByName,
    hasInstanceFeature: () => mocks.hasFeature,
    settingsByName: new Map(),
    get currentUser() {
      return { name: mocks.userName };
    },
    workspaceResourceName: () => mocks.workspaceName,
  };
  const useAppStore = Object.assign(
    (selector: (state: MockAppState) => unknown) => selector(state),
    {
      getState: () => ({
        ...state,
        getOrFetchSettingByName: mocks.getOrFetchSettingByName,
        getSettingByName: mocks.getSettingByName,
        upsertSetting: mocks.upsertSetting,
        setSettingByName: mocks.setSettingByName,
      }),
    }
  );
  return { useAppStore };
});

vi.mock("@/utils", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/utils")>()),
  hasWorkspacePermissionV2: () => mocks.hasPermission,
}));

const roots: Array<ReturnType<typeof createRoot>> = [];

beforeEach(() => {
  mocks.hasPermission = true;
  mocks.hasFeature = true;
  mocks.userName = "users/member@example.com";
  mocks.workspaceName = "workspaces/test";
  mocks.currentChatId = "chat-test";
  mocks.chats = [
    { id: "chat-test", workspaceName: "workspaces/test", archived: false },
  ];
  mocks.target = undefined;
  mocks.getSettingByName.mockReturnValue(undefined);
  mocks.getSetting.mockResolvedValue({
    value: { value: { case: "semanticType", value: { types: [] } } },
  });
  mocks.validateDraft.mockImplementation(async (draft) => draft);
  mocks.upsertSetting.mockResolvedValue(undefined);
});

afterEach(() => {
  for (const root of roots.splice(0)) {
    act(() => root.unmount());
  }
  mocks.getOrFetchSettingByName.mockClear();
  mocks.getSettingByName.mockClear();
  mocks.upsertSetting.mockClear();
  mocks.useUnsavedChangesGuard.mockClear();
  mocks.validateDraft.mockClear();
  mocks.getSetting.mockClear();
  mocks.setSettingByName.mockClear();
  mocks.recordHandoff.mockClear();
  vi.restoreAllMocks();
});

async function renderPage() {
  const container = document.createElement("div");
  const root = createRoot(container);
  roots.push(root);
  await act(async () => {
    root.render(<SemanticTypesPage />);
    await Promise.resolve();
  });
  return { container, root };
}

function draftFixture() {
  return {
    id: "329ea6d8-86f1-4c50-9541-c73b61403816",
    title: "Custom email",
    description: "Replacement masking",
    algorithm: {
      $typeName: "bytebase.v1.Algorithm",
      mask: {
        case: "fullMask",
        value: {
          $typeName: "bytebase.v1.Algorithm.FullMask",
          substitution: "hidden",
        },
      },
    },
  };
}

function originFixture() {
  return {
    workspaceName: mocks.workspaceName,
    userName: mocks.userName,
    chatId: "chat-test",
    messageId: "message-test",
    draftId: "draft-test",
  };
}

function changeInput(input: HTMLInputElement | null, value: string) {
  expect(input).not.toBeNull();
  Object.getOwnPropertyDescriptor(
    HTMLInputElement.prototype,
    "value"
  )?.set?.call(input, value);
  input?.dispatchEvent(new Event("input", { bubbles: true }));
}

describe("SemanticTypesPage", () => {
  test("keeps the MD5 salt editable by the member but hidden from Agent DOM inspection", async () => {
    const { container } = await renderPage();
    const secret = "PRIVATE_MD5_SALT";
    const draft = {
      ...draftFixture(),
      algorithm: {
        $typeName: "bytebase.v1.Algorithm",
        mask: {
          case: "md5Mask",
          value: { $typeName: "bytebase.v1.Algorithm.MD5Mask", salt: secret },
        },
      },
    };
    await act(async () => {
      await mocks.target?.open(draft, "chat-test", originFixture());
    });
    const row = Array.from(container.querySelectorAll("tbody tr")).find(
      (item) => item.textContent?.includes(draft.id)
    );
    await act(async () =>
      row?.querySelector<HTMLButtonElement>("td:nth-child(5) button")?.click()
    );
    const saltInput = document.querySelector<HTMLInputElement>(
      'input[placeholder="settings.sensitive-data.algorithms.md5-mask.salt"]'
    );
    expect(saltInput?.value).toBe(secret);
    expect(saltInput?.disabled).toBe(false);
    expect(saltInput?.hasAttribute("data-agent-sensitive")).toBe(true);
    expect((await lazyExtractDomTree()).tree).not.toContain(secret);
    expect(JSON.stringify(await lazyExtractDomRefSuggestions())).not.toContain(
      secret
    );
    act(() => changeInput(saltInput, "member-updated-private-salt"));
    expect(saltInput?.value).toBe("member-updated-private-salt");
    expect(mocks.upsertSetting).not.toHaveBeenCalled();
  });

  test("rejects an Agent draft when its selected chat changes during validation", async () => {
    const { container } = await renderPage();
    let finish!: (value: unknown) => void;
    mocks.validateDraft.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finish = resolve;
        })
    );
    const opening = mocks.target?.open(
      draftFixture(),
      "chat-test",
      originFixture()
    );
    mocks.currentChatId = "another-chat";
    finish(draftFixture());
    await expect(opening).rejects.toThrow();
    expect(
      container.querySelector(
        'input[placeholder="settings.sensitive-data.semantic-types.table.title"]'
      )
    ).toBeNull();
    expect(mocks.upsertSetting).not.toHaveBeenCalled();
  });

  test.each(["archived", "workspace", "missing"])(
    "rejects an Agent draft with a %s source chat",
    async (change) => {
      await renderPage();
      if (change === "archived") mocks.chats[0].archived = true;
      if (change === "workspace")
        mocks.chats[0].workspaceName = "workspaces/another";
      if (change === "missing") mocks.chats = [];
      await expect(
        mocks.target?.open(draftFixture(), "chat-test", originFixture())
      ).rejects.toThrow();
      expect(mocks.validateDraft).not.toHaveBeenCalled();
    }
  );

  test("preserves a member-edited icon when saving an Agent semantic type", async () => {
    const { container } = await renderPage();
    await act(async () => {
      await mocks.target?.open(draftFixture(), "chat-test", originFixture());
    });
    const row = Array.from(container.querySelectorAll("tbody tr")).find(
      (item) => item.textContent?.includes(draftFixture().id)
    );
    await act(async () =>
      row?.querySelector<HTMLButtonElement>("td:first-child button")?.click()
    );
    const icon = "data:image/png;base64,eA==";
    vi.spyOn(FileReader.prototype, "readAsDataURL").mockImplementation(
      function (this: FileReader) {
        Object.defineProperty(this, "result", { value: icon });
        this.onload?.call(
          this,
          new ProgressEvent("load") as ProgressEvent<FileReader>
        );
      }
    );
    const input =
      document.querySelector<HTMLInputElement>('input[type="file"]');
    expect(input).not.toBeNull();
    Object.defineProperty(input, "files", {
      configurable: true,
      value: [new File(["x"], "icon.png", { type: "image/png" })],
    });
    await act(async () => {
      input?.dispatchEvent(new Event("change", { bubbles: true }));
    });
    const popover = input?.parentElement?.parentElement;
    expect(popover).not.toBeNull();
    await act(async () =>
      Array.from(popover?.querySelectorAll("button") ?? [])
        .find((button) => button.textContent === "common.update")
        ?.click()
    );
    await act(async () => {
      container
        .querySelector<HTMLButtonElement>('button[aria-label="common.confirm"]')
        ?.click();
      await Promise.resolve();
    });
    const request = mocks.upsertSetting.mock.calls.at(-1)?.[0] as
      | { value: { value: { value: { types: Array<{ icon: string }> } } } }
      | undefined;
    expect(request?.value.value.value.types.at(-1)?.icon).toBe(icon);
  });

  test("opens an Agent type as an unsaved prefilled row and saves only after native confirmation", async () => {
    const { container } = await renderPage();
    expect(mocks.target).toBeDefined();
    const before = await mocks.target?.getSnapshot();
    expect(before?.busy).toBe(false);

    await act(async () => {
      await mocks.target?.open(draftFixture(), "chat-test", originFixture());
    });
    expect(mocks.upsertSetting).not.toHaveBeenCalled();
    expect(
      container.querySelector<HTMLInputElement>(
        'input[placeholder="settings.sensitive-data.semantic-types.table.title"]'
      )?.value
    ).toBe("Custom email");
    expect((await mocks.target?.getSnapshot())?.busy).toBe(true);

    await act(async () => {
      container
        .querySelector<HTMLButtonElement>('button[aria-label="common.confirm"]')
        ?.click();
      await Promise.resolve();
    });
    expect(mocks.upsertSetting).toHaveBeenCalledTimes(1);
    expect(mocks.recordHandoff).toHaveBeenCalledWith(
      "chat-test",
      "message-test",
      "draft-test",
      "created"
    );
    expect(
      container.querySelector(
        'input[placeholder="settings.sensitive-data.semantic-types.table.title"]'
      )
    ).toBeNull();
  });

  test("allows native form confirmation after the member switches Agent chats", async () => {
    const { container } = await renderPage();
    await act(async () => {
      await mocks.target?.open(draftFixture(), "chat-test", originFixture());
    });
    mocks.currentChatId = "another-chat";
    await act(async () => {
      container
        .querySelector<HTMLButtonElement>('button[aria-label="common.confirm"]')
        ?.click();
      await Promise.resolve();
    });
    expect(mocks.upsertSetting).toHaveBeenCalledTimes(1);
    expect(mocks.recordHandoff).toHaveBeenCalledWith(
      "chat-test",
      "message-test",
      "draft-test",
      "created"
    );
  });

  test("appends to freshly read semantic types and permits duplicate titles and algorithms", async () => {
    const { container } = await renderPage();
    await act(async () => {
      await mocks.target?.open(draftFixture(), "chat-test", originFixture());
    });
    const existing = { ...draftFixture(), id: "other-custom-type" };
    mocks.getSetting.mockResolvedValue({
      value: { value: { case: "semanticType", value: { types: [existing] } } },
    });
    await act(async () => {
      container
        .querySelector<HTMLButtonElement>('button[aria-label="common.confirm"]')
        ?.click();
      await Promise.resolve();
    });
    const request = mocks.upsertSetting.mock.calls.at(-1)?.[0] as
      | {
          value: {
            value: { value: { types: Array<{ id: string; title: string }> } };
          };
        }
      | undefined;
    expect(request?.value.value.value.types.map(({ id }) => id)).toEqual([
      "other-custom-type",
      draftFixture().id,
    ]);
    expect(request?.value.value.value.types.map(({ title }) => title)).toEqual([
      "Custom email",
      "Custom email",
    ]);
  });

  test.each(["create", "edit"])(
    "refuses Agent opening while a manual %s is active",
    async (mode) => {
      if (mode === "edit") {
        mocks.getSettingByName.mockReturnValue({
          value: {
            value: {
              case: "semanticType",
              value: {
                types: [{ id: "existing-custom", title: "Existing type" }],
              },
            },
          },
        });
      }
      const { container } = await renderPage();
      act(() => {
        if (mode === "create") {
          Array.from(container.querySelectorAll("button"))
            .find((button) => button.textContent === "common.create")
            ?.click();
        } else {
          const row = Array.from(container.querySelectorAll("tbody tr")).find(
            (row) => row.textContent?.includes("existing-custom")
          );
          Array.from(row?.querySelectorAll<HTMLButtonElement>("button") ?? [])
            .at(-1)
            ?.click();
        }
      });
      await expect(
        mocks.target?.open(draftFixture(), "chat-test", originFixture())
      ).rejects.toThrow();
      expect(
        container.querySelectorAll(
          'input[placeholder="settings.sensitive-data.semantic-types.table.title"]'
        )
      ).toHaveLength(1);
      expect(mocks.upsertSetting).not.toHaveBeenCalled();
    }
  );

  test("rejects a stale source identity before prefilling any row", async () => {
    await renderPage();
    await expect(
      mocks.target?.open(draftFixture(), "chat-test", {
        ...originFixture(),
        userName: "users/another@example.com",
      })
    ).rejects.toThrow();
    expect(mocks.validateDraft).not.toHaveBeenCalled();
    expect((await mocks.target?.getSnapshot())?.busy).toBe(false);
  });

  test("rejects an Agent open after the owning page unmounts", async () => {
    const { root } = await renderPage();
    const target = mocks.target;
    act(() => root.unmount());
    roots.splice(roots.indexOf(root), 1);
    await expect(
      target?.open(draftFixture(), "chat-test", originFixture())
    ).rejects.toThrow();
    expect(mocks.validateDraft).not.toHaveBeenCalled();
  });

  test("refuses to replace edits made while an Agent draft is being validated", async () => {
    const { container } = await renderPage();
    let resolve!: (value: unknown) => void;
    mocks.validateDraft.mockImplementationOnce(
      () =>
        new Promise((done) => {
          resolve = done;
        })
    );
    const opening = mocks.target?.open(
      draftFixture(),
      "chat-test",
      originFixture()
    );
    act(() =>
      Array.from(container.querySelectorAll("button"))
        .find((button) => button.textContent === "common.create")
        ?.click()
    );
    resolve(draftFixture());
    await expect(opening).rejects.toThrow();
    expect(
      container.querySelectorAll(
        'input[placeholder="settings.sensitive-data.semantic-types.table.title"]'
      )
    ).toHaveLength(1);
    expect(mocks.upsertSetting).not.toHaveBeenCalled();
  });

  test("retains an Agent draft and reports no Created receipt when save validation fails", async () => {
    const { container } = await renderPage();
    await act(async () => {
      await mocks.target?.open(draftFixture(), "chat-test", originFixture());
    });
    mocks.validateDraft.mockRejectedValueOnce(new Error("Permission revoked"));
    await act(async () => {
      container
        .querySelector<HTMLButtonElement>('button[aria-label="common.confirm"]')
        ?.click();
      await Promise.resolve();
    });
    expect(mocks.upsertSetting).not.toHaveBeenCalled();
    expect(mocks.recordHandoff).not.toHaveBeenCalled();
    expect(container.textContent).toContain(
      "agent.semantic-type-draft.save-failed"
    );
    expect(
      container.querySelector(
        'input[placeholder="settings.sensitive-data.semantic-types.table.title"]'
      )
    ).not.toBeNull();
  });

  test("preserves other unsaved rows when the Agent type is saved", async () => {
    const { container } = await renderPage();
    await act(async () => {
      await mocks.target?.open(draftFixture(), "chat-test", originFixture());
    });
    act(() =>
      Array.from(container.querySelectorAll("button"))
        .find((button) => button.textContent === "common.create")
        ?.click()
    );
    act(() =>
      changeInput(
        Array.from(
          container.querySelectorAll<HTMLInputElement>(
            'input[placeholder="settings.sensitive-data.semantic-types.table.title"]'
          )
        ).at(-1) ?? null,
        "Other local type"
      )
    );
    await act(async () => {
      container
        .querySelector<HTMLButtonElement>('button[aria-label="common.confirm"]')
        ?.click();
      await Promise.resolve();
    });
    expect(
      Array.from(
        container.querySelectorAll<HTMLInputElement>(
          'input[placeholder="settings.sensitive-data.semantic-types.table.title"]'
        )
      ).map(({ value }) => value)
    ).toEqual(["Other local type"]);
    expect(mocks.upsertSetting).toHaveBeenCalledTimes(1);
  });

  test("does not overwrite settings when the fresh semantic-type read is unavailable", async () => {
    const { container } = await renderPage();
    await act(async () => {
      await mocks.target?.open(draftFixture(), "chat-test", originFixture());
    });
    mocks.getSetting.mockResolvedValue(undefined);
    await act(async () => {
      container
        .querySelector<HTMLButtonElement>('button[aria-label="common.confirm"]')
        ?.click();
      await Promise.resolve();
    });
    expect(mocks.upsertSetting).not.toHaveBeenCalled();
    expect(mocks.recordHandoff).not.toHaveBeenCalled();
  });

  test("rejects a fresh type ID collision without saving", async () => {
    const { container } = await renderPage();
    await act(async () => {
      await mocks.target?.open(draftFixture(), "chat-test", originFixture());
    });
    mocks.getSetting.mockResolvedValue({
      value: {
        value: { case: "semanticType", value: { types: [draftFixture()] } },
      },
    });
    await act(async () => {
      container
        .querySelector<HTMLButtonElement>('button[aria-label="common.confirm"]')
        ?.click();
      await Promise.resolve();
    });
    expect(mocks.upsertSetting).not.toHaveBeenCalled();
    expect(mocks.recordHandoff).not.toHaveBeenCalled();
  });

  test("rechecks the owner after the final settings read before saving", async () => {
    const { container } = await renderPage();
    await act(async () => {
      await mocks.target?.open(draftFixture(), "chat-test", originFixture());
    });
    let finish!: (value: unknown) => void;
    mocks.getSetting.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finish = resolve;
        })
    );
    await act(async () => {
      container
        .querySelector<HTMLButtonElement>('button[aria-label="common.confirm"]')
        ?.click();
      await Promise.resolve();
    });
    mocks.userName = "users/another@example.com";
    await act(async () => {
      finish({
        value: { value: { case: "semanticType", value: { types: [] } } },
      });
      await Promise.resolve();
    });
    expect(mocks.upsertSetting).not.toHaveBeenCalled();
    expect(mocks.recordHandoff).not.toHaveBeenCalled();
  });

  test("keeps an Agent draft editable when the server rejects its save", async () => {
    const { container } = await renderPage();
    await act(async () => {
      await mocks.target?.open(draftFixture(), "chat-test", originFixture());
    });
    mocks.upsertSetting.mockRejectedValueOnce(new Error("Request failed"));
    await act(async () => {
      container
        .querySelector<HTMLButtonElement>('button[aria-label="common.confirm"]')
        ?.click();
      await Promise.resolve();
    });
    expect(
      container.querySelector<HTMLInputElement>(
        'input[placeholder="settings.sensitive-data.semantic-types.table.title"]'
      )?.disabled
    ).toBe(false);
    expect(mocks.recordHandoff).not.toHaveBeenCalled();
    expect(container.textContent).toContain(
      "agent.semantic-type-draft.save-failed"
    );
  });

  test("editing the Agent draft algorithm does not save its type", async () => {
    const { container } = await renderPage();
    await act(async () => {
      await mocks.target?.open(draftFixture(), "chat-test", originFixture());
    });
    const row = Array.from(container.querySelectorAll("tbody tr")).find(
      (item) => item.textContent?.includes(draftFixture().id)
    );
    await act(async () => {
      row?.querySelector<HTMLButtonElement>("td:nth-child(5) button")?.click();
    });
    const drawer = document.querySelector('[role="dialog"]');
    expect(drawer).not.toBeNull();
    act(() => changeInput(drawer?.querySelector("input") ?? null, "redacted"));
    await act(async () => {
      Array.from(drawer?.querySelectorAll("button") ?? [])
        .find((button) => button.textContent === "common.update")
        ?.click();
    });
    expect(mocks.upsertSetting).not.toHaveBeenCalled();
    expect(
      container.querySelector('button[aria-label="common.confirm"]')
    ).not.toBeNull();
    await act(async () => {
      container
        .querySelector<HTMLButtonElement>('button[aria-label="common.confirm"]')
        ?.click();
      await Promise.resolve();
    });
    expect(mocks.validateDraft).toHaveBeenLastCalledWith(
      expect.objectContaining({
        algorithm: expect.objectContaining({
          mask: expect.objectContaining({
            case: "fullMask",
            value: expect.objectContaining({ substitution: "redacted" }),
          }),
        }),
      }),
      originFixture()
    );
  });

  test.each(["user", "workspace", "permission", "feature"])(
    "blocks saving after %s eligibility changes",
    async (change) => {
      const { container } = await renderPage();
      await act(async () => {
        await mocks.target?.open(draftFixture(), "chat-test", originFixture());
      });
      if (change === "user") mocks.userName = "users/another@example.com";
      if (change === "workspace") mocks.workspaceName = "workspaces/another";
      if (change === "permission") mocks.hasPermission = false;
      if (change === "feature") mocks.hasFeature = false;
      await act(async () => {
        container
          .querySelector<HTMLButtonElement>(
            'button[aria-label="common.confirm"]'
          )
          ?.click();
        await Promise.resolve();
      });
      expect(mocks.upsertSetting).not.toHaveBeenCalled();
      expect(mocks.recordHandoff).not.toHaveBeenCalled();
    }
  );

  test("prevents duplicate saves and keeps the draft while the request is pending", async () => {
    const { container } = await renderPage();
    await act(async () => {
      await mocks.target?.open(draftFixture(), "chat-test", originFixture());
    });
    let finish!: () => void;
    mocks.upsertSetting.mockImplementationOnce(
      () =>
        new Promise<void>((resolve) => {
          finish = resolve;
        })
    );
    await act(async () => {
      const button = container.querySelector<HTMLButtonElement>(
        'button[aria-label="common.confirm"]'
      );
      button?.click();
      button?.click();
      await Promise.resolve();
    });
    expect(mocks.upsertSetting).toHaveBeenCalledTimes(1);
    expect(mocks.recordHandoff).not.toHaveBeenCalled();
    expect(
      container.querySelector(
        'input[placeholder="settings.sensitive-data.semantic-types.table.title"]'
      )
    ).not.toBeNull();
    await act(async () => {
      finish();
      await Promise.resolve();
    });
    expect(mocks.recordHandoff).toHaveBeenCalledTimes(1);
  });

  test("preserves the normal manual Create workflow", async () => {
    const { container } = await renderPage();
    act(() =>
      Array.from(container.querySelectorAll("button"))
        .find((button) => button.textContent === "common.create")
        ?.click()
    );
    act(() =>
      changeInput(
        container.querySelector(
          'input[placeholder="settings.sensitive-data.semantic-types.table.title"]'
        ),
        "Manual type"
      )
    );
    await act(async () => {
      container
        .querySelector<HTMLButtonElement>('button[aria-label="common.confirm"]')
        ?.click();
      await Promise.resolve();
    });
    expect(mocks.upsertSetting).toHaveBeenCalledTimes(1);
    expect(mocks.validateDraft).not.toHaveBeenCalled();
    expect(mocks.recordHandoff).not.toHaveBeenCalled();
  });

  test("lets the toolbar actions wrap within the available width", async () => {
    mocks.getSettingByName.mockReturnValue(undefined);
    const container = document.createElement("div");
    const root = createRoot(container);
    roots.push(root);

    await act(async () => {
      root.render(<SemanticTypesPage />);
      await Promise.resolve();
    });

    const createButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === "common.create"
    );
    const group = createButton?.parentElement;
    expect(group?.classList.contains("flex-wrap")).toBe(true);
    expect(group?.classList.contains("gap-2")).toBe(true);
    expect(group?.classList.contains("min-w-0")).toBe(true);
    expect(group?.classList.contains("max-w-full")).toBe(true);
  });

  test("keeps the semantic type table horizontally scrollable on narrow screens", async () => {
    mocks.getOrFetchSettingByName.mockReturnValueOnce(new Promise(() => {}));
    const container = document.createElement("div");
    const root = createRoot(container);
    roots.push(root);

    await act(async () => {
      root.render(<SemanticTypesPage />);
      await Promise.resolve();
    });

    const table = container.querySelector("table");
    expect(table?.classList.contains("min-w-5xl")).toBe(true);
    expect(table?.parentElement?.classList.contains("overflow-x-auto")).toBe(
      true
    );
  });

  test("shows built-in semantic types without installing templates", async () => {
    mocks.getOrFetchSettingByName.mockReturnValueOnce(new Promise(() => {}));
    const container = document.createElement("div");
    const root = createRoot(container);
    roots.push(root);

    await act(async () => {
      root.render(<SemanticTypesPage />);
      await Promise.resolve();
    });

    const text = container.textContent ?? "";
    expect(text).toContain("bb.default");
    expect(text).toContain("bb.default-partial");
    expect(text).not.toContain(
      "settings.sensitive-data.semantic-types.use-predefined-type"
    );
  });

  test("does not duplicate built-in semantic types already stored in the workspace setting", async () => {
    mocks.getSettingByName.mockReturnValue({
      value: {
        value: {
          case: "semanticType",
          value: {
            types: [
              { id: "bb.default", title: "Stored default" },
              { id: "bb.default-partial", title: "Stored partial default" },
              { id: "email", title: "Email" },
            ],
          },
        },
      },
    });
    const container = document.createElement("div");
    const root = createRoot(container);
    roots.push(root);

    await act(async () => {
      root.render(<SemanticTypesPage />);
      await Promise.resolve();
    });

    const renderedIds = Array.from(
      container.querySelectorAll("tbody tr td:nth-child(2)")
    ).map((cell) => cell.textContent);
    expect(renderedIds).toEqual(["bb.default", "bb.default-partial", "email"]);
  });

  test("explains why a new semantic type cannot be submitted", async () => {
    mocks.getSettingByName.mockReturnValue(undefined);
    const container = document.createElement("div");
    const root = createRoot(container);
    roots.push(root);

    await act(async () => {
      root.render(<SemanticTypesPage />);
      await Promise.resolve();
    });

    const createButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === "common.create"
    );
    expect(createButton).toBeDefined();
    act(() => createButton?.click());

    const confirmButton = container.querySelector<HTMLButtonElement>(
      'button[aria-label="common.confirm"]'
    );
    expect(confirmButton?.disabled).toBe(true);
    expect(
      confirmButton
        ?.closest("[data-tooltip-content]")
        ?.getAttribute("data-tooltip-content")
    ).toBe("settings.sensitive-data.semantic-types.error.title-required");
    expect(container.textContent).toContain(
      "settings.sensitive-data.semantic-types.table.title"
    );
  });

  test("guards navigation while a semantic type is being created or edited", async () => {
    mocks.getSettingByName.mockReturnValue({
      value: {
        value: {
          case: "semanticType",
          value: { types: [{ id: "email", title: "Email" }] },
        },
      },
    });
    const container = document.createElement("div");
    const root = createRoot(container);
    roots.push(root);

    await act(async () => {
      root.render(<SemanticTypesPage />);
      await Promise.resolve();
    });

    expect(mocks.useUnsavedChangesGuard).toHaveBeenLastCalledWith(false);

    const createButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === "common.create"
    );
    act(() => createButton?.click());
    expect(mocks.useUnsavedChangesGuard).toHaveBeenLastCalledWith(true);

    const cancelButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.querySelector("svg.lucide-undo-2")
    );
    act(() => cancelButton?.click());
    expect(mocks.useUnsavedChangesGuard).toHaveBeenLastCalledWith(false);

    const emailRow = Array.from(container.querySelectorAll("tbody tr")).find(
      (row) => row.querySelector("td:nth-child(2)")?.textContent === "email"
    );
    const editButton = Array.from(
      emailRow?.querySelectorAll<HTMLButtonElement>("button") ?? []
    ).at(-1);
    act(() => editButton?.click());
    expect(mocks.useUnsavedChangesGuard).toHaveBeenLastCalledWith(false);

    const titleInput = emailRow?.querySelector<HTMLInputElement>(
      'input[placeholder="settings.sensitive-data.semantic-types.table.title"]'
    );
    act(() => {
      const descriptor = Object.getOwnPropertyDescriptor(
        HTMLInputElement.prototype,
        "value"
      );
      descriptor?.set?.call(titleInput, "Email address");
      titleInput?.dispatchEvent(new Event("input", { bubbles: true }));
    });
    expect(mocks.useUnsavedChangesGuard).toHaveBeenLastCalledWith(true);
  });

  test("preserves non-built-in bb-prefixed semantic types when saving", async () => {
    mocks.getSettingByName.mockReturnValue({
      value: {
        value: {
          case: "semanticType",
          value: {
            types: [
              { id: "bb.customer", title: "Customer" },
              { id: "email", title: "Email" },
            ],
          },
        },
      },
    });
    const container = document.createElement("div");
    const root = createRoot(container);
    roots.push(root);

    await act(async () => {
      root.render(<SemanticTypesPage />);
      await Promise.resolve();
    });

    const emailRow = Array.from(container.querySelectorAll("tbody tr")).find(
      (row) => row.querySelector("td:nth-child(2)")?.textContent === "email"
    );
    expect(emailRow).toBeDefined();
    const editButton = Array.from(
      emailRow?.querySelectorAll<HTMLButtonElement>("button") ?? []
    ).at(-1);
    expect(editButton).toBeDefined();
    act(() => editButton?.click());

    const titleInput = emailRow?.querySelector<HTMLInputElement>(
      'input[placeholder="settings.sensitive-data.semantic-types.table.title"]'
    );
    expect(titleInput).toBeDefined();
    act(() => {
      const descriptor = Object.getOwnPropertyDescriptor(
        HTMLInputElement.prototype,
        "value"
      );
      descriptor?.set?.call(titleInput, "Email address");
      titleInput?.dispatchEvent(new Event("input", { bubbles: true }));
    });

    const confirmButton = emailRow?.querySelector<HTMLButtonElement>(
      'button[aria-label="common.confirm"]'
    );
    expect(confirmButton?.disabled).toBe(false);
    act(() => confirmButton?.click());
    await act(async () => {
      await Promise.resolve();
    });

    const request = mocks.upsertSetting.mock.calls.at(-1)?.[0] as
      | {
          value?: {
            value?: {
              value?: { types?: Array<{ id: string }> };
            };
          };
        }
      | undefined;
    expect(request?.value?.value?.value?.types?.map(({ id }) => id)).toEqual([
      "bb.customer",
      "email",
    ]);
  });
});
