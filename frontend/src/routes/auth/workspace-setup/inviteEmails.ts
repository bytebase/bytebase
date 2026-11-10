type InviteEntry = {
  address: string;
  // Why the entry is shown but not sent.
  problem?: "invalid" | "own";
};

// The server's pattern for an account's email, which is lowercase, with one
// addition: the domain has a dot, as an invitee's address always does.
const EMAIL =
  /^[a-z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+$/;

const sameAddress = (a: string, b: string) =>
  a.toLowerCase() === b.toLowerCase();

// Separators between addresses, the full-width ones an input method types
// included.
const SEPARATORS = /[,;\n，；、]+/;

// True once `text` holds a character that ends the address before it.
export const endsAnAddress = (text: string) =>
  /\s/.test(text) || SEPARATORS.test(text);

// What a mail client or a sentence wraps an address in.
const unwrap = (token: string) =>
  token
    .trim()
    .replace(/^mailto:/i, "")
    .replace(/^[<("'[]+/, "")
    .replace(/[>)"'\].]+$/, "");

// Adds what was typed or pasted, one entry per address, dropping repeats. A
// recipient copied as `Ann Lee <ann@example.com>` is its bracketed address;
// the name beside it is not an entry.
export const addInviteAddresses = (
  existing: readonly string[],
  text: string
): string[] => {
  const next = [...existing];
  for (const recipient of text.split(SEPARATORS)) {
    const bracketed = [...recipient.matchAll(/<([^<>]+)>/g)].map(
      (match) => match[1] ?? ""
    );
    const tokens = bracketed.length > 0 ? bracketed : recipient.split(/\s+/);
    for (const token of tokens) {
      const address = unwrap(token);
      if (address && !next.some((entry) => sameAddress(entry, address))) {
        next.push(address);
      }
    }
  }
  return next;
};

export const describeInviteAddresses = (
  addresses: readonly string[],
  ownerEmail: string
): InviteEntry[] =>
  addresses.map((address) => ({
    address,
    problem: !EMAIL.test(address.toLowerCase())
      ? "invalid"
      : sameAddress(address, ownerEmail)
        ? "own"
        : undefined,
  }));

// The addresses Finish writes, in the lowercase form the server keeps.
export const validInviteAddresses = (
  addresses: readonly string[],
  ownerEmail: string
): string[] =>
  describeInviteAddresses(addresses, ownerEmail)
    .filter((entry) => !entry.problem)
    .map((entry) => entry.address.toLowerCase());
