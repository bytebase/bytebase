-- SQL to create the initial tables for the MediaWiki database.
-- This is read and executed by the install script; you should
-- not have to run it by itself unless doing a manual install.

-- This is a shared schema file used for both MySQL and SQLite installs.
--
-- For more documentation on the database schema, see
-- https://www.mediawiki.org/wiki/Manual:Database_layout
--
-- General notes:
--
-- If possible, create tables as InnoDB to benefit from the
-- superior resiliency against crashes and ability to read
-- during writes (and write during reads!)
--
-- Only the 'searchindex' table requires MyISAM due to the
-- requirement for fulltext index support, which is missing
-- from InnoDB.
--
--
-- The MySQL table backend for MediaWiki currently uses
-- 14-character BINARY or VARBINARY fields to store timestamps.
-- The format is YYYYMMDDHHMMSS, which is derived from the
-- text format of MySQL's TIMESTAMP fields.
--
-- Historically TIMESTAMP fields were used, but abandoned
-- in early 2002 after a lot of trouble with the fields
-- auto-updating.
--
-- The Postgres backend uses TIMESTAMPTZ fields for timestamps,
-- and we will migrate the MySQL definitions at some point as
-- well.
--
--
-- The  comments in this and other files are
-- replaced with the defined table prefix by the installer
-- and updater scripts. If you are installing or running
-- updates manually, you will need to manually insert the
-- table prefix if any when running these scripts.
--


--
-- The user table contains basic account information,
-- authentication keys, etc.
--
-- Some multi-wiki sites may share a single central user table
-- between separate wikis using the $wgSharedDB setting.
--
-- Note that when a external authentication plugin is used,
-- user table entries still need to be created to store
-- preferences and to key tracking information in the other
-- tables.
--
CREATE TABLE user (
  user_id int unsigned NOT NULL PRIMARY KEY AUTO_INCREMENT,

  -- Usernames must be unique, must not be in the form of
  -- an IP address. _Shouldn't_ allow slashes or case
  -- conflicts. Spaces are allowed, and are _not_ converted
  -- to underscores like titles. See the User::newFromName() for
  -- the specific tests that usernames have to pass.
  user_name varchar(255) binary NOT NULL default '',

  -- Optional 'real name' to be displayed in credit listings
  user_real_name varchar(255) binary NOT NULL default '',

  -- Password hashes, see User::crypt() and User::comparePasswords()
  -- in User.php for the algorithm
  user_password tinyblob NOT NULL,

  -- When using 'mail me a new password', a random
  -- password is generated and the hash stored here.
  -- The previous password is left in place until
  -- someone actually logs in with the new password,
  -- at which point the hash is moved to user_password
  -- and the old password is invalidated.
  user_newpassword tinyblob NOT NULL,

  -- Timestamp of the last time when a new password was
  -- sent, for throttling and expiring purposes
  -- Emailed passwords will expire $wgNewPasswordExpiry
  -- (a week) after being set. If user_newpass_time is NULL
  -- (eg. created by mail) it doesn't expire.
  user_newpass_time binary(14),

  -- Note: email should be restricted, not public info.
  -- Same with passwords.
  user_email tinytext NOT NULL,

  -- If the browser sends an If-Modified-Since header, a 304 response is
  -- suppressed if the value in this field for the current user is later than
  -- the value in the IMS header. That is, this field is an invalidation timestamp
  -- for the browser cache of logged-in users. Among other things, it is used
  -- to prevent pages generated for a previously logged in user from being
  -- displayed after a session expiry followed by a fresh login.
  user_touched binary(14) NOT NULL default '',

  -- A pseudorandomly generated value that is stored in
  -- a cookie when the "remember password" feature is
  -- used (previously, a hash of the password was used, but
  -- this was vulnerable to cookie-stealing attacks)
  user_token binary(32) NOT NULL default '',

  -- Initially NULL; when a user's e-mail address has been
  -- validated by returning with a mailed token, this is
  -- set to the current timestamp.
  user_email_authenticated binary(14),

  -- Randomly generated token created when the e-mail address
  -- is set and a confirmation test mail sent.
  user_email_token binary(32),

  -- Expiration date for the user_email_token
  user_email_token_expires binary(14),

  -- Timestamp of account registration.
  -- Accounts predating this schema addition may contain NULL.
  user_registration binary(14),

  -- Count of edits and edit-like actions.
  --
  -- *NOT* intended to be an accurate copy of COUNT(*) WHERE rev_user=user_id
  -- May contain NULL for old accounts if batch-update scripts haven't been
  -- run, as well as listing deleted edits and other myriad ways it could be
  -- out of sync.
  --
  -- Meant primarily for heuristic checks to give an impression of whether
  -- the account has been used much.
  --
  user_editcount int,

  -- Expiration date for user password.
  user_password_expires varbinary(14) DEFAULT NULL

) ENGINE=InnoDB DEFAULT CHARSET=binary;

CREATE UNIQUE INDEX user_name ON user (user_name);
CREATE INDEX user_email_token ON user (user_email_token);
CREATE INDEX user_email ON user (user_email(50));


--
-- The "actor" table associates user names or IP addresses with integers for
-- the benefit of other tables that need to refer to either logged-in or
-- logged-out users. If something can only ever be done by logged-in users, it
-- can refer to the user table directly.
--
CREATE TABLE actor (
  -- Unique ID to identify each actor
  actor_id bigint unsigned NOT NULL PRIMARY KEY AUTO_INCREMENT,

  -- Key to user.user_id, or NULL for anonymous edits.
  actor_user int unsigned,

  -- Text username or IP address
  actor_name varchar(255) binary NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=binary;

-- User IDs and names must be unique.
CREATE UNIQUE INDEX actor_user ON actor (actor_user);
CREATE UNIQUE INDEX actor_name ON actor (actor_name);


--
-- User permissions have been broken out to a separate table;
-- this allows sites with a shared user table to have different
-- permissions assigned to a user in each project.
--
-- This table replaces the old user_rights field which used a
-- comma-separated blob.
--
CREATE TABLE user_groups (
  -- Key to user_id
  ug_user int unsigned NOT NULL default 0,

  -- Group names are short symbolic string keys.
  -- The set of group names is open-ended, though in practice
  -- only some predefined ones are likely to be used.
  --
  -- At runtime $wgGroupPermissions will associate group keys
  -- with particular permissions. A user will have the combined
  -- permissions of any group they're explicitly in, plus
  -- the implicit '*' and 'user' groups.
  ug_group varbinary(255) NOT NULL default '',

  -- Time at which the user group membership will expire. Set to
  -- NULL for a non-expiring (infinite) membership.
  ug_expiry varbinary(14) NULL default NULL,

  PRIMARY KEY (ug_user, ug_group)
) ENGINE=InnoDB DEFAULT CHARSET=binary;

CREATE INDEX ug_group ON user_groups (ug_group);
CREATE INDEX ug_expiry ON user_groups (ug_expiry);

-- Stores the groups the user has once belonged to.
-- The user may still belong to these groups (check user_groups).
-- Users are not autopromoted to groups from which they were removed.
CREATE TABLE user_former_groups (
  -- Key to user_id
  ufg_user int unsigned NOT NULL default 0,
  ufg_group varbinary(255) NOT NULL default '',
  PRIMARY KEY (ufg_user,ufg_group)
) ENGINE=InnoDB DEFAULT CHARSET=binary;

--
-- Stores notifications of user talk page changes, for the display
-- of the "you have new messages" box
--
CREATE TABLE user_newtalk (
  -- Key to user.user_id
  user_id int unsigned NOT NULL default 0,
  -- If the user is an anonymous user their IP address is stored here
  -- since the user_id of 0 is ambiguous
  user_ip varbinary(40) NOT NULL default '',
  -- The highest timestamp of revisions of the talk page viewed
  -- by this user
  user_last_timestamp varbinary(14) NULL default NULL
) ENGINE=InnoDB DEFAULT CHARSET=binary;

-- Indexes renamed for SQLite in 1.14
CREATE INDEX un_user_id ON user_newtalk (user_id);
CREATE INDEX un_user_ip ON user_newtalk (user_ip);


--
-- User preferences and perhaps other fun stuff. :)
-- Replaces the old user.user_options blob, with a couple nice properties:
--
-- 1) We only store non-default settings, so changes to the defauls
--    are now reflected for everybody, not just new accounts.
-- 2) We can more easily do bulk lookups, statistics, or modifications of
--    saved options since it's a sane table structure.
--
CREATE TABLE user_properties (
  -- Foreign key to user.user_id
  up_user int unsigned NOT NULL,

  -- Name of the option being saved. This is indexed for bulk lookup.
  up_property varbinary(255) NOT NULL,

  -- Property value as a string.
  up_value blob,
  PRIMARY KEY (up_user,up_property)
) ENGINE=InnoDB DEFAULT CHARSET=binary;

CREATE INDEX user_properties_property ON user_properties (up_property);

--
-- This table contains a user's bot passwords: passwords that allow access to
-- the account via the API with limited rights.
--
CREATE TABLE bot_passwords (
  -- User ID obtained from CentralIdLookup.
  bp_user int unsigned NOT NULL,

  -- Application identifier
  bp_app_id varbinary(32) NOT NULL,

  -- Password hashes, like user.user_password
  bp_password tinyblob NOT NULL,

  -- Like user.user_token
  bp_token binary(32) NOT NULL default '',

  -- JSON blob for MWRestrictions
  bp_restrictions blob NOT NULL,

  -- Grants allowed to the account when authenticated with this bot-password
  bp_grants blob NOT NULL,

  PRIMARY KEY ( bp_user, bp_app_id )
) ENGINE=InnoDB DEFAULT CHARSET=binary;

