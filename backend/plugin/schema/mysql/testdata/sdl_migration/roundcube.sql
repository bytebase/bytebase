-- Roundcube Webmail initial database structure


SET FOREIGN_KEY_CHECKS=0;

-- Table structure for table `session`

CREATE TABLE `session` (
 `sess_id` varchar(128) NOT NULL,
 `expires_at` datetime NOT NULL DEFAULT '1000-01-01 00:00:00',
 `ip` varchar(40) NOT NULL,
 `vars` mediumtext NOT NULL,
 PRIMARY KEY(`sess_id`),
 INDEX `expires_at_index` (`expires_at`)
) ROW_FORMAT=DYNAMIC ENGINE=INNODB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;


-- Table structure for table `users`

CREATE TABLE `users` (
 `user_id` int(10) UNSIGNED NOT NULL AUTO_INCREMENT,
 `username` varchar(128) BINARY NOT NULL,
 `mail_host` varchar(128) NOT NULL,
 `created` datetime NOT NULL DEFAULT '1000-01-01 00:00:00',
 `last_login` datetime,
 `failed_login` datetime,
 `failed_login_counter` int(10) UNSIGNED,
 `language` varchar(16),
 `preferences` longtext,
 PRIMARY KEY(`user_id`),
 UNIQUE `username` (`username`, `mail_host`)
) ROW_FORMAT=DYNAMIC ENGINE=INNODB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;


-- Table structure for table `cache`

CREATE TABLE `cache` (
 `user_id` int(10) UNSIGNED NOT NULL,
 `cache_key` varchar(128) BINARY NOT NULL,
 `expires` datetime,
 `data` longtext NOT NULL,
 PRIMARY KEY (`user_id`, `cache_key`),
 CONSTRAINT `user_id_fk_cache` FOREIGN KEY (`user_id`)
   REFERENCES `users`(`user_id`) ON DELETE CASCADE ON UPDATE CASCADE,
 INDEX `expires_index` (`expires`)
) ROW_FORMAT=DYNAMIC ENGINE=INNODB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;


-- Table structure for table `cache_shared`

CREATE TABLE `cache_shared` (
 `cache_key` varchar(255) BINARY NOT NULL,
 `expires` datetime,
 `data` longtext NOT NULL,
 PRIMARY KEY (`cache_key`),
 INDEX `expires_index` (`expires`)
) ROW_FORMAT=DYNAMIC ENGINE=INNODB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;


-- Table structure for table `cache_index`

CREATE TABLE `cache_index` (
 `user_id` int(10) UNSIGNED NOT NULL,
 `mailbox` varchar(255) BINARY NOT NULL,
 `expires` datetime,
 `valid` tinyint(1) NOT NULL DEFAULT '0',
 `data` longtext NOT NULL,
 CONSTRAINT `user_id_fk_cache_index` FOREIGN KEY (`user_id`)
   REFERENCES `users`(`user_id`) ON DELETE CASCADE ON UPDATE CASCADE,
 INDEX `expires_index` (`expires`),
 PRIMARY KEY (`user_id`, `mailbox`)
) ROW_FORMAT=DYNAMIC ENGINE=INNODB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;


-- Table structure for table `cache_thread`

CREATE TABLE `cache_thread` (
 `user_id` int(10) UNSIGNED NOT NULL,
 `mailbox` varchar(255) BINARY NOT NULL,
 `expires` datetime,
 `data` longtext NOT NULL,
 CONSTRAINT `user_id_fk_cache_thread` FOREIGN KEY (`user_id`)
   REFERENCES `users`(`user_id`) ON DELETE CASCADE ON UPDATE CASCADE,
 INDEX `expires_index` (`expires`),
 PRIMARY KEY (`user_id`, `mailbox`)
) ROW_FORMAT=DYNAMIC ENGINE=INNODB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;


-- Table structure for table `cache_messages`

CREATE TABLE `cache_messages` (
 `user_id` int(10) UNSIGNED NOT NULL,
 `mailbox` varchar(255) BINARY NOT NULL,
 `uid` int(11) UNSIGNED NOT NULL DEFAULT '0',
 `expires` datetime,
 `data` longtext NOT NULL,
 `flags` int(11) NOT NULL DEFAULT '0',
 CONSTRAINT `user_id_fk_cache_messages` FOREIGN KEY (`user_id`)
   REFERENCES `users`(`user_id`) ON DELETE CASCADE ON UPDATE CASCADE,
 INDEX `expires_index` (`expires`),
 PRIMARY KEY (`user_id`, `mailbox`, `uid`)
) ROW_FORMAT=DYNAMIC ENGINE=INNODB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;


