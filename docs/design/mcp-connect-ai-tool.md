# MCP settings page: connect your AI tool

Status: implemented in this PR · 2026-11-25 · [BYT-10594](https://linear.app/bytebase/issue/BYT-10594)

The workspace MCP settings page tells a user how to connect one AI tool to this Bytebase instance.
**The section becomes three numbered steps: add Bytebase to your AI tool, sign in, try a prompt.
Step 1 shows one way per tool, the easiest one that works for that tool. Every other way lives in
the setup guide in the docs.**
Frontend only: no proto, API, store or migration change.

The mock-ups and the four prototype variants are kept outside the repo.

## Problem

The reader is someone who wants an AI tool to work with their Bytebase. They have the tool open and
they want the shortest path from this page to a first answer.

| # | Problem | What it causes |
|---|---|---|
| 1 | The Claude Desktop tab says to add `<url>/mcp` as a custom connector on claude.ai. Claude reaches a connector from Anthropic's servers, not from the user's computer. | A self-hosted instance behind a firewall cannot be connected at all ([BYT-10594](https://linear.app/bytebase/issue/BYT-10594)). |
| 2 | Each command sits in a one-line field. | A command with a long External URL is cut off. |
| 3 | The page stops at the command. It does not say where sign-in happens or what to click. | The user runs the command and then does not know why the tool shows no Bytebase tools. |
| 4 | The page calls the tool a "client" and an "AI assistant". | "AI assistant" is the name of the AI features built into Bytebase, so the page reads as if it were about those. |

## Principle

> One way per tool on the page. The page says what to do, then what happens next. Anything a user
> could copy must be usable as copied.

## What each tool allows

The page can only offer what the tool supports. How each fact was checked is in the last column.

| Tool | How a remote MCP server is added | Checked by |
|---|---|---|
| Claude Code | `claude mcp add --transport http <name> <url>`. Sign-in starts from `/mcp`. | `claude mcp add --help`, Claude Code 2.1.330 |
| Claude Desktop | The config file `claude_desktop_config.json` takes only servers that Claude starts with a command. An entry with a URL and no command is skipped, and the app says so in a dialog. A connector takes a URL but connects from Anthropic's servers. An MCP Bundle (`.mcpb`) installs with a double-click. | Read in the app's code, build of 2026-11-21. The dialog was not triggered. |
| Codex | `codex mcp add <name> --url <url>`. The CLI, the IDE extension and the desktop app share one config file. | `codex mcp add --help`, CLI 0.171.0; OpenAI's Codex MCP docs |
| Copilot CLI | `copilot mcp add --transport http <name> <url>` | Carried over from the page before this change; not rechecked |
| Gemini CLI | `gemini mcp add --transport http <name> <url>` | Carried over; not rechecked. Its successor is tracked in [BYT-10613](https://linear.app/bytebase/issue/BYT-10613). |
| VS Code | A link that starts with `vscode:mcp/install?` and carries the server as URL-encoded JSON opens VS Code on that server's page, with an Install button. `code --add-mcp '<json>'` does the same from a terminal. | `code --help`, VS Code 1.145.2, for the command. The link handler was read in the installed app's code; the link was not clicked through. |

For Claude Desktop this leaves one path that works against a private instance today: a command
entry in the config file that starts the npm package
[`@bytebase/bytebase-mcp`](https://www.npmjs.com/package/@bytebase/bytebase-mcp). The package runs
on the user's computer and forwards to this instance's `/mcp`.

## Decisions

**D1 · Three numbered steps.** Add Bytebase to your AI tool, sign in, try a prompt. The tool picked
in step 1 decides what step 2 says, because how sign-in starts differs by tool. The former "Your
first prompt" section becomes step 3. Rejected: tabs with one action each and nothing after, since
sign-in is where users get lost and it had no place; and a list of all tools down the page, which is
long and mostly about tools the reader does not use.

**D2 · The page says "AI tool".** Not "client", which is the protocol's word and not the user's, and
not "AI assistant", which names Bytebase's own AI features. This covers the page description too.

**D3 · One way per tool on the page.** The page shows the easiest way that works. Other ways (a
config file edited by hand, a tool's settings screen) go in the setup guide. Rejected: a tab that
shows how to run the npm package on its own. Bytebase is already running, so every tab here is about
connecting a tool, and the package is only the means for a tool that cannot take a URL.

**D4 · Claude Desktop: an entry in its config file.** The tab gives three steps: open the config
file from the app's settings, open `claude_desktop_config.json` in a text editor, add the entry and
save. The entry starts `npx -y @bytebase/bytebase-mcp` with `BYTEBASE_URL` set to the External URL.
Rejected, each for now:

- *A connector.* It cannot reach a private instance, which is the bug this change fixes. It stays a
  valid way for an instance reachable from the internet, and moves to the setup guide.
- *A command that edits the file for the user.* The npm package has no such command.
- *The bundle.* It is the easiest way, one double-click and one field, but it is not released.
  When it is, it replaces the entry on this tab.

The tab does not say where in the file the entry goes. The setup guide's before-and-after example
does. See "Known limits".

**D5 · VS Code: an install link, with its setup guide beside it.** The tab shows an "Install in VS
Code" button and a "VS Code setup guide" link. Rejected: the `code --add-mcp` command alone, because
the `code` command is not on the path by default on macOS; the button and the command together,
because that is two ways on one tab; and choosing between them by whether VS Code is installed,
because a web page cannot know that. The command moves to the guide.

**D6 · The last tab is "Other".** It was "JSON". It holds the config for a tool that has no tab,
which is a normal path and not a fallback, so it is named for who uses it and not for its format.

**D7 · The setup guide is linked from the section heading.** One "Setup guide" link sits in the
heading row, where the access policy section above puts "Edit policy". Tabs whose way has a known
other way link to their own part of the guide.

| Link on the page | Where it goes |
|---|---|
| Setup guide | `https://docs.bytebase.com/integrations/mcp/setup` |
| See an example, on Claude Desktop | `https://docs.bytebase.com/integrations/mcp/setup#claude-desktop` |
| VS Code setup guide, on VS Code | `https://docs.bytebase.com/integrations/mcp/setup#vs-code` |

**D8 · A control that carries the External URL is off until the URL is set.** See "States". A tool
set up with the placeholder address cannot reach Bytebase, and the failure shows later, in the tool,
with no pointer back to this page. Rejected: leaving copy on and trusting the reader to replace the
placeholder, which is what the page did.

**D9 · Step 2 takes the button name from the consent page.** The sentence ends "click Allow access",
and the two words come from the consent page's own string, so the two cannot drift apart.

**D10 · A code block says where its text goes.** Each block has a heading strip: "Run in your
terminal", the file name, "Add to your AI tool's MCP config", "Ask your AI tool". The copy button
sits in the strip, so it never covers the text, and a long command wraps instead of being cut off.

## Copy

| Place | Text |
|---|---|
| Page description | Connect AI tools to Bytebase using the Model Context Protocol. |
| Section title | Connect your AI tool |
| Step 1 | Add Bytebase to your AI tool |
| Step 2 | Sign in |
| Step 2, default | The first time your AI tool uses Bytebase, a browser window opens. Sign in to Bytebase and click Allow access. |
| Step 2, Claude Code | In Claude Code, run /mcp, select bytebase, and choose Authenticate. In the browser window that opens, sign in to Bytebase and click Allow access. |
| Step 2, Claude Desktop | Quit and reopen Claude Desktop. In the browser window that opens, sign in to Bytebase and click Allow access. |
| Step 3 | Try a prompt |
| Claude Desktop, 1 | In Claude Desktop, click your name in the lower-left corner, then **Settings > Desktop app > Extensions > Edit config**. A folder opens. |
| Claude Desktop, 2 | From that folder, open `claude_desktop_config.json` in a text editor. |
| Claude Desktop, 3 | Add this to the file and save it. |
| Claude Desktop, below | Requires Node.js 20 or later. |

Claude Desktop's menu path and file name are the app's own words. They are constants in the
component and are not translated.

## States

Every value in step 1 carries the instance's External URL. When it is not set, values show the
placeholder `{https://your-bytebase-url.com}`.

| What the control does | External URL set | External URL not set |
|---|---|---|
| Copies text that holds the URL: a command or a config entry | Copy works | The text shows the placeholder. Copy is off, and a line below says "Copy is off until the External URL is set." |
| Opens a link that holds the URL: Install in VS Code | A link | A disabled button, and a line below says "Install is off until the External URL is set." No link holding the placeholder is in the page. |
| Holds no URL: the first prompt, the guide links | Enabled | Enabled |

The reason is always visible text. It is never only a tooltip, because it is the one thing that
tells the user why the control is off. The page's existing External URL alert, with its "Configure
now" link, stays the way to fix it.

## Known limits

- **A pasted entry can break or displace what is in the file.** `claude_desktop_config.json`
  usually holds other settings. A missing comma or brace makes it invalid, and a second
  `mcpServers` key displaces the servers already there. The tab leaves this to the setup guide's
  example (D4). The bundle removes the problem.
- **The npm package needs a release before this ships.** Version 0.1.0 does not give Bytebase tools
  to Code and Cowork sessions in Claude Desktop
  ([BYT-10611](https://linear.app/bytebase/issue/BYT-10611)). Chat works.
- **Starting the package through `npx` from Claude Desktop is not tested.** Claude Desktop may not
  find `npx` when Node.js was installed with a version manager.
- **The links into the setup guide lead nowhere until the guide is published.** The page and its two
  parts were checked on the docs preview. See "Scope".

## Scope

1. **This PR.** The section, its copy in all five locales, and its tests.
   - `frontend/src/routes/workspace/mcp/MCPConnectSection.tsx` holds the section;
     `frontend/src/routes/workspace/MCPPage.tsx` only composes the page.
   - `MCPConnectSection.test.tsx`: no command uses a flag its CLI does not have; the Claude Desktop
     entry starts the npm package with this instance's address; the VS Code link carries the
     endpoint; step 2 follows the selected tool; one test per row of "States".
   - `frontend/tests/e2e/workspace/workspace-mcp-policy.spec.ts`: the assertion that read the
     removed description now reads the "Sign in" step.
2. **The setup guide**, in the docs repository ([bytebase/bytebase.com#307](https://github.com/bytebase/bytebase.com/pull/307)),
   at `/integrations/mcp/setup`. A part per tool with the other ways to connect, a before-and-after
   example of the Claude Desktop file, the `code --add-mcp` command, and troubleshooting. It has to
   be published before this change ships, so the three links in D7 land on a page that exists.
3. **An npm release** of `@bytebase/bytebase-mcp` with the Claude Desktop fixes.

## Not in this PR

- The bundle and its release.
- Replacing the Gemini CLI tab with its successor ([BYT-10613](https://linear.app/bytebase/issue/BYT-10613)).
- A Cursor tab.
- Showing the connector on Bytebase Cloud, where it works and needs no Node.js. It is in the guide.
