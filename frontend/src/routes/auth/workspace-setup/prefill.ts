// Local parts that name a mailbox rather than a person.
const SHARED_MAILBOXES = new Set([
  "accounts",
  "admin",
  "administrator",
  "billing",
  "contact",
  "data",
  "dba",
  "dev",
  "devops",
  "engineering",
  "finance",
  "hello",
  "help",
  "hr",
  "info",
  "it",
  "mail",
  "marketing",
  "noreply",
  "office",
  "ops",
  "postmaster",
  "root",
  "sales",
  "security",
  "service",
  "support",
  "team",
  "tech",
  "test",
  "webmaster",
]);

// Mail providers, by the name part of their domains, which says nothing about
// the account's company. Matching the name covers their country domains too
// (yahoo.co.jp, hotmail.co.uk, outlook.fr).
const PUBLIC_MAIL_PROVIDERS = new Set([
  "126",
  "139",
  "163",
  "aliyun",
  "aol",
  "att",
  "btinternet",
  "comcast",
  "daum",
  "duck",
  "fastmail",
  "foxmail",
  "free",
  "gmail",
  "gmx",
  "googlemail",
  "hanmail",
  "hey",
  "hotmail",
  "icloud",
  "laposte",
  "libero",
  "live",
  "mac",
  "mail",
  "me",
  "msn",
  "naver",
  "orange",
  "outlook",
  "pm",
  "proton",
  "protonmail",
  "qq",
  "rediffmail",
  "seznam",
  "sina",
  "sohu",
  "t-online",
  "tuta",
  "tutanota",
  "verizon",
  "web",
  "yahoo",
  "yandex",
  "ymail",
  "zoho",
]);

// Labels that sit between a company and its country code, as in "acme.co.uk".
const SECOND_LEVEL_LABELS = new Set([
  "ac",
  "co",
  "com",
  "edu",
  "go",
  "gov",
  "ne",
  "net",
  "or",
  "org",
]);

// The title the server gives a workspace nobody has named yet. Its letter case
// differs by how the account signed up: "Default workspace" by email,
// "Default Workspace" through an identity provider.
const UNNAMED_WORKSPACE_TITLE = "Default workspace";

const capitalize = (word: string) =>
  word.charAt(0).toUpperCase() + word.slice(1);

const sameText = (a: string, b: string) => a.toLowerCase() === b.toLowerCase();

export const guessDisplayName = (
  email: string,
  title: string | undefined
): string => {
  const localPart = email.split("@")[0] ?? "";
  const given = title?.trim() ?? "";
  // An account created without a name carries its address as one.
  if (given && !sameText(given, email) && !sameText(given, localPart)) {
    return given;
  }
  const base = localPart.split("+")[0] ?? "";
  const words = base.split(/[._-]+/).filter(Boolean);
  const namesNobody =
    !/\p{L}/u.test(base) ||
    SHARED_MAILBOXES.has(words.join("").toLowerCase()) ||
    words.some((word) => SHARED_MAILBOXES.has(word.toLowerCase()));
  return namesNobody ? "" : words.map(capitalize).join(" ");
};

// The company part of the address's domain: the label before the top-level
// domain, or before a second-level one such as "co".
const guessCompany = (email: string): string | undefined => {
  const labels = (email.split("@")[1] ?? "").toLowerCase().split(".");
  labels.pop();
  const last = labels.at(-1);
  if (labels.length > 1 && last && SECOND_LEVEL_LABELS.has(last)) labels.pop();
  const company = labels.at(-1);
  if (!company || PUBLIC_MAIL_PROVIDERS.has(company)) return undefined;
  return company.split("-").map(capitalize).join(" ");
};

type WorkspaceTitleGuess =
  | { kind: "title"; title: string }
  | { kind: "personal"; firstName: string }
  | { kind: "anonymous" };

// A workspace that already has a real name keeps it; otherwise the company
// behind the address names it, and a personal address names it after its owner.
export const guessWorkspaceTitle = (
  email: string,
  displayName: string,
  currentTitle: string | undefined
): WorkspaceTitleGuess => {
  const existing = currentTitle?.trim();
  if (existing && !sameText(existing, UNNAMED_WORKSPACE_TITLE)) {
    return { kind: "title", title: existing };
  }
  const company = guessCompany(email);
  if (company) return { kind: "title", title: company };
  const firstName = displayName.split(/\s+/)[0];
  return firstName ? { kind: "personal", firstName } : { kind: "anonymous" };
};