-- Table structure for table `collected_addresses`

CREATE TABLE `collected_addresses` (
 `address_id` int(10) UNSIGNED NOT NULL AUTO_INCREMENT,
 `changed` datetime NOT NULL DEFAULT '1000-01-01 00:00:00',
 `name` varchar(255) NOT NULL DEFAULT '',
 `email` varchar(255) NOT NULL,
 `user_id` int(10) UNSIGNED NOT NULL,
 `type` int(10) UNSIGNED NOT NULL,
 PRIMARY KEY(`address_id`),
 CONSTRAINT `user_id_fk_collected_addresses` FOREIGN KEY (`user_id`)
   REFERENCES `users`(`user_id`) ON DELETE CASCADE ON UPDATE CASCADE,
 UNIQUE INDEX `user_email_collected_addresses_index` (`user_id`, `type`, `email`)
) ROW_FORMAT=DYNAMIC ENGINE=INNODB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;


-- Table structure for table `contacts`

CREATE TABLE `contacts` (
 `contact_id` int(10) UNSIGNED NOT NULL AUTO_INCREMENT,
 `changed` datetime NOT NULL DEFAULT '1000-01-01 00:00:00',
 `del` tinyint(1) NOT NULL DEFAULT '0',
 `name` varchar(128) NOT NULL DEFAULT '',
 `email` text NOT NULL,
 `firstname` varchar(128) NOT NULL DEFAULT '',
 `surname` varchar(128) NOT NULL DEFAULT '',
 `vcard` longtext,
 `words` text,
 `user_id` int(10) UNSIGNED NOT NULL,
 PRIMARY KEY(`contact_id`),
 CONSTRAINT `user_id_fk_contacts` FOREIGN KEY (`user_id`)
   REFERENCES `users`(`user_id`) ON DELETE CASCADE ON UPDATE CASCADE,
 INDEX `user_contacts_index` (`user_id`,`del`)
) ROW_FORMAT=DYNAMIC ENGINE=INNODB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;


-- Table structure for table `contactgroups`

CREATE TABLE `contactgroups` (
  `contactgroup_id` int(10) UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` int(10) UNSIGNED NOT NULL,
  `changed` datetime NOT NULL DEFAULT '1000-01-01 00:00:00',
  `del` tinyint(1) NOT NULL DEFAULT '0',
  `name` varchar(128) NOT NULL DEFAULT '',
  PRIMARY KEY(`contactgroup_id`),
  CONSTRAINT `user_id_fk_contactgroups` FOREIGN KEY (`user_id`)
    REFERENCES `users`(`user_id`) ON DELETE CASCADE ON UPDATE CASCADE,
  INDEX `contactgroups_user_index` (`user_id`,`del`)
) ROW_FORMAT=DYNAMIC ENGINE=INNODB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;


-- Table structure for table `contactgroupmembers`

CREATE TABLE `contactgroupmembers` (
  `contactgroup_id` int(10) UNSIGNED NOT NULL,
  `contact_id` int(10) UNSIGNED NOT NULL,
  `created` datetime NOT NULL DEFAULT '1000-01-01 00:00:00',
  PRIMARY KEY (`contactgroup_id`, `contact_id`),
  CONSTRAINT `contactgroup_id_fk_contactgroups` FOREIGN KEY (`contactgroup_id`)
    REFERENCES `contactgroups`(`contactgroup_id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `contact_id_fk_contacts` FOREIGN KEY (`contact_id`)
    REFERENCES `contacts`(`contact_id`) ON DELETE CASCADE ON UPDATE CASCADE,
  INDEX `contactgroupmembers_contact_index` (`contact_id`)
) ROW_FORMAT=DYNAMIC ENGINE=INNODB;


-- Table structure for table `identities`

CREATE TABLE `identities` (
 `identity_id` int(10) UNSIGNED NOT NULL AUTO_INCREMENT,
 `user_id` int(10) UNSIGNED NOT NULL,
 `changed` datetime NOT NULL DEFAULT '1000-01-01 00:00:00',
 `del` tinyint(1) NOT NULL DEFAULT '0',
 `standard` tinyint(1) NOT NULL DEFAULT '0',
 `name` varchar(128) NOT NULL,
 `organization` varchar(128) NOT NULL DEFAULT '',
 `email` varchar(128) NOT NULL,
 `reply-to` varchar(128) NOT NULL DEFAULT '',
 `bcc` varchar(128) NOT NULL DEFAULT '',
 `signature` longtext,
 `html_signature` tinyint(1) NOT NULL DEFAULT '0',
 PRIMARY KEY(`identity_id`),
 CONSTRAINT `user_id_fk_identities` FOREIGN KEY (`user_id`)
   REFERENCES `users`(`user_id`) ON DELETE CASCADE ON UPDATE CASCADE,
 INDEX `user_identities_index` (`user_id`, `del`),
 INDEX `email_identities_index` (`email`, `del`)
) ROW_FORMAT=DYNAMIC ENGINE=INNODB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;


