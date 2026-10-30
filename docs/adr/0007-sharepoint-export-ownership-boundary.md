# Leave delivered export governance with SharePoint administrators

For the proposed SharePoint export integration, delivering an export to SharePoint
means Bytebase sends the exported file directly to SharePoint instead of returning
its bytes to the caller. It does not promise to prevent copying visible query
results or downloading a delivered file through SharePoint.

Customer administrators provision destination sites, libraries, and folders and
configure their reader permissions. Authorized Bytebase users maintain each
project's destination mapping. Bytebase validates destinations and uploads files; it does not manage
SharePoint permissions, retention, or deletion. Reader policies may allow project-wide access or narrower
access, but Bytebase does not promise that file access matches database access.

After delivery, database grant expiration does not revoke access to the exported
file. This boundary leaves storage governance with the customer rather than
requiring Bytebase to reproduce and continuously synchronize database access
rules in SharePoint.

Live validation on October 21–22, 2026, using an application token that carried
only `Lists.SelectedOperations.Selected`, established how a library write grant
behaves:

- The grant is an ordinary SharePoint permission entry for the application. It
  does not bypass a folder's unique permissions; a folder without the entry
  returns `404 itemNotFound`, and an ungranted library returns `403`.
- Creating a grant adds the application to the library and to every folder in it
  that already has unique permissions. The SharePoint UI's Stop Inheriting copies
  the entry; scripted inheritance breaks that do not copy entries drop it.
- Re-issuing an existing grant changes nothing. Deleting and re-creating it
  restores the entry in every folder scope; deleting it removes the entry
  everywhere in the library.
- Delete unique permissions on a granted library removes Bytebase's write access,
  widens readers to the whole site, and removes the grant from Graph's permission
  list.
- The classic SharePoint permissions pages do not show the application.
- Libraries on a second site behaved the same as on the root site.
- After any grant on a site, the application can read the site name and every
  library's name and root on it, including libraries it cannot write to.
- A full SharePoint URL of a library root or folder, on the root site or another
  site, resolves to its drive item through Graph's `/shares` endpoint with only
  the Selected permission. A URL that does not exist returns the same
  `403 accessDenied` as one the application cannot access.

Customers are responsible for ensuring effective application write access on
each configured destination, including after permission inheritance changes.
Bytebase reports failures; it does not repair SharePoint permissions. Direct
per-folder application grants and end-user read access were not tested.
Evidence is in Linear issue BOT-156.

Each project has one configured destination: a document library root or a folder
within a document library. Projects may target different libraries across
multiple SharePoint sites using the workspace's shared application identity.
Bytebase does not select a different destination based on the exporting user.
Multiple Microsoft tenants or independently configured application identities are not
implied by support for multiple sites.

Configuring a storage destination approves exports to it. SharePoint is
offered when the workspace has the SharePoint plan feature and SharePoint
credentials and the project has a SharePoint folder. Download is the only
Export Destination with a policy switch: the workspace query data policy's
`disable_download` turns it off. It defaults to false, so an upgrade keeps
Download. A separate SharePoint switch was dropped because configuring a
storage destination is itself the approval. The Export Permission
(`bb.sql.export`, ADR 0006) decides who may export, and the available
destinations decide where exports may go. An approved export access grant
satisfies the Export Permission but never adds a destination, so with Download
disabled and no project SharePoint folder, export is refused for everyone,
grant holders too. The Export API enforces `disable_download` for every
caller, including MCP; hiding the UI is not enforcement. The setting stays in
effect if the workspace license lapses; only changing it requires the license.

Bytebase is neutral about how customers isolate projects. It uses whatever
library root or folder is configured and does not recommend one structure or
warn about folder destinations. Keeping Bytebase's access intact through later
permission changes is the customer's SharePoint administration. Customers make
grants with their own administrator account; Bytebase never holds
`Sites.FullControl.All` or performs grants itself.

Upload failures return an actionable error after bounded retries, without a
fallback to Download or durable background delivery in the initial version.

