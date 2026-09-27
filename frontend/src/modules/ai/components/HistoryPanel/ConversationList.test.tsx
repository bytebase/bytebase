import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { Sheet, SheetContent, SheetTitle } from "@/components/ui/sheet";
import type { Conversation } from "../../types";
import { ConversationList } from "./ConversationList";

const mocks = vi.hoisted(() => ({
  deleteConversation: vi.fn(),
  emit: vi.fn(),
  setSelected: vi.fn(),
  updateConversation: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("scroll-into-view-if-needed", () => ({
  default: vi.fn(),
}));

vi.mock("@/modules/sql-editor/store/tab", () => ({
  useCurrentSQLEditorTab: () => ({
    connection: {
      database: "instances/test/databases/test",
      instance: "instances/test",
    },
  }),
}));

vi.mock("../../store", () => ({
  useConversationStore: () => ({
    deleteConversation: mocks.deleteConversation,
    updateConversation: mocks.updateConversation,
  }),
}));

const conversation: Conversation = {
  id: "conversation-1",
  created_ts: 1,
  name: "Original title",
  instance: "instances/test",
  database: "instances/test/databases/test",
  messageList: [],
};

vi.mock("../context", () => ({
  useAIContext: () => ({
    events: { emit: mocks.emit },
    chat: {
      list: [conversation],
      ready: true,
      selected: conversation,
      setSelected: mocks.setSelected,
    },
  }),
}));

describe("ConversationList", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    conversation.name = "Original title";
    mocks.updateConversation.mockResolvedValue(undefined);
  });

  test("renames a conversation in place", async () => {
    render(<ConversationList />);

    fireEvent.click(
      screen.getByRole("button", {
        name: "plugin.ai.conversation.rename",
      })
    );

    const input = screen.getByRole("textbox", {
      name: "plugin.ai.conversation.rename",
    }) as HTMLTextAreaElement;
    await waitFor(() => expect(input).toHaveFocus());
    expect(input).toHaveValue("Original title");
    expect(input.selectionStart).toBe(0);
    expect(input.selectionEnd).toBe("Original title".length);

    fireEvent.change(input, { target: { value: "Updated title" } });
    fireEvent.keyDown(input, { key: "Enter" });

    await waitFor(() => {
      expect(mocks.updateConversation).toHaveBeenCalledWith({
        ...conversation,
        name: "Updated title",
      });
    });
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
  });

  test("uses an auto-height textarea for wrapped titles", () => {
    conversation.name =
      "Which users have created the most query history entries, saved queries, issues, and task runs?";
    render(<ConversationList />);

    fireEvent.click(
      screen.getByRole("button", {
        name: "plugin.ai.conversation.rename",
      })
    );

    const textarea = screen.getByRole("textbox", {
      name: "plugin.ai.conversation.rename",
    }) as HTMLTextAreaElement;
    expect(textarea.tagName).toBe("TEXTAREA");
    expect(textarea).toHaveAttribute("rows", "1");
    expect(textarea).toHaveClass("resize-none", "overflow-hidden");

    Object.defineProperty(textarea, "scrollHeight", {
      configurable: true,
      value: 72,
    });
    fireEvent.change(textarea, {
      target: { value: `${conversation.name} More details.` },
    });
    expect(textarea.style.height).toBe("72px");
  });

  test("keeps the inline rename open inside the history sheet", async () => {
    render(
      <Sheet open>
        <SheetContent>
          <SheetTitle>History</SheetTitle>
          <ConversationList />
        </SheetContent>
      </Sheet>
    );

    fireEvent.click(
      screen.getByRole("button", {
        name: "plugin.ai.conversation.rename",
      })
    );

    await waitFor(() => {
      expect(
        screen.getByRole("textbox", {
          name: "plugin.ai.conversation.rename",
        })
      ).toBeVisible();
    });
  });

  test("opens the inline rename on primary pointer down", () => {
    render(<ConversationList />);

    fireEvent.pointerDown(
      screen.getByRole("button", {
        name: "plugin.ai.conversation.rename",
      }),
      { button: 0 }
    );

    expect(
      screen.getByRole("textbox", {
        name: "plugin.ai.conversation.rename",
      })
    ).toBeVisible();
  });

  test("cancels an inline rename with Escape", async () => {
    render(<ConversationList />);

    fireEvent.click(
      screen.getByRole("button", {
        name: "plugin.ai.conversation.rename",
      })
    );
    const input = screen.getByRole("textbox", {
      name: "plugin.ai.conversation.rename",
    });
    fireEvent.change(input, { target: { value: "Discarded title" } });
    fireEvent.keyDown(input, { key: "Escape" });

    await waitFor(() => {
      expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
    });
    expect(mocks.updateConversation).not.toHaveBeenCalled();
    expect(screen.getByText("Original title")).toBeInTheDocument();
  });

  test("saves an inline rename on blur", async () => {
    render(<ConversationList />);

    fireEvent.click(
      screen.getByRole("button", {
        name: "plugin.ai.conversation.rename",
      })
    );
    const input = screen.getByRole("textbox", {
      name: "plugin.ai.conversation.rename",
    });
    fireEvent.change(input, { target: { value: "Blurred title" } });
    fireEvent.blur(input);

    await waitFor(() => {
      expect(mocks.updateConversation).toHaveBeenCalledWith({
        ...conversation,
        name: "Blurred title",
      });
    });
  });

  test("keeps an untitled conversation as a placeholder until it is edited", () => {
    conversation.name = "";
    render(<ConversationList />);

    fireEvent.click(
      screen.getByRole("button", {
        name: "plugin.ai.conversation.rename",
      })
    );
    const input = screen.getByRole("textbox", {
      name: "plugin.ai.conversation.rename",
    });
    expect(input).toHaveValue("");
    expect(input).toHaveAttribute(
      "placeholder",
      "plugin.ai.conversation.untitled"
    );

    fireEvent.blur(input);
    expect(mocks.updateConversation).not.toHaveBeenCalled();
  });

  test("confirms deletion in an anchored popover", async () => {
    render(<ConversationList />);

    fireEvent.click(
      screen.getByRole("button", {
        name: "plugin.ai.conversation.delete",
      })
    );

    expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument();
    const confirmation = screen.getByText(
      "bbkit.confirm-button.sure-to-delete"
    );
    expect(confirmation.closest("[data-open]")).not.toBeNull();

    fireEvent.click(screen.getByText("common.delete"));

    await waitFor(() => {
      expect(mocks.deleteConversation).toHaveBeenCalledWith(conversation.id);
    });
    expect(mocks.setSelected).toHaveBeenCalledWith(undefined);
    expect(
      screen.queryByText("bbkit.confirm-button.sure-to-delete")
    ).not.toBeInTheDocument();
  });
});