-- Table structure for table `responses`

CREATE TABLE `responses` (
 `response_id` int(10) UNSIGNED NOT NULL AUTO_INCREMENT,
 `user_id` int(10) UNSIGNED NOT NULL,
 `name` varchar(255) NOT NULL,
 `data` longtext NOT NULL,
 `is_html` tinyint(1) NOT NULL DEFAULT '0',
 `changed` datetime NOT NULL DEFAULT '1000-01-01 00:00:00',
 `del` tinyint(1) NOT NULL DEFAULT '0',
 PRIMARY KEY (`response_id`),
 CONSTRAINT `user_id_fk_responses` FOREIGN KEY (`user_id`)
   REFERENCES `users` (`user_id`) ON DELETE CASCADE ON UPDATE CASCADE,
 INDEX `user_responses_index` (`user_id`, `del`)
) ROW_FORMAT=DYNAMIC ENGINE=INNODB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;


-- Table structure for table `dictionary`

CREATE TABLE `dictionary` (
  `id` int(10) UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY, -- redundant, for compat. with Galera Cluster
  `user_id` int(10) UNSIGNED, -- NULL here is for "shared dictionaries"
  `language` varchar(16) NOT NULL,
  `data` longtext NOT NULL,
  CONSTRAINT `user_id_fk_dictionary` FOREIGN KEY (`user_id`)
    REFERENCES `users`(`user_id`) ON DELETE CASCADE ON UPDATE CASCADE,
  UNIQUE `uniqueness` (`user_id`, `language`)
) ROW_FORMAT=DYNAMIC ENGINE=INNODB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;


-- Table structure for table `searches`

CREATE TABLE `searches` (
 `search_id` int(10) UNSIGNED NOT NULL AUTO_INCREMENT,
 `user_id` int(10) UNSIGNED NOT NULL,
 `type` int(3) NOT NULL DEFAULT '0',
 `name` varchar(128) NOT NULL,
 `data` text,
 PRIMARY KEY(`search_id`),
 CONSTRAINT `user_id_fk_searches` FOREIGN KEY (`user_id`)
   REFERENCES `users`(`user_id`) ON DELETE CASCADE ON UPDATE CASCADE,
 UNIQUE `uniqueness` (`user_id`, `type`, `name`)
) ROW_FORMAT=DYNAMIC ENGINE=INNODB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Table structure for table `filestore`

CREATE TABLE `filestore` (
 `file_id` int(10) UNSIGNED NOT NULL AUTO_INCREMENT,
 `user_id` int(10) UNSIGNED NOT NULL,
 `context` varchar(32) NOT NULL,
 `filename` varchar(128) NOT NULL,
 `mtime` int(10) NOT NULL,
 `data` longtext NOT NULL,
 PRIMARY KEY (`file_id`),
 CONSTRAINT `user_id_fk_filestore` FOREIGN KEY (`user_id`)
   REFERENCES `users` (`user_id`) ON DELETE CASCADE ON UPDATE CASCADE,
 UNIQUE `uniqueness` (`user_id`, `context`, `filename`)
) ROW_FORMAT=DYNAMIC ENGINE=INNODB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Table structure for table `uploads`

CREATE TABLE `uploads` (
 `upload_id` varchar(64) NOT NULL,
 `session_id` varchar(128) NOT NULL,
 `group` varchar(128) NOT NULL,
 `metadata` mediumtext NOT NULL,
 `created` datetime NOT NULL DEFAULT '1000-01-01 00:00:00',
 PRIMARY KEY (`upload_id`),
 INDEX `uploads_session_group_index` (`session_id`, `group`, `created`)
) ROW_FORMAT=DYNAMIC ENGINE=INNODB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Table structure for table `system`

CREATE TABLE `system` (
 `name` varchar(64) NOT NULL,
 `value` mediumtext,
 PRIMARY KEY(`name`)
) ROW_FORMAT=DYNAMIC ENGINE=INNODB CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

SET FOREIGN_KEY_CHECKS=1;