A project's destination is part of the project's settings and is changed with
the existing `bb.projects.update` permission, so Project Owners can change it.

A user who can update the project can select any library root or folder writable by the
workspace application identity, including a destination used by another project.
There is no separate approved-site, library, or folder allowlist. Customers grant
the application access to the required libraries in SharePoint. Bytebase
resolves the destination and audits mapping changes; customers remain
responsible for its audience.

Each export is delivered into its own uniquely named subfolder of the destination
to preserve existing files. Retries of the same export target the same subfolder
and file names rather than creating additional copies.
The subfolder name contains the UTC export time, the exporter's email, the
database, and the export ID, because SharePoint attributes the files to the
Bytebase application rather than to the exporting user.

An export captures its destination when it starts: the configured URL and the
folder it resolved to. Mapping changes affect subsequent exports; the running
export and its retries upload to the captured folder. The audit log records the
actual destination.

Delivery is confirmed only when every file of the export is confirmed. Confirmed
delivery returns the URL of the export subfolder. Delivery success does not
imply that the exporting user can read the file; SharePoint
enforces that user's access independently, and Bytebase does not grant access.

If an upload response is lost, Bytebase attempts bounded reconciliation against
the same export file names. If delivery remains uncertain, Bytebase reports that
delivery could not be confirmed and files may exist in SharePoint. It does
not claim that no file was created.

Bytebase checks the destination once, before the query runs. A change made
while an export is running applies to the next export. A rejected destination
is never redirected to another one.

Internal upload retries reuse the generated files, file names, and captured
destination without rerunning the query. A new Export action starts a new export
and reruns the query.

The initial release supports self-hosted Bytebase on any hosting platform using
client-secret authentication. It does not require AKS or another platform
identity provider. Bytebase Cloud support, workload identity federation, and
certificate authentication are outside the initial scope.

The client secret is configured in the workspace export setting through a
write-only field. Workspace credential configuration uses the existing
`bb.settings.set` permission, following other workspace integrations. Workspace
Admin and Workspace DBA have that permission by default; custom roles can
delegate it with its existing broader settings authority.

A destination is configured as the full SharePoint URL of a library root or a
folder, which carries the site and hostname. Administrators may paste a browser
URL, including a library view with an `id` parameter or a copied folder link;
Bytebase normalizes it to the canonical URL and resolves it on save, rejecting a
URL that does not resolve. It stores and returns only the canonical URL; resolved
names go stale on rename and are not kept. A SharePoint browsing picker is
outside the initial scope.

Bytebase resolves the stored URL at each export rather than pinning item IDs.
Renaming or moving the folder makes exports fail as not found; a folder
recreated at the same URL receives subsequent exports.

Customers create and rotate Entra client secrets. Expired or invalid credentials
cause affected exports to fail with a clear credential error. An authorized user
can replace the secret without changing project mappings; Bytebase does not
create or renew Entra credentials.

When both destinations are available, Export defaults to SharePoint and offers
Download as an explicit alternative. Disabling Download removes that
alternative.

Destination validation in the initial release only resolves the configured URL.
A destination that resolves is accepted, not verified: the application can read a library root it
cannot write to, so a successful read does not prove upload access. The first
export is the actual write check. SharePoint does not distinguish a missing
destination from one Bytebase cannot access, so errors report both
possibilities. A write probe is a later improvement.

The initial release preserves existing export limits and result formats. Changing
the export destination does not expand export capacity.

SharePoint delivery uploads the export's statement and result files directly
instead of a zip, so readers can open each result in the browser without
downloading it. Readers of the destination therefore also see the query text.
Zip passwords apply only to Download. Batch export
is a client-side composition of per-database exports: each database delivers
its own files, and the batch reports delivered, failed, and unconfirmed
databases separately without rolling back delivered files.

The initial release supports the Microsoft global commercial cloud only.
Government and sovereign clouds use different sign-in and Graph endpoints.

SharePoint delivery is an Enterprise plan feature, on the same plan as
restricting data copying.