--
-- Core of the wiki: each page has an entry here which identifies
-- it by title and contains some essential metadata.
--
CREATE TABLE page (
  -- Unique identifier number. The page_id will be preserved across
  -- edits and rename operations, but not deletions and recreations.
  page_id int unsigned NOT NULL PRIMARY KEY AUTO_INCREMENT,

  -- A page name is broken into a namespace and a title.
  -- The namespace keys are UI-language-independent constants,
  -- defined in includes/Defines.php
  page_namespace int NOT NULL,

  -- The rest of the title, as text.
  -- Spaces are transformed into underscores in title storage.
  page_title varchar(255) binary NOT NULL,

  -- Comma-separated set of permission keys indicating who
  -- can move or edit the page.
  page_restrictions tinyblob NOT NULL,

  -- 1 indicates the article is a redirect.
  page_is_redirect tinyint unsigned NOT NULL default 0,

  -- 1 indicates this is a new entry, with only one edit.
  -- Not all pages with one edit are new pages.
  page_is_new tinyint unsigned NOT NULL default 0,

  -- Random value between 0 and 1, used for Special:Randompage
  page_random real unsigned NOT NULL,

  -- This timestamp is updated whenever the page changes in
  -- a way requiring it to be re-rendered, invalidating caches.
  -- Aside from editing this includes permission changes,
  -- creation or deletion of linked pages, and alteration
  -- of contained templates.
  page_touched binary(14) NOT NULL default '',

  -- This timestamp is updated whenever a page is re-parsed and
  -- it has all the link tracking tables updated for it. This is
  -- useful for de-duplicating expensive backlink update jobs.
  page_links_updated varbinary(14) NULL default NULL,

  -- Handy key to revision.rev_id of the current revision.
  -- This may be 0 during page creation, but that shouldn't
  -- happen outside of a transaction... hopefully.
  page_latest int unsigned NOT NULL,

  -- Uncompressed length in bytes of the page's current source text.
  page_len int unsigned NOT NULL,

  -- content model, see CONTENT_MODEL_XXX constants
  page_content_model varbinary(32) DEFAULT NULL,

  -- Page content language
  page_lang varbinary(35) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=binary;

-- The title index. Care must be taken to always specify a namespace when
-- by title, so that the index is used. Even listing all known namespaces
-- with IN() is better than omitting page_namespace from the WHERE clause.
CREATE UNIQUE INDEX name_title ON page (page_namespace,page_title);

-- The index for Special:Random
CREATE INDEX page_random ON page (page_random);

-- Questionable utility, used by ProofreadPage, possibly DynamicPageList.
-- ApiQueryAllPages unconditionally filters on namespace and so hopefully does
-- not use it.
CREATE INDEX page_len ON page (page_len);

-- The index for Special:Shortpages and Special:Longpages. Also SiteStats::articles()
-- in 'comma' counting mode, MessageCache::loadFromDB().
CREATE INDEX page_redirect_namespace_len ON page (page_is_redirect, page_namespace, page_len);

--
-- Every edit of a page creates also a revision row.
-- This stores metadata about the revision, and a reference
-- to the text storage backend.
--
CREATE TABLE revision (
  -- Unique ID to identify each revision
  rev_id int unsigned NOT NULL PRIMARY KEY AUTO_INCREMENT,

  -- Key to page_id. This should _never_ be invalid.
  rev_page int unsigned NOT NULL,

  -- Key to text.old_id, where the actual bulk text is stored.
  -- It's possible for multiple revisions to use the same text,
  -- for instance revisions where only metadata is altered
  -- or a rollback to a previous version.
  -- @deprecated since 1.31. If rows in the slots table with slot_revision_id = rev_id
  -- exist, this field should be ignored (and may be 0) in favor of the
  -- corresponding data from the slots and content tables
  rev_text_id int unsigned NOT NULL default 0,

  -- Text comment summarizing the change. Deprecated in favor of
  -- revision_comment_temp.revcomment_comment_id.
  rev_comment varbinary(767) NOT NULL default '',

  -- Key to user.user_id of the user who made this edit.
  -- Stores 0 for anonymous edits and for some mass imports.
  -- Deprecated in favor of revision_actor_temp.revactor_actor.
  rev_user int unsigned NOT NULL default 0,

  -- Text username or IP address of the editor.
  -- Deprecated in favor of revision_actor_temp.revactor_actor.
  rev_user_text varchar(255) binary NOT NULL default '',

  -- Timestamp of when revision was created
  rev_timestamp binary(14) NOT NULL default '',

  -- Records whether the user marked the 'minor edit' checkbox.
  -- Many automated edits are marked as minor.
  rev_minor_edit tinyint unsigned NOT NULL default 0,

  -- Restrictions on who can access this revision
  rev_deleted tinyint unsigned NOT NULL default 0,

  -- Length of this revision in bytes
  rev_len int unsigned,

  -- Key to revision.rev_id
  -- This field is used to add support for a tree structure (The Adjacency List Model)
  rev_parent_id int unsigned default NULL,

  -- SHA-1 text content hash in base-36
  rev_sha1 varbinary(32) NOT NULL default '',

  -- content model, see CONTENT_MODEL_XXX constants
  -- @deprecated since 1.31. If rows in the slots table with slot_revision_id = rev_id
  -- exist, this field should be ignored (and may be NULL) in favor of the
  -- corresponding data from the slots and content tables
  rev_content_model varbinary(32) DEFAULT NULL,

  -- content format, see CONTENT_FORMAT_XXX constants
  -- @deprecated since 1.31. If rows in the slots table with slot_revision_id = rev_id
  -- exist, this field should be ignored (and may be NULL).
  rev_content_format varbinary(64) DEFAULT NULL

) ENGINE=InnoDB DEFAULT CHARSET=binary MAX_ROWS=10000000 AVG_ROW_LENGTH=1024;
-- In case tables are created as MyISAM, use row hints for MySQL <5.0 to avoid 4GB limit

-- The index is proposed for removal, do not use it in new code: T163532.
-- Used for ordering revisions within a page by rev_id, which is usually
-- incorrect, since rev_timestamp is normally the correct order. It can also
-- be used by dumpBackup.php, if a page and rev_id range is specified.
CREATE INDEX rev_page_id ON revision (rev_page, rev_id);

-- Used by ApiQueryAllRevisions
CREATE INDEX rev_timestamp ON revision (rev_timestamp);

-- History index
CREATE INDEX page_timestamp ON revision (rev_page,rev_timestamp);

-- Logged-in user contributions index
CREATE INDEX user_timestamp ON revision (rev_user,rev_timestamp);

-- Anonymous user countributions index
CREATE INDEX usertext_timestamp ON revision (rev_user_text,rev_timestamp);

-- Credits index. This is scanned in order to compile credits lists for pages,
-- in ApiQueryContributors. Also for ApiQueryRevisions if rvuser is specified
-- and is a logged-in user.
CREATE INDEX page_user_timestamp ON revision (rev_page,rev_user,rev_timestamp);

--
-- Temporary table to avoid blocking on an alter of revision.
--
-- On large wikis like the English Wikipedia, altering the revision table is a
-- months-long process. This table is being created to avoid such an alter, and
-- will be merged back into revision in the future.
--
CREATE TABLE revision_comment_temp (
  -- Key to rev_id
  revcomment_rev int unsigned NOT NULL,
  -- Key to comment_id
  revcomment_comment_id bigint unsigned NOT NULL,
  PRIMARY KEY (revcomment_rev, revcomment_comment_id)
) ENGINE=InnoDB DEFAULT CHARSET=binary;
-- Ensure uniqueness
CREATE UNIQUE INDEX revcomment_rev ON revision_comment_temp (revcomment_rev);

--
-- Temporary table to avoid blocking on an alter of revision.
--
-- On large wikis like the English Wikipedia, altering the revision table is a
-- months-long process. This table is being created to avoid such an alter, and
-- will be merged back into revision in the future.
--
CREATE TABLE revision_actor_temp (
  -- Key to rev_id
  revactor_rev int unsigned NOT NULL,
  -- Key to actor_id
  revactor_actor bigint unsigned NOT NULL,
  -- Copy fields from revision for indexes
  revactor_timestamp binary(14) NOT NULL default '',
  revactor_page int unsigned NOT NULL,
  PRIMARY KEY (revactor_rev, revactor_actor)
) ENGINE=InnoDB DEFAULT CHARSET=binary;
-- Ensure uniqueness
CREATE UNIQUE INDEX revactor_rev ON revision_actor_temp (revactor_rev);
-- Match future indexes on revision
CREATE INDEX actor_timestamp ON revision_actor_temp (revactor_actor,revactor_timestamp);
CREATE INDEX page_actor_timestamp ON revision_actor_temp (revactor_page,revactor_actor,revactor_timestamp);

--
-- Every time an edit by a logged out user is saved,
-- a row is created in ip_changes. This stores
-- the IP as a hex representation so that we can more
-- easily find edits within an IP range.
--
CREATE TABLE ip_changes (
  -- Foreign key to the revision table, also serves as the unique primary key
  ipc_rev_id int unsigned NOT NULL PRIMARY KEY DEFAULT '0',

  -- The timestamp of the revision
  ipc_rev_timestamp binary(14) NOT NULL DEFAULT '',

  -- Hex representation of the IP address, as returned by IP::toHex()
  -- For IPv4 it will resemble: ABCD1234
  -- For IPv6: v6-ABCD1234000000000000000000000000
  -- BETWEEN is then used to identify revisions within a given range
  ipc_hex varbinary(35) NOT NULL DEFAULT ''

) ENGINE=InnoDB DEFAULT CHARSET=binary;

CREATE INDEX ipc_rev_timestamp ON ip_changes (ipc_rev_timestamp);
CREATE INDEX ipc_hex_time ON ip_changes (ipc_hex,ipc_rev_timestamp);

--
-- Holds text of individual page revisions.
--
-- Field names are a holdover from the 'old' revisions table in
-- MediaWiki 1.4 and earlier: an upgrade will transform that
-- table into the 'text' table to minimize unnecessary churning
-- and downtime. If upgrading, the other fields will be left unused.
--
CREATE TABLE text (
  -- Unique text storage key number.
  -- Note that the 'oldid' parameter used in URLs does *not*
  -- refer to this number anymore, but to rev_id.
  --
  -- revision.rev_text_id is a key to this column
  old_id int unsigned NOT NULL PRIMARY KEY AUTO_INCREMENT,

  -- Depending on the contents of the old_flags field, the text
  -- may be convenient plain text, or it may be funkily encoded.
  old_text mediumblob NOT NULL,

  -- Comma-separated list of flags:
  -- gzip: text is compressed with PHP's gzdeflate() function.
  -- utf-8: text was stored as UTF-8.
  --        If $wgLegacyEncoding option is on, rows *without* this flag
  --        will be converted to UTF-8 transparently at load time. Note
  --        that due to a bug in a maintenance script, this flag may
  --        have been stored as 'utf8' in some cases (T18841).
  -- object: text field contained a serialized PHP object.
  --         The object either contains multiple versions compressed
  --         together to achieve a better compression ratio, or it refers
  --         to another row where the text can be found.
  -- external: text was stored in an external location specified by old_text.
  --           Any additional flags apply to the data stored at that URL, not
  --           the URL itself. The 'object' flag is *not* set for URLs of the
  --           form 'DB://cluster/id/itemid', because the external storage
  --           system itself decompresses these.
  old_flags tinyblob NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=binary MAX_ROWS=10000000 AVG_ROW_LENGTH=10240;
-- In case tables are created as MyISAM, use row hints for MySQL <5.0 to avoid 4GB limit


--
-- Edits, blocks, and other actions typically have a textual comment describing
-- the action. They are stored here to reduce the size of the main tables, and
-- to allow for deduplication.
--
-- Deduplication is currently best-effort to avoid locking on inserts that
-- would be required for strict deduplication. There MAY be multiple rows with
-- the same comment_text and comment_data.
--
CREATE TABLE comment (
  -- Unique ID to identify each comment
  comment_id bigint unsigned NOT NULL PRIMARY KEY AUTO_INCREMENT,

  -- Hash of comment_text and comment_data, for deduplication
  comment_hash INT NOT NULL,

  -- Text comment summarizing the change.
  -- This text is shown in the history and other changes lists,
  -- rendered in a subset of wiki markup by Linker::formatComment()
  -- Size limits are enforced at the application level, and should
  -- take care to crop UTF-8 strings appropriately.
  comment_text BLOB NOT NULL,

  -- JSON data, intended for localizing auto-generated comments.
  -- This holds structured data that is intended to be used to provide
  -- localized versions of automatically-generated comments. When not empty,
  -- comment_text should be the generated comment localized using the wiki's
  -- content language.
  comment_data BLOB
) ENGINE=InnoDB DEFAULT CHARSET=binary;
-- Index used for deduplication.
CREATE INDEX comment_hash ON comment (comment_hash);


--
-- Archive area for deleted pages and their revisions.
-- These may be viewed (and restored) by admins through the Special:Undelete interface.
--
CREATE TABLE archive (
  -- Primary key
  ar_id int unsigned NOT NULL PRIMARY KEY AUTO_INCREMENT,

  -- Copied from page_namespace
  ar_namespace int NOT NULL default 0,
  -- Copied from page_title
  ar_title varchar(255) binary NOT NULL default '',

  -- Basic revision stuff...
  ar_comment varbinary(767) NOT NULL default '', -- Deprecated in favor of ar_comment_id
  ar_comment_id bigint unsigned NOT NULL DEFAULT 0, -- ("DEFAULT 0" is temporary, signaling that ar_comment should be used)
  ar_user int unsigned NOT NULL default 0, -- Deprecated in favor of ar_actor
  ar_user_text varchar(255) binary NOT NULL DEFAULT '', -- Deprecated in favor of ar_actor
  ar_actor bigint unsigned NOT NULL DEFAULT 0, -- ("DEFAULT 0" is temporary, signaling that ar_user/ar_user_text should be used)
  ar_timestamp binary(14) NOT NULL default '',
  ar_minor_edit tinyint NOT NULL default 0,

  -- Copied from rev_id.
  --
  -- @since 1.5 Entries from 1.4 will be NULL here. When restoring
  -- archive rows from before 1.5, a new rev_id is created.
  ar_rev_id int unsigned NOT NULL,

  -- Copied from rev_text_id, references text.old_id.
  -- To avoid breaking the block-compression scheme and otherwise making
  -- storage changes harder, the actual text is *not* deleted from the
  -- text storage. Instead, it is merely hidden from public view, by removal
  -- of the page and revision entries.
  --
  -- @deprecated since 1.31. If rows in the slots table with slot_revision_id = ar_rev_id
  -- exist, this field should be ignored (and may be 0) in favor of the
  -- corresponding data from the slots and content tables
  ar_text_id int unsigned NOT NULL DEFAULT 0,

  -- Copied from rev_deleted. Although this may be raised during deletion.
  -- Users with the "suppressrevision" right may "archive" and "suppress"
  -- content in a single action.
  -- @since 1.10
  ar_deleted tinyint unsigned NOT NULL default 0,

  -- Copied from rev_len, length of this revision in bytes.
  -- @since 1.10
  ar_len int unsigned,

  -- Copied from page_id. Restoration will attempt to use this as page ID if
  -- no current page with the same name exists. Otherwise, the revisions will
  -- be restored under the current page. Can be used for manual undeletion by
  -- developers if multiple pages by the same name were archived.
  --
  -- @since 1.11 Older entries will have NULL.
  ar_page_id int unsigned,

  -- Copied from rev_parent_id.
  -- @since 1.13
  ar_parent_id int unsigned default NULL,

  -- Copied from rev_sha1, SHA-1 text content hash in base-36
  -- @since 1.19
  ar_sha1 varbinary(32) NOT NULL default '',

  -- Copied from rev_content_model, see CONTENT_MODEL_XXX constants
  -- @since 1.21
  -- @deprecated since 1.31. If rows in the slots table with slot_revision_id = ar_rev_id
  -- exist, this field should be ignored (and may be NULL) in favor of the
  -- corresponding data from the slots and content tables
  ar_content_model varbinary(32) DEFAULT NULL,

  -- Copied from rev_content_format, see CONTENT_FORMAT_XXX constants
  -- @since 1.21
  -- @deprecated since 1.31. If rows in the slots table with slot_revision_id = ar_rev_id
  -- exist, this field should be ignored (and may be NULL).
  ar_content_format varbinary(64) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=binary;

-- Index for Special:Undelete to page through deleted revisions
CREATE INDEX name_title_timestamp ON archive (ar_namespace,ar_title,ar_timestamp);

-- Index for Special:DeletedContributions
CREATE INDEX ar_usertext_timestamp ON archive (ar_user_text,ar_timestamp);
CREATE INDEX ar_actor_timestamp ON archive (ar_actor,ar_timestamp);

-- Index for linking archive rows with tables that normally link with revision
-- rows, such as change_tag.
CREATE INDEX ar_revid ON archive (ar_rev_id);

--
-- Slots represent an n:m relation between revisions and content objects.
-- A content object can have a specific "role" in one or more revisions.
-- Each revision can have multiple content objects, each having a different role.
--
CREATE TABLE slots (

  -- reference to rev_id or ar_rev_id
  slot_revision_id bigint unsigned NOT NULL,

  -- reference to role_id
  slot_role_id smallint unsigned NOT NULL,

  -- reference to content_id
  slot_content_id bigint unsigned NOT NULL,

  -- The revision ID of the revision that originated the slot's content.
  -- To find revisions that changed slots, look for slot_origin = slot_revision_id.
  slot_origin bigint unsigned NOT NULL,

  PRIMARY KEY ( slot_revision_id, slot_role_id )
) ENGINE=InnoDB DEFAULT CHARSET=binary;

-- Index for finding revisions that modified a specific slot
CREATE INDEX slot_revision_origin_role ON slots (slot_revision_id, slot_origin, slot_role_id);

--
-- The content table represents content objects. It's primary purpose is to provide the necessary
-- meta-data for loading and interpreting a serialized data blob to create a content object.
--
CREATE TABLE content (

  -- ID of the content object
  content_id bigint unsigned PRIMARY KEY AUTO_INCREMENT,

  -- Nominal size of the content object (not necessarily of the serialized blob)
  content_size int unsigned NOT NULL,

  -- Nominal hash of the content object (not necessarily of the serialized blob)
  content_sha1 varbinary(32) NOT NULL,

  -- reference to model_id. Note the content format isn't specified; it should
  -- be assumed to be in the default format for the model unless auto-detected
  -- otherwise.
  content_model smallint unsigned NOT NULL,

  -- URL-like address of the content blob
  content_address varbinary(255) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=binary;

--
-- Normalization table for role names
--
CREATE TABLE slot_roles (
  role_id smallint PRIMARY KEY AUTO_INCREMENT,
  role_name varbinary(64) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=binary;

-- Index for looking of the internal ID of for a name
CREATE UNIQUE INDEX role_name ON slot_roles (role_name);

--
-- Normalization table for content model names
--
CREATE TABLE content_models (
  model_id smallint PRIMARY KEY AUTO_INCREMENT,
  model_name varbinary(64) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=binary;

-- Index for looking of the internal ID of for a name
CREATE UNIQUE INDEX model_name ON content_models (model_name);

--
-- Track page-to-page hyperlinks within the wiki.
--
CREATE TABLE pagelinks (
  -- Key to the page_id of the page containing the link.
  pl_from int unsigned NOT NULL default 0,
  -- Namespace for this page
  pl_from_namespace int NOT NULL default 0,

  -- Key to page_namespace/page_title of the target page.
  -- The target page may or may not exist, and due to renames
  -- and deletions may refer to different page records as time
  -- goes by.
  pl_namespace int NOT NULL default 0,
  pl_title varchar(255) binary NOT NULL default '',
  PRIMARY KEY (pl_from,pl_namespace,pl_title)
) ENGINE=InnoDB DEFAULT CHARSET=binary;

-- Reverse index, for Special:Whatlinkshere
CREATE INDEX pl_namespace ON pagelinks (pl_namespace,pl_title,pl_from);

-- Index for Special:Whatlinkshere with namespace filter
CREATE INDEX pl_backlinks_namespace ON pagelinks (pl_from_namespace,pl_namespace,pl_title,pl_from);


--
-- Track template inclusions.
--
CREATE TABLE templatelinks (
  -- Key to the page_id of the page containing the link.
  tl_from int unsigned NOT NULL default 0,
  -- Namespace for this page
  tl_from_namespace int NOT NULL default 0,

  -- Key to page_namespace/page_title of the target page.
  -- The target page may or may not exist, and due to renames
  -- and deletions may refer to different page records as time
  -- goes by.
  tl_namespace int NOT NULL default 0,
  tl_title varchar(255) binary NOT NULL default '',
  PRIMARY KEY (tl_from,tl_namespace,tl_title)
) ENGINE=InnoDB DEFAULT CHARSET=binary;

-- Reverse index, for Special:Whatlinkshere
CREATE INDEX tl_namespace ON templatelinks (tl_namespace,tl_title,tl_from);

-- Index for Special:Whatlinkshere with namespace filter
CREATE INDEX tl_backlinks_namespace ON templatelinks (tl_from_namespace,tl_namespace,tl_title,tl_from);


--
-- Track links to images *used inline*
-- We don't distinguish live from broken links here, so
-- they do not need to be changed on upload/removal.
--
CREATE TABLE imagelinks (
  -- Key to page_id of the page containing the image / media link.
  il_from int unsigned NOT NULL default 0,
  -- Namespace for this page
  il_from_namespace int NOT NULL default 0,

  -- Filename of target image.
  -- This is also the page_title of the file's description page;
  -- all such pages are in namespace 6 (NS_FILE).
  il_to varchar(255) binary NOT NULL default '',
  PRIMARY KEY (il_from,il_to)
) ENGINE=InnoDB DEFAULT CHARSET=binary;

-- Reverse index, for Special:Whatlinkshere and file description page local usage
CREATE INDEX il_to ON imagelinks (il_to,il_from);

-- Index for Special:Whatlinkshere with namespace filter
CREATE INDEX il_backlinks_namespace ON imagelinks (il_from_namespace,il_to,il_from);


--
-- Track category inclusions *used inline*
-- This tracks a single level of category membership
--
CREATE TABLE categorylinks (
  -- Key to page_id of the page defined as a category member.
  cl_from int unsigned NOT NULL default 0,

  -- Name of the category.
  -- This is also the page_title of the category's description page;
  -- all such pages are in namespace 14 (NS_CATEGORY).
  cl_to varchar(255) binary NOT NULL default '',

  -- A binary string obtained by applying a sortkey generation algorithm
  -- (Collation::getSortKey()) to page_title, or cl_sortkey_prefix . "\n"
  -- . page_title if cl_sortkey_prefix is nonempty.
  cl_sortkey varbinary(230) NOT NULL default '',

  -- A prefix for the raw sortkey manually specified by the user, either via
  -- [[Category:Foo|prefix]] or {{defaultsort:prefix}}.  If nonempty, it's
  -- concatenated with a line break followed by the page title before the sortkey
  -- conversion algorithm is run.  We store this so that we can update
  -- collations without reparsing all pages.
  -- Note: If you change the length of this field, you also need to change
  -- code in LinksUpdate.php. See T27254.
  cl_sortkey_prefix varchar(255) binary NOT NULL default '',

  -- This isn't really used at present. Provided for an optional
  -- sorting method by approximate addition time.
  cl_timestamp timestamp NOT NULL,

  -- Stores $wgCategoryCollation at the time cl_sortkey was generated.  This
  -- can be used to install new collation versions, tracking which rows are not
  -- yet updated.  '' means no collation, this is a legacy row that needs to be
  -- updated by updateCollation.php.  In the future, it might be possible to
  -- specify different collations per category.
  cl_collation varbinary(32) NOT NULL default '',

  -- Stores whether cl_from is a category, file, or other page, so we can
  -- paginate the three categories separately.  This never has to be updated
  -- after the page is created, since none of these page types can be moved to
  -- any other.
  cl_type ENUM('page', 'subcat', 'file') NOT NULL default 'page',
  PRIMARY KEY (cl_from,cl_to)
) ENGINE=InnoDB DEFAULT CHARSET=binary;


-- We always sort within a given category, and within a given type.  FIXME:
-- Formerly this index didn't cover cl_type (since that didn't exist), so old
-- callers won't be using an index: fix this?
CREATE INDEX cl_sortkey ON categorylinks (cl_to,cl_type,cl_sortkey,cl_from);

-- Used by the API (and some extensions)
CREATE INDEX cl_timestamp ON categorylinks (cl_to,cl_timestamp);

-- Used when updating collation (e.g. updateCollation.php)
CREATE INDEX cl_collation_ext ON categorylinks (cl_collation, cl_to, cl_type, cl_from);

--
-- Track all existing categories. Something is a category if 1) it has an entry
-- somewhere in categorylinks, or 2) it has a description page. Categories
-- might not have corresponding pages, so they need to be tracked separately.
--
CREATE TABLE category (
  -- Primary key
  cat_id int unsigned NOT NULL PRIMARY KEY AUTO_INCREMENT,

  -- Name of the category, in the same form as page_title (with underscores).
  -- If there is a category page corresponding to this category, by definition,
  -- it has this name (in the Category namespace).
  cat_title varchar(255) binary NOT NULL,

  -- The numbers of member pages (including categories and media), subcatego-
  -- ries, and Image: namespace members, respectively.  These are signed to
  -- make underflow more obvious.  We make the first number include the second
  -- two for better sorting: subtracting for display is easy, adding for order-
  -- ing is not.
  cat_pages int signed NOT NULL default 0,
  cat_subcats int signed NOT NULL default 0,
  cat_files int signed NOT NULL default 0
) ENGINE=InnoDB DEFAULT CHARSET=binary;

CREATE UNIQUE INDEX cat_title ON category (cat_title);

-- For Special:Mostlinkedcategories
CREATE INDEX cat_pages ON category (cat_pages);


--
-- Track links to external URLs
--
CREATE TABLE externallinks (
  -- Primary key
  el_id int unsigned NOT NULL PRIMARY KEY AUTO_INCREMENT,

  -- page_id of the referring page
  el_from int unsigned NOT NULL default 0,

  -- The URL
  el_to blob NOT NULL,

  -- In the case of HTTP URLs, this is the URL with any username or password
  -- removed, and with the labels in the hostname reversed and converted to
  -- lower case. An extra dot is added to allow for matching of either
  -- example.com or *.example.com in a single scan.
  -- Example:
  --      http://user:password@sub.example.com/page.html
  --   becomes
  --      http://com.example.sub./page.html
  -- which allows for fast searching for all pages under example.com with the
  -- clause:
  --      WHERE el_index LIKE 'http://com.example.%'
  el_index blob NOT NULL,

  -- This is el_index truncated to 60 bytes to allow for sortable queries that
  -- aren't supported by a partial index.
  -- @todo Drop the default once this is deployed everywhere and code is populating it.
  el_index_60 varbinary(60) NOT NULL default ''
) ENGINE=InnoDB DEFAULT CHARSET=binary;

-- Forward index, for page edit, save
CREATE INDEX el_from ON externallinks (el_from, el_to(40));

-- Index for Special:LinkSearch exact search
CREATE INDEX el_to ON externallinks (el_to(60), el_from);

-- For Special:LinkSearch wildcard search
CREATE INDEX el_index ON externallinks (el_index(60));

-- For Special:LinkSearch wildcard search with efficient paging by el_id
CREATE INDEX el_index_60 ON externallinks (el_index_60, el_id);
CREATE INDEX el_from_index_60 ON externallinks (el_from, el_index_60, el_id);

--
-- Track interlanguage links
--
CREATE TABLE langlinks (
  -- page_id of the referring page
  ll_from int unsigned NOT NULL default 0,

  -- Language code of the target
  ll_lang varbinary(20) NOT NULL default '',

  -- Title of the target, including namespace
  ll_title varchar(255) binary NOT NULL default '',
  PRIMARY KEY (ll_from,ll_lang)
) ENGINE=InnoDB DEFAULT CHARSET=binary;

-- Index for ApiQueryLangbacklinks
CREATE INDEX ll_lang ON langlinks (ll_lang, ll_title);


--
-- Track inline interwiki links
--
CREATE TABLE iwlinks (
  -- page_id of the referring page
  iwl_from int unsigned NOT NULL default 0,

  -- Interwiki prefix code of the target
  iwl_prefix varbinary(20) NOT NULL default '',

  -- Title of the target, including namespace
  iwl_title varchar(255) binary NOT NULL default '',
  PRIMARY KEY (iwl_from,iwl_prefix,iwl_title)
) ENGINE=InnoDB DEFAULT CHARSET=binary;

-- Index for ApiQueryIWBacklinks
CREATE INDEX iwl_prefix_title_from ON iwlinks (iwl_prefix, iwl_title, iwl_from);

-- Index for ApiQueryIWLinks
CREATE INDEX iwl_prefix_from_title ON iwlinks (iwl_prefix, iwl_from, iwl_title);


--
-- Contains a single row with some aggregate info
-- on the state of the site.
--
CREATE TABLE site_stats (
  -- The single row should contain 1 here.
  ss_row_id int unsigned NOT NULL PRIMARY KEY,

  -- Total number of edits performed.
  ss_total_edits bigint unsigned default NULL,

  -- See SiteStatsInit::articles().
  ss_good_articles bigint unsigned default NULL,

  -- Total pages, theoretically equal to SELECT COUNT(*) FROM page.
  ss_total_pages bigint unsigned default NULL,

  -- Number of users, theoretically equal to SELECT COUNT(*) FROM user.
  ss_users bigint unsigned default NULL,

  -- Number of users that still edit.
  ss_active_users bigint unsigned default NULL,

  -- Number of images, equivalent to SELECT COUNT(*) FROM image.
  ss_images bigint unsigned default NULL
) ENGINE=InnoDB DEFAULT CHARSET=binary;

--
-- The internet is full of jerks, alas. Sometimes it's handy
-- to block a vandal or troll account.
--
CREATE TABLE ipblocks (
  -- Primary key, introduced for privacy.
  ipb_id int NOT NULL PRIMARY KEY AUTO_INCREMENT,

  -- Blocked IP address in dotted-quad form or user name.
  ipb_address tinyblob NOT NULL,

  -- Blocked user ID or 0 for IP blocks.
  ipb_user int unsigned NOT NULL default 0,

  -- User ID who made the block.
  ipb_by int unsigned NOT NULL default 0, -- Deprecated in favor of ipb_by_actor

  -- User name of blocker
  ipb_by_text varchar(255) binary NOT NULL default '', -- Deprecated in favor of ipb_by_actor

  -- Actor who made the block.
  ipb_by_actor bigint unsigned NOT NULL DEFAULT 0, -- ("DEFAULT 0" is temporary, signaling that ipb_by/ipb_by_text should be used)

  -- Text comment made by blocker. Deprecated in favor of ipb_reason_id
  ipb_reason varbinary(767) NOT NULL default '',

  -- Key to comment_id. Text comment made by blocker.
  -- ("DEFAULT 0" is temporary, signaling that ipb_reason should be used)
  ipb_reason_id bigint unsigned NOT NULL DEFAULT 0,

  -- Creation (or refresh) date in standard YMDHMS form.
  -- IP blocks expire automatically.
  ipb_timestamp binary(14) NOT NULL default '',

  -- Indicates that the IP address was banned because a banned
  -- user accessed a page through it. If this is 1, ipb_address
  -- will be hidden, and the block identified by block ID number.
  ipb_auto bool NOT NULL default 0,

  -- If set to 1, block applies only to logged-out users
  ipb_anon_only bool NOT NULL default 0,

  -- Block prevents account creation from matching IP addresses
  ipb_create_account bool NOT NULL default 1,

  -- Block triggers autoblocks
  ipb_enable_autoblock bool NOT NULL default '1',

  -- Time at which the block will expire.
  -- May be "infinity"
  ipb_expiry varbinary(14) NOT NULL default '',

  -- Start and end of an address range, in hexadecimal
  -- Size chosen to allow IPv6
  -- FIXME: these fields were originally blank for single-IP blocks,
  -- but now they are populated. No migration was ever done. They
  -- should be fixed to be blank again for such blocks (T51504).
  ipb_range_start tinyblob NOT NULL,
  ipb_range_end tinyblob NOT NULL,

  -- Flag for entries hidden from users and Sysops
  ipb_deleted bool NOT NULL default 0,

  -- Block prevents user from accessing Special:Emailuser
  ipb_block_email bool NOT NULL default 0,

  -- Block allows user to edit their own talk page
  ipb_allow_usertalk bool NOT NULL default 0,

  -- ID of the block that caused this block to exist
  -- Autoblocks set this to the original block
  -- so that the original block being deleted also
  -- deletes the autoblocks
  ipb_parent_block_id int default NULL

) ENGINE=InnoDB DEFAULT CHARSET=binary;

-- Unique index to support "user already blocked" messages
-- Any new options which prevent collisions should be included
CREATE UNIQUE INDEX ipb_address ON ipblocks (ipb_address(255), ipb_user, ipb_auto, ipb_anon_only);

-- For querying whether a logged-in user is blocked
CREATE INDEX ipb_user ON ipblocks (ipb_user);

-- For querying whether an IP address is in any range
CREATE INDEX ipb_range ON ipblocks (ipb_range_start(8), ipb_range_end(8));

-- Index for Special:BlockList
CREATE INDEX ipb_timestamp ON ipblocks (ipb_timestamp);

-- Index for table pruning
CREATE INDEX ipb_expiry ON ipblocks (ipb_expiry);

-- Index for removing autoblocks when a parent block is removed
CREATE INDEX ipb_parent_block_id ON ipblocks (ipb_parent_block_id);


--
-- Uploaded images and other files.
--
CREATE TABLE image (
  -- Filename.
  -- This is also the title of the associated description page,
  -- which will be in namespace 6 (NS_FILE).
  img_name varchar(255) binary NOT NULL default '' PRIMARY KEY,

  -- File size in bytes.
  img_size int unsigned NOT NULL default 0,

  -- For images, size in pixels.
  img_width int NOT NULL default 0,
  img_height int NOT NULL default 0,

  -- Extracted Exif metadata stored as a serialized PHP array.
  img_metadata mediumblob NOT NULL,

  -- For images, bits per pixel if known.
  img_bits int NOT NULL default 0,

  -- Media type as defined by the MEDIATYPE_xxx constants
  img_media_type ENUM("UNKNOWN", "BITMAP", "DRAWING", "AUDIO", "VIDEO", "MULTIMEDIA", "OFFICE", "TEXT", "EXECUTABLE", "ARCHIVE", "3D") default NULL,

  -- major part of a MIME media type as defined by IANA
  -- see https://www.iana.org/assignments/media-types/
  -- for "chemical" cf. http://dx.doi.org/10.1021/ci9803233 by the ACS
  img_major_mime ENUM("unknown", "application", "audio", "image", "text", "video", "message", "model", "multipart", "chemical") NOT NULL default "unknown",

  -- minor part of a MIME media type as defined by IANA
  -- the minor parts are not required to adher to any standard
  -- but should be consistent throughout the database
  -- see https://www.iana.org/assignments/media-types/
  img_minor_mime varbinary(100) NOT NULL default "unknown",

  -- Description field as entered by the uploader.
  -- This is displayed in image upload history and logs.
  -- Deprecated in favor of img_description_id.
  img_description varbinary(767) NOT NULL default '',

  img_description_id bigint unsigned NOT NULL DEFAULT 0, -- ("DEFAULT 0" is temporary, signaling that img_description should be used)

  -- user_id and user_name of uploader.
  -- Deprecated in favor of img_actor.
  img_user int unsigned NOT NULL default 0,
  img_user_text varchar(255) binary NOT NULL DEFAULT '',

  -- actor_id of the uploader.
  -- ("DEFAULT 0" is temporary, signaling that img_user/img_user_text should be used)
  img_actor bigint unsigned NOT NULL DEFAULT 0,

  -- Time of the upload.
  img_timestamp varbinary(14) NOT NULL default '',

  -- SHA-1 content hash in base-36
  img_sha1 varbinary(32) NOT NULL default ''
) ENGINE=InnoDB DEFAULT CHARSET=binary;

-- Used by Special:Newimages and ApiQueryAllImages
CREATE INDEX img_user_timestamp ON image (img_user,img_timestamp);
CREATE INDEX img_usertext_timestamp ON image (img_user_text,img_timestamp);
CREATE INDEX img_actor_timestamp ON image (img_actor,img_timestamp);
-- Used by Special:ListFiles for sort-by-size
CREATE INDEX img_size ON image (img_size);
-- Used by Special:Newimages and Special:ListFiles
CREATE INDEX img_timestamp ON image (img_timestamp);
-- Used in API and duplicate search
CREATE INDEX img_sha1 ON image (img_sha1(10));
-- Used to get media of one type
CREATE INDEX img_media_mime ON image (img_media_type,img_major_mime,img_minor_mime);

--
-- Temporary table to avoid blocking on an alter of image.
--
-- On large wikis like Wikimedia Commons, altering the image table is a
-- months-long process. This table is being created to avoid such an alter, and
-- will be merged back into image in the future.
--
CREATE TABLE image_comment_temp (
  -- Key to img_name (ugh)
  imgcomment_name varchar(255) binary NOT NULL,
  -- Key to comment_id
  imgcomment_description_id bigint unsigned NOT NULL,
  PRIMARY KEY (imgcomment_name, imgcomment_description_id)
) ENGINE=InnoDB DEFAULT CHARSET=binary;
-- Ensure uniqueness
CREATE UNIQUE INDEX imgcomment_name ON image_comment_temp (imgcomment_name);


--
-- Previous revisions of uploaded files.
-- Awkwardly, image rows have to be moved into
-- this table at re-upload time.
--
CREATE TABLE oldimage (
  -- Base filename: key to image.img_name
  oi_name varchar(255) binary NOT NULL default '',

  -- Filename of the archived file.
  -- This is generally a timestamp and '!' prepended to the base name.
  oi_archive_name varchar(255) binary NOT NULL default '',

  -- Other fields as in image...
  oi_size int unsigned NOT NULL default 0,
  oi_width int NOT NULL default 0,
  oi_height int NOT NULL default 0,
  oi_bits int NOT NULL default 0,
  oi_description varbinary(767) NOT NULL default '', -- Deprecated.
  oi_description_id bigint unsigned NOT NULL DEFAULT 0, -- ("DEFAULT 0" is temporary, signaling that oi_description should be used)
  oi_user int unsigned NOT NULL default 0, -- Deprecated in favor of oi_actor
  oi_user_text varchar(255) binary NOT NULL DEFAULT '', -- Deprecated in favor of oi_actor
  oi_actor bigint unsigned NOT NULL DEFAULT 0, -- ("DEFAULT 0" is temporary, signaling that oi_user/oi_user_text should be used)
  oi_timestamp binary(14) NOT NULL default '',

  oi_metadata mediumblob NOT NULL,
  oi_media_type ENUM("UNKNOWN", "BITMAP", "DRAWING", "AUDIO", "VIDEO", "MULTIMEDIA", "OFFICE", "TEXT", "EXECUTABLE", "ARCHIVE", "3D") default NULL,
  oi_major_mime ENUM("unknown", "application", "audio", "image", "text", "video", "message", "model", "multipart", "chemical") NOT NULL default "unknown",
  oi_minor_mime varbinary(100) NOT NULL default "unknown",
  oi_deleted tinyint unsigned NOT NULL default 0,
  oi_sha1 varbinary(32) NOT NULL default ''
) ENGINE=InnoDB DEFAULT CHARSET=binary;

CREATE INDEX oi_usertext_timestamp ON oldimage (oi_user_text,oi_timestamp);
CREATE INDEX oi_actor_timestamp ON oldimage (oi_actor,oi_timestamp);
CREATE INDEX oi_name_timestamp ON oldimage (oi_name,oi_timestamp);
-- oi_archive_name truncated to 14 to avoid key length overflow
CREATE INDEX oi_name_archive_name ON oldimage (oi_name,oi_archive_name(14));
CREATE INDEX oi_sha1 ON oldimage (oi_sha1(10));


--
-- Record of deleted file data
--
CREATE TABLE filearchive (
  -- Unique row id
  fa_id int NOT NULL PRIMARY KEY AUTO_INCREMENT,

  -- Original base filename; key to image.img_name, page.page_title, etc
  fa_name varchar(255) binary NOT NULL default '',

  -- Filename of archived file, if an old revision
  fa_archive_name varchar(255) binary default '',

  -- Which storage bin (directory tree or object store) the file data
  -- is stored in. Should be 'deleted' for files that have been deleted;
  -- any other bin is not yet in use.
  fa_storage_group varbinary(16),

  -- SHA-1 of the file contents plus extension, used as a key for storage.
  -- eg 8f8a562add37052a1848ff7771a2c515db94baa9.jpg
  --
  -- If NULL, the file was missing at deletion time or has been purged
  -- from the archival storage.
  fa_storage_key varbinary(64) default '',

  -- Deletion information, if this file is deleted.
  fa_deleted_user int,
  fa_deleted_timestamp binary(14) default '',
  fa_deleted_reason varbinary(767) default '', -- Deprecated
  fa_deleted_reason_id bigint unsigned NOT NULL DEFAULT 0, -- ("DEFAULT 0" is temporary, signaling that fa_deleted_reason should be used)

  -- Duped fields from image
  fa_size int unsigned default 0,
  fa_width int default 0,
  fa_height int default 0,
  fa_metadata mediumblob,
  fa_bits int default 0,
  fa_media_type ENUM("UNKNOWN", "BITMAP", "DRAWING", "AUDIO", "VIDEO", "MULTIMEDIA", "OFFICE", "TEXT", "EXECUTABLE", "ARCHIVE", "3D") default NULL,
  fa_major_mime ENUM("unknown", "application", "audio", "image", "text", "video", "message", "model", "multipart", "chemical") default "unknown",
  fa_minor_mime varbinary(100) default "unknown",
  fa_description varbinary(767) default '', -- Deprecated
  fa_description_id bigint unsigned NOT NULL DEFAULT 0, -- ("DEFAULT 0" is temporary, signaling that fa_description should be used)
  fa_user int unsigned default 0, -- Deprecated in favor of fa_actor
  fa_user_text varchar(255) binary DEFAULT '', -- Deprecated in favor of fa_actor
  fa_actor bigint unsigned NOT NULL DEFAULT 0, -- ("DEFAULT 0" is temporary, signaling that fa_user/fa_user_text should be used)
  fa_timestamp binary(14) default '',

  -- Visibility of deleted revisions, bitfield
  fa_deleted tinyint unsigned NOT NULL default 0,

  -- sha1 hash of file content
  fa_sha1 varbinary(32) NOT NULL default ''
) ENGINE=InnoDB DEFAULT CHARSET=binary;

-- pick out by image name
CREATE INDEX fa_name ON filearchive (fa_name, fa_timestamp);
-- pick out dupe files
CREATE INDEX fa_storage_group ON filearchive (fa_storage_group, fa_storage_key);
-- sort by deletion time
CREATE INDEX fa_deleted_timestamp ON filearchive (fa_deleted_timestamp);
-- sort by uploader
CREATE INDEX fa_user_timestamp ON filearchive (fa_user_text,fa_timestamp);
CREATE INDEX fa_actor_timestamp ON filearchive (fa_actor,fa_timestamp);
-- find file by sha1, 10 bytes will be enough for hashes to be indexed
CREATE INDEX fa_sha1 ON filearchive (fa_sha1(10));


--
-- Store information about newly uploaded files before they're
-- moved into the actual filestore
--
CREATE TABLE uploadstash (
  us_id int unsigned NOT NULL PRIMARY KEY AUTO_INCREMENT,

  -- the user who uploaded the file.
  us_user int unsigned NOT NULL,

  -- file key. this is how applications actually search for the file.
  -- this might go away, or become the primary key.
  us_key varchar(255) NOT NULL,

  -- the original path
  us_orig_path varchar(255) NOT NULL,

  -- the temporary path at which the file is actually stored
  us_path varchar(255) NOT NULL,

  -- which type of upload the file came from (sometimes)
  us_source_type varchar(50),

  -- the date/time on which the file was added
  us_timestamp varbinary(14) NOT NULL,

  us_status varchar(50) NOT NULL,

  -- chunk counter starts at 0, current offset is stored in us_size
  us_chunk_inx int unsigned NULL,

  -- Serialized file properties from FSFile::getProps()
  us_props blob,

  -- file size in bytes
  us_size int unsigned NOT NULL,
  -- this hash comes from FSFile::getSha1Base36(), and is 31 characters
  us_sha1 varchar(31) NOT NULL,
  us_mime varchar(255),
  -- Media type as defined by the MEDIATYPE_xxx constants, should duplicate definition in the image table
  us_media_type ENUM("UNKNOWN", "BITMAP", "DRAWING", "AUDIO", "VIDEO", "MULTIMEDIA", "OFFICE", "TEXT", "EXECUTABLE", "ARCHIVE", "3D") default NULL,
  -- image-specific properties
  us_image_width int unsigned,
  us_image_height int unsigned,
  us_image_bits smallint unsigned

) ENGINE=InnoDB DEFAULT CHARSET=binary;

-- sometimes there's a delete for all of a user's stuff.
CREATE INDEX us_user ON uploadstash (us_user);
-- pick out files by key, enforce key uniqueness
CREATE UNIQUE INDEX us_key ON uploadstash (us_key);
-- the abandoned upload cleanup script needs this
CREATE INDEX us_timestamp ON uploadstash (us_timestamp);


--
-- Primarily a summary table for Special:Recentchanges,
-- this table contains some additional info on edits from
-- the last few days, see Article::editUpdates()
--
CREATE TABLE recentchanges (
  rc_id int NOT NULL PRIMARY KEY AUTO_INCREMENT,
  rc_timestamp varbinary(14) NOT NULL default '',

  -- As in revision
  rc_user int unsigned NOT NULL default 0, -- Deprecated in favor of rc_actor
  rc_user_text varchar(255) binary NOT NULL DEFAULT '', -- Deprecated in favor of rc_actor
  rc_actor bigint unsigned NOT NULL DEFAULT 0, -- ("DEFAULT 0" is temporary, signaling that rc_user/rc_user_text should be used)

  -- When pages are renamed, their RC entries do _not_ change.
  rc_namespace int NOT NULL default 0,
  rc_title varchar(255) binary NOT NULL default '',

  -- as in revision...
  rc_comment varbinary(767) NOT NULL default '', -- Deprecated.
  rc_comment_id bigint unsigned NOT NULL DEFAULT 0, -- ("DEFAULT 0" is temporary, signaling that rc_comment should be used)
  rc_minor tinyint unsigned NOT NULL default 0,

  -- Edits by user accounts with the 'bot' rights key are
  -- marked with a 1 here, and will be hidden from the
  -- default view.
  rc_bot tinyint unsigned NOT NULL default 0,

  -- Set if this change corresponds to a page creation
  rc_new tinyint unsigned NOT NULL default 0,

  -- Key to page_id (was cur_id prior to 1.5).
  -- This will keep links working after moves while
  -- retaining the at-the-time name in the changes list.
  rc_cur_id int unsigned NOT NULL default 0,

  -- rev_id of the given revision
  rc_this_oldid int unsigned NOT NULL default 0,

  -- rev_id of the prior revision, for generating diff links.
  rc_last_oldid int unsigned NOT NULL default 0,

  -- The type of change entry (RC_EDIT,RC_NEW,RC_LOG,RC_EXTERNAL)
  rc_type tinyint unsigned NOT NULL default 0,

  -- The source of the change entry (replaces rc_type)
  -- default of '' is temporary, needed for initial migration
  rc_source varchar(16) binary not null default '',

  -- If the Recent Changes Patrol option is enabled,
  -- users may mark edits as having been reviewed to
  -- remove a warning flag on the RC list.
  -- A value of 1 indicates the page has been reviewed.
  rc_patrolled tinyint unsigned NOT NULL default 0,

  -- Recorded IP address the edit was made from, if the
  -- $wgPutIPinRC option is enabled.
  rc_ip varbinary(40) NOT NULL default '',

  -- Text length in characters before
  -- and after the edit
  rc_old_len int,
  rc_new_len int,

  -- Visibility of recent changes items, bitfield
  rc_deleted tinyint unsigned NOT NULL default 0,

  -- Value corresponding to log_id, specific log entries
  rc_logid int unsigned NOT NULL default 0,
  -- Store log type info here, or null
  rc_log_type varbinary(255) NULL default NULL,
  -- Store log action or null
  rc_log_action varbinary(255) NULL default NULL,
  -- Log params
  rc_params blob NULL
) ENGINE=InnoDB DEFAULT CHARSET=binary;

-- Special:Recentchanges
CREATE INDEX rc_timestamp ON recentchanges (rc_timestamp);

-- Special:Watchlist
CREATE INDEX rc_namespace_title_timestamp ON recentchanges (rc_namespace, rc_title, rc_timestamp);

-- Special:Recentchangeslinked when finding changes in pages linked from a page
CREATE INDEX rc_cur_id ON recentchanges (rc_cur_id);

-- Special:Newpages
CREATE INDEX new_name_timestamp ON recentchanges (rc_new,rc_namespace,rc_timestamp);

-- Blank unless $wgPutIPinRC=true (false at WMF), possibly used by extensions,
-- but mostly replaced by CheckUser.
CREATE INDEX rc_ip ON recentchanges (rc_ip);

-- Probably intended for Special:NewPages namespace filter
CREATE INDEX rc_ns_usertext ON recentchanges (rc_namespace, rc_user_text);
CREATE INDEX rc_ns_actor ON recentchanges (rc_namespace, rc_actor);

-- SiteStats active user count, Special:ActiveUsers, Special:NewPages user filter
CREATE INDEX rc_user_text ON recentchanges (rc_user_text, rc_timestamp);
CREATE INDEX rc_actor ON recentchanges (rc_actor, rc_timestamp);

-- ApiQueryRecentChanges (T140108)
CREATE INDEX rc_name_type_patrolled_timestamp ON recentchanges (rc_namespace, rc_type, rc_patrolled, rc_timestamp);


CREATE TABLE watchlist (
  wl_id int unsigned NOT NULL PRIMARY KEY AUTO_INCREMENT,
  -- Key to user.user_id
  wl_user int unsigned NOT NULL,

  -- Key to page_namespace/page_title
  -- Note that users may watch pages which do not exist yet,
  -- or existed in the past but have been deleted.
  wl_namespace int NOT NULL default 0,
  wl_title varchar(255) binary NOT NULL default '',

  -- Timestamp used to send notification e-mails and show "updated since last visit" markers on
  -- history and recent changes / watchlist. Set to NULL when the user visits the latest revision
  -- of the page, which means that they should be sent an e-mail on the next change.
  wl_notificationtimestamp varbinary(14)

) ENGINE=InnoDB DEFAULT CHARSET=binary;

-- Special:Watchlist
CREATE UNIQUE INDEX wl_user ON watchlist (wl_user, wl_namespace, wl_title);

-- Special:Movepage (WatchedItemStore::duplicateEntry)
CREATE INDEX namespace_title ON watchlist (wl_namespace, wl_title);

-- ApiQueryWatchlistRaw changed filter
CREATE INDEX wl_user_notificationtimestamp ON watchlist (wl_user, wl_notificationtimestamp);


--
-- When using the default MySQL search backend, page titles
-- and text are munged to strip markup, do Unicode case folding,
-- and prepare the result for MySQL's fulltext index.
--
-- This table must be MyISAM; InnoDB does not support the needed
-- fulltext index.
--
CREATE TABLE searchindex (
  -- Key to page_id
  si_page int unsigned NOT NULL,

  -- Munged version of title
  si_title varchar(255) NOT NULL default '',

  -- Munged version of body text
  si_text mediumtext NOT NULL
) ENGINE=MyISAM DEFAULT CHARSET=utf8;

CREATE UNIQUE INDEX si_page ON searchindex (si_page);
CREATE FULLTEXT INDEX si_title ON searchindex (si_title);
CREATE FULLTEXT INDEX si_text ON searchindex (si_text);


--
-- Recognized interwiki link prefixes
--
CREATE TABLE interwiki (
  -- The interwiki prefix, (e.g. "Meatball", or the language prefix "de")
  iw_prefix varchar(32) NOT NULL,

  -- The URL of the wiki, with "$1" as a placeholder for an article name.
  -- Any spaces in the name will be transformed to underscores before
  -- insertion.
  iw_url blob NOT NULL,

  -- The URL of the file api.php
  iw_api blob NOT NULL,

  -- The name of the database (for a connection to be established with wfGetLB( 'wikiid' ))
  iw_wikiid varchar(64) NOT NULL,

  -- A boolean value indicating whether the wiki is in this project
  -- (used, for example, to detect redirect loops)
  iw_local bool NOT NULL,

  -- Boolean value indicating whether interwiki transclusions are allowed.
  iw_trans tinyint NOT NULL default 0
) ENGINE=InnoDB DEFAULT CHARSET=binary;

CREATE UNIQUE INDEX iw_prefix ON interwiki (iw_prefix);


--
-- Used for caching expensive grouped queries
--
CREATE TABLE querycache (
  -- A key name, generally the base name of of the special page.
  qc_type varbinary(32) NOT NULL,

  -- Some sort of stored value. Sizes, counts...
  qc_value int unsigned NOT NULL default 0,

  -- Target namespace+title
  qc_namespace int NOT NULL default 0,
  qc_title varchar(255) binary NOT NULL default ''
) ENGINE=InnoDB DEFAULT CHARSET=binary;

CREATE INDEX qc_type ON querycache (qc_type,qc_value);


--
-- For a few generic cache operations if not using Memcached
--
CREATE TABLE objectcache (
  keyname varbinary(255) NOT NULL default '' PRIMARY KEY,
  value mediumblob,
  exptime datetime
) ENGINE=InnoDB DEFAULT CHARSET=binary;
CREATE INDEX exptime ON objectcache (exptime);


--
-- Cache of interwiki transclusion
--
CREATE TABLE transcache (
  tc_url varbinary(255) NOT NULL PRIMARY KEY,
  tc_contents text,
  tc_time binary(14) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=binary;


CREATE TABLE logging (
  -- Log ID, for referring to this specific log entry, probably for deletion and such.
  log_id int unsigned NOT NULL PRIMARY KEY AUTO_INCREMENT,

  -- Symbolic keys for the general log type and the action type
  -- within the log. The output format will be controlled by the
  -- action field, but only the type controls categorization.
  log_type varbinary(32) NOT NULL default '',
  log_action varbinary(32) NOT NULL default '',

  -- Timestamp. Duh.
  log_timestamp binary(14) NOT NULL default '19700101000000',

  -- The user who performed this action; key to user_id
  log_user int unsigned NOT NULL default 0, -- Deprecated in favor of log_actor

  -- Name of the user who performed this action
  log_user_text varchar(255) binary NOT NULL default '', -- Deprecated in favor of log_actor

  -- The actor who performed this action
  log_actor bigint unsigned NOT NULL DEFAULT 0, -- ("DEFAULT 0" is temporary, signaling that log_user/log_user_text should be used)

  -- Key to the page affected. Where a user is the target,
  -- this will point to the user page.
  log_namespace int NOT NULL default 0,
  log_title varchar(255) binary NOT NULL default '',
  log_page int unsigned NULL,

  -- Freeform text. Interpreted as edit history comments.
  -- Deprecated in favor of log_comment_id.
  log_comment varbinary(767) NOT NULL default '',

  -- Key to comment_id. Comment summarizing the change.
  -- ("DEFAULT 0" is temporary, signaling that log_comment should be used)
  log_comment_id bigint unsigned NOT NULL DEFAULT 0,

  -- miscellaneous parameters:
  -- LF separated list (old system) or serialized PHP array (new system)
  log_params blob NOT NULL,

  -- rev_deleted for logs
  log_deleted tinyint unsigned NOT NULL default 0
) ENGINE=InnoDB DEFAULT CHARSET=binary;

-- Special:Log type filter
CREATE INDEX type_time ON logging (log_type, log_timestamp);

-- Special:Log performer filter
CREATE INDEX user_time ON logging (log_user, log_timestamp);
CREATE INDEX actor_time ON logging (log_actor, log_timestamp);

-- Special:Log title filter, log extract
CREATE INDEX page_time ON logging (log_namespace, log_title, log_timestamp);

-- Special:Log unfiltered
CREATE INDEX times ON logging (log_timestamp);

-- Special:Log filter by performer and type
CREATE INDEX log_user_type_time ON logging (log_user, log_type, log_timestamp);
CREATE INDEX log_actor_type_time ON logging (log_actor, log_type, log_timestamp);

-- Apparently just used for a few maintenance pages (findMissingFiles.php, Flow).
-- Could be removed?
CREATE INDEX log_page_id_time ON logging (log_page,log_timestamp);

-- Special:Log action filter
CREATE INDEX type_action ON logging (log_type, log_action, log_timestamp);

-- Special:Log filter by type and anonymous performer
CREATE INDEX log_user_text_type_time ON logging (log_user_text, log_type, log_timestamp);

-- Special:Log filter by anonymous performer
CREATE INDEX log_user_text_time ON logging (log_user_text, log_timestamp);


CREATE TABLE log_search (
  -- The type of ID (rev ID, log ID, rev timestamp, username)
  ls_field varbinary(32) NOT NULL,
  -- The value of the ID
  ls_value varchar(255) NOT NULL,
  -- Key to log_id
  ls_log_id int unsigned NOT NULL default 0,
  PRIMARY KEY (ls_field,ls_value,ls_log_id)
) ENGINE=InnoDB DEFAULT CHARSET=binary;
CREATE INDEX ls_log_id ON log_search (ls_log_id);


-- Jobs performed by parallel apache threads or a command-line daemon
CREATE TABLE job (
  job_id int unsigned NOT NULL PRIMARY KEY AUTO_INCREMENT,

  -- Command name
  -- Limited to 60 to prevent key length overflow
  job_cmd varbinary(60) NOT NULL default '',

  -- Namespace and title to act on
  -- Should be 0 and '' if the command does not operate on a title
  job_namespace int NOT NULL,
  job_title varchar(255) binary NOT NULL,

  -- Timestamp of when the job was inserted
  -- NULL for jobs added before addition of the timestamp
  job_timestamp varbinary(14) NULL default NULL,

  -- Any other parameters to the command
  -- Stored as a PHP serialized array, or an empty string if there are no parameters
  job_params blob NOT NULL,

  -- Random, non-unique, number used for job acquisition (for lock concurrency)
  job_random integer unsigned NOT NULL default 0,

  -- The number of times this job has been locked
  job_attempts integer unsigned NOT NULL default 0,

  -- Field that conveys process locks on rows via process UUIDs
  job_token varbinary(32) NOT NULL default '',

  -- Timestamp when the job was locked
  job_token_timestamp varbinary(14) NULL default NULL,

  -- Base 36 SHA1 of the job parameters relevant to detecting duplicates
  job_sha1 varbinary(32) NOT NULL default ''
) ENGINE=InnoDB DEFAULT CHARSET=binary;

CREATE INDEX job_sha1 ON job (job_sha1);
CREATE INDEX job_cmd_token ON job (job_cmd,job_token,job_random);
CREATE INDEX job_cmd_token_id ON job (job_cmd,job_token,job_id);
CREATE INDEX job_cmd ON job (job_cmd, job_namespace, job_title, job_params(128));
CREATE INDEX job_timestamp ON job (job_timestamp);


-- Details of updates to cached special pages
CREATE TABLE querycache_info (
  -- Special page name
  -- Corresponds to a qc_type value
  qci_type varbinary(32) NOT NULL default '' PRIMARY KEY,

  -- Timestamp of last update
  qci_timestamp binary(14) NOT NULL default '19700101000000'
) ENGINE=InnoDB DEFAULT CHARSET=binary;


-- For each redirect, this table contains exactly one row defining its target
CREATE TABLE redirect (
  -- Key to the page_id of the redirect page
  rd_from int unsigned NOT NULL default 0 PRIMARY KEY,

  -- Key to page_namespace/page_title of the target page.
  -- The target page may or may not exist, and due to renames
  -- and deletions may refer to different page records as time
  -- goes by.
  rd_namespace int NOT NULL default 0,
  rd_title varchar(255) binary NOT NULL default '',
  rd_interwiki varchar(32) default NULL,
  rd_fragment varchar(255) binary default NULL
) ENGINE=InnoDB DEFAULT CHARSET=binary;

CREATE INDEX rd_ns_title ON redirect (rd_namespace,rd_title,rd_from);


-- Used for caching expensive grouped queries that need two links (for example double-redirects)
CREATE TABLE querycachetwo (
  -- A key name, generally the base name of of the special page.
  qcc_type varbinary(32) NOT NULL,

  -- Some sort of stored value. Sizes, counts...
  qcc_value int unsigned NOT NULL default 0,

  -- Target namespace+title
  qcc_namespace int NOT NULL default 0,
  qcc_title varchar(255) binary NOT NULL default '',

  -- Target namespace+title2
  qcc_namespacetwo int NOT NULL default 0,
  qcc_titletwo varchar(255) binary NOT NULL default ''
) ENGINE=InnoDB DEFAULT CHARSET=binary;

CREATE INDEX qcc_type ON querycachetwo (qcc_type,qcc_value);
CREATE INDEX qcc_title ON querycachetwo (qcc_type,qcc_namespace,qcc_title);
CREATE INDEX qcc_titletwo ON querycachetwo (qcc_type,qcc_namespacetwo,qcc_titletwo);


-- Used for storing page restrictions (i.e. protection levels)
CREATE TABLE page_restrictions (
  -- Field for an ID for this restrictions row (sort-key for Special:ProtectedPages)
  pr_id int unsigned NOT NULL PRIMARY KEY AUTO_INCREMENT,
  -- Page to apply restrictions to (Foreign Key to page).
  pr_page int NOT NULL,
  -- The protection type (edit, move, etc)
  pr_type varbinary(60) NOT NULL,
  -- The protection level (Sysop, autoconfirmed, etc)
  pr_level varbinary(60) NOT NULL,
  -- Whether or not to cascade the protection down to pages transcluded.
  pr_cascade tinyint NOT NULL,
  -- Field for future support of per-user restriction.
  pr_user int unsigned NULL,
  -- Field for time-limited protection.
  pr_expiry varbinary(14) NULL
) ENGINE=InnoDB DEFAULT CHARSET=binary;

CREATE UNIQUE INDEX pr_pagetype ON page_restrictions (pr_page,pr_type);
CREATE INDEX pr_typelevel ON page_restrictions (pr_type,pr_level);
CREATE INDEX pr_level ON page_restrictions (pr_level);
CREATE INDEX pr_cascade ON page_restrictions (pr_cascade);


-- Protected titles - nonexistent pages that have been protected
CREATE TABLE protected_titles (
  pt_namespace int NOT NULL,
  pt_title varchar(255) binary NOT NULL,
  pt_user int unsigned NOT NULL,
  pt_reason varbinary(767) default '', -- Deprecated.
  pt_reason_id bigint unsigned NOT NULL DEFAULT 0, -- ("DEFAULT 0" is temporary, signaling that pt_reason should be used)
  pt_timestamp binary(14) NOT NULL,
  pt_expiry varbinary(14) NOT NULL default '',
  pt_create_perm varbinary(60) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=binary;

CREATE UNIQUE INDEX pt_namespace_title ON protected_titles (pt_namespace,pt_title);
CREATE INDEX pt_timestamp ON protected_titles (pt_timestamp);


-- Name/value pairs indexed by page_id
CREATE TABLE page_props (
  pp_page int NOT NULL,
  pp_propname varbinary(60) NOT NULL,
  pp_value blob NOT NULL,
  pp_sortkey float DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=binary;

CREATE UNIQUE INDEX pp_page_propname ON page_props (pp_page,pp_propname);
CREATE UNIQUE INDEX pp_propname_page ON page_props (pp_propname,pp_page);
CREATE UNIQUE INDEX pp_propname_sortkey_page ON page_props (pp_propname,pp_sortkey,pp_page);

-- A table to log updates, one text key row per update.
CREATE TABLE updatelog (
  ul_key varchar(255) NOT NULL PRIMARY KEY,
  ul_value blob
) ENGINE=InnoDB DEFAULT CHARSET=binary;


-- A table to track tags for revisions, logs and recent changes.
CREATE TABLE change_tag (
  ct_id int unsigned NOT NULL PRIMARY KEY AUTO_INCREMENT,
  -- RCID for the change
  ct_rc_id int NULL,
  -- LOGID for the change
  ct_log_id int unsigned NULL,
  -- REVID for the change
  ct_rev_id int unsigned NULL,
  -- Tag applied
  ct_tag varchar(255) NOT NULL,
  -- Parameters for the tag; used by some extensions
  ct_params blob NULL
) ENGINE=InnoDB DEFAULT CHARSET=binary;

CREATE UNIQUE INDEX change_tag_rc_tag ON change_tag (ct_rc_id,ct_tag);
CREATE UNIQUE INDEX change_tag_log_tag ON change_tag (ct_log_id,ct_tag);
CREATE UNIQUE INDEX change_tag_rev_tag ON change_tag (ct_rev_id,ct_tag);
-- Covering index, so we can pull all the info only out of the index.
CREATE INDEX change_tag_tag_id ON change_tag (ct_tag,ct_rc_id,ct_rev_id,ct_log_id);


-- Rollup table to pull a LIST of tags simply without ugly GROUP_CONCAT
-- that only works on MySQL 4.1+
CREATE TABLE tag_summary (
  ts_id int unsigned NOT NULL PRIMARY KEY AUTO_INCREMENT,
  -- RCID for the change
  ts_rc_id int NULL,
  -- LOGID for the change
  ts_log_id int unsigned NULL,
  -- REVID for the change
  ts_rev_id int unsigned NULL,
  -- Comma-separated list of tags
  ts_tags blob NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=binary;

CREATE UNIQUE INDEX tag_summary_rc_id ON tag_summary (ts_rc_id);
CREATE UNIQUE INDEX tag_summary_log_id ON tag_summary (ts_log_id);
CREATE UNIQUE INDEX tag_summary_rev_id ON tag_summary (ts_rev_id);


CREATE TABLE valid_tag (
  vt_tag varchar(255) NOT NULL PRIMARY KEY
) ENGINE=InnoDB DEFAULT CHARSET=binary;

-- Table for storing localisation data
CREATE TABLE l10n_cache (
  -- Language code
  lc_lang varbinary(32) NOT NULL,
  -- Cache key
  lc_key varchar(255) NOT NULL,
  -- Value
  lc_value mediumblob NOT NULL,
  PRIMARY KEY (lc_lang, lc_key)
) ENGINE=InnoDB DEFAULT CHARSET=binary;

-- Table caching which local files a module depends on that aren't
-- registered directly, used for fast retrieval of file dependency.
-- Currently only used for tracking images that CSS depends on
CREATE TABLE module_deps (
  -- Module name
  md_module varbinary(255) NOT NULL,
  -- Module context vary (includes skin and language; called "md_skin" for legacy reasons)
  md_skin varbinary(32) NOT NULL,
  -- JSON blob with file dependencies
  md_deps mediumblob NOT NULL,
  PRIMARY KEY (md_module,md_skin)
) ENGINE=InnoDB DEFAULT CHARSET=binary;

-- Holds all the sites known to the wiki.
CREATE TABLE sites (
  -- Numeric id of the site
  site_id                    INT UNSIGNED        NOT NULL PRIMARY KEY AUTO_INCREMENT,

  -- Global identifier for the site, ie 'enwiktionary'
  site_global_key            varbinary(32)       NOT NULL,

  -- Type of the site, ie 'mediawiki'
  site_type                  varbinary(32)       NOT NULL,

  -- Group of the site, ie 'wikipedia'
  site_group                 varbinary(32)       NOT NULL,

  -- Source of the site data, ie 'local', 'wikidata', 'my-magical-repo'
  site_source                varbinary(32)       NOT NULL,

  -- Language code of the sites primary language.
  site_language              varbinary(32)       NOT NULL,

  -- Protocol of the site, ie 'http://', 'irc://', '//'
  -- This field is an index for lookups and is build from type specific data in site_data.
  site_protocol              varbinary(32)       NOT NULL,

  -- Domain of the site in reverse order, ie 'org.mediawiki.www.'
  -- This field is an index for lookups and is build from type specific data in site_data.
  site_domain                VARCHAR(255)        NOT NULL,

  -- Type dependent site data.
  site_data                  BLOB                NOT NULL,

  -- If site.tld/path/key:pageTitle should forward users to  the page on
  -- the actual site, where "key" is the local identifier.
  site_forward              bool                NOT NULL,

  -- Type dependent site config.
  -- For instance if template transclusion should be allowed if it's a MediaWiki.
  site_config               BLOB                NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=binary;

CREATE UNIQUE INDEX sites_global_key ON sites (site_global_key);
CREATE INDEX sites_type ON sites (site_type);
CREATE INDEX sites_group ON sites (site_group);
CREATE INDEX sites_source ON sites (site_source);
CREATE INDEX sites_language ON sites (site_language);
CREATE INDEX sites_protocol ON sites (site_protocol);
CREATE INDEX sites_domain ON sites (site_domain);
CREATE INDEX sites_forward ON sites (site_forward);

-- Links local site identifiers to their corresponding site.
CREATE TABLE site_identifiers (
  -- Key on site.site_id
  si_site                    INT UNSIGNED        NOT NULL,

  -- local key type, ie 'interwiki' or 'langlink'
  si_type                    varbinary(32)       NOT NULL,

  -- local key value, ie 'en' or 'wiktionary'
  si_key                     varbinary(32)       NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=binary;

CREATE UNIQUE INDEX site_ids_type ON site_identifiers (si_type, si_key);
CREATE INDEX site_ids_site ON site_identifiers (si_site);
CREATE INDEX site_ids_key ON site_identifiers (si_key);

-- vim: sw=2 sts=2 et
