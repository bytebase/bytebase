# Separate self-service account settings from admin account recovery

Status: proposal · 2026-08-18

## Problem

One component serves two audiences. `ProfilePage` is mounted both at `/setting/profile` (editing
yourself) and at `/users/:email` (looking at someone else), and the create-user sheet on the Users
page doubles as an edit form for the same fields. That produces:

- In one directory row, **clicking the row opens an edit sheet and clicking the name in that row
  opens a page**. Both mean "open this user."
- A single `Edit` toggle flips name, email, phone and **password** into edit mode together. Changing
  a password signs the user out of every session — silently, inside a generic "Save".
- One permission flag decides both "can I edit myself" and "can I edit this other person", so a
  normal user sees their own email as dead text with no explanation, and an admin viewing someone
  else gets a 2FA card whose copy is addressed to the account owner.
- In cloud, admins can't edit other users at all, so the page quietly degrades into a read-only
  screen that still looks editable.

## What admins actually do here

Almost always one of two things: **a user forgot their password**, or **a user lost their
authenticator**. Correcting a name or an email is rare.

Recovery matters most self-hosted, where there is no guaranteed mail server — SMTP is optional and
unconfigured by default, so an admin setting the password directly is the only recovery path inside
the product. (The `bytebase recovery` CLI covers the case where no admin can sign in either.) Two
related facts:

- The **"Forgot password" link shows even with no mail configured**, and the request is deliberately
  silent so it can't reveal whether an account exists. Users click it, are told to check their email,
  and nothing arrives.
- **Recovery codes already work at sign-in**, single-use, so a user who lost their authenticator but
  kept their codes never needs an admin. True today, invisible today.

## What admins can change

Email address, password, turning MFA off, deactivate/reactivate, roles, and display name and phone.

Turning MFA *on* for someone else is impossible — enrollment needs a secret from that person's own
setup — so the only coherent admin action is a reset.

In cloud, every one of these is blocked against other users **except roles**: name, phone, password,
MFA, email and deactivation are all self-only there, while role assignment through the workspace IAM
policy still works — it is how members are added in the first place.

These aren't one uniform "edit user" capability. Name and phone are ordinary profile fields the user
also edits themselves. Email, password, deactivation and roles each go through their own API and
permission — email in particular changes the identity the person signs in with. Different
consequences deserve different confirmations, so each gets its own dialog rather than sharing one
form.

## Prior art

**Metabase** is the closest analog — self-hosted, local passwords, optional SMTP. Admin actions live
in a **row menu**: edit, reset password, deactivate. With no SMTP configured it **shows the admin a
temporary password to hand over**. Okta offers the same choice explicitly; Keycloak marks admin-set
passwords temporary so they must be changed at next login.

**Linear** answers the other half. It has **no admin user page** — role and suspend live in the
members table — and a **hover popover** on avatars for identity at a glance. It does have a profile
page, but that page shows the person's assigned issues: a work view, not an account view.

Grafana and GitLab do have per-user admin pages, but they carry sessions, identities and org
membership. Page-versus-menu tracks how much state there is to inspect, not who is acting.

## Design

**Admin actions move to the Users row menu.** Reset password and Reset MFA sit at the top, above
edit, change email and deactivate. Recovery becomes two clicks with no navigation.

The overflow menu becomes the row's only affordance — the row itself stops being clickable, which is
what removes the two-destinations-per-row problem. The create-user sheet goes back to creating: its
edit mode disappears, and with it the password field it currently reuses for edits. Role assignment
is not in the menu and does not move; roles are workspace IAM rather than user data, and they stay on
the members page where they are managed alongside groups.

*Reset password*: generate or type a password, then show it once with a copy button, stating that it
must be delivered out of band and that the user's sessions have all been signed out.

*Reset MFA*: leads with the self-service path — if they still have recovery codes, no admin action is
needed. The confirmation states the real consequence, which differs by policy: when the workspace
requires MFA the user is forced into setup at next sign-in; when it doesn't, they simply have no MFA
until they re-enroll. Named **Reset MFA**, not "Disable 2FA" — the intent is recovery, and "disable"
is also what you'd call deliberately weakening an account.

**`/users/:email` is deleted, replaced by a hover card** showing **avatar, name, email, and a
"deactivated" badge**. Nothing else. The card answers one question — "who is this?" — asked in
passing about a name in an issue line. Email disambiguates people with the same display name;
deactivated tells you they've left. Roles and groups don't answer that question and overflow a
popover; last login would leak everyone's activity pattern, since every member can read user records.
Keeping it to identity means the card needs no IAM lookup and looks the same to every viewer.

Names become hover triggers rather than links. No redirect is kept — nothing links to the old route
after this change.

**Your own account stays one page**, at `/account`. There are only two editable fields — name and
phone — plus email, password and two-factor. That is a page with sections, not a set of tabs.

What has to change isn't the page count but where the password lives: it moves out of the shared form
into its own dialog, so changing it is a deliberate act with its own confirmation rather than a side
effect of "Save". Two-factor gets the same treatment. The existing 2FA setup wizard keeps its own
route, since it is a genuine multi-step flow.

The route moves because today `/setting/profile` sits as a sibling of `/setting/general` and
`/setting/subscription`, making "my account" a peer of "this workspace's billing" — the same
conflation as everything above, in the URL tree. Nobody does that: Sentry namespaces personal
settings under `/settings/account/*`, GitHub splits `/settings/*` (you) from `/orgs/:org/settings/*`
(the org), Grafana keeps personal at `/profile/*` outside the admin tree. Workspace settings stay
where they are.

(Bytebase's singular `/setting` is also off-convention — everyone uses plural. Renaming it touches
every workspace settings URL and is unrelated to this change, so it belongs in a separate cleanup.)

## Phasing

**Phase 1 — frontend only.** Everything above.

**Phase 2 — two small backend changes.**

1. **Make an admin-set password temporary.** Forced password change is already built end to end — the
   login response, reset page and route guard all exist. It just never fires for an admin reset, so
   today an admin-set password is a permanent credential the admin knows. Only the trigger is
   missing. (Related: an admin reset currently postpones password-rotation expiry.)
2. **Hide "Forgot password" when no mail is configured**, pointing at the workspace admin instead.
   Independent of everything above and shippable on its own.

## Alternatives

Keeping `/users/:email` as a read-only account card — the content is popover-sized and every inbound
link is incidental. Giving admins a dedicated page — recovery is a two-click errand, not a
destination. Branching harder on "is this me" inside one component — the bugs above come precisely
from one flag standing in for two different authorization questions.
