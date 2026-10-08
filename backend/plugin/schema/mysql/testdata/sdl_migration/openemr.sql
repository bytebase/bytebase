--
-- Database: `openemr`
--
-- Keep v_database in sync with $v_database in version.php.
-- CI will fail if they don't match.
-- v_database: 541
--

--
-- Table structure for table `addresses`
--

DROP TABLE IF EXISTS `addresses`;
CREATE TABLE `addresses` (
  `id` int(11) NOT NULL default '0',
  `line1` varchar(255) default NULL,
  `line2` varchar(255) default NULL,
  `city` varchar(255) default NULL,
  `state` varchar(35) default NULL,
  `zip` varchar(10) default NULL,
  `plus_four` varchar(4) default NULL,
  `country` varchar(255) default NULL,
  `foreign_id` int(11) default NULL,
  `district` VARCHAR(255) DEFAULT NULL COMMENT 'The county or district of the address',
  PRIMARY KEY  (`id`),
  KEY `foreign_id` (`foreign_id`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `amc_misc_data`
--

DROP TABLE IF EXISTS `amc_misc_data`;
CREATE TABLE `amc_misc_data` (
  `amc_id` varchar(31) NOT NULL DEFAULT '' COMMENT 'Unique and maps to list_options list clinical_rules',
  `pid` bigint(20) default NULL,
  `map_category` varchar(255) NOT NULL default '' COMMENT 'Maps to an object category (such as prescriptions etc.)',
  `map_id` bigint(20) NOT NULL default '0' COMMENT 'Maps to an object id (such as prescription id etc.)',
  `date_created` datetime default NULL,
  `date_completed` datetime default NULL,
  `soc_provided` datetime default NULL,
  KEY  (`amc_id`,`pid`,`map_id`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `amendments`
--

DROP TABLE IF EXISTS `amendments`;
CREATE TABLE `amendments` (
  `amendment_id` int(11) NOT NULL AUTO_INCREMENT COMMENT 'Amendment ID',
  `amendment_date` date NOT NULL COMMENT 'Amendement request date',
  `amendment_by` varchar(50) NOT NULL COMMENT 'Amendment requested from',
  `amendment_status` varchar(50) NULL COMMENT 'Amendment status accepted/rejected/null',
  `pid` bigint(20) NOT NULL COMMENT 'Patient ID from patient_data',
  `amendment_desc` text COMMENT 'Amendment Details',
  `created_by` int(11) NOT NULL COMMENT 'references users.id for session owner',
  `modified_by` int(11) NULL COMMENT 'references users.id for session owner',
  `created_time` timestamp NULL COMMENT 'created time',
  `modified_time` timestamp NULL COMMENT 'modified time',
  PRIMARY KEY amendments_id(`amendment_id`),
  KEY amendment_pid(`pid`)
) ENGINE = InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `amendments_history`
--

DROP TABLE IF EXISTS `amendments_history`;
CREATE TABLE `amendments_history` (
  `amendment_id` int(11) NOT NULL AUTO_INCREMENT COMMENT 'Amendment ID',
  `amendment_note` text COMMENT 'Amendment requested from',
  `amendment_status` VARCHAR(50) NULL COMMENT 'Amendment Request Status',
  `created_by` int(11) NOT NULL COMMENT 'references users.id for session owner',
  `created_time` timestamp NULL COMMENT 'created time',
KEY amendment_history_id(`amendment_id`)
) ENGINE = InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `api_log`
--

DROP TABLE IF EXISTS `api_log`;
CREATE TABLE `api_log` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `log_id` int(11) NOT NULL,
  `user_id` bigint(20) NOT NULL,
  `patient_id` bigint(20) NOT NULL,
  `ip_address` varchar(255) NOT NULL,
  `method` varchar(20) NOT NULL,
  `request` varchar(255) NOT NULL,
  `request_url` text,
  `request_body` longtext,
  `response` longtext,
  `created_time` timestamp NULL,
  PRIMARY KEY (`id`)
) ENGINE = InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `api_token`
--

DROP TABLE IF EXISTS `api_token`;
CREATE TABLE `api_token` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `user_id` varchar(40) DEFAULT NULL,
  `token` varchar(128) DEFAULT NULL,
  `expiry` datetime DEFAULT NULL,
  `client_id` varchar(80) DEFAULT NULL,
  `scope` text COMMENT 'json encoded',
  `revoked` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '1=revoked,0=not revoked',
  `context` TEXT COMMENT 'context values that change/govern how access token are used',
  PRIMARY KEY (`id`),
  UNIQUE KEY `token` (`token`)
) ENGINE = InnoDB;

--
-- Table structure for table `api_refresh_token`
--
DROP TABLE IF EXISTS `api_refresh_token`;
CREATE TABLE `api_refresh_token` (
 `id` BIGINT(20) NOT NULL AUTO_INCREMENT,
 `user_id` VARCHAR(40) DEFAULT NULL,
 `client_id` VARCHAR(80) DEFAULT NULL,
 `token` VARCHAR(128) NOT NULL,
 `expiry` DATETIME DEFAULT NULL,
 `revoked` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '1=revoked,0=not revoked',
 PRIMARY KEY (`id`),
 UNIQUE KEY (`token`),
 INDEX `api_refresh_token_usr_client_idx` (`client_id`, `user_id`)
) ENGINE = InnoDB COMMENT = 'Holds information about api refresh tokens.';
-- --------------------------------------------------------

--
-- Table structure for table `audit_master`
--

DROP TABLE IF EXISTS `audit_master`;
CREATE TABLE `audit_master` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `pid` bigint(20) NOT NULL,
  `user_id` bigint(20) NOT NULL COMMENT 'The Id of the user who approves or denies',
  `approval_status` tinyint(4) NOT NULL COMMENT '1-Pending,2-Approved,3-Denied,4-Appointment directly updated to calendar table,5-Cancelled appointment',
  `comments` text,
  `created_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `modified_time` datetime NOT NULL,
  `ip_address` varchar(100) NOT NULL,
  `type` tinyint(4) NOT NULL COMMENT '1-new patient,2-existing patient,3-change is only in the document,4-Patient upload,5-random key,10-Appointment',
  `is_qrda_document` BOOLEAN NULL DEFAULT FALSE,
  `is_unstructured_document` BOOLEAN NULL DEFAULT FALSE,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `audit_details`
--

DROP TABLE IF EXISTS `audit_details`;
CREATE TABLE `audit_details` (
  `id` BIGINT(20) NOT NULL AUTO_INCREMENT,
  `table_name` VARCHAR(100) NOT NULL COMMENT 'openemr table name',
  `field_name` VARCHAR(100) NOT NULL COMMENT 'openemr table''s field name',
  `field_value` LONGTEXT COMMENT 'openemr table''s field value',
  `audit_master_id` BIGINT(20) NOT NULL COMMENT 'Id of the audit_master table',
  `entry_identification` VARCHAR(255) NOT NULL DEFAULT '1' COMMENT 'Used when multiple entry occurs from the same table.1 means no multiple entry',
  PRIMARY KEY (`id`),
  KEY `audit_master_id` (`audit_master_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `background_services`
--

DROP TABLE IF EXISTS `background_services`;
CREATE TABLE `background_services` (
  `name` varchar(31) NOT NULL,
  `title` varchar(127) NOT NULL COMMENT 'name for reports',
  `active` tinyint(1) NOT NULL default '0',
  `running` tinyint(1) NOT NULL default '-1' COMMENT 'True indicates managed service is busy. Skip this interval',
  `next_run` timestamp NOT NULL default CURRENT_TIMESTAMP,
  `execute_interval` int(11) NOT NULL default '0' COMMENT 'minimum number of minutes between function calls,0=manual mode',
  `function` varchar(127) NOT NULL COMMENT 'name of background service function',
  `require_once` varchar(255) default NULL COMMENT 'include file (if necessary)',
  `sort_order` int(11) NOT NULL default '100' COMMENT 'lower numbers will be run first',
  `lock_expires_at` datetime DEFAULT NULL COMMENT 'Lease expiration. Compared with NOW() on acquire, so the stored value uses whatever session timezone is in effect (OpenEMR syncs it to gbl_time_zone). Set on acquire, cleared on release. Expired leases are automatically stolen by the next worker.',
  PRIMARY KEY  (`name`)
) ENGINE=InnoDB;



--
-- Inserting data for table `background_services`
--

-- --------------------------------------------------------

--
-- Table structure for table `batchcom`
--

DROP TABLE IF EXISTS `batchcom`;
CREATE TABLE `batchcom` (
  `id` bigint(20) NOT NULL auto_increment,
  `patient_id` bigint(20) NOT NULL default '0',
  `sent_by` bigint(20) NOT NULL default '0',
  `msg_type` varchar(60) default NULL,
  `msg_subject` varchar(255) default NULL,
  `msg_text` mediumtext,
  `msg_date_sent` datetime NULL,
  PRIMARY KEY  (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `billing`
--

DROP TABLE IF EXISTS `billing`;
CREATE TABLE `billing` (
  `id` int(11) NOT NULL auto_increment,
  `date` datetime default NULL,
  `code_type` varchar(15) default NULL,
  `code` varchar(20) default NULL,
  `pid` bigint(20) default NULL,
  `provider_id` int(11) default NULL,
  `user` int(11) default NULL,
  `groupname` varchar(255) default NULL,
  `authorized` tinyint(1) default NULL,
  `encounter` int(11) default NULL,
  `code_text` longtext,
  `billed` tinyint(1) default NULL,
  `activity` tinyint(1) default NULL,
  `payer_id` int(11) default NULL,
  `bill_process` tinyint(2) NOT NULL default '0',
  `bill_date` datetime default NULL,
  `process_date` datetime default NULL,
  `process_file` varchar(255) default NULL,
  `modifier` varchar(12) default NULL,
  `units` int(11) default NULL,
  `fee` decimal(12,2) default NULL,
  `justify` varchar(255) default NULL,
  `target` varchar(30) default NULL,
  `x12_partner_id` int(11) default NULL,
  `ndc_info` varchar(255) default NULL,
  `notecodes` varchar(25) NOT NULL default '',
  `external_id` VARCHAR(20) DEFAULT NULL,
  `pricelevel` varchar(31) default '',
  `revenue_code` varchar(6) NOT NULL default '' COMMENT 'Item revenue code',
  `chargecat` varchar(31) default '' COMMENT 'Charge category or customer',
  PRIMARY KEY  (`id`),
  KEY `pid` (`pid`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `categories`
--

DROP TABLE IF EXISTS `categories`;
CREATE TABLE `categories` (
  `id` int(11) NOT NULL default '0',
  `name` varchar(255) default NULL,
  `value` varchar(255) default NULL,
  `parent` int(11) NOT NULL default '0',
  `lft` int(11) NOT NULL default '0',
  `rght` int(11) NOT NULL default '0',
  `aco_spec` varchar(63) NOT NULL DEFAULT 'patients|docs',
  `codes` varchar(255) NOT NULL DEFAULT '' COMMENT 'Category codes for documents stored in this category',
  PRIMARY KEY  (`id`),
  KEY `parent` (`parent`),
  KEY `lft` (`lft`,`rght`)
) ENGINE=InnoDB;

--
-- Inserting data for table `categories`
--

-- --------------------------------------------------------

--
-- Table structure for table `categories_seq`
--

DROP TABLE IF EXISTS `categories_seq`;
CREATE TABLE `categories_seq` (
  `id` int(11) NOT NULL default '0',
  PRIMARY KEY  (`id`)
) ENGINE=InnoDB;

--
-- Inserting data for table `categories_seq`
--

-- --------------------------------------------------------

--
-- Table structure for table `categories_to_documents`
--

DROP TABLE IF EXISTS `categories_to_documents`;
CREATE TABLE `categories_to_documents` (
  `category_id` int(11) NOT NULL default '0',
  `document_id` int(11) NOT NULL default '0',
  PRIMARY KEY  (`category_id`,`document_id`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `claims`
--

DROP TABLE IF EXISTS `claims`;
CREATE TABLE `claims` (
  `patient_id` bigint(20) NOT NULL,
  `encounter_id` int(11) NOT NULL,
  `version` int(10) unsigned NOT NULL COMMENT 'Claim version, incremented in code',
  `payer_id` int(11) NOT NULL default '0',
  `status` tinyint(2) NOT NULL default '0',
  `payer_type` tinyint(4) NOT NULL default '0',
  `bill_process` tinyint(2) NOT NULL default '0',
  `bill_time` datetime default NULL,
  `process_time` datetime default NULL,
  `process_file` varchar(255) default NULL,
  `target` varchar(30) default NULL,
  `x12_partner_id` int(11) NOT NULL default '0',
  `submitted_claim` text COMMENT 'This claims form claim data',
  PRIMARY KEY  (`patient_id`,`encounter_id`,`version`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `clinical_plans`
--

DROP TABLE IF EXISTS `clinical_plans`;
CREATE TABLE `clinical_plans` (
  `id` varchar(31) NOT NULL DEFAULT '' COMMENT 'Unique and maps to list_options list clinical_plans',
  `pid` bigint(20) NOT NULL DEFAULT '0' COMMENT '0 is default for all patients, while > 0 is id from patient_data table',
  `normal_flag` tinyint(1) COMMENT 'Normal Activation Flag',
  `cqm_flag` tinyint(1) COMMENT 'Clinical Quality Measure flag (unable to customize per patient)',
  `cqm_2011_flag` tinyint(1) COMMENT '2011 Clinical Quality Measure flag (unable to customize per patient)',
  `cqm_2014_flag` tinyint(1) COMMENT '2014 Clinical Quality Measure flag (unable to customize per patient)',
  `cqm_measure_group` varchar(10) NOT NULL default '' COMMENT 'Clinical Quality Measure Group Identifier',
  PRIMARY KEY  (`id`,`pid`)
) ENGINE=InnoDB;

--
-- Inserting data for Clinical Quality Measure (CQM) plans
--
--   Inserting data for Measure Group A: Diabetes Mellitus
--

--
--   Inserting data for Measure Group C: Chronic Kidney Disease (CKD)
--

--
--   Inserting data for Measure Group D: Preventative Care
--

--
--   Inserting data for Measure Group E: Perioperative Care
--

--
--   Inserting data for Measure Group F: Rheumatoid Arthritis
--

--
--    Inserting data for Measure Group G: Back Pain
--

--
--   Inserting data for Measure Group H: Coronary Artery Bypass Graft (CABG)
--

--
-- Inserting data for Standard clinical plans
--
--   Inserting data for Diabetes Mellitus
--

--
--   Inserting data for Prevention Plan
--

-- --------------------------------------------------------

--
-- Table structure for table `clinical_plans_rules`
--

DROP TABLE IF EXISTS `clinical_plans_rules`;
CREATE TABLE `clinical_plans_rules` (
  `plan_id` varchar(31) NOT NULL DEFAULT '' COMMENT 'Unique and maps to list_options list clinical_plans',
  `rule_id` varchar(31) NOT NULL DEFAULT '' COMMENT 'Unique and maps to list_options list clinical_rules',
  PRIMARY KEY  (`plan_id`,`rule_id`)
) ENGINE=InnoDB;

--
-- Inserting data for Clinical Quality Measure (CQM) plans to rules mappings
--
--   Inserting data for Measure Group A: Diabetes Mellitus
--
--     Inserting data for NQF 0059 (PQRI 1)   Diabetes: HbA1c Poor Control
--

--
--     Inserting data for NQF 0064 (PQRI 2)   Diabetes: LDL Management & Control
--

--
--     Inserting data for NQF 0061 (PQRI 3)   Diabetes: Blood Pressure Management
--

--
--     Inserting data for NQF 0055 (PQRI 117) Diabetes: Eye Exam
--

--
--     Inserting data for NQF 0056 (PQRI 163) Diabetes: Foot Exam
--

--
-- Inserting data for Measure Group D: Preventative Care
--
--   Inserting data for NQF 0041 (PQRI 110) Influenza Immunization for Patients >= 50 Years Old
--

--
--   Inserting data for NQF 0043 (PQRI 111) Pneumonia Vaccination Status for Older Adults
--

--
--   Inserting data for NQF 0421 (PQRI 128) Adult Weight Screening and Follow-Up
--

--
-- Inserting data for Standard clinical plans to rules mappings
--
--   Inserting data for Diabetes Mellitus
--
--     Inserting data for Hemoglobin A1C

--
--     Inserting data for Urine Microalbumin
--

--
--     Inserting data for Eye Exam

--
--     Inserting data for Foot Exam
--

--
--   Inserting data for Preventative Care
--
--     Inserting data for Hypertension: Blood Pressure Measurement
--

--
--     Inserting data for Tobacco Use Assessment
--

--
--     Inserting data for Tobacco Cessation Intervention
--

--
--     Inserting data for Adult Weight Screening and Follow-Up
--

--
--     Inserting data for Weight Assessment and Counseling for Children and Adolescents
--

--
--    Inserting data for Influenza Immunization for Patients >= 50 Years Old
--

--
--    Inserting data for Pneumonia Vaccination Status for Older Adults
--

--
--    Inserting data for Cancer Screening: Mammogram
--

--
--    Inserting data for Cancer Screening: Pap Smear
--

--
--    Inserting data for Cancer Screening: Colon Cancer Screening
--

--
--    Inserting data for Cancer Screening: Prostate Cancer Screening
--

-- --------------------------------------------------------

--
-- Table structure for table `clinical_rules`
--

DROP TABLE IF EXISTS `clinical_rules`;
CREATE TABLE `clinical_rules` (
  `id` varchar(31) NOT NULL DEFAULT '' COMMENT 'Unique and maps to list_options list clinical_rules',
  `pid` bigint(20) NOT NULL DEFAULT '0' COMMENT '0 is default for all patients, while > 0 is id from patient_data table',
  `active_alert_flag` tinyint(1) COMMENT 'Active Alert Widget Module flag - note not yet utilized',
  `passive_alert_flag` tinyint(1) COMMENT 'Passive Alert Widget Module flag',
  `cqm_flag` tinyint(1) COMMENT 'Clinical Quality Measure flag (unable to customize per patient)',
  `cqm_2011_flag` tinyint(1) COMMENT '2011 Clinical Quality Measure flag (unable to customize per patient)',
  `cqm_2014_flag` tinyint(1) COMMENT '2014 Clinical Quality Measure flag (unable to customize per patient)',
  `cqm_nqf_code` varchar(10) NOT NULL default '' COMMENT 'Clinical Quality Measure NQF identifier',
  `cqm_pqri_code` varchar(10) NOT NULL default '' COMMENT 'Clinical Quality Measure PQRI identifier',
  `amc_flag` tinyint(1) COMMENT 'Automated Measure Calculation flag (unable to customize per patient)',
  `amc_2011_flag` tinyint(1) COMMENT '2011 Automated Measure Calculation flag for (unable to customize per patient)',
  `amc_2014_flag` tinyint(1) COMMENT '2014 Automated Measure Calculation flag for (unable to customize per patient)',
  `amc_2015_flag` TINYINT(1) NULL DEFAULT NULL COMMENT '2015 Automated Measure Calculation flag for (unable to customize per patient)',
  `amc_code` varchar(10) NOT NULL default '' COMMENT 'Automated Measure Calculation identifier (MU rule)',
  `amc_code_2014` varchar(30) NOT NULL default '' COMMENT 'Automated Measure Calculation 2014 identifier (MU rule)',
  `amc_code_2015` VARCHAR(30) NOT NULL DEFAULT '' COMMENT 'Automated Measure Calculation 2014 identifier (MU rule)',
  `amc_2014_stage1_flag` tinyint(1) COMMENT '2014 Stage 1 - Automated Measure Calculation flag for (unable to customize per patient)',
  `amc_2014_stage2_flag` tinyint(1) COMMENT '2014 Stage 2 - Automated Measure Calculation flag for (unable to customize per patient)',
  `patient_reminder_flag` tinyint(1) COMMENT 'Clinical Reminder Module flag',
  `bibliographic_citation` VARCHAR(255) NOT NULL DEFAULT '',
  `developer` VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'Clinical Rule Developer',
  `funding_source` VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'Clinical Rule Funding Source',
  `release_version` VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'Clinical Rule Release Version',
  `web_reference` VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'Clinical Rule Web Reference',
  `linked_referential_cds` VARCHAR(50) NOT NULL DEFAULT '',
  `access_control` VARCHAR(255) NOT NULL DEFAULT 'patients:med' COMMENT 'ACO link for access control',
  `patient_dob_usage` TEXT COMMENT 'Description of how patient DOB is used by this rule',
  `patient_ethnicity_usage` TEXT COMMENT 'Description of how patient ethnicity is used by this rule',
  `patient_health_status_usage` TEXT COMMENT 'Description of how patient health status assessments are used by this rule',
  `patient_gender_identity_usage` TEXT COMMENT 'Description of how patient gender identity information is used by this rule',
  `patient_language_usage` TEXT COMMENT 'Description of how patient language information is used by this rule',
  `patient_race_usage` TEXT COMMENT 'Description of how patient race information is used by this rule',
  `patient_sex_usage` TEXT COMMENT 'Description of how patient birth sex information is used by this rule',
  `patient_sexual_orientation_usage` TEXT COMMENT 'Description of how patient sexual orientation is used by this rule',
  `patient_sodh_usage` TEXT COMMENT 'Description of how patient social determinants of health are used by this rule',
  PRIMARY KEY  (`id`,`pid`)
) ENGINE=InnoDB;

--
-- Inserting data for Automated Measure Calculation (AMC) rules
--
--   Inserting data for MU 170.302(c) Maintain an up-to-date problem list of current and active diagnoses (2014-MU-AMC:170.314(g)(1)/(2)–4)
--

--
--   Inserting data for MU 170.302(d) Maintain active medication list
--

--
--   Inserting data for MU 170.302(e) Maintain active medication allergy list
--

--
--   Inserting data for MU 170.302(f) Record and chart changes in vital signs
--

--
--   Inserting data for MU 170.302(g) Record smoking status for patients 13 years old or older
--

--
--   Inserting data for MU 170.302(h) Incorporate clinical lab-test results into certified EHR technology as structured data
--

--
--   Inserting data for MU 170.302(j) The EP, eligible hospital or CAH who receives a patient from another
--     setting of care or provider of care or believes an encounter is relevant
--     should perform medication reconciliation
--

--
--   Inserting data for MU 170.302(m) Use certified EHR technology to identify patient-specific education resources
--     and provide those resources to the patient if appropriate
--

--
--   Inserting data for MU 170.304(a) Use CPOE for medication orders directly entered by any licensed healthcare
--     professional who can enter orders into the medical record per state, local
--     and professional guidelines
--

--
--   Inserting data for MU 170.304(b) Generate and transmit permissible prescriptions electronically (eRx)
--

--
--   Inserting data for MU 170.304(c) Record demographics
--

--
--   Inserting data for MU 170.304(d) Send reminders to patients per patient preference for preventive/follow up care
--

--
--   Inserting data for MU 170.304(f) Provide patients with an electronic copy of their health information
--               (including diagnostic test results, problem list, medication lists,
--               medication allergies), upon request
--

--
--   Inserting data for MU 170.304(g) Provide patients with timely electronic access to their health information
--              (including lab results, problem list, medication lists, medication allergies)
--              within four business days of the information being available to the EP
--

--
--   Inserting data for MU 170.304(h) Provide clinical summaries for patients for each office visit
--

--
--   Inserting data for MU 170.304(i) The EP, eligible hospital or CAH who transitions their patient to
--               another setting of care or provider of care or refers their patient to
--               another provider of care should provide summary of care record for
--               each transition of care or referral
--

--
-- Inserting data for Clinical Quality Measure (CQM) rules
--
--   Inserting data for NQF 0013 Hypertension: Blood Pressure Measurement
--

--
--   Inserting data for NQF 0028a Tobacco Use Assessment
--

--
--   Inserting data for NQF 0028b Tobacco Cessation Intervention
--

--
--   Inserting data for NQF 0421 (PQRI 128) Adult Weight Screening and Follow-Up
--

--
--   Inserting data for NQF 0024 Weight Assessment and Counseling for Children and Adolescents
--

--
--   Inserting data for NQF 0041 (PQRI 110) Influenza Immunization for Patients >= 50 Years Old
--

--
--   Inserting data for NQF 0038 Childhood immunization Status
--


--
--   Inserting data for NQF 0043 (PQRI 111) Pneumonia Vaccination Status for Older Adults
--

--
--   Inserting data for NQF 0055 (PQRI 117) Diabetes: Eye Exam
--

--
--   Inserting data for NQF 0056 (PQRI 163) Diabetes: Foot Exam
--

--
--   Inserting data for NQF 0059 (PQRI 1) Diabetes: HbA1c Poor Control
--

--
--   Inserting data for NQF 0061 (PQRI 3) Diabetes: Blood Pressure Management
--

--
--   Inserting data for NQF 0064 (PQRI 2) Diabetes: LDL Management & Control
--

--
--   Inserting data for NQF 0002 Rule Children Pharyngitis
--

--
--   Inserting data for NQF 0101 Rule Fall Screening
--

--
--   Inserting data for NQF 0384 Rule Pain Intensity
--

--
--   Inserting data for NQF 0038 Rule Child Immunization Status
--

--
--   Inserting data for NQF 0028 Rule Tobacco Use
--

--
-- Inserting data for Standard clinical rules
--
--   Inserting data for Hypertension: Blood Pressure Measurement
--

--
--   Inserting data for Tobacco Use Assessment
--

--
--   Inserting data for Tobacco Cessation Intervention
--

--
--   Inserting data for Adult Weight Screening and Follow-Up
--

--
--   Inserting data for Weight Assessment and Counseling for Children and Adolescents
--

--
--   Inserting data for Influenza Immunization for Patients >= 50 Years Old
--

--
--   Inserting data for Pneumonia Vaccination Status for Older Adults
--

--
--   Inserting data for Diabetes: Hemoglobin A1C
--

--
--   Inserting data for Diabetes: Urine Microalbumin
--

--
--   Inserting data for Diabetes: Eye Exam
--

--
--   Inserting data for Diabetes: Foot Exam
--

--
--   Inserting data for Cancer Screening: Mammogram
--

--
--   Inserting data for Cancer Screening: Pap Smear
--

--
--   Inserting data for Cancer Screening: Colon Cancer Screening
--

--
--   Inserting data for Cancer Screening: Prostate Cancer Screening
--

--
-- Inserting data for Rules to specifically demonstrate passing of NIST criteria
--
--   Inserting data for Coumadin Management - INR Monitoring
--

--
-- Inserting data for Rule to specifically demonstrate MU2 for CDR engine
--

--
-- Inserting data for MU2 AMC rules
--

-- 2015 AMC Rules
-- --------------------------------------------------------

--
-- Table structure for table `clinical_rules_log
--

DROP TABLE IF EXISTS `clinical_rules_log`;
CREATE TABLE `clinical_rules_log` (
  `id` bigint(20) NOT NULL auto_increment,
  `date` datetime DEFAULT NULL,
  `pid` bigint(20) NOT NULL DEFAULT '0',
  `uid` bigint(20) NOT NULL DEFAULT '0',
  `category` VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'An example category is clinical_reminder_widget',
  `value` TEXT,
  `new_value` TEXT,
  `facility_id` INT(11) DEFAULT '0' COMMENT 'facility where the rule was executed, 0 if unknown',
  PRIMARY KEY (`id`),
  KEY `pid` (`pid`),
  KEY `uid` (`uid`),
  KEY `category` (`category`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `codes`
--

DROP TABLE IF EXISTS `codes`;
CREATE TABLE `codes` (
  `id` int(11) NOT NULL auto_increment,
  `code_text` text,
  `code_text_short` text,
  `code` varchar(25) NOT NULL default '',
  `code_type` smallint(6) default NULL,
  `modifier` varchar(12) NOT NULL default '',
  `units` int(11) default NULL,
  `fee` decimal(12,2) default NULL,
  `superbill` varchar(31) NOT NULL default '',
  `related_code` varchar(255) NOT NULL default '',
  `taxrates` varchar(255) NOT NULL default '',
  `cyp_factor` float NOT NULL DEFAULT 0 COMMENT 'quantity representing a years supply',
  `active` TINYINT(1) DEFAULT 1 COMMENT '0 = inactive, 1 = active',
  `reportable` TINYINT(1) DEFAULT 0 COMMENT '0 = non-reportable, 1 = reportable',
  `financial_reporting` TINYINT(1) DEFAULT 0 COMMENT '0 = negative, 1 = considered important code in financial reporting',
  `revenue_code` varchar(6) NOT NULL default '' COMMENT 'Item revenue code',
  PRIMARY KEY  (`id`),
  KEY `code` (`code`),
  KEY `code_type` (`code_type`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------
--
-- Table structure for table `contact`
--
DROP TABLE IF EXISTS `contact`;
CREATE TABLE `contact` (
   `id` BIGINT(20) NOT NULL auto_increment,
   `foreign_table_name` VARCHAR(255) NOT NULL DEFAULT '',
   `foreign_id` BIGINT(20) NOT NULL DEFAULT '0',
   PRIMARY KEY (`id`),
   KEY (`foreign_id`)
) ENGINE = InnoDB;

-- --------------------------------------------------------
--
-- Table structure for table `contact_address`
--
DROP TABLE IF EXISTS `contact_address`;
    CREATE TABLE `contact_address` (
    `id` BIGINT(20) NOT NULL auto_increment,
    `contact_id` BIGINT(20) NOT NULL,
    `address_id` BIGINT(20) NOT NULL,
    `priority` INT(11) NULL,
    `type` VARCHAR(255) NULL COMMENT 'FK to list_options.option_id for list_id address-types',
    `use` VARCHAR(255) NULL COMMENT 'FK to list_options.option_id for list_id address-uses',
    `notes` TINYTEXT,
    `status` CHAR(1) NULL COMMENT 'A=active,I=inactive',
    `is_primary` CHAR(1) NULL COMMENT 'Y=yes,N=no',
    `period_start` DATETIME NULL COMMENT 'Date the address became active',
    `period_end` DATETIME NULL COMMENT 'Date the address became deactivated',
    `inactivated_reason` VARCHAR(45) NULL DEFAULT NULL COMMENT '[Values: Moved, Mail Returned, etc]',
    `created_date` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `created_by` BIGINT(20) DEFAULT NULL COMMENT 'users.id',
    `updated_date` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `updated_by` BIGINT(20) DEFAULT NULL COMMENT 'users.id',
    PRIMARY KEY (`id`),
    KEY (`contact_id`),
    KEY (`address_id`),
    KEY contact_address_idx (`contact_id`,`address_id`)
) ENGINE = InnoDB ;


DROP TABLE IF EXISTS `contact_telecom`;
CREATE TABLE `contact_telecom` (
    `id` BIGINT(20) NOT NULL auto_increment,
    `contact_id` BIGINT(20) NOT NULL,
    `rank` INT(11) NULL COMMENT 'Specify preferred order of use (1 = highest)',
    `system` VARCHAR(255) NULL
    	COMMENT 'FK to list_options.option_id for list_id telecom_systems [phone, fax, email, pager, url, sms, other]',
    `use` VARCHAR(255) NULL
    	COMMENT 'FK to list_options.option_id for list_id telecom_uses [home, work, temp, old, mobile]',
    `value` varchar(255) default NULL,
    `status` CHAR(1) NULL COMMENT 'A=active,I=inactive',
    `is_primary` CHAR(1) NULL COMMENT 'Y=yes,N=no',
    `notes` TINYTEXT,
    `period_start` DATETIME NULL COMMENT 'Date the telecom became active',
    `period_end` DATETIME NULL COMMENT 'Date the telecom became deactivated',
    `inactivated_reason` VARCHAR(45) DEFAULT NULL COMMENT '[Values: ???, etc]',
    `created_date` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `created_by` BIGINT(20) DEFAULT NULL COMMENT 'users.id',
    `updated_date` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `updated_by` BIGINT(20) DEFAULT NULL COMMENT 'users.id',
   PRIMARY KEY (`id`),
    KEY (`contact_id`)
) ENGINE = InnoDB ;

DROP TABLE IF EXISTS `person`;
CREATE TABLE `person` (
    `id` BIGINT(20) NOT NULL AUTO_INCREMENT,
    `uuid` BINARY(16) DEFAULT NULL,
    `title` VARCHAR(31) DEFAULT NULL COMMENT 'Mr., Mrs., Dr., etc.',
    `first_name` VARCHAR(63) DEFAULT NULL,
    `middle_name` VARCHAR(63) DEFAULT NULL,
    `last_name` VARCHAR(63) DEFAULT NULL,
    `preferred_name` VARCHAR(63) DEFAULT NULL COMMENT 'Name person prefers to be called',
    `gender` VARCHAR(31) DEFAULT NULL,
    `birth_date` DATE DEFAULT NULL,
    `death_date` DATE DEFAULT NULL,
    `marital_status` VARCHAR(31) DEFAULT NULL,
    `race` VARCHAR(63) DEFAULT NULL,
    `ethnicity` VARCHAR(63) DEFAULT NULL,
    `preferred_language` VARCHAR(63) DEFAULT NULL COMMENT 'ISO 639-1 code',
    `communication` VARCHAR(254) DEFAULT NULL COMMENT 'Communication preferences/needs',
    `ssn` VARCHAR(31) DEFAULT NULL COMMENT 'Should be encrypted in application',
    `active` TINYINT(1) DEFAULT 1 COMMENT '1=active, 0=inactive',
    `inactive_reason` VARCHAR(255) DEFAULT NULL,
    `inactive_date` DATETIME DEFAULT NULL,
    `notes` TEXT,
    `created_date` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `created_by` BIGINT(20) DEFAULT NULL COMMENT 'users.id',
    `updated_date` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `updated_by` BIGINT(20) DEFAULT NULL COMMENT 'users.id',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uuid` (`uuid`),
    KEY `idx_person_name` (`last_name`, `first_name`),
    KEY `idx_person_dob` (`birth_date`),
    KEY `idx_person_search` (`last_name`, `first_name`, `birth_date`),
    KEY `idx_person_active` (`active`)
) ENGINE=InnoDB COMMENT='Core person demographics - contact info in contact_telecom';

DROP TABLE IF EXISTS `contact_relation`;
CREATE TABLE `contact_relation` (
    `id`  BIGINT(20) NOT NULL auto_increment,
    `contact_id`  BIGINT(20) NOT NULL,
    `target_table`  VARCHAR(255) NOT NULL DEFAULT '',
    `target_id`  BIGINT(20) NOT NULL,
    `active` BOOLEAN DEFAULT TRUE,
    `role` VARCHAR(63)  DEFAULT NULL,
    `relationship` VARCHAR(63)  DEFAULT NULL,
    `contact_priority` INT DEFAULT 1 COMMENT '1=highest priority',
    `is_primary_contact` BOOLEAN DEFAULT FALSE,
    `is_emergency_contact` BOOLEAN DEFAULT FALSE,
    `can_make_medical_decisions` BOOLEAN DEFAULT FALSE,
    `can_receive_medical_info` BOOLEAN DEFAULT FALSE,
    `start_date` DATETIME DEFAULT NULL,
    `end_date` DATETIME DEFAULT NULL,
    `notes` TEXT,
    `created_date` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `created_by` BIGINT(20) DEFAULT NULL COMMENT 'users.id',
    `updated_date` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `updated_by` BIGINT(20) DEFAULT NULL COMMENT 'users.id',
   PRIMARY KEY (`id`),
   KEY (`contact_id`),
   INDEX idx_contact_target_table (target_table, target_id)
) ENGINE = InnoDB;

DROP TABLE IF EXISTS `person_patient_link`;
CREATE TABLE `person_patient_link` (
    `id` BIGINT(20) NOT NULL AUTO_INCREMENT,
    `person_id` BIGINT(20) NOT NULL COMMENT 'FK to person.id',
    `patient_id` BIGINT(20) NOT NULL COMMENT 'FK to patient_data.id',
    `linked_date` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'When the link was created',
    `linked_by` BIGINT(20) DEFAULT NULL COMMENT 'FK to users.id - who created the link',
    `link_method` VARCHAR(50) DEFAULT 'manual' COMMENT 'How link was created: manual, auto_detected, migrated, import',
    `notes` TEXT COMMENT 'Optional notes about why/how they were linked',
    `active` TINYINT(1) NOT NULL DEFAULT 1 COMMENT 'Whether link is active (allows soft delete)',
    PRIMARY KEY (`id`),
    UNIQUE KEY `unique_active_link` (`person_id`, `patient_id`, `active`),
    KEY `idx_ppl_person` (`person_id`),
    KEY `idx_ppl_patient` (`patient_id`),
    KEY `idx_ppl_active` (`active`),
    KEY `idx_ppl_linked_date` (`linked_date`),
    KEY `idx_ppl_method` (`link_method`)
) ENGINE=InnoDB COMMENT='Links person records to patient_data records when person becomes patient';
-- --------------------------------------------------------

--
-- Table structure for table `syndromic_surveillance`
--

DROP TABLE IF EXISTS `syndromic_surveillance`;
CREATE TABLE `syndromic_surveillance` (
  `id` bigint(20) NOT NULL auto_increment,
  `lists_id` bigint(20) NOT NULL,
  `submission_date` datetime NOT NULL,
  `filename` varchar(255) NOT NULL default '',
  PRIMARY KEY  (`id`),
  KEY (`lists_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `dated_reminders`
--

DROP TABLE IF EXISTS `dated_reminders`;
CREATE TABLE `dated_reminders` (
  `dr_id` int(11) NOT NULL AUTO_INCREMENT,
  `dr_from_ID` int(11) NOT NULL,
  `dr_message_text` varchar(160) NOT NULL,
  `dr_message_sent_date` datetime NOT NULL,
  `dr_message_due_date` date NOT NULL,
  `pid` bigint(20) NOT NULL,
  `message_priority` tinyint(1) NOT NULL,
  `message_processed` tinyint(1) NOT NULL DEFAULT '0',
  `processed_date` timestamp NULL DEFAULT NULL,
  `dr_processed_by` int(11) NOT NULL,
  PRIMARY KEY (`dr_id`),
  KEY `dr_from_ID` (`dr_from_ID`,`dr_message_due_date`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `dated_reminders_link`
--

DROP TABLE IF EXISTS `dated_reminders_link`;
CREATE TABLE `dated_reminders_link` (
  `dr_link_id` int(11) NOT NULL AUTO_INCREMENT,
  `dr_id` int(11) NOT NULL,
  `to_id` int(11) NOT NULL,
  PRIMARY KEY (`dr_link_id`),
  KEY `to_id` (`to_id`),
  KEY `dr_id` (`dr_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `direct_message_log`
--

DROP TABLE IF EXISTS `direct_message_log`;
CREATE TABLE `direct_message_log` (
  `id` bigint(20) NOT NULL auto_increment,
  `msg_type` char(1) NOT NULL COMMENT 'S=sent,R=received',
  `msg_id` varchar(127) NOT NULL,
  `sender` varchar(255) NOT NULL,
  `recipient` varchar(255) NOT NULL,
  `create_ts` timestamp NOT NULL default CURRENT_TIMESTAMP,
  `status` char(1) NOT NULL COMMENT 'Q=queued,D=dispatched,R=received,F=failed',
  `status_info` varchar(511) default NULL,
  `status_ts` timestamp NULL default NULL,
  `patient_id` bigint(20) default NULL,
  `user_id` bigint(20) default NULL,
  PRIMARY KEY  (`id`),
  KEY `msg_id` (`msg_id`),
  KEY `patient_id` (`patient_id`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `documents`
--

DROP TABLE IF EXISTS `documents`;
CREATE TABLE `documents` (
  `id` int(11) NOT NULL default '0',
  `uuid` binary(16) DEFAULT NULL,
  `type` enum('file_url','blob','web_url') default NULL,
  `size` int(11) default NULL,
  `date` datetime default NULL,
  `date_expires` datetime default NULL,
  `url` varchar(255) default NULL,
  `thumb_url` varchar(255) default NULL,
  `mimetype` varchar(255) default NULL,
  `pages` int(11) default NULL,
  `owner` int(11) default NULL,
  `revision` timestamp NOT NULL,
  `foreign_id` bigint(20) default NULL,
  `docdate` date default NULL,
  `hash` varchar(255) DEFAULT NULL,
  `list_id` bigint(20) NOT NULL default '0',
  `name` varchar(255) DEFAULT NULL,
  `drive_uuid` binary(16) DEFAULT NULL,
  `couch_docid` VARCHAR(100) DEFAULT NULL,
  `couch_revid` VARCHAR(100) DEFAULT NULL,
  `storagemethod` TINYINT(4) NOT NULL DEFAULT '0' COMMENT '0->Harddisk,1->CouchDB',
  `path_depth` TINYINT DEFAULT '1' COMMENT 'Depth of path to use in url to find document. Not applicable for CouchDB.',
  `imported` TINYINT DEFAULT 0 NULL COMMENT 'Parsing status for CCR/CCD/CCDA importing',
  `encounter_id` bigint(20) NOT NULL DEFAULT '0' COMMENT 'Encounter id if tagged',
  `encounter_check` TINYINT(1) NOT NULL DEFAULT '0' COMMENT 'If encounter is created while tagging',
  `audit_master_approval_status` TINYINT NOT NULL DEFAULT 1 COMMENT 'approval_status from audit_master table',
  `audit_master_id` int(11) default NULL,
  `documentationOf` varchar(255) DEFAULT NULL,
  `encrypted` TINYINT(4) NOT NULL DEFAULT '0' COMMENT '0->No,1->Yes',
  `document_data` MEDIUMTEXT,
  `deleted` tinyint(1) NOT NULL DEFAULT '0',
  `foreign_reference_id` bigint(20) default NULL,
  `foreign_reference_table` VARCHAR(40) default NULL,
  PRIMARY KEY  (`id`),
  UNIQUE KEY `drive_uuid` (`drive_uuid`),
  UNIQUE KEY `uuid` (`uuid`),
  KEY `revision` (`revision`),
  KEY `foreign_id` (`foreign_id`),
  KEY `foreign_reference` (`foreign_reference_id`, `foreign_reference_table`),
  KEY `owner` (`owner`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `documents_legal_detail`
--

DROP TABLE IF EXISTS `documents_legal_detail`;
CREATE TABLE `documents_legal_detail` (
  `dld_id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `dld_pid` int(10) unsigned DEFAULT NULL,
  `dld_facility` int(10) unsigned DEFAULT NULL,
  `dld_provider` int(10) unsigned DEFAULT NULL,
  `dld_encounter` int(10) unsigned DEFAULT NULL,
  `dld_master_docid` int(10) unsigned NOT NULL,
  `dld_signed` smallint(5) unsigned NOT NULL COMMENT '0-Not Signed or Cannot Sign(Layout),1-Signed,2-Ready to sign,3-Denied(Pat Regi),4-Patient Upload,10-Save(Layout)',
  `dld_signed_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `dld_filepath` varchar(75) DEFAULT NULL,
  `dld_filename` varchar(45) NOT NULL,
  `dld_signing_person` varchar(50) NOT NULL,
  `dld_sign_level` int(11) NOT NULL COMMENT 'Sign flow level',
  `dld_content` varchar(50) NOT NULL COMMENT 'Layout sign position',
  `dld_file_for_pdf_generation` blob NOT NULL COMMENT 'The filled details in the fdf file is stored here.Patient Registration Screen',
  `dld_denial_reason` longtext,
  `dld_moved` tinyint(4) NOT NULL DEFAULT '0',
  `dld_patient_comments` text COMMENT 'Patient comments stored here',
  PRIMARY KEY (`dld_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `documents_legal_master`
--

DROP TABLE IF EXISTS `documents_legal_master`;
CREATE TABLE `documents_legal_master` (
  `dlm_category` int(10) unsigned DEFAULT NULL,
  `dlm_subcategory` int(10) unsigned DEFAULT NULL,
  `dlm_document_id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `dlm_document_name` varchar(75) NOT NULL,
  `dlm_filepath` varchar(75) NOT NULL,
  `dlm_facility` int(10) unsigned DEFAULT NULL,
  `dlm_provider` int(10) unsigned DEFAULT NULL,
  `dlm_sign_height` double NOT NULL,
  `dlm_sign_width` double NOT NULL,
  `dlm_filename` varchar(45) NOT NULL,
  `dlm_effective_date` datetime NOT NULL,
  `dlm_version` int(10) unsigned NOT NULL,
  `content` varchar(255) NOT NULL,
  `dlm_savedsign` varchar(255) DEFAULT NULL COMMENT '0-Yes 1-No',
  `dlm_review` varchar(255) DEFAULT NULL COMMENT '0-Yes 1-No',
  `dlm_upload_type` tinyint(4) DEFAULT '0' COMMENT '0-Provider Uploaded,1-Patient Uploaded',
  PRIMARY KEY (`dlm_document_id`)
) ENGINE=InnoDB COMMENT='List of Master Docs to be signed' AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `documents_legal_categories`
--

DROP TABLE IF EXISTS `documents_legal_categories`;
CREATE TABLE `documents_legal_categories` (
  `dlc_id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `dlc_category_type` int(10) unsigned NOT NULL COMMENT '1 category 2 subcategory',
  `dlc_category_name` varchar(45) NOT NULL,
  `dlc_category_parent` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`dlc_id`)
) ENGINE=InnoDB AUTO_INCREMENT=7;

--
-- Inserting data for table `documents_legal_categories`
--

--
-- Table structure for table `drug_inventory`
--

DROP TABLE IF EXISTS `drug_inventory`;
CREATE TABLE `drug_inventory` (
  `inventory_id` int(11) NOT NULL auto_increment,
  `drug_id` int(11) NOT NULL,
  `lot_number` varchar(20) default NULL,
  `expiration` date default NULL,
  `manufacturer` varchar(255) default NULL,
  `on_hand` int(11) NOT NULL default '0',
  `warehouse_id` varchar(31) NOT NULL DEFAULT '',
  `vendor_id` bigint(20) NOT NULL DEFAULT 0,
  `last_notify` date NULL,
  `destroy_date` date default NULL,
  `destroy_method` varchar(255) default NULL,
  `destroy_witness` varchar(255) default NULL,
  `destroy_notes` varchar(255) default NULL,
  PRIMARY KEY  (`inventory_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `drug_sales`
--

DROP TABLE IF EXISTS `drug_sales`;
CREATE TABLE `drug_sales` (
  `uuid` binary(16) DEFAULT NULL COMMENT 'UUID for this drug sales record, for data exchange purposes',
  `sale_id` int(11) NOT NULL auto_increment,
  `drug_id` int(11) NOT NULL,
  `inventory_id` int(11) NOT NULL,
  `prescription_id` int(11) NOT NULL default '0',
  `pid` bigint(20) NOT NULL default '0',
  `encounter` int(11) NOT NULL default '0',
  `user` varchar(255) default NULL,
  `sale_date` date NOT NULL,
  `quantity` int(11) NOT NULL default '0',
  `fee` decimal(12,2) NOT NULL default '0.00',
  `billed` tinyint(1) NOT NULL default '0' COMMENT 'indicates if the sale is posted to accounting',
  `xfer_inventory_id` int(11) NOT NULL DEFAULT 0,
  `distributor_id` bigint(20) NOT NULL DEFAULT 0 COMMENT 'references users.id',
  `notes` varchar(255) NOT NULL DEFAULT '',
  `bill_date` datetime default NULL,
  `pricelevel` varchar(31) default '',
  `selector` varchar(255) default '' comment 'references drug_templates.selector',
  `trans_type` tinyint NOT NULL DEFAULT 1 COMMENT '1=sale, 2=purchase, 3=return, 4=transfer, 5=adjustment',
  `chargecat` varchar(31) default '',
  `pharmacy_supply_type` VARCHAR(50) DEFAULT NULL COMMENT 'fk to list_options.option_id where list_id=pharmacy_supply_type to indicate type of dispensing first order, refil, emergency, partial order, etc',
  `last_updated` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `date_created` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_by` BIGINT(20) DEFAULT NULL COMMENT 'fk to users.id for user that last updated this entry',
  `created_by` BIGINT(20) DEFAULT NULL COMMENT 'fk to users.id for user that created this entry',
  PRIMARY KEY  (`sale_id`),
  UNIQUE INDEX `uuid` (`uuid`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `drug_templates`
--

DROP TABLE IF EXISTS `drug_templates`;
CREATE TABLE `drug_templates` (
  `drug_id` int(11) NOT NULL,
  `selector` varchar(255) NOT NULL default '',
  `dosage` varchar(10) default NULL,
  `period` int(11) NOT NULL default '0',
  `quantity` int(11) NOT NULL default '0',
  `refills` int(11) NOT NULL default '0',
  `taxrates` varchar(255) default NULL,
  `pkgqty` float NOT NULL DEFAULT 1.0 COMMENT 'Number of product items per template item',
  PRIMARY KEY  (`drug_id`,`selector`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `drugs`
--

DROP TABLE IF EXISTS `drugs`;
CREATE TABLE `drugs` (
  `drug_id` int(11) NOT NULL auto_increment,
  `uuid` binary(16) DEFAULT NULL,
  `name` varchar(255) NOT NULL DEFAULT '',
  `ndc_number` varchar(20) NOT NULL DEFAULT '',
  `on_order` int(11) NOT NULL default '0',
  `reorder_point` float NOT NULL DEFAULT 0.0,
  `max_level` float NOT NULL DEFAULT 0.0,
  `last_notify` date NULL,
  `reactions` text,
  `form` varchar(31) NOT NULL default '0',
  `size` varchar(25) NOT NULL default '',
  `unit` varchar(31) NOT NULL default '0',
  `route` varchar(31) NOT NULL default '0',
  `substitute` int(11) NOT NULL default '0',
  `related_code` varchar(255) NOT NULL DEFAULT '' COMMENT 'may reference a related codes.code',
  `cyp_factor` float NOT NULL DEFAULT 0 COMMENT 'quantity representing a years supply',
  `active` TINYINT(1) DEFAULT 1 COMMENT '0 = inactive, 1 = active',
  `allow_combining` tinyint(1) NOT NULL DEFAULT 0 COMMENT '1 = allow filling an order from multiple lots',
  `allow_multiple`  tinyint(1) NOT NULL DEFAULT 1 COMMENT '1 = allow multiple lots at one warehouse',
  `drug_code` varchar(25) NULL,
  `consumable` tinyint(1) NOT NULL DEFAULT 0 COMMENT '1 = will not show on the fee sheet',
  `dispensable` tinyint(1) NOT NULL DEFAULT 1 COMMENT '0 = pharmacy elsewhere, 1 = dispensed here',
  `date_created` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `last_updated` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY  (`drug_id`),
  UNIQUE KEY `uuid` (`uuid`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `edi_sequences`
--

DROP TABLE IF EXISTS `edi_sequences`;
CREATE TABLE `edi_sequences` (
  `id` int(9) unsigned NOT NULL default '0'
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `eligibility_verification`
--

DROP TABLE IF EXISTS `eligibility_verification`;
CREATE TABLE `eligibility_verification` (
  `verification_id` bigint(20) NOT NULL auto_increment,
  `response_id` varchar(32) default NULL,
  `insurance_id` bigint(20) default NULL,
  `eligibility_check_date` datetime default NULL,
  `copay` int(11) default NULL,
  `deductible` int(11) default NULL,
  `deductiblemet` enum('Y','N') default 'Y',
  `create_date` date default NULL,
  PRIMARY KEY  (`verification_id`),
  KEY `insurance_id` (`insurance_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `email_queue`
--
DROP TABLE IF EXISTS `email_queue`;
CREATE TABLE `email_queue` (
  `id` bigint NOT NULL auto_increment,
  `sender` varchar(255) DEFAULT '',
  `recipient` varchar(255) DEFAULT '',
  `subject` varchar(255) DEFAULT '',
  `body` text,
  `datetime_queued` datetime default NULL,
  `sent` tinyint DEFAULT 0,
  `datetime_sent` datetime default NULL,
  `error` tinyint DEFAULT 0,
  `error_message` text,
  `datetime_error` datetime default NULL,
  `template_name` VARCHAR(255) DEFAULT NULL COMMENT 'The folder prefix and base filename (w/o extension) of the twig template file to use for this email',
PRIMARY KEY (`id`),
KEY `sent` (`sent`)
) ENGINE=InnoDb AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `employer_data`
--

DROP TABLE IF EXISTS `employer_data`;
CREATE TABLE `employer_data` (
  `id` bigint(20) NOT NULL auto_increment,
  `uuid` binary(16) DEFAULT NULL COMMENT 'UUID for this employer record, for data exchange purposes',
  `name` varchar(255) default NULL,
  `street` varchar(255) default NULL,
  `street_line_2` TINYTEXT,
  `postal_code` varchar(255) default NULL,
  `city` varchar(255) default NULL,
  `state` varchar(255) default NULL,
  `country` varchar(255) default NULL,
  `date` datetime default NULL,
  `pid` bigint(20) NOT NULL default '0',
  `start_date` datetime DEFAULT NULL COMMENT 'Employment start date for patient',
  `end_date` datetime DEFAULT NULL COMMENT 'Employment end date for patient',
  `occupation` longtext COMMENT 'Employment Occupation fk to list_options.option_id where list_id=OccupationODH',
  `industry` text COMMENT 'Employment Industry fk to list_options.option_id where list_id=IndustryODH',
  `created_by` int DEFAULT NULL COMMENT 'fk to users.id for the user that entered in the employer data',
  PRIMARY KEY  (`id`),
  KEY `pid` (`pid`),
  UNIQUE KEY `uuid_unique` (`uuid`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `enc_category_map`
--
--   Mapping of rule encounter categories to category ids
--   from the event category in openemr_postcalendar_categories
--

DROP TABLE IF EXISTS `enc_category_map`;
CREATE TABLE `enc_category_map` (
  `rule_enc_id` varchar(31) NOT NULL DEFAULT '' COMMENT 'encounter id from rule_enc_types list in list_options',
  `main_cat_id` int(11) NOT NULL DEFAULT 0 COMMENT 'category id from event category in openemr_postcalendar_categories',
  KEY  (`rule_enc_id`,`main_cat_id`)
) ENGINE=InnoDB;

--
-- Inserting data for table `enc_category_map`
--

-- --------------------------------------------------------

--
-- Table structure for table `erx_ttl_touch`
--   Store records last update per patient data process
--

DROP TABLE IF EXISTS `erx_ttl_touch`;
CREATE  TABLE `erx_ttl_touch` (
  `patient_id` BIGINT(20) UNSIGNED NOT NULL COMMENT 'Patient record Id' ,
  `process` ENUM('allergies','medications') NOT NULL COMMENT 'NewCrop eRx SOAP process' ,
  `updated` DATETIME NOT NULL COMMENT 'Date and time of last process update for patient' ,
  PRIMARY KEY (`patient_id`, `process`)
) ENGINE = InnoDB COMMENT = 'Store records last update per patient data process';

-- --------------------------------------------------------

--
-- Table structure for table `erx_rx_log`
--

DROP TABLE IF EXISTS `erx_rx_log`;
CREATE TABLE `erx_rx_log` (
 `id` int(20) NOT NULL AUTO_INCREMENT,
 `prescription_id` int(6) NOT NULL,
 `date` varchar(25) NOT NULL,
 `time` varchar(15) NOT NULL,
 `code` int(6) NOT NULL,
 `status` text,
 `message_id` varchar(100) DEFAULT NULL,
 `read` int(1) DEFAULT NULL,
 PRIMARY KEY (`id`)
  ) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `erx_narcotics`
--

DROP TABLE IF EXISTS `erx_narcotics`;
CREATE TABLE `erx_narcotics` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `drug` varchar(255) NOT NULL,
  `dea_number` varchar(5) NOT NULL,
  `csa_sch` varchar(2) NOT NULL,
  `narc` varchar(2) NOT NULL,
  `other_names` varchar(255) NOT NULL,
   PRIMARY KEY (`id`)
  ) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `standardized_tables_track`
--

DROP TABLE IF EXISTS `standardized_tables_track`;
CREATE TABLE `standardized_tables_track` (
  `id` int(11) NOT NULL auto_increment,
  `imported_date` datetime default NULL,
  `name` varchar(255) NOT NULL default '' COMMENT 'name of standardized tables such as RXNORM',
  `revision_version` varchar(255) NOT NULL default '' COMMENT 'revision of standardized tables that were imported',
  `revision_date` datetime default NULL COMMENT 'revision of standardized tables that were imported',
  `file_checksum` varchar(32) NOT NULL DEFAULT '',
  PRIMARY KEY  (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `facility`
--

DROP TABLE IF EXISTS `facility`;
CREATE TABLE `facility` (
  `id` int(11) NOT NULL auto_increment,
  `uuid` binary(16) DEFAULT NULL,
  `name` varchar(255) default NULL,
  `phone` varchar(30) default NULL,
  `fax` varchar(30) default NULL,
  `street` varchar(255) default NULL,
  `city` varchar(255) default NULL,
  `state` varchar(50) default NULL,
  `postal_code` varchar(11) default NULL,
  `country_code` varchar(30) NOT NULL default '',
  `federal_ein` varchar(15) default NULL,
  `website` varchar(255) default NULL,
  `email` varchar(255) default NULL,
  `service_location` tinyint(1) NOT NULL default '1',
  `billing_location` tinyint(1) NOT NULL default '1',
  `accepts_assignment` tinyint(1) NOT NULL default '1',
  `pos_code` tinyint(4) default NULL,
  `x12_sender_id` varchar(25) default NULL,
  `attn` varchar(65) default NULL,
  `domain_identifier` varchar(60) default NULL,
  `facility_npi` varchar(15) default NULL,
  `facility_taxonomy` varchar(15) default NULL,
  `tax_id_type` VARCHAR(31) NOT NULL DEFAULT '',
  `color` VARCHAR(7) NOT NULL DEFAULT '',
  `primary_business_entity` INT(10) NOT NULL DEFAULT '1' COMMENT '0-Not Set as business entity 1-Set as business entity',
  `facility_code` VARCHAR(31) default NULL,
  `extra_validation` tinyint(1) NOT NULL DEFAULT '1',
  `mail_street` varchar(30) default NULL,
  `mail_street2` varchar(30) default NULL,
  `mail_city` varchar(50) default NULL,
  `mail_state` varchar(3) default NULL,
  `mail_zip` varchar(10) default NULL,
  `oid` VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'HIEs CCDA and FHIR an OID is required/wanted',
  `iban` varchar(50) default NULL,
  `info` TEXT,
  `weno_id` VARCHAR(10) DEFAULT NULL,
  `inactive` tinyint(1) NOT NULL DEFAULT '0',
  `date_created` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `last_updated` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `organization_type` VARCHAR(50) NOT NULL DEFAULT 'prov' COMMENT 'Organization type as defined by HL7 Value Set: OrganizationType',
  UNIQUE KEY `uuid` (`uuid`),
  PRIMARY KEY  (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=4;

--
-- Inserting data for table `facility`
--

-- --------------------------------------------------------

--
-- Table structure for table `facility_user_ids`
--

DROP TABLE IF EXISTS `facility_user_ids`;
CREATE TABLE  `facility_user_ids` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `uid` bigint(20) DEFAULT NULL,
  `facility_id` bigint(20) DEFAULT NULL,
  `uuid` binary(16) DEFAULT NULL,
  `field_id`    varchar(31)  NOT NULL COMMENT 'references layout_options.field_id',
  `field_value` TEXT,
  `date_created` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `last_updated` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `uid` (`uid`,`facility_id`,`field_id`),
  KEY `uuid` (`uuid`)
) ENGINE=InnoDB  AUTO_INCREMENT=1;

-- ---------------------------------------------------------

--
-- Table structure for table `fee_schedule`
--

DROP TABLE IF EXISTS `fee_schedule`;
CREATE TABLE `fee_schedule` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `insurance_company_id` INT(11) NOT NULL DEFAULT 0,
    `plan` VARCHAR(20) DEFAULT '',
    `code` VARCHAR(10) DEFAULT '',
    `modifier` VARCHAR(2) DEFAULT '',
    `type` VARCHAR(20) DEFAULT '',
    `fee` decimal(12,2) DEFAULT NULL,
    `effective_date` date DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `ins_plan_code_mod_type_date` (`insurance_company_id`, `plan`, `code`, `modifier`, `type`, `effective_date`)
) ENGINE=InnoDb AUTO_INCREMENT=1;

-- ---------------------------------------------------------

--
-- Table structure for table `fee_sheet_options`
--

DROP TABLE IF EXISTS `fee_sheet_options`;
CREATE TABLE `fee_sheet_options` (
  `fs_category` varchar(63) default NULL,
  `fs_option` varchar(63) default NULL,
  `fs_codes` varchar(255) default NULL
) ENGINE=InnoDB;

--
-- Inserting data for table `fee_sheet_options`
--

-- --------------------------------------------------------

--
-- Table structure for table `form_clinical_notes`
--

DROP TABLE IF EXISTS `form_clinical_notes`;
CREATE TABLE `form_clinical_notes` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT,
    `form_id` bigint(20) NOT NULL,
    `uuid` binary(16) DEFAULT NULL,
    `date` DATE DEFAULT NULL,
    `pid` bigint(20) DEFAULT NULL,
    `encounter` varchar(255) DEFAULT NULL,
    `user` varchar(255) DEFAULT NULL,
    `groupname` varchar(255) DEFAULT NULL,
    `authorized` tinyint(4) DEFAULT NULL,
    `activity` tinyint(4) DEFAULT NULL,
    `code` varchar(255) DEFAULT NULL,
    `codetext` text,
    `description` text,
    `external_id` VARCHAR(30) DEFAULT NULL,
    `clinical_notes_type` varchar(100) DEFAULT NULL,
    `clinical_notes_category` varchar(100) DEFAULT NULL,
    `note_related_to` text,
    `last_updated` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uuid` (`uuid`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `form_dictation`
--

DROP TABLE IF EXISTS `form_dictation`;
CREATE TABLE `form_dictation` (
  `id` bigint(20) NOT NULL auto_increment,
  `date` datetime default NULL,
  `pid` bigint(20) default NULL,
  `user` varchar(255) default NULL,
  `groupname` varchar(255) default NULL,
  `authorized` tinyint(4) default NULL,
  `activity` tinyint(4) default NULL,
  `dictation` longtext,
  `additional_notes` longtext,
  PRIMARY KEY  (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `form_encounter`
--

DROP TABLE IF EXISTS `form_encounter`;
CREATE TABLE `form_encounter` (
  `id` bigint(20) NOT NULL auto_increment,
  `uuid` binary(16) DEFAULT NULL,
  `date` datetime default NULL,
  `reason` longtext,
  `facility` longtext,
  `facility_id` int(11) NOT NULL default '0',
  `pid` bigint(20) default NULL,
  `encounter` bigint(20) default NULL,
  `onset_date` datetime default NULL,
  `sensitivity` varchar(30) default NULL,
  `billing_note` text,
  `pc_catid` int(11) NOT NULL default '5' COMMENT 'event category from openemr_postcalendar_categories',
  `last_level_billed` int  NOT NULL DEFAULT 0 COMMENT '0=none, 1=ins1, 2=ins2, etc',
  `last_level_closed` int  NOT NULL DEFAULT 0 COMMENT '0=none, 1=ins1, 2=ins2, etc',
  `last_stmt_date`    date DEFAULT NULL,
  `stmt_count`        int  NOT NULL DEFAULT 0,
  `provider_id` INT(11) DEFAULT '0' COMMENT 'default and main provider for this visit',
  `supervisor_id` INT(11) DEFAULT '0' COMMENT 'supervising provider, if any, for this visit',
  `invoice_refno` varchar(31) NOT NULL DEFAULT '',
  `referral_source` varchar(31) NOT NULL DEFAULT '',
  `billing_facility` INT(11) NOT NULL DEFAULT 0,
  `external_id` VARCHAR(20) DEFAULT NULL,
  `pos_code` tinyint(4) default NULL,
  `parent_encounter_id` BIGINT(20) NULL DEFAULT NULL,
  `class_code` VARCHAR(10) NOT NULL DEFAULT "AMB",
  `shift` varchar(31) NOT NULL DEFAULT '',
  `voucher_number` varchar(255) NOT NULL DEFAULT '' COMMENT 'also called referral number',
  `discharge_disposition` varchar(100) NULL DEFAULT NULL,
  `encounter_type_code` VARCHAR(31) NULL DEFAULT NULL COMMENT 'not all types are categories',
  `encounter_type_description` TEXT,
  `referring_provider_id` INT(11) DEFAULT '0' COMMENT 'referring provider, if any, for this visit',
  `date_end` DATETIME DEFAULT NULL,
  `in_collection` tinyint(1) default NULL,
  `last_update` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `ordering_provider_id` INT(11) DEFAULT '0' COMMENT 'referring provider, if any, for this visit',
  PRIMARY KEY  (`id`),
  UNIQUE KEY `uuid` (`uuid`),
  KEY `pid_encounter` (`pid`, `encounter`),
  KEY `encounter_date` (`date`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `form_misc_billing_options`
--

DROP TABLE IF EXISTS `form_misc_billing_options`;
CREATE TABLE `form_misc_billing_options` (
  `id` bigint(20) NOT NULL auto_increment,
  `date` datetime default NULL,
  `pid` bigint(20) default NULL,
  `user` varchar(255) default NULL,
  `groupname` varchar(255) default NULL,
  `authorized` tinyint(4) default NULL,
  `activity` tinyint(4) default NULL,
  `employment_related` tinyint(1) default NULL,
  `auto_accident` tinyint(1) default NULL,
  `accident_state` varchar(2) default NULL,
  `other_accident` tinyint(1) default NULL,
  `medicaid_referral_code` varchar(2)   default NULL,
  `epsdt_flag` tinyint(1) default NULL,
  `provider_qualifier_code` varchar(2) default NULL,
  `provider_id` int(11) default NULL,
  `outside_lab` tinyint(1) default NULL,
  `lab_amount` decimal(5,2) default NULL,
  `is_unable_to_work` tinyint(1) default NULL,
  `onset_date` date default NULL,
  `date_initial_treatment` date default NULL,
  `off_work_from` date default NULL,
  `off_work_to` date default NULL,
  `is_hospitalized` tinyint(1) default NULL,
  `hospitalization_date_from` date default NULL,
  `hospitalization_date_to` date default NULL,
  `medicaid_resubmission_code` varchar(10) default NULL,
  `medicaid_original_reference` varchar(15) default NULL,
  `prior_auth_number` varchar(20) default NULL,
  `comments` varchar(255) default NULL,
  `replacement_claim` tinyint(1) default 0,
  `icn_resubmission_number` varchar(35) default NULL,
  `box_14_date_qual` char(3) default NULL,
  `box_15_date_qual` char(3) default NULL,
  `encounter` bigint(20) default NULL,
  PRIMARY KEY  (`id`),
  UNIQUE KEY `encounter` (`encounter`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `form_reviewofs`
--

DROP TABLE IF EXISTS `form_reviewofs`;
CREATE TABLE `form_reviewofs` (
  `id` bigint(20) NOT NULL auto_increment,
  `date` datetime default NULL,
  `pid` bigint(20) default NULL,
  `user` varchar(255) default NULL,
  `groupname` varchar(255) default NULL,
  `authorized` tinyint(4) default NULL,
  `activity` tinyint(4) default NULL,
  `fever` varchar(5) default NULL,
  `chills` varchar(5) default NULL,
  `night_sweats` varchar(5) default NULL,
  `weight_loss` varchar(5) default NULL,
  `poor_appetite` varchar(5) default NULL,
  `insomnia` varchar(5) default NULL,
  `fatigued` varchar(5) default NULL,
  `depressed` varchar(5) default NULL,
  `hyperactive` varchar(5) default NULL,
  `exposure_to_foreign_countries` varchar(5) default NULL,
  `cataracts` varchar(5) default NULL,
  `cataract_surgery` varchar(5) default NULL,
  `glaucoma` varchar(5) default NULL,
  `double_vision` varchar(5) default NULL,
  `blurred_vision` varchar(5) default NULL,
  `poor_hearing` varchar(5) default NULL,
  `headaches` varchar(5) default NULL,
  `ringing_in_ears` varchar(5) default NULL,
  `bloody_nose` varchar(5) default NULL,
  `sinusitis` varchar(5) default NULL,
  `sinus_surgery` varchar(5) default NULL,
  `dry_mouth` varchar(5) default NULL,
  `strep_throat` varchar(5) default NULL,
  `tonsillectomy` varchar(5) default NULL,
  `swollen_lymph_nodes` varchar(5) default NULL,
  `throat_cancer` varchar(5) default NULL,
  `throat_cancer_surgery` varchar(5) default NULL,
  `heart_attack` varchar(5) default NULL,
  `irregular_heart_beat` varchar(5) default NULL,
  `chest_pains` varchar(5) default NULL,
  `shortness_of_breath` varchar(5) default NULL,
  `high_blood_pressure` varchar(5) default NULL,
  `heart_failure` varchar(5) default NULL,
  `poor_circulation` varchar(5) default NULL,
  `vascular_surgery` varchar(5) default NULL,
  `cardiac_catheterization` varchar(5) default NULL,
  `coronary_artery_bypass` varchar(5) default NULL,
  `heart_transplant` varchar(5) default NULL,
  `stress_test` varchar(5) default NULL,
  `emphysema` varchar(5) default NULL,
  `chronic_bronchitis` varchar(5) default NULL,
  `interstitial_lung_disease` varchar(5) default NULL,
  `shortness_of_breath_2` varchar(5) default NULL,
  `lung_cancer` varchar(5) default NULL,
  `lung_cancer_surgery` varchar(5) default NULL,
  `pheumothorax` varchar(5) default NULL,
  `stomach_pains` varchar(5) default NULL,
  `peptic_ulcer_disease` varchar(5) default NULL,
  `gastritis` varchar(5) default NULL,
  `endoscopy` varchar(5) default NULL,
  `polyps` varchar(5) default NULL,
  `colonoscopy` varchar(5) default NULL,
  `colon_cancer` varchar(5) default NULL,
  `colon_cancer_surgery` varchar(5) default NULL,
  `ulcerative_colitis` varchar(5) default NULL,
  `crohns_disease` varchar(5) default NULL,
  `appendectomy` varchar(5) default NULL,
  `divirticulitis` varchar(5) default NULL,
  `divirticulitis_surgery` varchar(5) default NULL,
  `gall_stones` varchar(5) default NULL,
  `cholecystectomy` varchar(5) default NULL,
  `hepatitis` varchar(5) default NULL,
  `cirrhosis_of_the_liver` varchar(5) default NULL,
  `splenectomy` varchar(5) default NULL,
  `kidney_failure` varchar(5) default NULL,
  `kidney_stones` varchar(5) default NULL,
  `kidney_cancer` varchar(5) default NULL,
  `kidney_infections` varchar(5) default NULL,
  `bladder_infections` varchar(5) default NULL,
  `bladder_cancer` varchar(5) default NULL,
  `prostate_problems` varchar(5) default NULL,
  `prostate_cancer` varchar(5) default NULL,
  `kidney_transplant` varchar(5) default NULL,
  `sexually_transmitted_disease` varchar(5) default NULL,
  `burning_with_urination` varchar(5) default NULL,
  `discharge_from_urethra` varchar(5) default NULL,
  `rashes` varchar(5) default NULL,
  `infections` varchar(5) default NULL,
  `ulcerations` varchar(5) default NULL,
  `pemphigus` varchar(5) default NULL,
  `herpes` varchar(5) default NULL,
  `osetoarthritis` varchar(5) default NULL,
  `rheumotoid_arthritis` varchar(5) default NULL,
  `lupus` varchar(5) default NULL,
  `ankylosing_sondlilitis` varchar(5) default NULL,
  `swollen_joints` varchar(5) default NULL,
  `stiff_joints` varchar(5) default NULL,
  `broken_bones` varchar(5) default NULL,
  `neck_problems` varchar(5) default NULL,
  `back_problems` varchar(5) default NULL,
  `back_surgery` varchar(5) default NULL,
  `scoliosis` varchar(5) default NULL,
  `herniated_disc` varchar(5) default NULL,
  `shoulder_problems` varchar(5) default NULL,
  `elbow_problems` varchar(5) default NULL,
  `wrist_problems` varchar(5) default NULL,
  `hand_problems` varchar(5) default NULL,
  `hip_problems` varchar(5) default NULL,
  `knee_problems` varchar(5) default NULL,
  `ankle_problems` varchar(5) default NULL,
  `foot_problems` varchar(5) default NULL,
  `insulin_dependent_diabetes` varchar(5) default NULL,
  `noninsulin_dependent_diabetes` varchar(5) default NULL,
  `hypothyroidism` varchar(5) default NULL,
  `hyperthyroidism` varchar(5) default NULL,
  `cushing_syndrom` varchar(5) default NULL,
  `addison_syndrom` varchar(5) default NULL,
  `additional_notes` longtext,
  PRIMARY KEY  (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `form_ros`
--

DROP TABLE IF EXISTS `form_ros`;
CREATE TABLE `form_ros` (
  `id` int(11) NOT NULL auto_increment,
  `pid` bigint(20) NOT NULL,
  `activity` int(11) NOT NULL default '1',
  `date` datetime default NULL,
  `weight_change` varchar(3) default NULL,
  `weakness` varchar(3) default NULL,
  `fatigue` varchar(3) default NULL,
  `anorexia` varchar(3) default NULL,
  `fever` varchar(3) default NULL,
  `chills` varchar(3) default NULL,
  `night_sweats` varchar(3) default NULL,
  `insomnia` varchar(3) default NULL,
  `irritability` varchar(3) default NULL,
  `heat_or_cold` varchar(3) default NULL,
  `intolerance` varchar(3) default NULL,
  `change_in_vision` varchar(3) default NULL,
  `glaucoma_history` varchar(3) default NULL,
  `eye_pain` varchar(3) default NULL,
  `irritation` varchar(3) default NULL,
  `redness` varchar(3) default NULL,
  `excessive_tearing` varchar(3) default NULL,
  `double_vision` varchar(3) default NULL,
  `blind_spots` varchar(3) default NULL,
  `photophobia` varchar(3) default NULL,
  `hearing_loss` varchar(3) default NULL,
  `discharge` varchar(3) default NULL,
  `pain` varchar(3) default NULL,
  `vertigo` varchar(3) default NULL,
  `tinnitus` varchar(3) default NULL,
  `frequent_colds` varchar(3) default NULL,
  `sore_throat` varchar(3) default NULL,
  `sinus_problems` varchar(3) default NULL,
  `post_nasal_drip` varchar(3) default NULL,
  `nosebleed` varchar(3) default NULL,
  `snoring` varchar(3) default NULL,
  `apnea` varchar(3) default NULL,
  `breast_mass` varchar(3) default NULL,
  `breast_discharge` varchar(3) default NULL,
  `biopsy` varchar(3) default NULL,
  `abnormal_mammogram` varchar(3) default NULL,
  `cough` varchar(3) default NULL,
  `sputum` varchar(3) default NULL,
  `shortness_of_breath` varchar(3) default NULL,
  `wheezing` varchar(3) default NULL,
  `hemoptsyis` varchar(3) default NULL,
  `asthma` varchar(3) default NULL,
  `copd` varchar(3) default NULL,
  `chest_pain` varchar(3) default NULL,
  `palpitation` varchar(3) default NULL,
  `syncope` varchar(3) default NULL,
  `pnd` varchar(3) default NULL,
  `doe` varchar(3) default NULL,
  `orthopnea` varchar(3) default NULL,
  `peripheal` varchar(3) default NULL,
  `edema` varchar(3) default NULL,
  `legpain_cramping` varchar(3) default NULL,
  `history_murmur` varchar(3) default NULL,
  `arrythmia` varchar(3) default NULL,
  `heart_problem` varchar(3) default NULL,
  `dysphagia` varchar(3) default NULL,
  `heartburn` varchar(3) default NULL,
  `bloating` varchar(3) default NULL,
  `belching` varchar(3) default NULL,
  `flatulence` varchar(3) default NULL,
  `nausea` varchar(3) default NULL,
  `vomiting` varchar(3) default NULL,
  `hematemesis` varchar(3) default NULL,
  `gastro_pain` varchar(3) default NULL,
  `food_intolerance` varchar(3) default NULL,
  `hepatitis` varchar(3) default NULL,
  `jaundice` varchar(3) default NULL,
  `hematochezia` varchar(3) default NULL,
  `changed_bowel` varchar(3) default NULL,
  `diarrhea` varchar(3) default NULL,
  `constipation` varchar(3) default NULL,
  `polyuria` varchar(3) default NULL,
  `polydypsia` varchar(3) default NULL,
  `dysuria` varchar(3) default NULL,
  `hematuria` varchar(3) default NULL,
  `frequency` varchar(3) default NULL,
  `urgency` varchar(3) default NULL,
  `incontinence` varchar(3) default NULL,
  `renal_stones` varchar(3) default NULL,
  `utis` varchar(3) default NULL,
  `hesitancy` varchar(3) default NULL,
  `dribbling` varchar(3) default NULL,
  `stream` varchar(3) default NULL,
  `nocturia` varchar(3) default NULL,
  `erections` varchar(3) default NULL,
  `ejaculations` varchar(3) default NULL,
  `g` varchar(3) default NULL,
  `p` varchar(3) default NULL,
  `ap` varchar(3) default NULL,
  `lc` varchar(3) default NULL,
  `mearche` varchar(3) default NULL,
  `menopause` varchar(3) default NULL,
  `lmp` varchar(3) default NULL,
  `f_frequency` varchar(3) default NULL,
  `f_flow` varchar(3) default NULL,
  `f_symptoms` varchar(3) default NULL,
  `abnormal_hair_growth` varchar(3) default NULL,
  `f_hirsutism` varchar(3) default NULL,
  `joint_pain` varchar(3) default NULL,
  `swelling` varchar(3) default NULL,
  `m_redness` varchar(3) default NULL,
  `m_warm` varchar(3) default NULL,
  `m_stiffness` varchar(3) default NULL,
  `muscle` varchar(3) default NULL,
  `m_aches` varchar(3) default NULL,
  `fms` varchar(3) default NULL,
  `arthritis` varchar(3) default NULL,
  `loc` varchar(3) default NULL,
  `seizures` varchar(3) default NULL,
  `stroke` varchar(3) default NULL,
  `tia` varchar(3) default NULL,
  `n_numbness` varchar(3) default NULL,
  `n_weakness` varchar(3) default NULL,
  `paralysis` varchar(3) default NULL,
  `intellectual_decline` varchar(3) default NULL,
  `memory_problems` varchar(3) default NULL,
  `dementia` varchar(3) default NULL,
  `n_headache` varchar(3) default NULL,
  `s_cancer` varchar(3) default NULL,
  `psoriasis` varchar(3) default NULL,
  `s_acne` varchar(3) default NULL,
  `s_other` varchar(3) default NULL,
  `s_disease` varchar(3) default NULL,
  `p_diagnosis` varchar(3) default NULL,
  `p_medication` varchar(3) default NULL,
  `depression` varchar(3) default NULL,
  `anxiety` varchar(3) default NULL,
  `social_difficulties` varchar(3) default NULL,
  `thyroid_problems` varchar(3) default NULL,
  `diabetes` varchar(3) default NULL,
  `abnormal_blood` varchar(3) default NULL,
  `anemia` varchar(3) default NULL,
  `fh_blood_problems` varchar(3) default NULL,
  `bleeding_problems` varchar(3) default NULL,
  `allergies` varchar(3) default NULL,
  `frequent_illness` varchar(3) default NULL,
  `hiv` varchar(3) default NULL,
  `hai_status` varchar(3) default NULL,
  PRIMARY KEY  (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `form_soap`
--

DROP TABLE IF EXISTS `form_soap`;
CREATE TABLE `form_soap` (
  `id` bigint(20) NOT NULL auto_increment,
  `date` datetime default NULL,
  `pid` bigint(20) default '0',
  `user` varchar(255) default NULL,
  `groupname` varchar(255) default NULL,
  `authorized` tinyint(4) default '0',
  `activity` tinyint(4) default '0',
  `subjective` text,
  `objective` text,
  `assessment` text,
  `plan` text,
  PRIMARY KEY  (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `form_vitals`
--

DROP TABLE IF EXISTS `form_vitals`;
CREATE TABLE `form_vitals` (
  `id` bigint(20) NOT NULL auto_increment,
  `uuid` BINARY(16) DEFAULT NULL,
  `date` datetime default NULL,
  `pid` bigint(20) default '0',
  `user` varchar(255) default NULL,
  `groupname` varchar(255) default NULL,
  `authorized` tinyint(4) default '0',
  `activity` tinyint(4) default '0',
  `bps` varchar(40) default NULL,
  `bpd` varchar(40) default NULL,
  `weight` DECIMAL(12,6) default '0.00',
  `height` DECIMAL(12,6) default '0.00',
  `temperature` DECIMAL(12,6) default '0.00',
  `temp_method` varchar(255) default NULL,
  `pulse` DECIMAL(12,6) default '0.00',
  `respiration` DECIMAL(12,6) default '0.00',
  `note` varchar(255) default NULL,
  `BMI` DECIMAL(12,6) default '0.0',
  `BMI_status` varchar(255) default NULL,
  `waist_circ` DECIMAL(12,6) default '0.00',
  `head_circ` DECIMAL(12,6) default '0.00',
  `oxygen_saturation` DECIMAL(6,2) default '0.00',
  `oxygen_flow_rate` DECIMAL(12,6) default '0.00',
  `external_id` VARCHAR(20) DEFAULT NULL,
  `ped_weight_height` DECIMAL(6,2) default '0.00',
  `ped_bmi` DECIMAL(6,2) default '0.00',
  `ped_head_circ` DECIMAL(6,2) default '0.00',
  `inhaled_oxygen_concentration` DECIMAL(6,2) DEFAULT '0.00',
  `last_updated` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY  (`id`),
  KEY `pid` (`pid`),
  UNIQUE KEY `uuid` (`uuid`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `forms`
--

DROP TABLE IF EXISTS `forms`;
CREATE TABLE `forms` (
  `id` bigint(20) NOT NULL auto_increment,
  `date` datetime default NULL,
  `encounter` bigint(20) default NULL,
  `form_name` longtext,
  `form_id` bigint(20) default NULL,
  `pid` bigint(20) default NULL,
  `user` varchar(255) default NULL,
  `groupname` varchar(255) default NULL,
  `authorized` tinyint(4) default NULL,
  `deleted` tinyint(4) DEFAULT '0' NOT NULL COMMENT 'flag indicates form has been deleted',
  `formdir` longtext,
  `therapy_group_id` INT(11) DEFAULT NULL,
  `issue_id` bigint(20) NOT NULL default 0 COMMENT 'references lists.id to identify a case',
  `provider_id` bigint(20) NOT NULL default 0 COMMENT 'references users.id to identify a provider',
  PRIMARY KEY  (`id`),
  KEY `pid_encounter` (`pid`, `encounter`),
  KEY `form_id` (`form_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `gacl_acl`
--

DROP TABLE IF EXISTS `gacl_acl`;
CREATE TABLE `gacl_acl` (
  `id` int(11) NOT NULL DEFAULT 0,
  `section_value` varchar(150) NOT NULL DEFAULT 'system',
  `allow` int(11) NOT NULL DEFAULT 0,
  `enabled` int(11) NOT NULL DEFAULT 0,
  `return_value` text,
  `note` text,
  `updated_date` int(11) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  KEY `gacl_enabled_acl` (`enabled`),
  KEY `gacl_section_value_acl` (`section_value`),
  KEY `gacl_updated_date_acl` (`updated_date`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `gacl_acl_sections`
--

DROP TABLE IF EXISTS `gacl_acl_sections`;
CREATE TABLE `gacl_acl_sections` (
  `id` int(11) NOT NULL DEFAULT 0,
  `value` varchar(150) NOT NULL,
  `order_value` int(11) NOT NULL DEFAULT 0,
  `name` varchar(230) NOT NULL,
  `hidden` int(11) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  UNIQUE KEY `gacl_value_acl_sections` (`value`),
  KEY `gacl_hidden_acl_sections` (`hidden`)
) ENGINE=InnoDB;

--
-- Dumping data for table `gacl_acl_sections`
--

-- --------------------------------------------------------

--
-- Table structure for table `gacl_acl_seq`
--

DROP TABLE IF EXISTS `gacl_acl_seq`;
CREATE TABLE `gacl_acl_seq` (
  `id` int(11) NOT NULL
) ENGINE=InnoDB;

--
-- Inserting data for table `gacl_acl_seq`
--

-- --------------------------------------------------------

--
-- Table structure for table `gacl_aco`
--

DROP TABLE IF EXISTS `gacl_aco`;
CREATE TABLE `gacl_aco` (
  `id` int(11) NOT NULL DEFAULT 0,
  `section_value` varchar(150) NOT NULL DEFAULT '0',
  `value` varchar(150) NOT NULL,
  `order_value` int(11) NOT NULL DEFAULT 0,
  `name` varchar(255) NOT NULL,
  `hidden` int(11) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  UNIQUE KEY `gacl_section_value_value_aco` (`section_value`,`value`),
  KEY `gacl_hidden_aco` (`hidden`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `gacl_aco_map`
--

DROP TABLE IF EXISTS `gacl_aco_map`;
CREATE TABLE `gacl_aco_map` (
  `acl_id` int(11) NOT NULL DEFAULT 0,
  `section_value` varchar(150) NOT NULL DEFAULT '0',
  `value` varchar(150) NOT NULL,
  PRIMARY KEY (`acl_id`,`section_value`,`value`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `gacl_aco_sections`
--

DROP TABLE IF EXISTS `gacl_aco_sections`;
CREATE TABLE `gacl_aco_sections` (
  `id` int(11) NOT NULL DEFAULT 0,
  `value` varchar(150) NOT NULL,
  `order_value` int(11) NOT NULL DEFAULT 0,
  `name` varchar(230) NOT NULL,
  `hidden` int(11) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  UNIQUE KEY `gacl_value_aco_sections` (`value`),
  KEY `gacl_hidden_aco_sections` (`hidden`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `gacl_aco_sections_seq`
--

DROP TABLE IF EXISTS `gacl_aco_sections_seq`;
CREATE TABLE `gacl_aco_sections_seq` (
  `id` int(11) NOT NULL
) ENGINE=InnoDB;

--
-- Inserting data for table `gacl_aco_sections_seq`
--

-- --------------------------------------------------------

--
-- Table structure for table `gacl_aco_seq`
--

DROP TABLE IF EXISTS `gacl_aco_seq`;
CREATE TABLE `gacl_aco_seq` (
  `id` int(11) NOT NULL
) ENGINE=InnoDB;


--
-- Inserting data for table `gacl_aco_seq`
--

-- --------------------------------------------------------

--
-- Table structure for table `gacl_aro`
--

DROP TABLE IF EXISTS `gacl_aro`;
CREATE TABLE `gacl_aro` (
  `id` int(11) NOT NULL DEFAULT 0,
  `section_value` varchar(150) NOT NULL DEFAULT '0',
  `value` varchar(150) NOT NULL,
  `order_value` int(11) NOT NULL DEFAULT 0,
  `name` varchar(255) NOT NULL,
  `hidden` int(11) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  UNIQUE KEY `gacl_section_value_value_aro` (`section_value`,`value`),
  KEY `gacl_hidden_aro` (`hidden`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `gacl_aro_groups`
--

DROP TABLE IF EXISTS `gacl_aro_groups`;
CREATE TABLE `gacl_aro_groups` (
  `id` int(11) NOT NULL DEFAULT 0,
  `parent_id` int(11) NOT NULL DEFAULT 0,
  `lft` int(11) NOT NULL DEFAULT 0,
  `rgt` int(11) NOT NULL DEFAULT 0,
  `name` varchar(255) NOT NULL,
  `value` varchar(150) NOT NULL,
  PRIMARY KEY (`id`,`value`),
  UNIQUE KEY `gacl_value_aro_groups` (`value`),
  KEY `gacl_parent_id_aro_groups` (`parent_id`),
  KEY `gacl_lft_rgt_aro_groups` (`lft`,`rgt`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `gacl_aro_groups_id_seq`
--

DROP TABLE IF EXISTS `gacl_aro_groups_id_seq`;
CREATE TABLE `gacl_aro_groups_id_seq` (
  `id` int(11) NOT NULL
) ENGINE=InnoDB;

--
-- Inserting data for table `gacl_aro_groups_id_seq`
--

-- --------------------------------------------------------

--
-- Table structure for table `gacl_aro_groups_map`
--

DROP TABLE IF EXISTS `gacl_aro_groups_map`;
CREATE TABLE `gacl_aro_groups_map` (
  `acl_id` int(11) NOT NULL DEFAULT 0,
  `group_id` int(11) NOT NULL DEFAULT 0,
  PRIMARY KEY (`acl_id`,`group_id`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `gacl_aro_map`
--

DROP TABLE IF EXISTS `gacl_aro_map`;
CREATE TABLE `gacl_aro_map` (
  `acl_id` int(11) NOT NULL DEFAULT 0,
  `section_value` varchar(150) NOT NULL DEFAULT '0',
  `value` varchar(150) NOT NULL,
  PRIMARY KEY (`acl_id`,`section_value`,`value`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `gacl_aro_sections`
--

DROP TABLE IF EXISTS `gacl_aro_sections`;
CREATE TABLE `gacl_aro_sections` (
  `id` int(11) NOT NULL DEFAULT 0,
  `value` varchar(150) NOT NULL,
  `order_value` int(11) NOT NULL DEFAULT 0,
  `name` varchar(230) NOT NULL,
  `hidden` int(11) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  UNIQUE KEY `gacl_value_aro_sections` (`value`),
  KEY `gacl_hidden_aro_sections` (`hidden`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `gacl_aro_sections_seq`
--

DROP TABLE IF EXISTS `gacl_aro_sections_seq`;
CREATE TABLE `gacl_aro_sections_seq` (
  `id` int(11) NOT NULL
) ENGINE=InnoDB;

--
-- Inserting data for table `gacl_aro_sections_seq`
--

-- --------------------------------------------------------

--
-- Table structure for table `gacl_aro_seq`
--

DROP TABLE IF EXISTS `gacl_aro_seq`;
CREATE TABLE `gacl_aro_seq` (
  `id` int(11) NOT NULL
) ENGINE=InnoDB;

--
-- Inserting data for table `gacl_aro_seq`
--

-- --------------------------------------------------------

--
-- Table structure for table `gacl_axo`
--

DROP TABLE IF EXISTS `gacl_axo`;
CREATE TABLE `gacl_axo` (
  `id` int(11) NOT NULL DEFAULT 0,
  `section_value` varchar(150) NOT NULL DEFAULT '0',
  `value` varchar(150) NOT NULL,
  `order_value` int(11) NOT NULL DEFAULT 0,
  `name` varchar(255) NOT NULL,
  `hidden` int(11) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  UNIQUE KEY `gacl_section_value_value_axo` (`section_value`,`value`),
  KEY `gacl_hidden_axo` (`hidden`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `gacl_axo_groups`
--

DROP TABLE IF EXISTS `gacl_axo_groups`;
CREATE TABLE `gacl_axo_groups` (
  `id` int(11) NOT NULL DEFAULT 0,
  `parent_id` int(11) NOT NULL DEFAULT 0,
  `lft` int(11) NOT NULL DEFAULT 0,
  `rgt` int(11) NOT NULL DEFAULT 0,
  `name` varchar(255) NOT NULL,
  `value` varchar(150) NOT NULL,
  PRIMARY KEY (`id`,`value`),
  UNIQUE KEY `gacl_value_axo_groups` (`value`),
  KEY `gacl_parent_id_axo_groups` (`parent_id`),
  KEY `gacl_lft_rgt_axo_groups` (`lft`,`rgt`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `gacl_axo_groups_map`
--

DROP TABLE IF EXISTS `gacl_axo_groups_map`;
CREATE TABLE `gacl_axo_groups_map` (
  `acl_id` int(11) NOT NULL DEFAULT 0,
  `group_id` int(11) NOT NULL DEFAULT 0,
  PRIMARY KEY (`acl_id`,`group_id`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `gacl_axo_map`
--

DROP TABLE IF EXISTS `gacl_axo_map`;
CREATE TABLE `gacl_axo_map` (
  `acl_id` int(11) NOT NULL DEFAULT 0,
  `section_value` varchar(150) NOT NULL DEFAULT '0',
  `value` varchar(150) NOT NULL,
  PRIMARY KEY (`acl_id`,`section_value`,`value`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `gacl_axo_sections`
--

DROP TABLE IF EXISTS `gacl_axo_sections`;
CREATE TABLE `gacl_axo_sections` (
  `id` int(11) NOT NULL DEFAULT 0,
  `value` varchar(150) NOT NULL,
  `order_value` int(11) NOT NULL DEFAULT 0,
  `name` varchar(230) NOT NULL,
  `hidden` int(11) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  UNIQUE KEY `gacl_value_axo_sections` (`value`),
  KEY `gacl_hidden_axo_sections` (`hidden`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `gacl_groups_aro_map`
--

DROP TABLE IF EXISTS `gacl_groups_aro_map`;
CREATE TABLE `gacl_groups_aro_map` (
  `group_id` int(11) NOT NULL DEFAULT 0,
  `aro_id` int(11) NOT NULL DEFAULT 0,
  PRIMARY KEY (`group_id`,`aro_id`),
  KEY `gacl_aro_id` (`aro_id`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `gacl_groups_axo_map`
--

DROP TABLE IF EXISTS `gacl_groups_axo_map`;
CREATE TABLE `gacl_groups_axo_map` (
  `group_id` int(11) NOT NULL DEFAULT 0,
  `axo_id` int(11) NOT NULL DEFAULT 0,
  PRIMARY KEY (`group_id`,`axo_id`),
  KEY `gacl_axo_id` (`axo_id`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `gacl_phpgacl`
--

DROP TABLE IF EXISTS `gacl_phpgacl`;
CREATE TABLE `gacl_phpgacl` (
  `name` varchar(230) NOT NULL,
  `value` varchar(150) NOT NULL,
  PRIMARY KEY (`name`)
) ENGINE=InnoDB;


--
-- Dumping data for table `gacl_phpgacl`
--

-- --------------------------------------------------------

--
-- Table structure for table `groups`
--

DROP TABLE IF EXISTS `groups`;
CREATE TABLE `groups` (
  `id` bigint(20) NOT NULL auto_increment,
  `name` longtext,
  `user` longtext,
  PRIMARY KEY  (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `history_data`
--

DROP TABLE IF EXISTS `history_data`;
CREATE TABLE `history_data` (
  `id` bigint(20) NOT NULL auto_increment,
  `uuid` binary(16) DEFAULT NULL,
  `coffee` longtext,
  `tobacco` longtext,
  `alcohol` longtext,
  `sleep_patterns` longtext,
  `exercise_patterns` longtext,
  `seatbelt_use` longtext,
  `counseling` longtext,
  `hazardous_activities` longtext,
  `recreational_drugs` longtext,
  `last_breast_exam` varchar(255) default NULL,
  `last_mammogram` varchar(255) default NULL,
  `last_gynocological_exam` varchar(255) default NULL,
  `last_rectal_exam` varchar(255) default NULL,
  `last_prostate_exam` varchar(255) default NULL,
  `last_physical_exam` varchar(255) default NULL,
  `last_sigmoidoscopy_colonoscopy` varchar(255) default NULL,
  `last_ecg` varchar(255) default NULL,
  `last_cardiac_echo` varchar(255) default NULL,
  `last_retinal` varchar(255) default NULL,
  `last_fluvax` varchar(255) default NULL,
  `last_pneuvax` varchar(255) default NULL,
  `last_ldl` varchar(255) default NULL,
  `last_hemoglobin` varchar(255) default NULL,
  `last_psa` varchar(255) default NULL,
  `last_exam_results` varchar(255) default NULL,
  `history_mother` longtext,
  `dc_mother` text,
  `history_father` longtext,
  `dc_father`  text,
  `history_siblings` longtext,
  `dc_siblings` text,
  `history_offspring` longtext,
  `dc_offspring` text,
  `history_spouse` longtext,
  `dc_spouse` text,
  `relatives_cancer` longtext,
  `relatives_tuberculosis` longtext,
  `relatives_diabetes` longtext,
  `relatives_high_blood_pressure` longtext,
  `relatives_heart_problems` longtext,
  `relatives_stroke` longtext,
  `relatives_epilepsy` longtext,
  `relatives_mental_illness` longtext,
  `relatives_suicide` longtext,
  `cataract_surgery` datetime default NULL,
  `tonsillectomy` datetime default NULL,
  `cholecystestomy` datetime default NULL,
  `heart_surgery` datetime default NULL,
  `hysterectomy` datetime default NULL,
  `hernia_repair` datetime default NULL,
  `hip_replacement` datetime default NULL,
  `knee_replacement` datetime default NULL,
  `appendectomy` datetime default NULL,
  `date` datetime default NULL,
  `pid` bigint(20) NOT NULL default '0',
  `name_1` varchar(255) default NULL,
  `value_1` varchar(255) default NULL,
  `name_2` varchar(255) default NULL,
  `value_2` varchar(255) default NULL,
  `additional_history` text,
  `exams` text,
  `usertext11` TEXT,
  `usertext12` varchar(255) NOT NULL DEFAULT '',
  `usertext13` varchar(255) NOT NULL DEFAULT '',
  `usertext14` varchar(255) NOT NULL DEFAULT '',
  `usertext15` varchar(255) NOT NULL DEFAULT '',
  `usertext16` varchar(255) NOT NULL DEFAULT '',
  `usertext17` varchar(255) NOT NULL DEFAULT '',
  `usertext18` varchar(255) NOT NULL DEFAULT '',
  `usertext19` varchar(255) NOT NULL DEFAULT '',
  `usertext20` varchar(255) NOT NULL DEFAULT '',
  `usertext21` varchar(255) NOT NULL DEFAULT '',
  `usertext22` varchar(255) NOT NULL DEFAULT '',
  `usertext23` varchar(255) NOT NULL DEFAULT '',
  `usertext24` varchar(255) NOT NULL DEFAULT '',
  `usertext25` varchar(255) NOT NULL DEFAULT '',
  `usertext26` varchar(255) NOT NULL DEFAULT '',
  `usertext27` varchar(255) NOT NULL DEFAULT '',
  `usertext28` varchar(255) NOT NULL DEFAULT '',
  `usertext29` varchar(255) NOT NULL DEFAULT '',
  `usertext30` varchar(255) NOT NULL DEFAULT '',
  `userdate11` date DEFAULT NULL,
  `userdate12` date DEFAULT NULL,
  `userdate13` date DEFAULT NULL,
  `userdate14` date DEFAULT NULL,
  `userdate15` date DEFAULT NULL,
  `userarea11` text,
  `userarea12` text,
  `created_by` BIGINT(20) DEFAULT NULL COMMENT 'users.id the user that first created this record',
  PRIMARY KEY  (`id`),
  KEY `pid` (`pid`),
  UNIQUE KEY `uuid` (`uuid`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `icd9_dx_code`
--

DROP TABLE IF EXISTS `icd9_dx_code`;
CREATE TABLE `icd9_dx_code` (
  `dx_id` SERIAL,
  `dx_code`             varchar(5),
  `formatted_dx_code`   varchar(6),
  `short_desc`          varchar(60),
  `long_desc`           varchar(300),
  `active` tinyint default 0,
  `revision` int default 0,
  KEY `dx_code` (`dx_code`),
  KEY `formatted_dx_code` (`formatted_dx_code`),
  KEY `active` (`active`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `icd9_sg_code`
--

DROP TABLE IF EXISTS `icd9_sg_code`;
CREATE TABLE `icd9_sg_code` (
  `sg_id` SERIAL,
  `sg_code`             varchar(5),
  `formatted_sg_code`   varchar(6),
  `short_desc`          varchar(60),
  `long_desc`           varchar(300),
  `active` tinyint default 0,
  `revision` int default 0,
  KEY `sg_code` (`sg_code`),
  KEY `formatted_sg_code` (`formatted_sg_code`),
  KEY `active` (`active`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `icd9_dx_long_code`
--

DROP TABLE IF EXISTS `icd9_dx_long_code`;
CREATE TABLE `icd9_dx_long_code` (
  `dx_id` SERIAL,
  `dx_code`             varchar(5),
  `long_desc`           varchar(300),
  `active` tinyint default 0,
  `revision` int default 0
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `icd9_sg_long_code`
--

DROP TABLE IF EXISTS `icd9_sg_long_code`;
CREATE TABLE `icd9_sg_long_code` (
  `sq_id` SERIAL,
  `sg_code`             varchar(5),
  `long_desc`           varchar(300),
  `active` tinyint default 0,
  `revision` int default 0
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `icd10_dx_order_code`
--

DROP TABLE IF EXISTS `icd10_dx_order_code`;
CREATE TABLE `icd10_dx_order_code` (
  `dx_id`               SERIAL,
  `dx_code`             varchar(7),
  `formatted_dx_code`   varchar(10),
  `valid_for_coding`    char,
  `short_desc`          varchar(60),
  `long_desc`           text,
  `active` tinyint default 0,
  `revision` int default 0,
  KEY `formatted_dx_code` (`formatted_dx_code`),
  KEY `active` (`active`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `icd10_pcs_order_code`
--

DROP TABLE IF EXISTS `icd10_pcs_order_code`;
CREATE TABLE `icd10_pcs_order_code` (
  `pcs_id`              SERIAL,
  `pcs_code`            varchar(7),
  `valid_for_coding`    char,
  `short_desc`          varchar(60),
  `long_desc`           text,
  `active` tinyint default 0,
  `revision` int default 0,
  KEY `pcs_code` (`pcs_code`),
  KEY `active` (`active`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `icd10_gem_pcs_9_10`
--

DROP TABLE IF EXISTS `icd10_gem_pcs_9_10`;
CREATE TABLE `icd10_gem_pcs_9_10` (
  `map_id` SERIAL,
  `pcs_icd9_source` varchar(5) default NULL,
  `pcs_icd10_target` varchar(7) default NULL,
  `flags` varchar(5) default NULL,
  `active` tinyint default 0,
  `revision` int default 0
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `icd10_gem_pcs_10_9`
--

DROP TABLE IF EXISTS `icd10_gem_pcs_10_9`;
CREATE TABLE `icd10_gem_pcs_10_9` (
  `map_id` SERIAL,
  `pcs_icd10_source` varchar(7) default NULL,
  `pcs_icd9_target` varchar(5) default NULL,
  `flags` varchar(5) default NULL,
  `active` tinyint default 0,
  `revision` int default 0
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `icd10_gem_dx_9_10`
--

DROP TABLE IF EXISTS `icd10_gem_dx_9_10`;
CREATE TABLE `icd10_gem_dx_9_10` (
  `map_id` SERIAL,
  `dx_icd9_source` varchar(5) default NULL,
  `dx_icd10_target` varchar(7) default NULL,
  `flags` varchar(5) default NULL,
  `active` tinyint default 0,
  `revision` int default 0
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `icd10_gem_dx_10_9`
--

DROP TABLE IF EXISTS `icd10_gem_dx_10_9`;
CREATE TABLE `icd10_gem_dx_10_9` (
  `map_id` SERIAL,
  `dx_icd10_source` varchar(7) default NULL,
  `dx_icd9_target` varchar(5) default NULL,
  `flags` varchar(5) default NULL,
  `active` tinyint default 0,
  `revision` int default 0
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `icd10_reimbr_dx_9_10`
--

DROP TABLE IF EXISTS `icd10_reimbr_dx_9_10`;
CREATE TABLE `icd10_reimbr_dx_9_10` (
  `map_id` SERIAL,
  `code`        varchar(8),
  `code_cnt`    tinyint,
  `ICD9_01`     varchar(5),
  `ICD9_02`     varchar(5),
  `ICD9_03`     varchar(5),
  `ICD9_04`     varchar(5),
  `ICD9_05`     varchar(5),
  `ICD9_06`     varchar(5),
  `active` tinyint default 0,
  `revision` int default 0
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `icd10_reimbr_pcs_9_10`
--

DROP TABLE IF EXISTS `icd10_reimbr_pcs_9_10`;
CREATE TABLE `icd10_reimbr_pcs_9_10` (
  `map_id`      SERIAL,
  `code`        varchar(8),
  `code_cnt`    tinyint,
  `ICD9_01`     varchar(5),
  `ICD9_02`     varchar(5),
  `ICD9_03`     varchar(5),
  `ICD9_04`     varchar(5),
  `ICD9_05`     varchar(5),
  `ICD9_06`     varchar(5),
  `active` tinyint default 0,
  `revision` int default 0
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `immunizations`
--

DROP TABLE IF EXISTS `immunizations`;
CREATE TABLE `immunizations` (
  `id` bigint(20) NOT NULL auto_increment,
  `uuid` binary(16) DEFAULT NULL,
  `patient_id` bigint(20) default NULL,
  `administered_date` datetime default NULL,
  `immunization_id` int(11) default NULL,
  `cvx_code` varchar(64) default NULL,
  `manufacturer` varchar(100) default NULL,
  `lot_number` varchar(50) default NULL,
  `administered_by_id` bigint(20) default NULL,
  `administered_by` VARCHAR( 255 ) default NULL COMMENT 'Alternative to administered_by_id',
  `education_date` date default NULL,
  `vis_date` date default NULL COMMENT 'Date of VIS Statement',
  `note` text,
  `create_date` datetime default NULL,
  `update_date` timestamp NOT NULL,
  `created_by` bigint(20) default NULL,
  `updated_by` bigint(20) default NULL,
  `amount_administered` float DEFAULT NULL,
  `amount_administered_unit` varchar(50) DEFAULT NULL,
  `expiration_date` date DEFAULT NULL,
  `route` varchar(100) DEFAULT NULL,
  `administration_site` varchar(100) DEFAULT NULL,
  `added_erroneously` tinyint(1) NOT NULL DEFAULT '0',
  `external_id` VARCHAR(20) DEFAULT NULL,
  `completion_status` VARCHAR(50) DEFAULT NULL,
  `information_source` VARCHAR(31) DEFAULT NULL,
  `refusal_reason` VARCHAR(31) DEFAULT NULL,
  `ordering_provider` INT(11) DEFAULT NULL,
  `reason_code` varchar(31) DEFAULT NULL COMMENT 'Medical code explaining reason of the vital observation value in form codesystem:codetype;...;',
  `reason_description` text COMMENT 'Human readable text description of the reason_code column',
  `encounter_id` BIGINT(20) DEFAULT NULL COMMENT 'fk to form_encounter.encounter to link immunization to encounter record',
  PRIMARY KEY  (`id`),
  KEY `patient_id` (`patient_id`),
  UNIQUE KEY `uuid` (`uuid`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `insurance_companies`
--

DROP TABLE IF EXISTS `insurance_companies`;
CREATE TABLE `insurance_companies` (
  `id` int(11) NOT NULL default '0',
  `uuid` binary(16)   DEFAULT NULL,
  `name` varchar(255) default NULL,
  `attn` varchar(255) default NULL,
  `cms_id` varchar(15) default NULL,
  `ins_type_code` int(11) default NULL,
  `x12_receiver_id` varchar(25) default NULL,
  `x12_default_partner_id` int(11) default NULL,
  `alt_cms_id` varchar(15) default NULL,
  `inactive` tinyint(1) NOT NULL DEFAULT '0',
  `eligibility_id` VARCHAR(32) default NULL,
  `x12_default_eligibility_id` INT(11) default NULL,
  `cqm_sop` int DEFAULT NULL COMMENT 'HL7 Source of Payment for eCQMs',
  `date_created` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `last_updated` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY  (`id`),
  UNIQUE KEY `uuid` (`uuid`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `insurance_data`
--

DROP TABLE IF EXISTS `insurance_data`;
CREATE TABLE `insurance_data` (
  `id` bigint(20) NOT NULL auto_increment,
  `uuid` binary(16)   DEFAULT NULL,
  `type` enum('primary','secondary','tertiary') default NULL,
  `provider` varchar(255) default NULL,
  `plan_name` varchar(255) default NULL,
  `policy_number` varchar(255) default NULL,
  `group_number` varchar(255) default NULL,
  `subscriber_lname` varchar(255) default NULL,
  `subscriber_mname` varchar(255) default NULL,
  `subscriber_fname` varchar(255) default NULL,
  `subscriber_relationship` varchar(255) default NULL,
  `subscriber_ss` varchar(255) default NULL,
  `subscriber_DOB` date default NULL,
  `subscriber_street` varchar(255) default NULL,
  `subscriber_postal_code` varchar(255) default NULL,
  `subscriber_city` varchar(255) default NULL,
  `subscriber_state` varchar(255) default NULL,
  `subscriber_country` varchar(255) default NULL,
  `subscriber_phone` varchar(255) default NULL,
  `subscriber_employer` varchar(255) default NULL,
  `subscriber_employer_street` varchar(255) default NULL,
  `subscriber_employer_postal_code` varchar(255) default NULL,
  `subscriber_employer_state` varchar(255) default NULL,
  `subscriber_employer_country` varchar(255) default NULL,
  `subscriber_employer_city` varchar(255) default NULL,
  `copay` varchar(255) default NULL,
  `date` date NULL,
  `pid` bigint(20) NOT NULL default '0',
  `subscriber_sex` varchar(25) default NULL,
  `accept_assignment` varchar(5) NOT NULL DEFAULT 'TRUE',
  `policy_type` varchar(25) NOT NULL default '',
  `subscriber_street_line_2` TINYTEXT,
  `subscriber_employer_street_line_2` TINYTEXT,
  `date_end` date NULL,
  PRIMARY KEY  (`id`),
  UNIQUE KEY `uuid` (`uuid`),
  UNIQUE KEY `pid_type_date` (`pid`,`type`,`date`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `insurance_numbers`
--

DROP TABLE IF EXISTS `insurance_numbers`;
CREATE TABLE `insurance_numbers` (
  `id` int(11) NOT NULL default '0',
  `provider_id` int(11) NOT NULL default '0',
  `insurance_company_id` int(11) default NULL,
  `provider_number` varchar(20) default NULL,
  `rendering_provider_number` varchar(20) default NULL,
  `group_number` varchar(20) default NULL,
  `provider_number_type` varchar(4) default NULL,
  `rendering_provider_number_type` varchar(4) default NULL,
  PRIMARY KEY  (`id`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `insurance_type_codes`
--

DROP TABLE IF EXISTS `insurance_type_codes`;
CREATE TABLE `insurance_type_codes` (
  `id` int(2) NOT NULL,
  `type` varchar(60) NOT NULL,
  `claim_type` text,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB;

--
-- Inserting data for table `insurance_type_codes`
--

-- --------------------------------------------------------

--
-- Table structure for table `ip_tracking`
--
DROP TABLE IF EXISTS `ip_tracking`;
CREATE TABLE `ip_tracking` (
    `id` bigint NOT NULL auto_increment,
    `ip_string` varchar(255) DEFAULT '',
    `total_ip_login_fail_counter` bigint DEFAULT 0,
    `ip_login_fail_counter` bigint DEFAULT 0,
    `ip_last_login_fail` datetime DEFAULT NULL,
    `ip_auto_block_emailed` tinyint DEFAULT 0,
    `ip_force_block` tinyint DEFAULT 0,
    `ip_no_prevent_timing_attack` tinyint DEFAULT 0,
    PRIMARY KEY (`id`),
    UNIQUE KEY `ip_string` (`ip_string`)
) ENGINE=InnoDb AUTO_INCREMENT=1;


-- --------------------------------------------------------

--
-- Table structure for table `issue_encounter`
--

DROP TABLE IF EXISTS `issue_encounter`;
CREATE TABLE `issue_encounter` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `uuid` binary(16) DEFAULT NULL COMMENT 'UUID for this issue encounter record, for data exchange purposes',
  `pid` bigint(20) NOT NULL,
  `list_id` int(11) NOT NULL,
  `encounter` int(11) NOT NULL,
  `resolved` tinyint(1) NOT NULL,
  `created_by` bigint(20) DEFAULT NULL COMMENT 'fk to users.id for the user that entered in the issue encounter data',
  `updated_by` bigint(20) DEFAULT NULL COMMENT 'fk to users.id for the user that last updated the issue encounter data',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT 'timestamp when this issue encounter record was created',
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'timestamp when this issue encounter record was last updated',
  UNIQUE KEY `uniq_issue_key`(`pid`,`list_id`,`encounter`),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uuid_unique` (`uuid`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `issue_types`
--

DROP TABLE IF EXISTS `issue_types`;
CREATE TABLE `issue_types` (
    `active` tinyint(1) NOT NULL DEFAULT '1',
    `category` varchar(75) NOT NULL DEFAULT '',
    `type` varchar(75) NOT NULL DEFAULT '',
    `plural` varchar(75) NOT NULL DEFAULT '',
    `singular` varchar(75) NOT NULL DEFAULT '',
    `abbreviation` varchar(75) NOT NULL DEFAULT '',
    `style` smallint(6) NOT NULL DEFAULT '0',
    `force_show` smallint(6) NOT NULL DEFAULT '0',
    `ordering` int(11) NOT NULL DEFAULT '0',
    `aco_spec` varchar(63) NOT NULL default 'patients|med',
    PRIMARY KEY (`category`,`type`)
) ENGINE=InnoDB;

--
-- Inserting data for table `issue_types`
--

-- --------------------------------------------------------

CREATE TABLE IF NOT EXISTS `form_history_sdoh_health_concerns` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT,
    `sdoh_history_id` bigint(20) UNSIGNED NOT NULL COMMENT 'FK to form_history_sdoh.id',
    `health_concern_id` bigint(20) NOT NULL COMMENT 'FK to lists.id where type=health_concern or medical_problem',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `created_by` bigint(20) DEFAULT NULL COMMENT 'FK to users.id',
    PRIMARY KEY (`id`),
    UNIQUE KEY `unique_sdoh_concern` (`sdoh_history_id`, `health_concern_id`),
    KEY `idx_sdoh_history` (`sdoh_history_id`),
    KEY `idx_health_concern` (`health_concern_id`)
) ENGINE=InnoDB COMMENT='Links SDOH assessments to health concern conditions';

--
-- Table structure for table `keys`
--

DROP TABLE IF EXISTS `keys`;
CREATE TABLE `keys` (
  `id` bigint(20) NOT NULL auto_increment,
  `name` varchar(20) NOT NULL DEFAULT '',
  `value` text,
  PRIMARY KEY (`id`),
  UNIQUE KEY (`name`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `lang_constants`
--

DROP TABLE IF EXISTS `lang_constants`;
CREATE TABLE `lang_constants` (
  `cons_id` int(11) NOT NULL auto_increment,
  `constant_name` mediumtext BINARY,
  UNIQUE KEY `cons_id` (`cons_id`),
  KEY `constant_name` (`constant_name`(100))
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `lang_definitions`
--

DROP TABLE IF EXISTS `lang_definitions`;
CREATE TABLE `lang_definitions` (
  `def_id` int(11) NOT NULL auto_increment,
  `cons_id` int(11) NOT NULL default '0',
  `lang_id` int(11) NOT NULL default '0',
  `definition` mediumtext,
  UNIQUE KEY `def_id` (`def_id`),
  KEY `cons_id` (`cons_id`),
  KEY `lang_cons` (`lang_id`, `cons_id`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `lang_languages`
--

DROP TABLE IF EXISTS `lang_languages`;
CREATE TABLE `lang_languages` (
  `lang_id` int(11) NOT NULL auto_increment,
  `lang_code` char(2) NOT NULL default '',
  `lang_description` varchar(100) default NULL,
  `lang_is_rtl` TINYINT DEFAULT 0 COMMENT 'Set this to 1 for RTL languages Arabic, Farsi, Hebrew, Urdu etc.',
  UNIQUE KEY `lang_id` (`lang_id`)
) ENGINE=InnoDB AUTO_INCREMENT=2;

--
-- Inserting data for table `lang_languages`
--

-- --------------------------------------------------------

--
-- Table structure for table `lang_custom`
--

DROP TABLE IF EXISTS `lang_custom`;
CREATE TABLE `lang_custom` (
  `lang_description` varchar(100) NOT NULL default '',
  `lang_code` char(2) NOT NULL default '',
  `constant_name` mediumtext,
  `definition` mediumtext
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `layout_group_properties`
--

DROP TABLE IF EXISTS `layout_group_properties`;
CREATE TABLE `layout_group_properties` (
  grp_form_id     varchar(31)    not null,
  grp_group_id    varchar(31)    not null default '' comment 'empty when representing the whole form',
  grp_title       varchar(63)    not null default '' comment 'descriptive name of the form or group',
  grp_subtitle    varchar(63)    not null default '' comment 'for display under the title',
  grp_mapping     varchar(31)    not null default '' comment 'the form category',
  grp_seq         int(11)        not null default 0  comment 'optional order within mapping',
  grp_activity    tinyint(1)     not null default 1,
  grp_repeats     int(11)        not null default 0,
  grp_columns     int(11)        not null default 0,
  grp_size        int(11)        not null default 0,
  grp_issue_type  varchar(75)    not null default '',
  grp_aco_spec    varchar(63)    not null default '',
  grp_save_close  tinyint(1)     not null default 0,
  grp_init_open   tinyint(1)     not null default 0,
  grp_referrals   tinyint(1)     not null default 0,
  grp_unchecked   tinyint(1)     not null default 0,
  grp_services    varchar(4095)  not null default '',
  grp_products    varchar(4095)  not null default '',
  grp_diags       varchar(4095)  not null default '',
  grp_last_update timestamp      NULL,
  PRIMARY KEY (grp_form_id, grp_group_id)
) ENGINE=InnoDB;

--
-- Inserting data for table `layout_group_properties`
--

-- --------------------------------------------------------

--
-- Table structure for table `layout_options`
--

DROP TABLE IF EXISTS `layout_options`;
CREATE TABLE `layout_options` (
  `form_id` varchar(31) NOT NULL default '',
  `field_id` varchar(31) NOT NULL default '',
  `group_id` varchar(31) NOT NULL default '',
  `title` text,
  `seq` int(11) NOT NULL default '0',
  `data_type` tinyint(3) NOT NULL default '0',
  `uor` tinyint(1) NOT NULL default '1',
  `fld_length` int(11) NOT NULL default '15',
  `max_length` int(11) NOT NULL default '0',
  `list_id` varchar(100) NOT NULL default '',
  `titlecols` tinyint(3) NOT NULL default '1',
  `datacols` tinyint(3) NOT NULL default '1',
  `default_value` varchar(255) NOT NULL default '',
  `edit_options` varchar(36) NOT NULL default '',
  `description` text,
  `fld_rows` int(11) NOT NULL default '0',
  `list_backup_id` varchar(100) NOT NULL default '',
  `source` char(1) NOT NULL default 'F' COMMENT 'F=Form, D=Demographics, H=History, E=Encounter',
  `conditions` text COMMENT 'serialized array of skip conditions',
  `validation` varchar(100) default NULL,
  `codes` varchar(255) NOT NULL DEFAULT '',
  PRIMARY KEY  (`form_id`,`field_id`,`seq`)
) ENGINE=InnoDB;

--
-- Inserting data for table `layout_options`
--

--
-- choices
--
-- Stats
--
-- ------------------------------------
-- --------------------------------------------------------

--
-- Table structure for table `list_options`
--

DROP TABLE IF EXISTS `list_options`;
CREATE TABLE `list_options` (
  `list_id` varchar(100) NOT NULL default '',
  `option_id` varchar(100) NOT NULL default '',
  `title` varchar(255) NOT NULL default '',
  `seq` int(11) NOT NULL default '0',
  `is_default` tinyint(1) NOT NULL default '0',
  `option_value` float NOT NULL default '0',
  `mapping` varchar(31) NOT NULL DEFAULT '',
  `notes` TEXT,
  `codes` varchar(255) NOT NULL DEFAULT '',
  `toggle_setting_1` tinyint(1) NOT NULL default '0',
  `toggle_setting_2` tinyint(1) NOT NULL default '0',
  `activity` TINYINT DEFAULT 1 NOT NULL,
  `subtype` varchar(31) NOT NULL DEFAULT '',
  `edit_options` tinyint(1) NOT NULL DEFAULT '1',
  `timestamp` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `last_updated` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY  (`list_id`,`option_id`)
) ENGINE=InnoDB;

--
-- Inserting data for table `list_options`
--

-- there isn't really an NCI code for affected area as it depends on what area it is, so this code is unmapeable
--
--  Clinical Plan Titles
--

--
-- Clinical Rule Titles
--

--
-- order types
--

--
-- Clinical Rule Target Methods
--

-- Clinical Rule Target Intervals

-- Clinical Rule Comparisons
-- Clinical Rule Filter Methods
-- Clinical Rule Age Intervals
-- Encounter Types (needed for mapping encounters for CQM rules)
-- Clinical Rule Action Categories
-- Clinical Rule Action Items
-- Clinical Rule Reminder Intervals
-- Clinical Rule Reminder Methods
-- Clinical Rule Reminder Due Options
-- Clinical Rule Reminder Inactivate Options
-- eRx User Roles
-- MSP remit codes
-- Medical Problem Issue List
-- Ophthalmology: Medical Problem Issue List
-- Medication Issue List
-- Allergy Issue List
-- Surgery Issue List
-- Dental Issue List
-- General Issue List
-- Issue Types List
-- Issue Subtypes List
-- Insurance Types List
-- Amendment Statuses
-- Amendment request from
-- Patient Flow Board Rooms
-- Religious Affiliation
-- Relationship
-- Severity
-- Physician Type

-- Industry

-- Occupation

-- Industry ODH

-- Occupation ODH

-- Reaction

-- County

-- Immunization Manufacturers

-- Immunization Completion Status

-- Immunization Registry Status

-- Publicity Code

-- Immunization Refusal Reason

-- Immunization Information Source

-- Next of kin Relationship
-- Immunization Administered Site
-- Immunization Observation Criteria
-- Immunization Vaccine Eligibility Results
--  LBF Validation

--  Form Keys

-- provider_qualifier_code

-- Files type white list

-- Sample Apps (Disabled)

-- Sort Directions

-- ActEncounterCode [FHIR Encounter.class]
-- us-core-provider-role [FHIR PractitionerRole.role]
-- us-core-provider-specialty [FHIR PractitionerRole.specialty]
-- AllergyIntolerance Verification Status Codes [FHIR AllergyIntolerance.verification]
-- Condition Verification Status Codes [FHIR Condition.verification]
-- Vitals Interpretation Values
-- Discharge Disposition (for encounters and eventually appointments)
-- External Patient Education
-- Observation Types
-- --------------------------------------------------------

-- Observation Statii
-- Add list options of observation-status codes
-- --------------------------------------------------------

-- CCDA Sections for sort orders
-- Address Use Types
-- Address Types
-- -------------------------------------------------------------------------------------------------------------------------------------------------------
-- Social History SDOHValuesets

-- -----------------------------------------------------------------------------------------------------------------------------------------------------------

-- Vital Signs Answers (single set; no duplicate parent)
-- Tribal Affiliations
-- Disability Status parent exists above; now the answers
-- SDOH Problems (single set)
-- SDOH Interventions (fixed duplicate option_id for 467681000124101)
-- --------------------------------------------------------------------------------------------------------------------------------------------------------------
-- Intentional missing create list. Appends
-- Yes/No/Unknown List
-- Administrative Sex list used for patient_data.sex_identified field.
-- This list seems to constantly update with USCDI versions and new administrations so expect the values here to change frequently
-- note USCDI V3 has a ton more options here, but USCDI V4 reverts to M/F/nonbinary/asked-decline with expansion allowed so adding in unknown to map values from patient_data.sex column
-- Insert pronoun list
-- Add v3-ActPharmacySupplyType for tracking the supply type of drug dispensing
-- this is an example value set which means the value set can be nearly anything we want here so we can expand in the future if needed
-- Related Person relationships
-- Spouse/Partner
-- Parents
-- Children
-- Siblings
-- Grandparents
-- Great Grandparents
-- Grandchildren
-- Extended Family
 -- In-Laws
-- Other Relationships
-- Self
-- Related Person Roles
-- Telecom System types
-- Telecome Uses
-- Person Patient Link Method
--
-- Table structure for table `lists`
--

DROP TABLE IF EXISTS `lists`;
CREATE TABLE `lists` (
  `id` bigint(20) NOT NULL auto_increment,
  `uuid` binary(16) DEFAULT NULL,
  `date` datetime default NULL,
  `type` varchar(255) default NULL,
  `subtype` varchar(31) NOT NULL DEFAULT '',
  `title` varchar(255) default NULL,
  `udi` varchar(255) default NULL,
  `udi_data` text,
  `begdate` datetime default NULL,
  `enddate` datetime default NULL,
  `returndate` date default NULL,
  `occurrence` int(11) default '0' COMMENT "Reference to list_options option_id='occurrence'",
  `classification` int(11) default '0',
  `referredby` varchar(255) default NULL,
  `extrainfo` varchar(255) default NULL,
  `diagnosis` varchar(255) default NULL,
  `activity` tinyint(4) default NULL,
  `comments` longtext,
  `pid` bigint(20) default NULL,
  `user` varchar(255) default NULL,
  `groupname` varchar(255) default NULL,
  `outcome` int(11) NOT NULL default '0',
  `destination` varchar(255) default NULL,
  `reinjury_id` bigint(20)  NOT NULL DEFAULT 0,
  `injury_part` varchar(31) NOT NULL DEFAULT '',
  `injury_type` varchar(31) NOT NULL DEFAULT '',
  `injury_grade` varchar(31) NOT NULL DEFAULT '',
  `reaction` varchar(255) NOT NULL DEFAULT '',
  `verification` VARCHAR(36) NOT NULL DEFAULT '' COMMENT 'Reference to list_options option_id = allergyintolerance-verification',
  `external_allergyid` INT(11) DEFAULT NULL,
  `erx_source` ENUM('0','1') DEFAULT '0' NOT NULL  COMMENT '0-OpenEMR 1-External',
  `erx_uploaded` ENUM('0','1') DEFAULT '0' NOT NULL  COMMENT '0-Pending NewCrop upload 1-Uploaded TO NewCrop',
  `modifydate` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `severity_al` VARCHAR( 50 ) DEFAULT NULL,
  `external_id` VARCHAR(20) DEFAULT NULL,
  `list_option_id` VARCHAR(100) DEFAULT NULL COMMENT 'Reference to list_options table',
  PRIMARY KEY  (`id`),
  KEY `pid` (`pid`),
  KEY `type` (`type`),
  UNIQUE KEY `uuid` (`uuid`)
) ENGINE=InnoDB AUTO_INCREMENT=1;


-- --------------------------------------------------------
DROP TABLE IF EXISTS `lists_medication`;
CREATE TABLE `lists_medication` (
    `id` BIGINT(20) NOT NULL AUTO_INCREMENT
    , `list_id` BIGINT(20) NULL COMMENT 'FK Reference to lists.id'
    , `drug_dosage_instructions` LONGTEXT COMMENT 'Free text dosage instructions for taking the drug'
    , `usage_category` VARCHAR(100) NULL COMMENT 'option_id in list_options.list_id=medication-usage-category'
    , `usage_category_title` VARCHAR(255) NOT NULL COMMENT 'title in list_options.list_id=medication-usage-category'
    , `request_intent` VARCHAR(100) NULL COMMENT 'option_id in list_options.list_id=medication-request-intent'
    , `request_intent_title` VARCHAR(255) NOT NULL COMMENT 'title in list_options.list_id=medication-request-intent'
    , `medication_adherence_information_source` VARCHAR(50) DEFAULT NULL COMMENT 'fk to list_options.option_id where list_id=medication_adherence_information_source to indicate who provided the medication adherence information'
    , `medication_adherence` VARCHAR(50) DEFAULT NULL COMMENT 'fk to list_options.option_id where list_id=medication_adherence to indicate if patient is complying with medication regimen'
    , `medication_adherence_date_asserted` DATETIME DEFAULT NULL COMMENT 'Date when the medication adherence information was asserted'
    , `prescription_id` BIGINT(20) DEFAULT NULL COMMENT 'fk to prescriptions.prescription_id to link medication to prescription record'
    , `is_primary_record` TINYINT(1) DEFAULT '1' COMMENT 'Indicates if this medication is a primary record(1) or a reported record(0)'
    , `reporting_source_record_id` BIGINT(20) DEFAULT NULL COMMENT 'If this is a reported record, this is the fk to the users.id column for the address book user that the medication was reported by'
    , PRIMARY KEY (`id`)
    , INDEX `lists_med_usage_category_idx`(`usage_category`)
    , INDEX `lists_med_request_intent_idx`(`request_intent`)
    , INDEX `lists_medication_list_idx` (`list_id`)
) ENGINE = InnoDB COMMENT = 'Holds additional data about patient medications.';

-- --------------------------------------------------------

--
-- Table structure for table `lists_touch`
--

DROP TABLE IF EXISTS `lists_touch`;
CREATE TABLE `lists_touch` (
  `pid` bigint(20) NOT NULL default '0',
  `type` varchar(255) NOT NULL default '',
  `date` datetime default NULL,
  PRIMARY KEY  (`pid`,`type`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `log`
--

DROP TABLE IF EXISTS `log`;
CREATE TABLE `log` (
  `id` bigint(20) NOT NULL auto_increment,
  `date` datetime default NULL,
  `event` varchar(255) default NULL,
  `category` varchar(255) default NULL,
  `user` varchar(255) default NULL,
  `groupname` varchar(255) default NULL,
  `comments` longtext,
  `user_notes` longtext,
  `patient_id` bigint(20) default NULL,
  `success` tinyint(1) default 1,
  `checksum` longtext,
  `crt_user` varchar(255) default NULL,
  `log_from` VARCHAR(20) DEFAULT 'open-emr',
  `menu_item_id` INT(11) DEFAULT NULL,
  `ccda_doc_id` INT(11) DEFAULT NULL COMMENT 'CCDA document id from ccda',
  PRIMARY KEY  (`id`),
  KEY `patient_id` (`patient_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1;


-- --------------------------------------------------------

--
-- Table structure for table `modules`
--

DROP TABLE IF EXISTS `modules`;
CREATE TABLE `modules` (
  `mod_id` INT(11) NOT NULL AUTO_INCREMENT,
  `mod_name` VARCHAR(64) NOT NULL DEFAULT '0',
  `mod_directory` VARCHAR(64) NOT NULL DEFAULT '',
  `mod_parent` VARCHAR(64) NOT NULL DEFAULT '',
  `mod_type` VARCHAR(64) NOT NULL DEFAULT '',
  `mod_active` INT(1) UNSIGNED NOT NULL DEFAULT '0',
  `mod_ui_name` VARCHAR(64) NOT NULL DEFAULT '',
  `mod_relative_link` VARCHAR(64) NOT NULL DEFAULT '',
  `mod_ui_order` TINYINT(3) NOT NULL DEFAULT '0',
  `mod_ui_active` INT(1) UNSIGNED NOT NULL DEFAULT '0',
  `mod_description` VARCHAR(255) NOT NULL DEFAULT '',
  `mod_nick_name` VARCHAR(25) NOT NULL DEFAULT '',
  `mod_enc_menu` VARCHAR(10) NOT NULL DEFAULT 'no',
  `permissions_item_table` CHAR(100) DEFAULT NULL,
  `directory` VARCHAR(255) NOT NULL,
  `date` DATETIME NOT NULL,
  `sql_run` TINYINT(4) DEFAULT '0',
  `type` TINYINT(4) DEFAULT '0',
  `sql_version` VARCHAR(150) NOT NULL,
  `acl_version` VARCHAR(150) NOT NULL,
  PRIMARY KEY (`mod_id`,`mod_directory`)
) ENGINE=InnoDB;

--
-- Inserting data for table `modules`
--

-- --------------------------------------------------------

--
-- Table structure for table `module_acl_group_settings`
--

DROP TABLE IF EXISTS `module_acl_group_settings`;
CREATE TABLE `module_acl_group_settings` (
  `module_id` int(11) NOT NULL,
  `group_id` int(11) NOT NULL,
  `section_id` int(11) NOT NULL,
  `allowed` tinyint(1) DEFAULT NULL,
  PRIMARY KEY (`module_id`,`group_id`,`section_id`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `module_acl_sections`
--

DROP TABLE IF EXISTS `module_acl_sections`;
CREATE TABLE `module_acl_sections` (
  `section_id` int(11) DEFAULT NULL,
  `section_name` varchar(255) DEFAULT NULL,
  `parent_section` int(11) DEFAULT NULL,
  `section_identifier` varchar(50) DEFAULT NULL,
  `module_id` int(11) DEFAULT NULL
) ENGINE=InnoDB;

--
-- Inserting data for table `module_acl_sections`
--

-- --------------------------------------------------------

--
-- Table structure for table `module_acl_user_settings`
--

DROP TABLE IF EXISTS `module_acl_user_settings`;
CREATE TABLE `module_acl_user_settings` (
  `module_id` int(11) NOT NULL,
  `user_id` int(11) NOT NULL,
  `section_id` int(11) NOT NULL,
  `allowed` int(1) DEFAULT NULL,
  PRIMARY KEY (`module_id`,`user_id`,`section_id`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `module_configuration`
--

DROP TABLE IF EXISTS `module_configuration`;
CREATE TABLE `module_configuration` (
  `module_config_id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `module_id` int(10) unsigned NOT NULL,
  `field_name` varchar(45) NOT NULL,
  `field_value` varchar(255) NOT NULL,
  `created_by` BIGINT(20) DEFAULT NULL COMMENT 'users.id the user that first created this record',
  `date_added` DATETIME DEFAULT NULL COMMENT 'Datetime the record was initially created',
  `updated_by` BIGINT(20) DEFAULT NULL COMMENT 'users.id the user that last modified this record',
  `date_modified` DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT 'Datetime the record was last modified',
  `date_created` DATETIME DEFAULT NULL COMMENT 'Datetime the record was created',
  PRIMARY KEY (`module_config_id`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `modules_hooks_settings`
--

DROP TABLE IF EXISTS `modules_hooks_settings`;
CREATE TABLE `modules_hooks_settings` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `mod_id` int(11) DEFAULT NULL,
  `enabled_hooks` varchar(255) DEFAULT NULL,
  `attached_to` varchar(45) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `modules_settings`
--

DROP TABLE IF EXISTS `modules_settings`;
CREATE TABLE `modules_settings` (
  `mod_id` INT(11) DEFAULT NULL,
  `fld_type` SMALLINT(6) DEFAULT NULL COMMENT '1=>ACL,2=>preferences,3=>hooks',
  `obj_name` VARCHAR(255) DEFAULT NULL,
  `menu_name` VARCHAR(255) DEFAULT NULL,
  `path` VARCHAR(255) DEFAULT NULL
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `notes`
--

DROP TABLE IF EXISTS `notes`;
CREATE TABLE `notes` (
  `id` int(11) NOT NULL default '0',
  `foreign_id` int(11) NOT NULL default '0',
  `note` varchar(255) default NULL,
  `owner` int(11) default NULL,
  `date` datetime default NULL,
  `revision` timestamp NOT NULL,
  PRIMARY KEY  (`id`),
  KEY `foreign_id` (`owner`),
  KEY `foreign_id_2` (`foreign_id`),
  KEY `date` (`date`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `onotes`
--

DROP TABLE IF EXISTS `onotes`;
CREATE TABLE `onotes` (
  `id` bigint(20) NOT NULL auto_increment,
  `date` datetime default NULL,
  `body` longtext,
  `user` varchar(255) default NULL,
  `groupname` varchar(255) default NULL,
  `activity` tinyint(4) default NULL,
  PRIMARY KEY  (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `onsite_documents`
--

DROP TABLE IF EXISTS `onsite_documents`;
CREATE TABLE `onsite_documents` (
  `id` int(10) UNSIGNED NOT NULL AUTO_INCREMENT,
  `pid` bigint(20) UNSIGNED DEFAULT NULL,
  `facility` int(10) UNSIGNED DEFAULT NULL,
  `provider` int(10) UNSIGNED DEFAULT NULL,
  `encounter` int(10) UNSIGNED DEFAULT NULL,
  `create_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `doc_type` varchar(255) NOT NULL,
  `patient_signed_status` smallint(5) UNSIGNED NOT NULL,
  `patient_signed_time` datetime NULL,
  `authorize_signed_time` datetime DEFAULT NULL,
  `accept_signed_status` smallint(5) NOT NULL,
  `authorizing_signator` varchar(50) NOT NULL,
  `review_date` datetime NULL,
  `denial_reason` varchar(255) NOT NULL,
  `authorized_signature` text,
  `patient_signature` text,
  `full_document` mediumblob,
  `file_name` varchar(255) NOT NULL,
  `file_path` varchar(255) NOT NULL,
  `template_data` longtext,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=1;


-- --------------------------------------------------------

--
-- Table structure for table `onsite_mail`
--

DROP TABLE IF EXISTS `onsite_mail`;
CREATE TABLE `onsite_mail` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `date` datetime DEFAULT NULL,
  `owner` varchar(128) DEFAULT NULL,
  `user` varchar(255) DEFAULT NULL,
  `groupname` varchar(255) DEFAULT NULL,
  `activity` tinyint(4) DEFAULT NULL,
  `authorized` tinyint(4) DEFAULT NULL,
  `header` varchar(255) DEFAULT NULL,
  `title` varchar(255) DEFAULT NULL,
  `body` longtext,
  `recipient_id` varchar(128) DEFAULT NULL,
  `recipient_name` varchar(255) DEFAULT NULL,
  `sender_id` varchar(128) DEFAULT NULL,
  `sender_name` varchar(255) DEFAULT NULL,
  `assigned_to` varchar(255) DEFAULT NULL,
  `deleted` tinyint(4) DEFAULT '0' COMMENT 'flag indicates note is deleted',
  `delete_date` datetime DEFAULT NULL,
  `mtype` varchar(128) DEFAULT NULL,
  `message_status` varchar(20) NOT NULL DEFAULT 'New',
  `mail_chain` int(11) DEFAULT NULL,
  `reply_mail_chain` int(11) DEFAULT NULL,
  `is_msg_encrypted` tinyint(2) DEFAULT '0' COMMENT 'Whether messsage encrypted 0-Not encrypted, 1-Encrypted',
  PRIMARY KEY (`id`),
  KEY `pid` (`owner`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `onsite_messages`
--

DROP TABLE IF EXISTS `onsite_messages`;
CREATE TABLE `onsite_messages` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `username` varchar(64) NOT NULL,
  `message` longtext,
  `ip` varchar(15) NOT NULL,
  `date` datetime NOT NULL,
  `sender_id` VARCHAR(64) NULL COMMENT 'who sent id',
  `recip_id` varchar(255) NOT NULL COMMENT 'who to id array',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB COMMENT='Portal messages' AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `onsite_online`
--

DROP TABLE IF EXISTS `onsite_online`;
CREATE TABLE `onsite_online` (
  `hash` varchar(32) NOT NULL,
  `ip` varchar(15) NOT NULL,
  `last_update` datetime NOT NULL,
  `username` varchar(64) NOT NULL,
  `userid` int(11) UNSIGNED DEFAULT NULL,
  PRIMARY KEY (`hash`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `onsite_portal_activity`
--

DROP TABLE IF EXISTS `onsite_portal_activity`;
CREATE TABLE `onsite_portal_activity` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `date` datetime DEFAULT NULL,
  `patient_id` bigint(20) DEFAULT NULL,
  `activity` varchar(255) DEFAULT NULL,
  `require_audit` tinyint(1) DEFAULT '1',
  `pending_action` varchar(255) DEFAULT NULL,
  `action_taken` varchar(255) DEFAULT NULL,
  `status` varchar(255) DEFAULT NULL,
  `narrative` longtext,
  `table_action` longtext,
  `table_args` longtext,
  `action_user` int(11) DEFAULT NULL,
  `action_taken_time` datetime DEFAULT NULL,
  `checksum` longtext,
  PRIMARY KEY (`id`),
  KEY `date` (`date`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `onsite_signatures`
--

DROP TABLE IF EXISTS `onsite_signatures`;
CREATE TABLE `onsite_signatures` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `status` varchar(128) NOT NULL DEFAULT 'waiting',
  `type` varchar(128) NOT NULL,
  `created` int(11) NOT NULL,
  `lastmod` datetime NOT NULL,
  `pid` bigint(20) DEFAULT NULL,
  `encounter` int(11) DEFAULT NULL,
  `user` varchar(255) DEFAULT NULL,
  `activity` tinyint(4) NOT NULL DEFAULT '0',
  `authorized` tinyint(4) DEFAULT NULL,
  `signator` varchar(255) NOT NULL,
  `sig_image` text,
  `signature` text,
  `sig_hash` varchar(255) NOT NULL,
  `ip` varchar(46) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `pid` (`pid`,`user`),
  KEY `encounter` (`encounter`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `openemr_module_vars`
--

DROP TABLE IF EXISTS `openemr_module_vars`;
CREATE TABLE `openemr_module_vars` (
  `pn_id` int(11) unsigned NOT NULL auto_increment,
  `pn_modname` varchar(64) default NULL,
  `pn_name` varchar(64) default NULL,
  `pn_value` longtext,
  PRIMARY KEY  (`pn_id`),
  KEY `pn_modname` (`pn_modname`),
  KEY `pn_name` (`pn_name`)
) ENGINE=InnoDB AUTO_INCREMENT=235;

--
-- Inserting data for table `openemr_module_vars`
--

-- --------------------------------------------------------

--
-- Table structure for table `openemr_modules`
--

DROP TABLE IF EXISTS `openemr_modules`;
CREATE TABLE `openemr_modules` (
  `pn_id` int(11) unsigned NOT NULL auto_increment,
  `pn_name` varchar(64) default NULL,
  `pn_type` int(6) NOT NULL default '0',
  `pn_displayname` varchar(64) default NULL,
  `pn_description` varchar(255) default NULL,
  `pn_regid` int(11) unsigned NOT NULL default '0',
  `pn_directory` varchar(64) default NULL,
  `pn_version` varchar(10) default NULL,
  `pn_admin_capable` tinyint(1) NOT NULL default '0',
  `pn_user_capable` tinyint(1) NOT NULL default '0',
  `pn_state` tinyint(1) NOT NULL default '0',
  PRIMARY KEY  (`pn_id`)
) ENGINE=InnoDB AUTO_INCREMENT=47;

--
-- Inserting data for table `openemr_modules`
--

-- --------------------------------------------------------

--
-- Table structure for table `openemr_postcalendar_categories`
--

DROP TABLE IF EXISTS `openemr_postcalendar_categories`;
CREATE TABLE `openemr_postcalendar_categories` (
  `pc_catid` int(11) unsigned NOT NULL auto_increment,
  `pc_constant_id` VARCHAR (255) default NULL,
  `pc_catname` varchar(100) default NULL,
  `pc_catcolor` varchar(50) default NULL,
  `pc_catdesc` text,
  `pc_recurrtype` int(1) NOT NULL default '0',
  `pc_enddate` date default NULL,
  `pc_recurrspec` text,
  `pc_recurrfreq` int(3) NOT NULL default '0',
  `pc_duration` bigint(20) NOT NULL default '0',
  `pc_end_date_flag` tinyint(1) NOT NULL default '0',
  `pc_end_date_type` int(2) default NULL,
  `pc_end_date_freq` int(11) NOT NULL default '0',
  `pc_end_all_day` tinyint(1) NOT NULL default '0',
  `pc_dailylimit` int(2) NOT NULL default '0',
  `pc_cattype` INT( 11 ) NOT NULL COMMENT 'Used in grouping categories',
  `pc_active` tinyint(1) NOT NULL default 1,
  `pc_seq` int(11) NOT NULL default '0',
  `aco_spec` VARCHAR(63) NOT NULL default 'encounters|notes',
  `pc_last_updated` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY  (`pc_catid`),
  UNIQUE KEY (`pc_constant_id`),
  KEY `basic_cat` (`pc_catname`,`pc_catcolor`)
) ENGINE=InnoDB AUTO_INCREMENT=11;

--
-- Inserting data for table `openemr_postcalendar_categories`
--


-- --------------------------------------------------------

--
-- Table structure for table `openemr_postcalendar_events`
--

DROP TABLE IF EXISTS `openemr_postcalendar_events`;
CREATE TABLE `openemr_postcalendar_events` (
  `pc_eid` int(11) unsigned NOT NULL auto_increment,
  `pc_catid` int(11) NOT NULL default '0',
  `pc_multiple` int(10) unsigned NOT NULL,
  `pc_aid` varchar(30) default NULL,
  `pc_pid` varchar(11) default NULL,
  `pc_gid` int(11) default 0,
  `pc_title` varchar(150) default NULL,
  `pc_time` datetime default NULL,
  `pc_hometext` text,
  `pc_comments` int(11) default '0',
  `pc_counter` mediumint(8) unsigned default '0',
  `pc_topic` int(3) NOT NULL default '1',
  `pc_informant` varchar(20) default NULL,
  `pc_eventDate` date NOT NULL,
  `pc_endDate` date DEFAULT NULL,
  `pc_duration` bigint(20) NOT NULL default '0',
  `pc_recurrtype` int(1) NOT NULL default '0',
  `pc_recurrspec` text,
  `pc_recurrfreq` int(3) NOT NULL default '0',
  `pc_startTime` time default NULL,
  `pc_endTime` time default NULL,
  `pc_alldayevent` int(1) NOT NULL default '0',
  `pc_location` text,
  `pc_conttel` varchar(50) default NULL,
  `pc_contname` varchar(50) default NULL,
  `pc_contemail` varchar(255) default NULL,
  `pc_website` varchar(255) default NULL,
  `pc_fee` varchar(50) default NULL,
  `pc_eventstatus` int(11) NOT NULL default '0',
  `pc_sharing` int(11) NOT NULL default '0',
  `pc_language` varchar(30) default NULL,
  `pc_apptstatus` varchar(15) NOT NULL default '-',
  `pc_prefcatid` int(11) NOT NULL default '0',
  `pc_facility` int(11) NOT NULL default '0' COMMENT 'facility id for this event',
  `pc_sendalertsms` VARCHAR(3) NOT NULL DEFAULT 'NO',
  `pc_sendalertemail` VARCHAR( 3 ) NOT NULL DEFAULT 'NO',
  `pc_billing_location` SMALLINT (6) NOT NULL DEFAULT '0',
  `pc_room` varchar(20) NOT NULL DEFAULT '',
  `uuid` binary(16) DEFAULT NULL,
  PRIMARY KEY  (`pc_eid`),
  KEY `basic_event` (`pc_catid`,`pc_aid`,`pc_eventDate`,`pc_endDate`,`pc_eventstatus`,`pc_sharing`,`pc_topic`),
  KEY `pc_eventDate` (`pc_eventDate`),
  UNIQUE KEY `uuid` (`uuid`)
) ENGINE=InnoDB AUTO_INCREMENT=7;

-- --------------------------------------------------------

--
-- Table structure for table `patient_access_onsite`
--

DROP TABLE IF EXISTS `patient_access_onsite`;
CREATE TABLE `patient_access_onsite`(
  `id` INT NOT NULL AUTO_INCREMENT,
  `pid` bigint(20),
  `portal_username` VARCHAR(100),
  `portal_pwd` VARCHAR(255),
  `portal_pwd_status` TINYINT DEFAULT '1' COMMENT '0=>Password Created Through Demographics by The provider or staff. Patient Should Change it at first time it.1=>Pwd updated or created by patient itself',
  `portal_login_username` VARCHAR(100) DEFAULT NULL COMMENT 'User entered username',
  `portal_onetime`  VARCHAR(255) DEFAULT NULL,
  `date_created` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `pid` (`pid`)
)ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `patient_data`
--

DROP TABLE IF EXISTS `patient_data`;
CREATE TABLE `patient_data` (
  `id` bigint(20) NOT NULL auto_increment,
  `uuid` binary(16) DEFAULT NULL,
  `title` varchar(255) NOT NULL default '',
  `language` varchar(255) NOT NULL default '',
  `financial` varchar(255) NOT NULL default '',
  `fname` varchar(255) NOT NULL default '',
  `lname` varchar(255) NOT NULL default '',
  `mname` varchar(255) NOT NULL default '',
  `DOB` date default NULL,
  `street` varchar(255) NOT NULL default '',
  `postal_code` varchar(255) NOT NULL default '',
  `city` varchar(255) NOT NULL default '',
  `state` varchar(255) NOT NULL default '',
  `country_code` varchar(255) NOT NULL default '',
  `drivers_license` varchar(255) NOT NULL default '',
  `ss` varchar(255) NOT NULL default '',
  `occupation` longtext,
  `phone_home` varchar(255) NOT NULL default '',
  `phone_biz` varchar(255) NOT NULL default '',
  `phone_contact` varchar(255) NOT NULL default '',
  `phone_cell` varchar(255) NOT NULL default '',
  `pharmacy_id` int(11) NOT NULL default '0',
  `status` varchar(255) NOT NULL default '',
  `contact_relationship` varchar(255) NOT NULL default '',
  `date` datetime default NULL,
  `sex` varchar(255) NOT NULL default '' COMMENT 'Sex at birth',
  `referrer` varchar(255) NOT NULL default '',
  `referrerID` varchar(255) NOT NULL default '',
  `providerID` int(11) default NULL,
  `ref_providerID` int(11) default NULL,
  `email` varchar(255) NOT NULL default '',
  `email_direct` varchar(255) NOT NULL default '',
  `ethnoracial` varchar(255) NOT NULL default '',
  `race` varchar(255) NOT NULL default '',
  `ethnicity` varchar(255) NOT NULL default '',
  `religion` varchar(40) NOT NULL default '',
  `interpreter` varchar(255) NOT NULL default '' COMMENT 'original field used for determining if patient needs an interpreter, now used for additional notes about need for interpreter',
  `interpreter_needed` TEXT COMMENT 'fk to list_options.option_id where list_id=yes_no_unknown used to determine if patient needs an interpreter',
  `migrantseasonal` varchar(255) NOT NULL default '',
  `family_size` varchar(255) NOT NULL default '',
  `monthly_income` varchar(255) NOT NULL default '',
  `billing_note` text,
  `homeless` varchar(255) NOT NULL default '',
  `financial_review` datetime default NULL,
  `pubpid` varchar(255) NOT NULL default '',
  `pid` bigint(20) NOT NULL default '0',
  `genericname1` varchar(255) NOT NULL default '',
  `genericval1` varchar(255) NOT NULL default '',
  `genericname2` varchar(255) NOT NULL default '',
  `genericval2` varchar(255) NOT NULL default '',
  `hipaa_mail` varchar(3) NOT NULL default '',
  `hipaa_voice` varchar(3) NOT NULL default '',
  `hipaa_notice` varchar(3) NOT NULL default '',
  `hipaa_message` varchar(20) NOT NULL default '',
  `hipaa_allowsms` VARCHAR(3) NOT NULL DEFAULT 'NO',
  `hipaa_allowemail` VARCHAR(3) NOT NULL DEFAULT 'NO',
  `squad` varchar(32) NOT NULL default '',
  `fitness` int(11) NOT NULL default '0',
  `referral_source` varchar(30) NOT NULL default '',
  `usertext1` varchar(255) NOT NULL DEFAULT '',
  `usertext2` varchar(255) NOT NULL DEFAULT '',
  `usertext3` varchar(255) NOT NULL DEFAULT '',
  `usertext4` varchar(255) NOT NULL DEFAULT '',
  `usertext5` varchar(255) NOT NULL DEFAULT '',
  `usertext6` varchar(255) NOT NULL DEFAULT '',
  `usertext7` varchar(255) NOT NULL DEFAULT '',
  `usertext8` varchar(255) NOT NULL DEFAULT '',
  `userlist1` varchar(255) NOT NULL DEFAULT '',
  `userlist2` varchar(255) NOT NULL DEFAULT '',
  `userlist3` varchar(255) NOT NULL DEFAULT '',
  `userlist4` varchar(255) NOT NULL DEFAULT '',
  `userlist5` varchar(255) NOT NULL DEFAULT '',
  `userlist6` varchar(255) NOT NULL DEFAULT '',
  `userlist7` varchar(255) NOT NULL DEFAULT '',
  `pricelevel` varchar(255) NOT NULL default 'standard',
  `regdate`     DATETIME DEFAULT NULL COMMENT 'Registration Date',
  `contrastart` date DEFAULT NULL COMMENT 'Date contraceptives initially used',
  `completed_ad` VARCHAR(3) NOT NULL DEFAULT 'NO',
  `ad_reviewed` DATETIME DEFAULT NULL COMMENT 'Date and time the advance care directive was reviewed and validated by the authenticator user.',
  `advance_directive_user_authenticator` BIGINT(20) COMMENT 'fk to users.id of the user who authenticates that the advance care directive is valid.',
  `vfc` varchar(255) NOT NULL DEFAULT '',
  `mothersname` varchar(255) NOT NULL DEFAULT '',
  `guardiansname` TEXT,
  `allow_imm_reg_use` varchar(255) NOT NULL DEFAULT '',
  `allow_imm_info_share` varchar(255) NOT NULL DEFAULT '',
  `allow_health_info_ex` varchar(255) NOT NULL DEFAULT '',
  `allow_patient_portal` varchar(31) NOT NULL DEFAULT '',
  `deceased_date` datetime default NULL,
  `deceased_reason` varchar(255) NOT NULL default '',
  `soap_import_status` TINYINT(4) DEFAULT NULL COMMENT '1-Prescription Press 2-Prescription Import 3-Allergy Press 4-Allergy Import',
  `cmsportal_login` varchar(60) NOT NULL default '',
  `care_team_provider` TEXT,
  `care_team_facility` TEXT,
  `care_team_status` TEXT,
  `county` varchar(40) NOT NULL default '',
  `industry` TEXT,
  `imm_reg_status` TEXT,
  `imm_reg_stat_effdate` TEXT,
  `publicity_code` TEXT,
  `publ_code_eff_date` TEXT,
  `protect_indicator` TEXT,
  `prot_indi_effdate` TEXT,
  `guardianrelationship` TEXT,
  `guardiansex` TEXT,
  `guardianaddress` TEXT,
  `guardiancity` TEXT,
  `guardianstate` TEXT,
  `guardianpostalcode` TEXT,
  `guardiancountry` TEXT,
  `guardianphone` TEXT,
  `guardianworkphone` TEXT,
  `guardianemail` TEXT,
  `sexual_orientation` TEXT,
  `gender_identity` TEXT,
  `birth_fname` TEXT,
  `birth_lname` TEXT,
  `birth_mname` TEXT,
  `dupscore` INT NOT NULL default -9,
  `name_history` TINYTEXT,
  `suffix` TINYTEXT,
  `street_line_2` TINYTEXT,
  `patient_groups` TEXT,
  `prevent_portal_apps` TEXT,
  `provider_since_date` TINYTEXT,
  `created_by` BIGINT(20) DEFAULT NULL COMMENT 'users.id the user that first created this record',
  `updated_by` BIGINT(20) DEFAULT NULL COMMENT 'users.id the user that last modified this record',
  `preferred_name` TINYTEXT,
  `nationality_country` TINYTEXT,
  `last_updated` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `tribal_affiliations` TEXT,
  `sex_identified` TEXT COMMENT 'Patient reported current sex',
  `pronoun` TEXT,
  UNIQUE KEY `pid` (`pid`),
  UNIQUE KEY `uuid` (`uuid`),
  KEY `idx_patient_name` (`lname`, `fname`),
  KEY `idx_patient_dob` (`DOB`),
  KEY `id` (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=1;
-- --------------------------------------------------------

--
-- Table structure for table `patient_history` that is a dependent table on `patient_data`
DROP TABLE IF EXISTS `patient_history`;
CREATE TABLE `patient_history` (
    `id` BIGINT(20) NOT NULL AUTO_INCREMENT
    , `uuid` BINARY(16) NULL
    , `date` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
    , `care_team_provider` TEXT
    , `care_team_facility` TEXT
    , `pid` BIGINT(20) NOT NULL
    , `history_type_key` varchar(36) DEFAULT NULL
    , `previous_name_prefix` TEXT
    , `previous_name_first` TEXT
    , `previous_name_middle` TEXT
    , `previous_name_last` TEXT
    , `previous_name_suffix` TEXT
    , `previous_name_enddate` date DEFAULT NULL
    ,`created_by` BIGINT(20) DEFAULT NULL COMMENT 'users.id the user that first created this record'
    , PRIMARY KEY (`id`)
    , UNIQUE `uuid` (`uuid`)
    , KEY `pid_idx` (`pid`)
) ENGINE = InnoDB;
--
-- Table structure for table `patient_portal_menu`
--

DROP TABLE IF EXISTS `patient_portal_menu`;
CREATE TABLE `patient_portal_menu` (
  `patient_portal_menu_id` INT(11) NOT NULL AUTO_INCREMENT,
  `patient_portal_menu_group_id` INT(11) DEFAULT NULL,
  `menu_name` VARCHAR(40) DEFAULT NULL,
  `menu_order` SMALLINT(4) DEFAULT NULL,
  `menu_status` TINYINT(2) DEFAULT '1',
  PRIMARY KEY (`patient_portal_menu_id`)
) ENGINE=INNODB AUTO_INCREMENT=14;

-- --------------------------------------------------------

--
-- Table structure for table `patient_reminders`
--

DROP TABLE IF EXISTS `patient_reminders`;
CREATE TABLE `patient_reminders` (
  `id` bigint(20) NOT NULL auto_increment,
  `active` tinyint(1) NOT NULL default 1 COMMENT '1 if active and 0 if not active',
  `date_inactivated` datetime DEFAULT NULL,
  `reason_inactivated` varchar(31) NOT NULL DEFAULT '' COMMENT 'Maps to list_options list rule_reminder_inactive_opt',
  `due_status` varchar(31) NOT NULL DEFAULT '' COMMENT 'Maps to list_options list rule_reminder_due_opt',
  `pid` bigint(20) NOT NULL COMMENT 'id from patient_data table',
  `category` varchar(31) NOT NULL DEFAULT '' COMMENT 'Maps to the category item in the rule_action_item table',
  `item` varchar(31) NOT NULL DEFAULT '' COMMENT 'Maps to the item column in the rule_action_item table',
  `date_created` datetime DEFAULT NULL,
  `date_sent` datetime DEFAULT NULL,
  `voice_status` tinyint(1) NOT NULL default 0 COMMENT '0 if not sent and 1 if sent',
  `sms_status` tinyint(1) NOT NULL default 0 COMMENT '0 if not sent and 1 if sent',
  `email_status` tinyint(1) NOT NULL default 0 COMMENT '0 if not sent and 1 if sent',
  `mail_status` tinyint(1) NOT NULL default 0 COMMENT '0 if not sent and 1 if sent',
  PRIMARY KEY (`id`),
  KEY `pid` (`pid`),
  KEY (`category`,`item`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `patient_tracker`
--

DROP TABLE IF EXISTS `patient_tracker`;
CREATE TABLE `patient_tracker` (
  `id`                     bigint(20)   NOT NULL auto_increment,
  `date`                   datetime     DEFAULT NULL,
  `apptdate`               date         DEFAULT NULL,
  `appttime`               time         DEFAULT NULL,
  `eid`                    bigint(20)   NOT NULL default '0',
  `pid`                    bigint(20)   NOT NULL default '0',
  `original_user`          varchar(255) NOT NULL default '' COMMENT 'This is the user that created the original record',
  `encounter`              bigint(20)   NOT NULL default '0',
  `lastseq`                varchar(4)   NOT NULL default '' COMMENT 'The element file should contain this number of elements',
  `random_drug_test`       TINYINT(1)   DEFAULT NULL COMMENT 'NULL if not randomized. If randomized, 0 is no, 1 is yes',
  `drug_screen_completed`  TINYINT(1)   NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  KEY (`eid`),
  KEY (`pid`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

--
-- Table structure for table `patient_tracker_element`
--

DROP TABLE IF EXISTS `patient_tracker_element`;
CREATE TABLE `patient_tracker_element` (
  `pt_tracker_id`      bigint(20)   NOT NULL default '0' COMMENT 'maps to id column in patient_tracker table',
  `start_datetime`     datetime     DEFAULT NULL,
  `room`               varchar(20)  NOT NULL default '',
  `status`             varchar(31)  NOT NULL default '',
  `seq`                varchar(4)   NOT NULL default '' COMMENT 'This is a numerical sequence for this pt_tracker_id events',
  `user`               varchar(255) NOT NULL default '' COMMENT 'This is the user that created this element',
  KEY  (`pt_tracker_id`,`seq`)
) ENGINE=InnoDB;

--
-- Table structure for table `payments`
--

DROP TABLE IF EXISTS `payments`;
CREATE TABLE `payments` (
  `id` bigint(20) NOT NULL auto_increment,
  `pid` bigint(20) NOT NULL default '0',
  `dtime` datetime NOT NULL,
  `encounter` bigint(20) NOT NULL default '0',
  `user` varchar(255) default NULL,
  `method` varchar(255) default NULL,
  `source` varchar(255) default NULL,
  `amount1` decimal(12,2) NOT NULL default '0.00',
  `amount2` decimal(12,2) NOT NULL default '0.00',
  `posted1` decimal(12,2) NOT NULL default '0.00',
  `posted2` decimal(12,2) NOT NULL default '0.00',
  PRIMARY KEY  (`id`),
  KEY `pid` (`pid`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `payment_gateway_details`
--

DROP TABLE IF EXISTS `payment_gateway_details`;
CREATE TABLE `payment_gateway_details` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `service_name` varchar(100) DEFAULT NULL,
  `login_id` varchar(255) DEFAULT NULL,
  `transaction_key` varchar(255) DEFAULT NULL,
  `md5` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `pharmacies`
--

DROP TABLE IF EXISTS `pharmacies`;
CREATE TABLE `pharmacies` (
  `id` int(11) NOT NULL default '0',
  `name` varchar(255) default NULL,
  `transmit_method` int(11) NOT NULL default '1',
  `email` varchar(255) default NULL,
  `ncpdp` int(12) DEFAULT NULL,
  `npi` int(12) DEFAULT NULL,
  PRIMARY KEY  (`id`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `phone_numbers`
--

DROP TABLE IF EXISTS `phone_numbers`;
CREATE TABLE `phone_numbers` (
  `id` int(11) NOT NULL default '0',
  `country_code` varchar(5) default NULL,
  `area_code` char(3) default NULL,
  `prefix` char(3) default NULL,
  `number` varchar(4) default NULL,
  `type` int(11) default NULL,
  `foreign_id` int(11) default NULL,
  PRIMARY KEY  (`id`),
  KEY `foreign_id` (`foreign_id`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `pnotes`
--

DROP TABLE IF EXISTS `pnotes`;
CREATE TABLE `pnotes` (
  `id` bigint(20) NOT NULL auto_increment,
  `date` datetime default NULL,
  `body` longtext,
  `pid` bigint(20) default NULL,
  `user` varchar(255) default NULL,
  `groupname` varchar(255) default NULL,
  `activity` tinyint(4) default NULL,
  `authorized` tinyint(4) default NULL,
  `title` varchar(255) default NULL,
  `assigned_to` varchar(255) default NULL,
  `deleted` tinyint(4) default 0 COMMENT 'flag indicates note is deleted',
  `message_status` VARCHAR(20) NOT NULL DEFAULT 'New',
  `portal_relation` VARCHAR(100) NULL,
  `is_msg_encrypted` TINYINT(2) DEFAULT '0' COMMENT 'Whether messsage encrypted 0-Not encrypted, 1-Encrypted',
  `update_by` bigint(20) default NULL,
  `update_date` DATETIME DEFAULT NULL,
  PRIMARY KEY  (`id`),
  KEY `pid` (`pid`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `prescriptions`
--

DROP TABLE IF EXISTS `prescriptions`;
CREATE TABLE `prescriptions` (
  `id` int(11) NOT NULL auto_increment,
  `uuid` binary(16) DEFAULT NULL,
  `patient_id` bigint(20) default NULL,
  `filled_by_id` int(11) default NULL,
  `pharmacy_id` int(11) default NULL,
  `date_added` DATETIME DEFAULT NULL COMMENT 'Datetime the prescriptions was initially created',
  `date_modified` DATETIME DEFAULT NULL COMMENT 'Datetime the prescriptions was last modified',
  `provider_id` int(11) default NULL,
  `encounter` int(11) default NULL,
  `start_date` date default NULL,
  `drug` varchar(150) default NULL,
  `drug_id` int(11) NOT NULL default '0',
  `rxnorm_drugcode` varchar(25) DEFAULT NULL,
  `form` int(3) default NULL,
  `dosage` varchar(100) default NULL,
  `quantity` varchar(31) default NULL,
  `size` varchar(25) DEFAULT NULL,
  `unit` int(11) default NULL,
  `route` varchar(100) default NULL COMMENT 'Max size 100 characters is same max as immunizations',
  `interval` int(11) default NULL,
  `substitute` int(11) default NULL,
  `refills` int(11) default NULL,
  `per_refill` int(11) default NULL,
  `filled_date` date default NULL,
  `medication` int(11) default NULL,
  `note` text,
  `active` int(11) NOT NULL default '1',
  `datetime` DATETIME DEFAULT NULL,
  `user` VARCHAR(50) DEFAULT NULL,
  `site` VARCHAR(50) DEFAULT NULL,
  `prescriptionguid` VARCHAR(50) DEFAULT NULL,
  `erx_source` TINYINT(4) NOT NULL DEFAULT '0' COMMENT '0-OpenEMR 1-External',
  `erx_uploaded` TINYINT(4) NOT NULL DEFAULT '0' COMMENT '0-Pending NewCrop upload 1-Uploaded to NewCrop',
  `drug_info_erx` TEXT,
  `external_id` VARCHAR(20) DEFAULT NULL,
  `end_date` date default NULL,
  `indication` text,
  `prn` VARCHAR(30) DEFAULT NULL,
  `ntx` INT(2) DEFAULT NULL,
  `rtx` INT(2) DEFAULT NULL,
  `txDate` DATE NOT NULL,
  `usage_category` VARCHAR(100) NULL COMMENT 'option_id in list_options.list_id=medication-usage-category',
  `usage_category_title` VARCHAR(255) NOT NULL COMMENT 'title in list_options.list_id=medication-usage-category',
  `request_intent` VARCHAR(100) NULL COMMENT 'option_id in list_options.list_id=medication-request-intent',
  `request_intent_title` VARCHAR(255) NOT NULL COMMENT 'title in list_options.list_id=medication-request-intent',
  `drug_dosage_instructions` longtext COMMENT 'Medication dosage instructions',
  `diagnosis` TEXT COMMENT 'Diagnosis or reason for the prescription',
  `created_by` BIGINT(20) DEFAULT NULL COMMENT 'users.id the user that first created this record',
  `updated_by` BIGINT(20) DEFAULT NULL COMMENT 'users.id the user that last modified this record',
  PRIMARY KEY  (`id`),
  KEY `patient_id` (`patient_id`),
  UNIQUE INDEX `uuid` (`uuid`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `prices`
--

DROP TABLE IF EXISTS `prices`;
CREATE TABLE `prices` (
  `pr_id` varchar(11) NOT NULL default '',
  `pr_selector` varchar(255) NOT NULL default '' COMMENT 'template selector for drugs, empty for codes',
  `pr_level` varchar(31) NOT NULL default '',
  `pr_price` decimal(12,2) NOT NULL default '0.00' COMMENT 'price in local currency',
  PRIMARY KEY  (`pr_id`,`pr_selector`,`pr_level`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `pro_assessments`
--

DROP TABLE IF EXISTS `pro_assessments`;
CREATE TABLE `pro_assessments` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `form_oid` varchar(255) NOT NULL COMMENT 'unique id for specific instrument, pulled from assessment center API',
  `form_name` varchar (255) NOT NULL COMMENT 'pulled from assessment center API',
  `user_id` int(11) NOT NULL COMMENT 'ID for user that orders the form',
  `deadline` datetime NOT NULL COMMENT 'deadline to complete the form, will be used when sending notification and reminders',
  `patient_id` int(11) NOT NULL COMMENT 'ID for patient to order the form for',
  `assessment_oid` varchar(255) NOT NULL COMMENT 'unique id for this specific assessment, pulled from assessment center API',
  `status` varchar(255) NOT NULL COMMENT 'ordered or completed',
  `score` double NOT NULL COMMENT 'T-Score for the assessment',
  `error` double NOT NULL COMMENT 'Standard error for the score',
  `created_at` datetime NOT NULL COMMENT 'timestamp recording the creation time of this assessment',
  `updated_at` datetime NOT NULL COMMENT 'this field indicates the completion time when the status is completed',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `registry`
--

DROP TABLE IF EXISTS `registry`;
CREATE TABLE `registry` (
  `name` varchar(255) default NULL,
  `state` tinyint(4) default NULL,
  `directory` varchar(255) default NULL,
  `id` bigint(20) NOT NULL auto_increment,
  `sql_run` tinyint(4) default NULL,
  `unpackaged` tinyint(4) default NULL,
  `date` datetime default NULL,
  `priority` int(11) default '0',
  `category` varchar(255) default NULL,
  `nickname` varchar(255) default NULL,
  `patient_encounter` TINYINT NOT NULL DEFAULT '1',
  `therapy_group_encounter` TINYINT NOT NULL DEFAULT '0',
  `aco_spec` varchar(63) NOT NULL default 'encounters|notes',
  `form_foreign_id` BIGINT(21) NULL DEFAULT NULL COMMENT 'An id to a form repository. Primarily questionnaire_repository.',
  PRIMARY KEY  (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=25;

--
-- Inserting data for table `registry`
--

-- --------------------------------------------------------

--
-- Table structure for table `report_itemized`
-- (goal is optimize insert performance, so only one key)
--

DROP TABLE IF EXISTS `report_itemized`;
CREATE TABLE `report_itemized` (
  `report_id` bigint(20) NOT NULL,
  `itemized_test_id` smallint(6) NOT NULL,
  `numerator_label` varchar(25) NOT NULL DEFAULT '' COMMENT 'Only used in special cases',
  `pass` tinyint(1) NOT NULL DEFAULT '0' COMMENT '0 is fail, 1 is pass, 2 is excluded',
  `pid` bigint(20) NOT NULL,
  `rule_id` VARCHAR(31) DEFAULT NULL  COMMENT 'fk to clinical_rules.rule_id',
  `item_details` TEXT COMMENT 'JSON with specific sub item results for a clinical rule',
  KEY (`report_id`,`itemized_test_id`,`numerator_label`,`pass`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `report_results`
--

DROP TABLE IF EXISTS `report_results`;
CREATE TABLE `report_results` (
  `report_id` bigint(20) NOT NULL,
  `field_id` varchar(31) NOT NULL default '',
  `field_value` text,
  PRIMARY KEY (`report_id`,`field_id`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `rule_action`
--

DROP TABLE IF EXISTS `rule_action`;
CREATE TABLE `rule_action` (
  `id` varchar(31) NOT NULL DEFAULT '' COMMENT 'Maps to the id column in the clinical_rules table',
  `group_id` bigint(20) NOT NULL DEFAULT 1 COMMENT 'Contains group id to identify collection of targets in a rule',
  `category` varchar(31) NOT NULL DEFAULT '' COMMENT 'Maps to the category item in the rule_action_item table',
  `item` varchar(31) NOT NULL DEFAULT '' COMMENT 'Maps to the item column in the rule_action_item table',
  KEY  (`id`)
) ENGINE=InnoDB;

--
-- Standard clinical rule actions
--

-- --------------------------------------------------------

--
-- Table structure for table `rule_action_item`
--

DROP TABLE IF EXISTS `rule_action_item`;
CREATE TABLE `rule_action_item` (
  `category` varchar(31) NOT NULL DEFAULT '' COMMENT 'Maps to list_options list rule_action_category',
  `item` varchar(31) NOT NULL DEFAULT '' COMMENT 'Maps to list_options list rule_action',
  `clin_rem_link` varchar(255) NOT NULL DEFAULT '' COMMENT 'Custom html link in clinical reminder widget',
  `reminder_message` TEXT COMMENT 'Custom message in patient reminder',
  `custom_flag` tinyint(1) NOT NULL default 0 COMMENT '1 indexed to rule_patient_data, 0 indexed within main schema',
  PRIMARY KEY  (`category`,`item`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `rule_filter`
--

DROP TABLE IF EXISTS `rule_filter`;
CREATE TABLE `rule_filter` (
  `id` varchar(31) NOT NULL DEFAULT '' COMMENT 'Maps to the id column in the clinical_rules table',
  `include_flag` tinyint(1) NOT NULL default 0 COMMENT '0 is exclude and 1 is include',
  `required_flag` tinyint(1) NOT NULL default 0 COMMENT '0 is optional and 1 is required',
  `method` varchar(31) NOT NULL DEFAULT '' COMMENT 'Maps to list_options list rule_filters',
  `method_detail` varchar(31) NOT NULL DEFAULT '' COMMENT 'Maps to list_options lists rule__intervals',
  `value` varchar(255) NOT NULL DEFAULT '',
  KEY  (`id`)
) ENGINE=InnoDB;

--
-- Standard clinical rule filters
--
-- Hypertension: Blood Pressure Measurement
--

-- Tobacco Use Assessment
-- no filters
-- Tobacco Cessation Intervention

-- Adult Weight Screening and Follow-Up

-- Weight Assessment and Counseling for Children and Adolescents

-- Influenza Immunization for Patients >= 50 Years Old

-- Pneumonia Vaccination Status for Older Adults

-- Diabetes: Hemoglobin A1C

-- Diabetes: Urine Microalbumin

-- Diabetes: Eye Exam

-- Diabetes: Foot Exam

-- Cancer Screening: Mammogram

-- Cancer Screening: Pap Smear

-- Cancer Screening: Colon Cancer Screening

-- Cancer Screening: Prostate Cancer Screening

--
-- Rule filters to specifically demonstrate passing of NIST criteria
--
-- Coumadin Management - INR Monitoring
--

-- Penicillin Allergy Assessment

-- --------------------------------------------------------

--
-- Table structure for table `rule_patient_data`
--

DROP TABLE IF EXISTS `rule_patient_data`;
CREATE TABLE `rule_patient_data` (
  `id` bigint(20) NOT NULL auto_increment,
  `date` datetime DEFAULT NULL,
  `pid` bigint(20) NOT NULL,
  `category` varchar(31) NOT NULL DEFAULT '' COMMENT 'Maps to the category item in the rule_action_item table',
  `item` varchar(31) NOT NULL DEFAULT '' COMMENT 'Maps to the item column in the rule_action_item table',
  `complete` varchar(31) NOT NULL DEFAULT '' COMMENT 'Maps to list_options list yesno',
  `result` varchar(255) NOT NULL DEFAULT '',
  PRIMARY KEY  (`id`),
  KEY (`pid`),
  KEY (`category`,`item`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `rule_reminder`
--

DROP TABLE IF EXISTS `rule_reminder`;
CREATE TABLE `rule_reminder` (
  `id` varchar(31) NOT NULL DEFAULT '' COMMENT 'Maps to the id column in the clinical_rules table',
  `method` varchar(31) NOT NULL DEFAULT '' COMMENT 'Maps to list_options list rule_reminder_methods',
  `method_detail` varchar(31) NOT NULL DEFAULT '' COMMENT 'Maps to list_options list rule_reminder_intervals',
  `value` varchar(255) NOT NULL DEFAULT '',
  KEY  (`id`)
) ENGINE=InnoDB;

-- Hypertension: Blood Pressure Measurement
-- Tobacco Use Assessment
-- Tobacco Cessation Intervention
-- Adult Weight Screening and Follow-Up
-- Weight Assessment and Counseling for Children and Adolescents
-- Influenza Immunization for Patients >= 50 Years Old
-- Pneumonia Vaccination Status for Older Adults
-- Diabetes: Hemoglobin A1C
-- Diabetes: Urine Microalbumin
-- Diabetes: Eye Exam
-- Diabetes: Foot Exam
-- Cancer Screening: Mammogram
-- Cancer Screening: Pap Smear
-- Cancer Screening: Colon Cancer Screening
-- Cancer Screening: Prostate Cancer Screening
-- Coumadin Management - INR Monitoring
-- Data Entry - Social Security Number
-- Penicillin Allergy Assessment
-- Blood Pressure Measurement
-- INR Measurement
-- --------------------------------------------------------

--
-- Table structure for table `rule_target`
--

DROP TABLE IF EXISTS `rule_target`;
CREATE TABLE `rule_target` (
  `id` varchar(31) NOT NULL DEFAULT '' COMMENT 'Maps to the id column in the clinical_rules table',
  `group_id` bigint(20) NOT NULL DEFAULT 1 COMMENT 'Contains group id to identify collection of targets in a rule',
  `include_flag` tinyint(1) NOT NULL default 0 COMMENT '0 is exclude and 1 is include',
  `required_flag` tinyint(1) NOT NULL default 0 COMMENT '0 is required and 1 is optional',
  `method` varchar(31) NOT NULL DEFAULT '' COMMENT 'Maps to list_options list rule_targets',
  `value` varchar(255) NOT NULL DEFAULT '' COMMENT 'Data is dependent on the method',
  `interval` bigint(20) NOT NULL DEFAULT 0 COMMENT 'Only used in interval entries',
  KEY  (`id`)
) ENGINE=InnoDB;

--
-- Standard clinical rule targets
--
-- Hypertension: Blood Pressure Measurement
-- Tobacco Use Assessment
-- Tobacco Cessation Intervention
-- Adult Weight Screening and Follow-Up
-- Weight Assessment and Counseling for Children and Adolescents
-- Influenza Immunization for Patients >= 50 Years Old
-- Pneumonia Vaccination Status for Older Adults
-- Diabetes: Hemoglobin A1C
-- Diabetes: Urine Microalbumin
-- Diabetes: Eye Exam
-- Diabetes: Foot Exam
-- Cancer Screening: Mammogram
-- Cancer Screening: Pap Smear
-- Cancer Screening: Colon Cancer Screening
-- Cancer Screening: Prostate Cancer Screening
--
-- Rule targets to specifically demonstrate passing of NIST criteria
--
-- Coumadin Management - INR Monitoring
-- Data entry - Social security number.
-- Penicillin allergy assessment.
-- Blood Pressure Measurement
-- INR Measurement
-- --------------------------------------------------------

--
-- Table structure for table `sequences`
--

DROP TABLE IF EXISTS `sequences`;
CREATE TABLE `sequences` (
  `id` int(11) unsigned NOT NULL default '0'
) ENGINE=InnoDB;

--
-- Inserting data for table `sequences`
--

-- --------------------------------------------------------

--
-- Table structure for table `session_tracker`
--

DROP TABLE IF EXISTS `session_tracker`;
CREATE TABLE `session_tracker` (
  `uuid` binary(16) NOT NULL DEFAULT '',
  `created` timestamp NULL,
  `last_updated` timestamp NULL,
  `number_scripts` bigint DEFAULT 1,
  PRIMARY KEY (`uuid`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `payment_processing_audit`
--

DROP TABLE IF EXISTS `payment_processing_audit`;
CREATE TABLE `payment_processing_audit` (
  `uuid` binary(16) NOT NULL DEFAULT '',
  `service` varchar(50) DEFAULT NULL,
  `pid` bigint NOT NULL,
  `success` tinyint DEFAULT 0,
  `action_name` varchar(50) DEFAULT NULL,
  `amount` varchar(20) DEFAULT NULL,
  `ticket` varchar(100) DEFAULT NULL,
  `transaction_id` varchar(100) DEFAULT NULL,
  `audit_data` text,
  `date` datetime DEFAULT NULL,
  `map_uuid` binary(16) DEFAULT NULL,
  `map_transaction_id` varchar(100) DEFAULT NULL,
  `reverted` tinyint DEFAULT 0,
  `revert_action_name` varchar(50) DEFAULT NULL,
  `revert_transaction_id` varchar(100) DEFAULT NULL,
  `revert_audit_data` text,
  `revert_date` datetime DEFAULT NULL,
  PRIMARY KEY (`uuid`),
  KEY (`pid`),
  KEY (`success`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `supported_external_dataloads`
--

DROP TABLE IF EXISTS `supported_external_dataloads`;
CREATE TABLE `supported_external_dataloads` (
  `load_id` SERIAL,
  `load_type` varchar(24) NOT NULL DEFAULT '',
  `load_source` varchar(24) NOT NULL DEFAULT 'CMS',
  `load_release_date` date NOT NULL,
  `load_filename` varchar(256) NOT NULL DEFAULT '',
  `load_checksum` varchar(32) NOT NULL DEFAULT ''
) ENGINE=InnoDB;

--
-- Inserting data for table `supported_external_dataloads`
--

-- --------------------------------------------------------

--
-- Table structure for table `transactions`
--

DROP TABLE IF EXISTS `transactions`;
CREATE TABLE `transactions` (
  `id`                      bigint(20)   NOT NULL auto_increment,
  `date`                    datetime     default NULL,
  `title`                   varchar(255) NOT NULL DEFAULT '',
  `pid`                     bigint(20)   default NULL,
  `user`                    varchar(255) NOT NULL DEFAULT '',
  `groupname`               varchar(255) NOT NULL DEFAULT '',
  `authorized`              tinyint(4)   default NULL,
  PRIMARY KEY  (`id`),
  KEY `pid` (`pid`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `users`
--

DROP TABLE IF EXISTS `users`;
CREATE TABLE `users` (
  `id` bigint(20) NOT NULL auto_increment,
  `uuid` binary(16) DEFAULT NULL,
  `username` varchar(255) default NULL,
  `password` longtext,
  `authorized` tinyint(4) default NULL,
  `info` longtext,
  `source` tinyint(4) default NULL,
  `fname` varchar(255) default NULL,
  `mname` varchar(255) default NULL,
  `lname` varchar(255) default NULL,
  `suffix` varchar(255) default NULL,
  `federaltaxid` varchar(255) default NULL,
  `federaldrugid` varchar(255) default NULL,
  `upin` varchar(255) default NULL,
  `facility` varchar(255) default NULL,
  `facility_id` int(11) NOT NULL default '0',
  `see_auth` int(11) NOT NULL default '1',
  `active` tinyint(1) NOT NULL default '1',
  `npi` varchar(15) default NULL,
  `title` varchar(30) default NULL,
  `specialty` varchar(255) default NULL,
  `billname` varchar(255) default NULL,
  `email` varchar(255) default NULL,
  `email_direct` varchar(255) NOT NULL default '',
  `google_signin_email` VARCHAR(255) UNIQUE DEFAULT NULL,
  `url` varchar(255) default NULL,
  `assistant` varchar(255) default NULL,
  `organization` varchar(255) default NULL,
  `valedictory` varchar(255) default NULL,
  `street` varchar(60) default NULL,
  `streetb` varchar(60) default NULL,
  `city` varchar(30) default NULL,
  `state` varchar(30) default NULL,
  `zip` varchar(20) default NULL,
  `country_code` varchar(255) COMMENT 'ISO 3166-1 alpha-2 country code for address but can take entire country name for now',
  `street2` varchar(60) default NULL,
  `streetb2` varchar(60) default NULL,
  `city2` varchar(30) default NULL,
  `state2` varchar(30) default NULL,
  `zip2` varchar(20) default NULL,
  `country_code2` varchar(255) COMMENT 'ISO 3166-1 alpha-2 country code for address but can take entire country name for now',
  `phone` varchar(30) default NULL,
  `fax` varchar(30) default NULL,
  `phonew1` varchar(30) default NULL,
  `phonew2` varchar(30) default NULL,
  `phonecell` varchar(30) default NULL,
  `notes` text,
  `cal_ui` tinyint(4) NOT NULL default '1',
  `taxonomy` varchar(30) NOT NULL DEFAULT '207Q00000X',
  `calendar` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '1 = appears in calendar',
  `abook_type` varchar(31) NOT NULL DEFAULT '',
  `default_warehouse` varchar(31) NOT NULL DEFAULT '',
  `irnpool` varchar(31) NOT NULL DEFAULT '',
  `state_license_number` VARCHAR(25) DEFAULT NULL,
  `weno_prov_id` VARCHAR(15) DEFAULT NULL,
  `newcrop_user_role` VARCHAR(30) DEFAULT NULL,
  `cpoe` tinyint(1) NULL DEFAULT NULL,
  `physician_type` VARCHAR(50) DEFAULT NULL,
  `main_menu_role` VARCHAR(50) NOT NULL DEFAULT 'standard',
  `patient_menu_role` VARCHAR(50) NOT NULL DEFAULT 'standard',
  `portal_user` tinyint(1) NOT NULL DEFAULT '0',
  `supervisor_id` int(11) NOT NULL DEFAULT '0',
  `billing_facility` TEXT,
  `billing_facility_id` INT(11) NOT NULL DEFAULT '0',
  `date_created` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `last_updated` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY  (`id`),
  UNIQUE KEY `uuid` (`uuid`),
  KEY `abook_type` (`abook_type`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

--
-- Inserting data for table `users`
--
-- NOTE THIS IS DONE AFTER INSTALLATION WHERE THE sql/official_additional_users.sql script is called durig setup
--  (so these inserts can be found in the sql/official_additional_users.sql script)
--

-- --------------------------------------------------------

--
-- Table structure for table `user_secure`
--

DROP TABLE IF EXISTS `users_secure`;
CREATE TABLE `users_secure` (
  `id` bigint(20) NOT NULL,
  `username` varchar(255) DEFAULT NULL,
  `password` varchar(255),
  `last_update_password` datetime DEFAULT NULL,
  `last_update` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `password_history1` varchar(255),
  `password_history2` varchar(255),
  `password_history3` varchar(255),
  `password_history4` varchar(255),
  `last_challenge_response` datetime DEFAULT NULL,
  `login_work_area` text,
  `total_login_fail_counter` bigint DEFAULT 0,
  `login_fail_counter` INT(11) DEFAULT '0',
  `last_login_fail` datetime DEFAULT NULL,
  `auto_block_emailed` tinyint DEFAULT 0,
  PRIMARY KEY (`id`),
  UNIQUE KEY `USERNAME_ID` (`id`,`username`)
) ENGINE=InnoDb;

-- --------------------------------------------------------

--
-- Table structure for table `user_settings`
--

DROP TABLE IF EXISTS `user_settings`;
CREATE TABLE `user_settings` (
  `setting_user`  bigint(20)   NOT NULL DEFAULT 0,
  `setting_label` varchar(100)  NOT NULL,
  `setting_value` varchar(255) NOT NULL DEFAULT '',
  PRIMARY KEY (`setting_user`, `setting_label`)
) ENGINE=InnoDB;

--
-- Inserting data for table `user_settings`
--

-- --------------------------------------------------------

--
-- Table structure for table `uuid_mapping`
--

DROP TABLE IF EXISTS `uuid_mapping`;
CREATE TABLE `uuid_mapping` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `uuid` binary(16) NOT NULL DEFAULT '',
  `resource` varchar(255) NOT NULL DEFAULT '',
  `resource_path` VARCHAR(255) DEFAULT NULL,
  `table` varchar(255) NOT NULL DEFAULT '',
  `target_uuid` binary(16) NOT NULL DEFAULT '',
  `created` timestamp NULL,
  PRIMARY KEY (`id`),
  KEY `uuid` (`uuid`),
  KEY `resource` (`resource`),
  KEY `table` (`table`),
  KEY `target_uuid` (`target_uuid`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `uuid_registry`
--

DROP TABLE IF EXISTS `uuid_registry`;
CREATE TABLE `uuid_registry` (
  `uuid` binary(16) NOT NULL DEFAULT '',
  `table_name` varchar(255) NOT NULL DEFAULT '',
  `table_id` varchar(255) NOT NULL DEFAULT '',
  `table_vertical` varchar(255) NOT NULL DEFAULT '',
  `couchdb` varchar(255) NOT NULL DEFAULT '',
  `document_drive` tinyint(4) NOT NULL DEFAULT '0',
  `mapped` tinyint(4) NOT NULL DEFAULT '0',
  `created` timestamp NULL,
  PRIMARY KEY (`uuid`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `validate_email`
--

DROP TABLE IF EXISTS `verify_email`;
CREATE TABLE `verify_email` (
  `id` bigint NOT NULL auto_increment,
  `pid_holder` bigint DEFAULT NULL,
  `email` varchar(255) DEFAULT NULL,
  `language` varchar(100) DEFAULT NULL,
  `fname` varchar(255) DEFAULT NULL,
  `mname` varchar(255) DEFAULT NULL,
  `lname` varchar(255) DEFAULT NULL,
  `dob` date DEFAULT NULL,
  `token_onetime`  VARCHAR(255) DEFAULT NULL,
  `active` tinyint NOT NULL default 1,
  PRIMARY KEY (`id`),
  UNIQUE KEY (`email`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `voids`
--

DROP TABLE IF EXISTS `voids`;
CREATE TABLE `voids` (
  `void_id`                bigint(20)    NOT NULL AUTO_INCREMENT,
  `patient_id`             bigint(20)    NOT NULL            COMMENT 'references patient_data.pid',
  `encounter_id`           bigint(20)    NOT NULL DEFAULT 0  COMMENT 'references form_encounter.encounter',
  `what_voided`            varchar(31)   NOT NULL            COMMENT 'checkout,receipt and maybe other options later',
  `date_original`          datetime      DEFAULT NULL        COMMENT 'time of original action that is now voided',
  `date_voided`            datetime      NOT NULL            COMMENT 'time of void action',
  `user_id`                bigint(20)    NOT NULL            COMMENT 'references users.id',
  `amount1`                decimal(12,2) NOT NULL DEFAULT 0  COMMENT 'for checkout,receipt total voided adjustments',
  `amount2`                decimal(12,2) NOT NULL DEFAULT 0  COMMENT 'for checkout,receipt total voided payments',
  `other_info`             text                              COMMENT 'for checkout,receipt the old invoice refno',
  `reason`                 VARCHAR(31)   default '',
  `notes`                  VARCHAR(255)  default '',
  PRIMARY KEY (`void_id`),
  KEY datevoided (date_voided),
  KEY pidenc (patient_id, encounter_id)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `x12_partners`
--

DROP TABLE IF EXISTS `x12_partners`;
CREATE TABLE `x12_partners` (
  `id` int(11) NOT NULL default '0',
  `name` varchar(255) default NULL,
  `id_number` varchar(255) default NULL,
  `x12_sender_id` varchar(255) default NULL,
  `x12_receiver_id` varchar(255) default NULL,
  `processing_format` enum('standard','medi-cal','cms','proxymed','oa_eligibility','availity_eligibility') default NULL,
  `x12_isa01` VARCHAR( 2 ) NOT NULL DEFAULT '00' COMMENT 'User logon Required Indicator',
  `x12_isa02` VARCHAR( 10 ) NOT NULL DEFAULT '          ' COMMENT 'User Logon',
  `x12_isa03` VARCHAR( 2 ) NOT NULL DEFAULT '00' COMMENT 'User password required Indicator',
  `x12_isa04` VARCHAR( 10 ) NOT NULL DEFAULT '          ' COMMENT 'User Password',
  `x12_isa05` char(2)     NOT NULL DEFAULT 'ZZ',
  `x12_isa07` char(2)     NOT NULL DEFAULT 'ZZ',
  `x12_isa14` char(1)     NOT NULL DEFAULT '0',
  `x12_isa15` char(1)     NOT NULL DEFAULT 'P',
  `x12_gs02`  varchar(15) NOT NULL DEFAULT '',
  `x12_per06` varchar(80) NOT NULL DEFAULT '',
  `x12_dtp03` char(1)     NOT NULL DEFAULT 'A',
  `x12_gs03` varchar(15) DEFAULT NULL,
  `x12_submitter_id` smallint(6) DEFAULT NULL,
  `x12_submitter_name` varchar(255) DEFAULT NULL,
  `x12_sftp_login` varchar(255) DEFAULT NULL,
  `x12_sftp_pass` varchar(255) DEFAULT NULL,
  `x12_sftp_host` varchar(255) DEFAULT NULL,
  `x12_sftp_port` varchar(255) DEFAULT NULL,
  `x12_sftp_local_dir` varchar(255) DEFAULT NULL,
  `x12_sftp_remote_dir` varchar(255) DEFAULT NULL,
  `x12_token_endpoint` tinytext,
  `x12_eligibility_endpoint` tinytext,
  `x12_claim_status_endpoint` tinytext,
  `x12_attachment_endpoint` tinytext,
  `x12_client_id` tinytext,
  `x12_client_secret` tinytext,
  PRIMARY KEY  (`id`)
) ENGINE=InnoDB;

-- -----------------------------------------------------------------------------------

--
-- Table structure for table `automatic_notification`
--

DROP TABLE IF EXISTS `automatic_notification`;
CREATE TABLE `automatic_notification` (
  `notification_id` int(5) NOT NULL auto_increment,
  `sms_gateway_type` varchar(255) NOT NULL,
  `provider_name` varchar(100) NOT NULL,
  `message` text,
  `email_sender` varchar(100) NOT NULL,
  `email_subject` varchar(100) NOT NULL,
  `type` enum('SMS','Email') NOT NULL default 'SMS',
  PRIMARY KEY  (`notification_id`)
) ENGINE=InnoDB AUTO_INCREMENT=3;

--
-- Inserting data for table `automatic_notification`
--

-- --------------------------------------------------------

--
-- Table structure for table `notification_log`
--

DROP TABLE IF EXISTS `notification_log`;
CREATE TABLE `notification_log` (
  `iLogId` int(11) NOT NULL auto_increment,
  `pid` bigint(20) NOT NULL,
  `pc_eid` int(11) unsigned NULL,
  `sms_gateway_type` varchar(50) NOT NULL,
  `smsgateway_info` varchar(255) NOT NULL,
  `message` text,
  `email_sender` varchar(255) NOT NULL,
  `email_subject` varchar(255) NOT NULL,
  `type` enum('SMS','Email') NOT NULL,
  `patient_info` text,
  `pc_eventDate` date NOT NULL,
  `pc_endDate` date NOT NULL,
  `pc_startTime` time NOT NULL,
  `pc_endTime` time NOT NULL,
  `dSentDateTime` datetime NOT NULL,
  PRIMARY KEY  (`iLogId`)
) ENGINE=InnoDB AUTO_INCREMENT=5;

-- --------------------------------------------------------

--
-- Table structure for table `notification_settings`
--

DROP TABLE IF EXISTS `notification_settings`;
CREATE TABLE `notification_settings` (
  `SettingsId` int(3) NOT NULL auto_increment,
  `Send_SMS_Before_Hours` int(3) NOT NULL,
  `Send_Email_Before_Hours` int(3) NOT NULL,
  `SMS_gateway_username` varchar(100) NOT NULL,
  `SMS_gateway_password` varchar(100) NOT NULL,
  `SMS_gateway_apikey` varchar(100) NOT NULL,
  `type` varchar(50) NOT NULL,
  PRIMARY KEY  (`SettingsId`)
) ENGINE=InnoDB AUTO_INCREMENT=2;

--
-- Inserting data for table `notification_settings`
--

-- -------------------------------------------------------------------

--
-- Table structure for table `chart_tracker`
--

DROP TABLE IF EXISTS `chart_tracker`;
CREATE TABLE chart_tracker (
  ct_pid            int(11)       NOT NULL,
  ct_when           datetime      NOT NULL,
  ct_userid         bigint(20)    NOT NULL DEFAULT 0,
  ct_location       varchar(31)   NOT NULL DEFAULT '',
  PRIMARY KEY (ct_pid, ct_when)
) ENGINE=InnoDB;

-- -----------------------------------------------------------------------------------

--
-- Table structure for table `ar_session`
--

DROP TABLE IF EXISTS `ar_session`;
CREATE TABLE ar_session (
  session_id     int unsigned  NOT NULL AUTO_INCREMENT,
  payer_id       int(11)       NOT NULL            COMMENT '0=pt else references insurance_companies.id',
  user_id        int(11)       NOT NULL            COMMENT 'references users.id for session owner',
  closed         tinyint(1)    NOT NULL DEFAULT 0  COMMENT '0=no, 1=yes',
  reference      varchar(255)  NOT NULL DEFAULT '' COMMENT 'check or EOB number',
  check_date     date          DEFAULT NULL,
  deposit_date   date          DEFAULT NULL,
  pay_total      decimal(12,2) NOT NULL DEFAULT 0,
  created_time timestamp NOT NULL default CURRENT_TIMESTAMP,
  modified_time datetime NOT NULL,
  global_amount decimal( 12, 2 ) NOT NULL ,
  payment_type varchar( 50 ) NOT NULL ,
  description text,
  adjustment_code varchar( 50 ) NOT NULL ,
  post_to_date date NOT NULL ,
  patient_id bigint(20) NOT NULL,
  payment_method varchar( 25 ) NOT NULL,
  PRIMARY KEY (session_id),
  KEY user_closed (user_id, closed),
  KEY deposit_date (deposit_date)
) ENGINE=InnoDB;

-- -----------------------------------------------------------------------------------

--
-- Table structure for table `ar_activity`
--

DROP TABLE IF EXISTS `ar_activity`;
CREATE TABLE ar_activity (
  pid            int(11)       NOT NULL,
  encounter      int(11)       NOT NULL,
  sequence_no    int unsigned  NOT NULL            COMMENT 'Ar_activity sequence_no, incremented in code',
  `code_type`    varchar(12)   NOT NULL DEFAULT '',
  code           varchar(20)   NOT NULL            COMMENT 'empty means claim level',
  modifier       varchar(12)   NOT NULL DEFAULT '',
  payer_type     int           NOT NULL            COMMENT '0=pt, 1=ins1, 2=ins2, etc',
  post_time      datetime      NOT NULL,
  post_user      int(11)       NOT NULL            COMMENT 'references users.id',
  session_id     int unsigned  NOT NULL            COMMENT 'references ar_session.session_id',
  memo           varchar(255)  NOT NULL DEFAULT '' COMMENT 'adjustment reasons go here',
  pay_amount     decimal(12,2) NOT NULL DEFAULT 0  COMMENT 'either pay or adj will always be 0',
  adj_amount     decimal(12,2) NOT NULL DEFAULT 0,
  modified_time datetime NOT NULL,
  follow_up char(1) NOT NULL,
  follow_up_note text,
  account_code varchar(15) NOT NULL,
  reason_code varchar(255) DEFAULT NULL COMMENT 'Use as needed to show the primary payer adjustment reason code',
  deleted        datetime DEFAULT NULL COMMENT 'NULL if active, otherwise when voided',
  post_date      date DEFAULT NULL COMMENT 'Posting date if specified at payment time',
  payer_claim_number varchar(30) DEFAULT NULL,
  PRIMARY KEY (pid, encounter, sequence_no),
  KEY session_id (session_id)
) ENGINE=InnoDB;

-- -----------------------------------------------------------------------------------

--
-- Table structure for table `users_facility`
--

DROP TABLE IF EXISTS `users_facility`;
CREATE TABLE `users_facility` (
  `tablename` varchar(64) NOT NULL,
  `table_id` int(11) NOT NULL,
  `facility_id` int(11) NOT NULL,
  `warehouse_id` varchar(31) NOT NULL DEFAULT '',
  PRIMARY KEY (`tablename`,`table_id`,`facility_id`,`warehouse_id`)
) ENGINE=InnoDB COMMENT='joins users or patient_data to facility table';

-- -----------------------------------------------------------------------------------

--
-- Table structure for table `lbf_data`
--

DROP TABLE IF EXISTS `lbf_data`;
CREATE TABLE `lbf_data` (
  `form_id`     int(11)      NOT NULL AUTO_INCREMENT COMMENT 'references forms.form_id',
  `field_id`    varchar(31)  NOT NULL COMMENT 'references layout_options.field_id',
  `field_value` LONGTEXT,
  PRIMARY KEY (`form_id`,`field_id`)
) ENGINE=InnoDB COMMENT='contains all data from layout-based forms';

-- -----------------------------------------------------------------------------------

--
-- Table structure for table `lbt_data`
--

DROP TABLE IF EXISTS `lbt_data`;
CREATE TABLE `lbt_data` (
  `form_id`     bigint(20)   NOT NULL COMMENT 'references transactions.id',
  `field_id`    varchar(31)  NOT NULL COMMENT 'references layout_options.field_id',
  `field_value` TEXT,
  PRIMARY KEY (`form_id`,`field_id`)
) ENGINE=InnoDB COMMENT='contains all data from layout-based transactions';

-- -----------------------------------------------------------------------------------

--
-- Table structure for table `gprelations`
--

DROP TABLE IF EXISTS `gprelations`;
CREATE TABLE gprelations (
  type1 int(2)     NOT NULL,
  id1   bigint(20) NOT NULL,
  type2 int(2)     NOT NULL,
  id2   bigint(20) NOT NULL,
  PRIMARY KEY (type1,id1,type2,id2),
  KEY key2  (type2,id2)
) ENGINE=InnoDB COMMENT='general purpose relations';

-- -----------------------------------------------------------------------------------

--
-- Table structure for table `procedure_providers`
--

DROP TABLE IF EXISTS `procedure_providers`;
CREATE TABLE `procedure_providers` (
  `ppid`         bigint(20)   NOT NULL auto_increment,
  `uuid`         binary(16)   DEFAULT NULL,
  `name`         varchar(255) NOT NULL DEFAULT '',
  `npi`          varchar(15)  NOT NULL DEFAULT '',
  `send_app_id`  varchar(255) NOT NULL DEFAULT ''  COMMENT 'Sending application ID (MSH-3.1)',
  `send_fac_id`  varchar(255) NOT NULL DEFAULT ''  COMMENT 'Sending facility ID (MSH-4.1)',
  `recv_app_id`  varchar(255) NOT NULL DEFAULT ''  COMMENT 'Receiving application ID (MSH-5.1)',
  `recv_fac_id`  varchar(255) NOT NULL DEFAULT ''  COMMENT 'Receiving facility ID (MSH-6.1)',
  `DorP`         char(1)      NOT NULL DEFAULT 'D' COMMENT 'Debugging or Production (MSH-11)',
  `direction`    char(1)      NOT NULL DEFAULT 'B' COMMENT 'Bidirectional or Results-only',
  `protocol`     varchar(15)  NOT NULL DEFAULT 'DL',
  `remote_host`  varchar(255) NOT NULL DEFAULT '',
  `login`        varchar(255) NOT NULL DEFAULT '',
  `password`     varchar(255) NOT NULL DEFAULT '',
  `orders_path`  varchar(255) NOT NULL DEFAULT '',
  `results_path` varchar(255) NOT NULL DEFAULT '',
  `notes`        text,
  `lab_director` bigint(20)   NOT NULL DEFAULT '0',
  `active`       tinyint(1)   NOT NULL DEFAULT '1',
  `type`         varchar(31)  DEFAULT NULL,
  `date_created` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `last_updated` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`ppid`),
  UNIQUE KEY `uuid` (`uuid`)
) ENGINE=InnoDB;

-- -----------------------------------------------------------------------------------

--
-- Table structure for table `procedure_type`
--

DROP TABLE IF EXISTS `procedure_type`;
CREATE TABLE `procedure_type` (
  `procedure_type_id`   bigint(20)   NOT NULL AUTO_INCREMENT,
  `parent`              bigint(20)   NOT NULL DEFAULT 0  COMMENT 'references procedure_type.procedure_type_id',
  `name`                varchar(63)  NOT NULL DEFAULT '' COMMENT 'name for this category, procedure or result type',
  `lab_id`              bigint(20)   NOT NULL DEFAULT 0  COMMENT 'references procedure_providers.ppid, 0 means default to parent',
  `procedure_code`      varchar(64)  NOT NULL DEFAULT '' COMMENT 'code identifying this procedure',
  `procedure_type`      varchar(31)  NOT NULL DEFAULT '' COMMENT 'see list proc_type',
  `body_site`           varchar(31)  NOT NULL DEFAULT '' COMMENT 'where to do injection, e.g. arm, buttock',
  `specimen`            varchar(31)  NOT NULL DEFAULT '' COMMENT 'blood, urine, saliva, etc.',
  `route_admin`         varchar(31)  NOT NULL DEFAULT '' COMMENT 'oral, injection',
  `laterality`          varchar(31)  NOT NULL DEFAULT '' COMMENT 'left, right, ...',
  `description`         varchar(255) NOT NULL DEFAULT '' COMMENT 'descriptive text for procedure_code',
  `standard_code`       varchar(255) NOT NULL DEFAULT '' COMMENT 'industry standard code type and code (e.g. CPT4:12345)',
  `related_code`        varchar(255) NOT NULL DEFAULT '' COMMENT 'suggested code(s) for followup services if result is abnormal',
  `units`               varchar(31)  NOT NULL DEFAULT '' COMMENT 'default for procedure_result.units',
  `range`               varchar(255) NOT NULL DEFAULT '' COMMENT 'default for procedure_result.range',
  `seq`                 int(11)      NOT NULL default 0  COMMENT 'sequence number for ordering',
  `activity`            tinyint(1)   NOT NULL default 1  COMMENT '1=active, 0=inactive',
  `notes`               varchar(255) NOT NULL default '' COMMENT 'additional notes to enhance description',
  `transport`           varchar(31)  DEFAULT NULL,
  `procedure_type_name` varchar(64)  NULL,
  PRIMARY KEY (`procedure_type_id`),
  KEY parent (parent),
  KEY `ptype_procedure_code` (`procedure_code`)
) ENGINE=InnoDB;

-- -----------------------------------------------------------------------------------

--
-- Table structure for table `procedure_questions`
--

DROP TABLE IF EXISTS `procedure_questions`;
CREATE TABLE `procedure_questions` (
  `lab_id`              bigint(20)   NOT NULL DEFAULT 0   COMMENT 'references procedure_providers.ppid to identify the lab',
  `procedure_code`      varchar(31)  NOT NULL DEFAULT ''  COMMENT 'references procedure_type.procedure_code to identify this order type',
  `question_code`       varchar(31)  NOT NULL DEFAULT ''  COMMENT 'code identifying this question',
  `seq`                 int(11)      NOT NULL default 0   COMMENT 'sequence number for ordering',
  `question_text`       varchar(255) NOT NULL DEFAULT ''  COMMENT 'descriptive text for question_code',
  `required`            tinyint(1)   NOT NULL DEFAULT 0   COMMENT '1 = required, 0 = not',
  `maxsize`             int          NOT NULL DEFAULT 0   COMMENT 'maximum length if text input field',
  `fldtype`             char(1)      NOT NULL DEFAULT 'T' COMMENT 'Text, Number, Select, Multiselect, Date, Gestational-age',
  `options`             text                              COMMENT 'choices for fldtype S and T',
  `tips`                varchar(255) NOT NULL DEFAULT ''  COMMENT 'Additional instructions for answering the question',
  `activity`            tinyint(1)   NOT NULL DEFAULT 1   COMMENT '1 = active, 0 = inactive',
  PRIMARY KEY (`lab_id`, `procedure_code`, `question_code`)
) ENGINE=InnoDB;

-- -----------------------------------------------------------------------------------

--
-- Table structure for table `procedure_order`
--

DROP TABLE IF EXISTS `procedure_order`;
CREATE TABLE `procedure_order` (
  `procedure_order_id`     bigint(20)       NOT NULL AUTO_INCREMENT,
  `uuid`                   binary(16)       DEFAULT NULL,
  `provider_id`            bigint(20)       NOT NULL DEFAULT 0  COMMENT 'references users.id, the ordering provider',
  `patient_id`             bigint(20)       NOT NULL            COMMENT 'references patient_data.pid',
  `encounter_id`           bigint(20)       NOT NULL DEFAULT 0  COMMENT 'references form_encounter.encounter',
  `date_collected`         datetime         DEFAULT NULL        COMMENT 'time specimen collected',
  `date_ordered`           datetime         DEFAULT NULL,
  `order_priority`         varchar(31)      NOT NULL DEFAULT '',
  `order_status`           varchar(31)      NOT NULL DEFAULT '' COMMENT 'pending,routed,complete,canceled',
  `patient_instructions`   text,
  `activity`               tinyint(1)       NOT NULL DEFAULT 1  COMMENT '0 if deleted',
  `control_id`             varchar(255)     NOT NULL DEFAULT '' COMMENT 'This is the CONTROL ID that is sent back from lab',
  `lab_id`                 bigint(20)       NOT NULL DEFAULT 0  COMMENT 'references procedure_providers.ppid',
  `specimen_type`          varchar(31)      NOT NULL DEFAULT '' COMMENT 'from the Specimen_Type list',
  `specimen_location`      varchar(31)      NOT NULL DEFAULT '' COMMENT 'from the Specimen_Location list',
  `specimen_volume`        varchar(30)      NOT NULL DEFAULT '' COMMENT 'from a text input field',
  `date_transmitted`       datetime         DEFAULT NULL        COMMENT 'time of order transmission, null if unsent',
  `clinical_hx`            varchar(255)     NOT NULL DEFAULT '' COMMENT 'clinical history text that may be relevant to the order',
  `external_id`            varchar(20)      DEFAULT NULL,
  `history_order`          enum('0','1')    DEFAULT '0'         COMMENT 'references order is added for history purpose only.',
  `order_diagnosis`        varchar(255)     DEFAULT ''          COMMENT 'primary order diagnosis',
  `billing_type`           varchar(4)       DEFAULT NULL,
  `specimen_fasting`       varchar(31)      DEFAULT NULL,
  `order_psc`              tinyint(4)       DEFAULT NULL,
  `order_abn`              varchar(31)      NOT NULL DEFAULT 'not_required',
  `collector_id`           bigint(11)       NOT NULL DEFAULT 0,
  `account`                varchar(60)      DEFAULT NULL,
  `account_facility`       int(11)          DEFAULT NULL,
  `provider_number`        varchar(30)      DEFAULT NULL,
  `procedure_order_type`   varchar(32)      NOT NULL DEFAULT 'laboratory_test',
  `scheduled_date` datetime DEFAULT NULL COMMENT 'Scheduled date for service (FHIR occurrence[x])',
  `scheduled_start` datetime DEFAULT NULL COMMENT 'Scheduled start time (FHIR occurrencePeriod.start)',
  `scheduled_end` datetime DEFAULT NULL COMMENT 'Scheduled end time (FHIR occurrencePeriod.end)',
  `performer_type` varchar(50) DEFAULT NULL COMMENT 'Type of performer: laboratory, radiology, pathology (SNOMED CT)',
  `order_intent` varchar(31) NOT NULL DEFAULT 'order' COMMENT 'FHIR intent: order, plan, directive, proposal',
  `location_id` int(11) DEFAULT NULL COMMENT 'References facility.id for service location (FHIR locationReference)',
  PRIMARY KEY (`procedure_order_id`),
  UNIQUE KEY `uuid` (`uuid`),
  KEY `datepid` (`date_ordered`,`patient_id`),
  KEY `patient_id` (`patient_id`),
  KEY `idx_specimen_type` (`specimen_type`),
  KEY `idx_scheduled_date` (`scheduled_date`),
  KEY `idx_order_intent` (`order_intent`),
  KEY `idx_location_id` (`location_id`)
) ENGINE=InnoDB;

-- -----------------------------------------------------------------------------------

--
-- Table structure for table `procedure_order_code`
--

DROP TABLE IF EXISTS `procedure_order_code`;
CREATE TABLE `procedure_order_code` (
  `procedure_order_id`      bigint(20)  NOT NULL                COMMENT 'references procedure_order.procedure_order_id',
  `procedure_order_seq`     int(11)     NOT NULL COMMENT 'Supports multiple tests per order. Procedure_order_seq, incremented in code',
  `procedure_code`          varchar(64) NOT NULL DEFAULT ''     COMMENT 'like procedure_type.procedure_code',
  `procedure_name`          varchar(255) NOT NULL DEFAULT ''    COMMENT 'descriptive name of the procedure code',
  `procedure_source`        char(1)     NOT NULL DEFAULT '1'    COMMENT '1=original order, 2=added after order sent',
  `diagnoses`               text                                COMMENT 'diagnoses and maybe other coding (e.g. ICD9:111.11)',
  `do_not_send`             tinyint(1)  NOT NULL DEFAULT '0'    COMMENT '0 = normal, 1 = do not transmit to lab',
  `procedure_order_title`   varchar( 255 ) NULL DEFAULT NULL,
  `procedure_type`          varchar(31) DEFAULT NULL,
  `transport`               varchar(31) DEFAULT NULL,
  `date_end` datetime DEFAULT NULL,
  `reason_code` varchar(31) DEFAULT NULL,
  `reason_description` text,
  `reason_date_low` datetime DEFAULT NULL,
  `reason_date_high` datetime DEFAULT NULL,
  `reason_status` varchar(31) DEFAULT NULL,
  PRIMARY KEY (`procedure_order_id`, `procedure_order_seq`)
) ENGINE=InnoDB;

-- -----------------------------------------------------------------------------------

--
-- Table structure for table `procedure_answers`
--

DROP TABLE IF EXISTS `procedure_answers`;
CREATE TABLE `procedure_answers` (
  `procedure_order_id`  bigint(20)   NOT NULL DEFAULT 0  COMMENT 'references procedure_order.procedure_order_id',
  `procedure_order_seq` int(11)      NOT NULL DEFAULT 0  COMMENT 'references procedure_order_code.procedure_order_seq',
  `question_code`       varchar(31)  NOT NULL DEFAULT '' COMMENT 'references procedure_questions.question_code',
  `answer_seq`          int(11)      NOT NULL COMMENT 'supports multiple-choice questions. answer_seq, incremented in code',
  `answer`              varchar(255) NOT NULL DEFAULT '' COMMENT 'answer data',
  `procedure_code`      varchar(31)  DEFAULT NULL,
  PRIMARY KEY (`procedure_order_id`, `procedure_order_seq`, `question_code`, `answer_seq`)
) ENGINE=InnoDB;

-- -----------------------------------------------------------------------------------

--
-- Table structure for table `procedure_report`
--

DROP TABLE IF EXISTS `procedure_report`;
CREATE TABLE `procedure_report` (
  `procedure_report_id` bigint(20)     NOT NULL AUTO_INCREMENT,
  `uuid`                binary(16)     DEFAULT NULL,
  `procedure_order_id`  bigint(20)     DEFAULT NULL   COMMENT 'references procedure_order.procedure_order_id',
  `procedure_order_seq` int(11)        NOT NULL DEFAULT 1  COMMENT 'references procedure_order_code.procedure_order_seq',
  `date_collected`      datetime       DEFAULT NULL,
  `date_collected_tz`   varchar(5)     DEFAULT ''          COMMENT '+-hhmm offset from UTC',
  `date_report`         datetime       DEFAULT NULL,
  `date_report_tz`      varchar(5)     DEFAULT ''          COMMENT '+-hhmm offset from UTC',
  `source`              bigint(20)     NOT NULL DEFAULT 0  COMMENT 'references users.id, who entered this data',
  `specimen_num`        varchar(63)    NOT NULL DEFAULT '',
  `report_status`       varchar(31)    NOT NULL DEFAULT '' COMMENT 'received,complete,error',
  `review_status`       varchar(31)    NOT NULL DEFAULT 'received' COMMENT 'pending review status: received,reviewed',
  `report_notes`        text           COMMENT 'notes from the lab',
  PRIMARY KEY (`procedure_report_id`),
  KEY procedure_order_id (procedure_order_id),
  UNIQUE KEY `uuid` (`uuid`)
) ENGINE=InnoDB;

-- -----------------------------------------------------------------------------------

--
-- Table structure for table `procedure_result`
--

DROP TABLE IF EXISTS `procedure_result`;
CREATE TABLE `procedure_result` (
  `procedure_result_id` bigint(20)   NOT NULL AUTO_INCREMENT,
  `uuid`                binary(16)   DEFAULT NULL,
  `procedure_report_id` bigint(20)   NOT NULL            COMMENT 'references procedure_report.procedure_report_id',
  `result_data_type`    char(1)      NOT NULL DEFAULT 'S' COMMENT 'N=Numeric, S=String, F=Formatted, E=External, L=Long text as first line of comments',
  `result_code`         varchar(31)  NOT NULL DEFAULT '' COMMENT 'LOINC code, might match a procedure_type.procedure_code',
  `result_text`         varchar(255) NOT NULL DEFAULT '' COMMENT 'Description of result_code',
  `date`                datetime     DEFAULT NULL        COMMENT 'lab-provided date specific to this result',
  `facility`            varchar(255) NOT NULL DEFAULT '' COMMENT 'lab-provided testing facility ID',
  `units`               varchar(31)  NOT NULL DEFAULT '',
  `result`              varchar(255) NOT NULL DEFAULT '',
  `range`               varchar(255) NOT NULL DEFAULT '',
  `abnormal`            varchar(31)  NOT NULL DEFAULT '' COMMENT 'no,yes,high,low',
  `comments`            text                             COMMENT 'comments from the lab',
  `document_id`         bigint(20)   NOT NULL DEFAULT 0  COMMENT 'references documents.id if this result is a document',
  `result_status`       varchar(31)  NOT NULL DEFAULT '' COMMENT 'preliminary, cannot be done, final, corrected, incomplete...etc.',
  `date_end`            datetime     DEFAULT NULL        COMMENT 'lab-provided end date specific to this result',
  PRIMARY KEY (`procedure_result_id`),
  UNIQUE KEY `uuid` (`uuid`),
  KEY procedure_report_id (procedure_report_id)
) ENGINE=InnoDB;

-- -----------------------------------------------------------------------------------

--
-- Table structure for table `procedure_specimen`
--

DROP TABLE IF EXISTS `procedure_specimen`;
CREATE TABLE `procedure_specimen` (
  `procedure_specimen_id` BIGINT(20) NOT NULL AUTO_INCREMENT COMMENT 'record id',
  `uuid` binary(16) DEFAULT NULL COMMENT 'FHIR Specimen id',
  `procedure_order_id` BIGINT(20) NOT NULL COMMENT 'links to procedure_order.procedure_order_id',
  `procedure_order_seq` INT(11) NOT NULL COMMENT 'links to procedure_order_code.procedure_order_seq (per test line)',
  `specimen_identifier` VARCHAR(128) DEFAULT NULL COMMENT 'tube/barcode/internal id',
  `accession_identifier` VARCHAR(128) DEFAULT NULL COMMENT 'lab accession number',
  `specimen_type_code` VARCHAR(64) DEFAULT NULL COMMENT 'prefer SNOMED CT code',
  `specimen_type` VARCHAR(255) DEFAULT NULL COMMENT 'display/text',
  `collection_method_code` VARCHAR(64) DEFAULT NULL,
  `collection_method` VARCHAR(255) DEFAULT NULL,
  `specimen_location_code` VARCHAR(64) DEFAULT NULL,
  `specimen_location` VARCHAR(255) DEFAULT NULL,
  `collected_date` DATETIME DEFAULT NULL COMMENT 'single instant',
  `collection_date_low` DATETIME DEFAULT NULL COMMENT 'period start',
  `collection_date_high` DATETIME DEFAULT NULL COMMENT 'period end',
  `volume_value` DECIMAL(10,3) DEFAULT NULL,
  `volume_unit` VARCHAR(32) DEFAULT 'mL',
  `condition_code` VARCHAR(32) DEFAULT NULL COMMENT 'HL7 v2 0493 (e.g., ACT, HEM)',
  `specimen_condition` VARCHAR(64) DEFAULT NULL,
  `comments` TEXT,
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `created_by` BIGINT(20) DEFAULT NULL,
  `updated_by` BIGINT(20) DEFAULT NULL,
   `deleted` TINYINT(1) DEFAULT 0,
  PRIMARY KEY (`procedure_specimen_id`),
  UNIQUE KEY `uuid_unique` (`uuid`),
  KEY `idx_order_line` (`procedure_order_id`,`procedure_order_seq`),
  KEY `idx_identifier` (`specimen_identifier`),
  KEY `idx_accession` (`accession_identifier`)
) ENGINE=InnoDB;

-- ------------------------------------------------------------------------

--
-- Table structure for table `procedure_order_relationships`
--

DROP TABLE IF EXISTS `procedure_order_relationships`;
CREATE TABLE `procedure_order_relationships` (
 `id` INT AUTO_INCREMENT PRIMARY KEY,
 `procedure_order_id` BIGINT(20) NOT NULL COMMENT 'Links to procedure_order.procedure_order_id',
 `resource_type` VARCHAR(50) NOT NULL COMMENT 'FHIR resource type (Observation, Condition, etc.)',
 `resource_uuid` BINARY(16) NOT NULL COMMENT 'UUID of the related resource',
 `relationship` VARCHAR(50) DEFAULT NULL COMMENT 'Type of relationship',
 `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
 `created_by` BIGINT(20) DEFAULT NULL COMMENT 'User who created this link',
 INDEX `idx_order_id` (`procedure_order_id`),
 INDEX `idx_resource` (`resource_type`, `resource_uuid`),
 INDEX `idx_created_at` (`created_at`)
) ENGINE=InnoDB COMMENT='Links ServiceRequests to supporting clinical information';

-- -----------------------------------------------------------------------------------

--
-- Table structure for table `globals`
--

DROP TABLE IF EXISTS `globals`;
CREATE TABLE `globals` (
  `gl_name`             varchar(63)    NOT NULL,
  `gl_index`            int(11)        NOT NULL DEFAULT 0,
  `gl_value`            varchar(255)   NOT NULL DEFAULT '',
  PRIMARY KEY (`gl_name`, `gl_index`)
) ENGINE=InnoDB;

-- -----------------------------------------------------------------------------------

--
-- Table structure for table `code_types`
--

DROP TABLE IF EXISTS `code_types`;
CREATE TABLE code_types (
  ct_key  varchar(15) NOT NULL           COMMENT 'short alphanumeric name',
  ct_id   int(11)     UNIQUE NOT NULL    COMMENT 'numeric identifier',
  ct_seq  int(11)     NOT NULL DEFAULT 0 COMMENT 'sort order',
  ct_mod  int(11)     NOT NULL DEFAULT 0 COMMENT 'length of modifier field',
  ct_just varchar(15) NOT NULL DEFAULT ''COMMENT 'ct_key of justify type, if any',
  ct_mask varchar(9)  NOT NULL DEFAULT ''COMMENT 'formatting mask for code values',
  ct_fee  tinyint(1)  NOT NULL default 0 COMMENT '1 if fees are used',
  ct_rel  tinyint(1)  NOT NULL default 0 COMMENT '1 if can relate to other code types',
  ct_nofs tinyint(1)  NOT NULL default 0 COMMENT '1 if to be hidden in the fee sheet',
  ct_diag tinyint(1)  NOT NULL default 0 COMMENT '1 if this is a diagnosis type',
  ct_active tinyint(1) NOT NULL default 1 COMMENT '1 if this is active',
  ct_label varchar(31) NOT NULL default '' COMMENT 'label of this code type',
  ct_external tinyint(1) NOT NULL default 0 COMMENT '0 if stored codes in codes tables, 1 or greater if codes stored in external tables',
  ct_claim tinyint(1) NOT NULL default 0 COMMENT '1 if this is used in claims',
  ct_proc tinyint(1) NOT NULL default 0 COMMENT '1 if this is a procedure type',
  ct_term tinyint(1) NOT NULL default 0 COMMENT '1 if this is a clinical term',
  ct_problem tinyint(1) NOT NULL default 0 COMMENT '1 if this code type is used as a medical problem',
  ct_drug tinyint(1) NOT NULL default 0 COMMENT '1 if this code type is used as a medication',
  PRIMARY KEY (ct_key)
) ENGINE=InnoDB;

-- Race List
-- Ethnicity List
-- void reasons list

-- payment methods list

-- shift list

-- Charge categories list (Customers), used by IPPF checkout

-- list_options for `form_eye`

-- we leave in the old one in case people are still referring to it but we deactivate it
-- Insert Medication Usage Intent
-- Insert Medication Request Intent
-- Insert 2021 eCQM Reporting Measures
-- Insert encounter_type list
-- --------------------------------------------------------

--
-- Table structure for table `extended_log`
--

DROP TABLE IF EXISTS `extended_log`;
CREATE TABLE `extended_log` (
  `id` bigint(20) NOT NULL auto_increment,
  `date` datetime default NULL,
  `event` varchar(255) default NULL,
  `user` varchar(255) default NULL,
  `recipient` varchar(255) default NULL,
  `description` longtext,
  `patient_id` bigint(20) default NULL,
  PRIMARY KEY  (`id`),
  KEY `patient_id` (`patient_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `version`
--

DROP TABLE IF EXISTS `version`;
CREATE TABLE version (
  v_major    int(11)     NOT NULL DEFAULT 0,
  v_minor    int(11)     NOT NULL DEFAULT 0,
  v_patch    int(11)     NOT NULL DEFAULT 0,
  v_realpatch int(11)    NOT NULL DEFAULT 0,
  v_tag      varchar(31) NOT NULL DEFAULT '',
  v_database int(11)     NOT NULL DEFAULT 0,
  v_acl      int(11)     NOT NULL DEFAULT 0
) ENGINE=InnoDB;

--
-- Inserting data for table `version`
--

-- --------------------------------------------------------

--
-- Table structure for table `customlists`
--

DROP TABLE IF EXISTS `customlists`;
CREATE TABLE `customlists` (
  `cl_list_slno` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `cl_list_id` int(10) unsigned NOT NULL COMMENT 'ID OF THE lIST FOR NEW TAKE SELECT MAX(cl_list_id)+1',
  `cl_list_item_id` int(10) unsigned DEFAULT NULL COMMENT 'ID OF THE lIST FOR NEW TAKE SELECT MAX(cl_list_item_id)+1',
  `cl_list_type` int(10) unsigned NOT NULL COMMENT '0=>List Name 1=>list items 2=>Context 3=>Template 4=>Sentence 5=> SavedTemplate 6=>CustomButton',
  `cl_list_item_short` varchar(10) DEFAULT NULL,
  `cl_list_item_long` text,
  `cl_list_item_level` int(11) DEFAULT NULL COMMENT 'Flow level for List Designation',
  `cl_order` int(11) DEFAULT NULL,
  `cl_deleted` tinyint(1) DEFAULT '0',
  `cl_creator` int(11) DEFAULT NULL,
  PRIMARY KEY (`cl_list_slno`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

--
-- Inserting data for table `customlists`
--

-- --------------------------------------------------------

--
-- Table structure for table `template_users`
--

DROP TABLE IF EXISTS `template_users`;
CREATE TABLE `template_users` (
  `tu_id` int(11) NOT NULL AUTO_INCREMENT,
  `tu_user_id` int(11) DEFAULT NULL,
  `tu_facility_id` int(11) DEFAULT NULL,
  `tu_template_id` int(11) DEFAULT NULL,
  `tu_template_order` int(11) DEFAULT NULL,
  PRIMARY KEY (`tu_id`),
  UNIQUE KEY `templateuser` (`tu_user_id`,`tu_template_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `product_warehouse`
--

DROP TABLE IF EXISTS `product_warehouse`;
CREATE TABLE `product_warehouse` (
  `pw_drug_id`   int(11) NOT NULL,
  `pw_warehouse` varchar(31) NOT NULL,
  `pw_min_level` float       DEFAULT 0,
  `pw_max_level` float       DEFAULT 0,
  PRIMARY KEY  (`pw_drug_id`,`pw_warehouse`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `misc_address_book`
--

DROP TABLE IF EXISTS `misc_address_book`;
CREATE TABLE `misc_address_book` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `fname` varchar(255) DEFAULT NULL,
  `mname` varchar(255) DEFAULT NULL,
  `lname` varchar(255) DEFAULT NULL,
  `street` varchar(60) DEFAULT NULL,
  `city` varchar(30) DEFAULT NULL,
  `state` varchar(30) DEFAULT NULL,
  `zip` varchar(20) DEFAULT NULL,
  `phone` varchar(30) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `esign_signatures`
--

DROP TABLE IF EXISTS `esign_signatures`;
CREATE TABLE `esign_signatures` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `tid` int(11) NOT NULL COMMENT 'Table row ID for signature',
  `table` varchar(255) NOT NULL COMMENT 'table name for the signature',
  `uid` int(11) NOT NULL COMMENT 'user id for the signing user',
  `datetime` datetime NOT NULL COMMENT 'datetime of the signature action',
  `is_lock` tinyint(1) NOT NULL DEFAULT '0' COMMENT 'sig, lock or amendment',
  `amendment` text COMMENT 'amendment text, if any',
  `hash` varchar(255) NOT NULL COMMENT 'hash of signed data',
  `signature_hash` varchar(255) NOT NULL COMMENT 'hash of signature itself',
  PRIMARY KEY (`id`),
  KEY `tid` (`tid`),
  KEY `table` (`table`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `log_comment_encrypt`
--

DROP TABLE IF EXISTS `log_comment_encrypt`;
CREATE TABLE `log_comment_encrypt` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `log_id` int(11) NOT NULL,
  `encrypt` enum('Yes','No') NOT NULL DEFAULT 'No',
  `checksum` longtext,
  `checksum_api` longtext,
  `version` tinyint(4) NOT NULL DEFAULT '0' COMMENT '0 for mycrypt and 1 for openssl',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `shared_attributes`
--

DROP TABLE IF EXISTS `shared_attributes`;
CREATE TABLE `shared_attributes` (
  `pid`          bigint(20)   NOT NULL,
  `encounter`    bigint(20)   NOT NULL COMMENT '0 if patient attribute, else encounter attribute',
  `field_id`     varchar(31)  NOT NULL COMMENT 'references layout_options.field_id',
  `last_update`  datetime     NOT NULL COMMENT 'time of last update',
  `user_id`      bigint(20)   NOT NULL COMMENT 'user who last updated',
  `field_value`  TEXT,
  PRIMARY KEY (`pid`, `encounter`, `field_id`)
);

-- --------------------------------------------------------

--
-- Table structure for table `ccda_components`
--

DROP TABLE IF EXISTS `ccda_components`;
CREATE TABLE ccda_components (
  ccda_components_id int(11) NOT NULL AUTO_INCREMENT,
  ccda_components_field varchar(100) DEFAULT NULL,
  ccda_components_name varchar(100) DEFAULT NULL,
  ccda_type int(11) NOT NULL COMMENT '0=>sections,1=>components',
  PRIMARY KEY (ccda_components_id)
) ENGINE=InnoDB AUTO_INCREMENT=23;

--
-- Inserting data for table `ccda_components`
--

-- --------------------------------------------------------

--
-- Table structure for table `ccda_sections`
--

DROP TABLE IF EXISTS `ccda_sections`;
CREATE TABLE ccda_sections (
  ccda_sections_id int(11) NOT NULL AUTO_INCREMENT,
  ccda_components_id int(11) DEFAULT NULL,
  ccda_sections_field varchar(100) DEFAULT NULL,
  ccda_sections_name varchar(100) DEFAULT NULL,
  ccda_sections_req_mapping tinyint(4) NOT NULL DEFAULT '1',
  PRIMARY KEY (ccda_sections_id)
) ENGINE=InnoDB AUTO_INCREMENT=46;

--
-- Inserting data for table `ccda_sections`
--

-- --------------------------------------------------------

--
-- Table structure for table `ccda_field_mapping`
--

DROP TABLE IF EXISTS `ccda_field_mapping`;
CREATE TABLE ccda_field_mapping (
  id int(11) NOT NULL AUTO_INCREMENT,
  table_id int(11) DEFAULT NULL,
  ccda_field varchar(100) DEFAULT NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `ccda`
--

DROP TABLE IF EXISTS `ccda`;
CREATE TABLE `ccda` (
  `id` INT(11) NOT NULL AUTO_INCREMENT,
  `uuid` binary(16) DEFAULT NULL,
  `pid` BIGINT(20) DEFAULT NULL,
  `encounter` BIGINT(20) DEFAULT NULL,
  `ccda_data` LONGTEXT,
  `time` VARCHAR(50) DEFAULT NULL,
  `status` SMALLINT(6) DEFAULT NULL,
  `updated_date` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `user_id` VARCHAR(50) null,
  `couch_docid` VARCHAR(100) NULL,
  `couch_revid` VARCHAR(100) NULL,
  `hash` varchar(255) DEFAULT NULL,
  `view` tinyint(4) NOT NULL DEFAULT '0',
  `transfer` tinyint(4) NOT NULL DEFAULT '0',
  `emr_transfer` tinyint(4) NOT NULL DEFAULT '0',
  `encrypted` TINYINT(4) NOT NULL DEFAULT '0' COMMENT '0->No,1->Yes',
  `transaction_id` BIGINT(20) COMMENT 'fk to transaction referral record',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uuid` (`uuid`),
  UNIQUE KEY `unique_key` (`pid`,`encounter`,`time`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `ccda_table_mapping`
--

DROP TABLE IF EXISTS `ccda_table_mapping`;
CREATE TABLE ccda_table_mapping (
  id int(11) NOT NULL AUTO_INCREMENT,
  ccda_component varchar(100) DEFAULT NULL,
  ccda_component_section varchar(100) DEFAULT NULL,
  form_dir varchar(100) DEFAULT NULL,
  form_type smallint(6) DEFAULT NULL,
  form_table varchar(100) DEFAULT NULL,
  user_id int(11) DEFAULT NULL,
  deleted tinyint(4) NOT NULL DEFAULT '0',
  timestamp timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- --------------------------------------------------------

--
-- Table structure for table `external_procedures`
--

DROP TABLE IF EXISTS `external_procedures`;
CREATE TABLE `external_procedures` (
  `ep_id` int(11) NOT NULL AUTO_INCREMENT,
  `ep_date` date DEFAULT NULL,
  `ep_code_type` varchar(20) DEFAULT NULL,
  `ep_code` varchar(9) DEFAULT NULL,
  `ep_pid` int(11) DEFAULT NULL,
  `ep_encounter` int(11) DEFAULT NULL,
  `ep_code_text` longtext,
  `ep_facility_id` varchar(255) DEFAULT NULL,
  `ep_external_id` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`ep_id`),
  KEY `ep_pid` (`ep_pid`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `external_encounters`
--

DROP TABLE IF EXISTS `external_encounters`;
CREATE TABLE `external_encounters` (
  `ee_id` int(11) NOT NULL AUTO_INCREMENT,
  `ee_date` date DEFAULT NULL,
  `ee_pid` int(11) DEFAULT NULL,
  `ee_provider_id` varchar(255) DEFAULT NULL,
  `ee_facility_id` varchar(255) DEFAULT NULL,
  `ee_encounter_diagnosis` varchar(255) DEFAULT NULL,
  `ee_external_id` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`ee_id`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `form_care_plan`
--

DROP TABLE IF EXISTS `form_care_plan`;
CREATE TABLE `form_care_plan` (
  `id` bigint(20) NOT NULL,
  `date` datetime DEFAULT NULL,
  `pid` bigint(20) DEFAULT NULL,
  `encounter` varchar(255) DEFAULT NULL,
  `user` varchar(255) DEFAULT NULL,
  `groupname` varchar(255) DEFAULT NULL,
  `authorized` tinyint(4) DEFAULT NULL,
  `activity` tinyint(4) DEFAULT NULL,
  `code` varchar(255) DEFAULT NULL,
  `codetext` text,
  `description` text,
  `external_id` varchar(30) DEFAULT NULL,
  `care_plan_type` varchar(30) DEFAULT NULL,
  `note_related_to` text,
  `date_end` datetime DEFAULT NULL,
  `reason_code` varchar(31) DEFAULT NULL,
  `reason_description` text,
  `reason_date_low` datetime DEFAULT NULL COMMENT 'The date the reason was recorded',
  `reason_date_high` datetime DEFAULT NULL COMMENT 'The date the explanation reason for the care plan entry value ends',
  `reason_status` varchar(31) DEFAULT NULL,
  `plan_status` varchar(32) DEFAULT NULL COMMENT 'Care Plan status (e.g., draft, active, completed, etc)',
  `proposed_date` DATETIME NULL COMMENT 'Target or Achieve-by date for the goal',
  KEY `idx_status_date` (`plan_status`,`date`,`date_end`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `form_functional_cognitive_status`
--

DROP TABLE IF EXISTS `form_functional_cognitive_status`;
CREATE TABLE `form_functional_cognitive_status` (
  `id` bigint(20) NOT NULL,
  `date` date DEFAULT NULL,
  `pid` bigint(20) DEFAULT NULL,
  `encounter` varchar(255) DEFAULT NULL,
  `user` varchar(255) DEFAULT NULL,
  `groupname` varchar(255) DEFAULT NULL,
  `authorized` tinyint(4) DEFAULT NULL,
  `activity` tinyint(4) DEFAULT NULL,
  `code` varchar(255) DEFAULT NULL,
  `codetext` text,
  `description` text,
  `external_id` varchar(30) DEFAULT NULL
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `form_observation`
--

DROP TABLE IF EXISTS `form_observation`;
CREATE TABLE `form_observation` (
   `id` bigint(20) NOT NULL AUTO_INCREMENT,
   `uuid` binary(16) DEFAULT NULL COMMENT 'UUID for the observation, used as unique logical identifier',
   `form_id` bigint(20) NOT NULL COMMENT 'FK to forms.form_id',
  `date` DATETIME DEFAULT NULL,
  `pid` bigint(20) DEFAULT NULL,
  `encounter` varchar(255) DEFAULT NULL,
  `user` varchar(255) DEFAULT NULL,
  `groupname` varchar(255) DEFAULT NULL,
  `authorized` tinyint(4) DEFAULT NULL,
  `activity` tinyint(4) DEFAULT NULL,
  `code` varchar(255) DEFAULT NULL,
  `observation` varchar(255) DEFAULT NULL,
  `ob_value` varchar(255),
  `ob_unit` varchar(255),
  `description` varchar(255),
  `code_type` varchar(255),
  `table_code` varchar(255),
  `ob_code` VARCHAR(64) DEFAULT NULL,
  `ob_type` VARCHAR(64) DEFAULT NULL,
  `ob_status` varchar(32) DEFAULT NULL,
  `result_status` varchar(32) DEFAULT NULL,
  `ob_reason_status` varchar(32) DEFAULT NULL,
  `ob_reason_code` varchar(64) DEFAULT NULL,
  `ob_reason_text` text,
  `ob_documentationof_table` varchar(255) DEFAULT NULL,
  `ob_documentationof_table_id` bigint(21) DEFAULT NULL,
   `date_end` DATETIME DEFAULT NULL,
   `parent_observation_id` bigint(20) DEFAULT NULL COMMENT 'FK to parent observation for sub-observations',
   `category` varchar(64) DEFAULT NULL COMMENT 'FK to list_options.option_id for observation category (SDOH, Functional, Cognitive, Physical, etc)',
   `questionnaire_response_id` bigint(21) DEFAULT NULL COMMENT 'FK to questionnaire_response table',
   `ob_value_code_description` VARCHAR(255) DEFAULT NULL,
   PRIMARY KEY (`id`),
   KEY `idx_form_id` (`form_id`),
   KEY `idx_parent_observation` (`parent_observation_id`),
   KEY `idx_category` (`category`),
   KEY `idx_questionnaire_response` (`questionnaire_response_id`),
   KEY `idx_pid_encounter` (`pid`, `encounter`),
   KEY `idx_date` (`date`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `form_clinical_instructions`
--

DROP TABLE IF EXISTS `form_clinical_instructions`;
CREATE TABLE `form_clinical_instructions` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `pid` bigint(20) DEFAULT NULL,
  `encounter` varchar(255) DEFAULT NULL,
  `user` varchar(255) DEFAULT NULL,
  `instruction` text,
  `date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `activity` TINYINT DEFAULT 1 NULL,
  PRIMARY KEY (`id`)
)ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table 'valueset'
--

DROP TABLE IF EXISTS `valueset`;
CREATE TABLE `valueset` (
  `nqf_code` varchar(255) NOT NULL DEFAULT '',
  `code` varchar(255) NOT NULL DEFAULT '',
  `code_system` varchar(255) NOT NULL DEFAULT '',
  `code_type` varchar(255) DEFAULT NULL,
  `valueset` varchar(255) NOT NULL DEFAULT '',
  `description` varchar(255) DEFAULT NULL,
  `valueset_name` varchar(500) DEFAULT NULL,
  PRIMARY KEY (`nqf_code`,`code`,`valueset`)
) ENGINE=InnoDB;

-- -------------------------------------------------------

--
-- Table structure for table `immunization_observation`
--

DROP TABLE IF EXISTS `immunization_observation`;
CREATE TABLE `immunization_observation` (
  `imo_id` int(11) NOT NULL AUTO_INCREMENT,
  `imo_im_id` int(11) NOT NULL,
  `imo_pid` int(11) DEFAULT NULL,
  `imo_criteria` varchar(255) DEFAULT NULL,
  `imo_criteria_value` varchar(255) DEFAULT NULL,
  `imo_user` int(11) DEFAULT NULL,
  `imo_code` varchar(255) DEFAULT NULL,
  `imo_codetext` varchar(255) DEFAULT NULL,
  `imo_codetype` varchar(255) DEFAULT NULL,
  `imo_vis_date_published` date DEFAULT NULL,
  `imo_vis_date_presented` date DEFAULT NULL,
  `imo_date_observation` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`imo_id`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table 'calendar external'
--

DROP TABLE IF EXISTS `calendar_external`;
CREATE TABLE calendar_external (
  `id` INT NOT NULL AUTO_INCREMENT,
  `date` DATE NOT NULL,
  `description` VARCHAR(45) NOT NULL,
  `source` VARCHAR(45) NULL,
  PRIMARY KEY (`id`)) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `form_eye_mag_dispense`
--

DROP TABLE IF EXISTS `form_eye_mag_dispense`;
CREATE TABLE `form_eye_mag_dispense` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `date` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `encounter` bigint(20) NULL,
  `pid` bigint(20) DEFAULT NULL,
  `user` varchar(255) DEFAULT NULL,
  `groupname` varchar(255) DEFAULT NULL,
  `authorized` tinyint(4) DEFAULT NULL,
  `activity` tinyint(4) DEFAULT NULL,
  `REFDATE` DATETIME NULL DEFAULT NULL,
  `REFTYPE` varchar(10) DEFAULT NULL,
  `RXTYPE` varchar(20)DEFAULT NULL,
  `ODSPH` varchar(10) DEFAULT NULL,
  `ODCYL` varchar(10) DEFAULT NULL,
  `ODAXIS` varchar(10) DEFAULT NULL,
  `OSSPH` varchar(10) DEFAULT NULL,
  `OSCYL` varchar(10) DEFAULT NULL,
  `OSAXIS` varchar(10) DEFAULT NULL,
  `ODMIDADD` varchar(10) DEFAULT NULL,
  `OSMIDADD` varchar(10) DEFAULT NULL,
  `ODADD` varchar(10) DEFAULT NULL,
  `OSADD` varchar(10) DEFAULT NULL,
  `ODHPD` varchar(20) DEFAULT NULL,
  `ODHBASE` varchar(20) DEFAULT NULL,
  `ODVPD` varchar(20) DEFAULT NULL,
  `ODVBASE` varchar(20) DEFAULT NULL,
  `ODSLABOFF` varchar(20) DEFAULT NULL,
  `ODVERTEXDIST` varchar(20) DEFAULT NULL,
  `OSHPD` varchar(20) DEFAULT NULL,
  `OSHBASE` varchar(20) DEFAULT NULL,
  `OSVPD` varchar(20) DEFAULT NULL,
  `OSVBASE` varchar(20) DEFAULT NULL,
  `OSSLABOFF` varchar(20) DEFAULT NULL,
  `OSVERTEXDIST` varchar(20) DEFAULT NULL,
  `ODMPDD` varchar(20) DEFAULT NULL,
  `ODMPDN` varchar(20) DEFAULT NULL,
  `OSMPDD` varchar(20) DEFAULT NULL,
  `OSMPDN` varchar(20) DEFAULT NULL,
  `BPDD` varchar(20) DEFAULT NULL,
  `BPDN` varchar(20) DEFAULT NULL,
  `LENS_MATERIAL` varchar(20) DEFAULT NULL,
  `LENS_TREATMENTS` varchar(100) DEFAULT NULL,
  `CTLMANUFACTUREROD` varchar(25) DEFAULT NULL,
  `CTLMANUFACTUREROS` varchar(25) DEFAULT NULL,
  `CTLSUPPLIEROD` varchar(25) DEFAULT NULL,
  `CTLSUPPLIEROS` varchar(25) DEFAULT NULL,
  `CTLBRANDOD` varchar(50) DEFAULT NULL,
  `CTLBRANDOS` varchar(50) DEFAULT NULL,
  `CTLODQUANTITY` varchar(255) DEFAULT NULL,
  `CTLOSQUANTITY` varchar(255) DEFAULT NULL,
  `ODDIAM` varchar(50) DEFAULT NULL,
  `ODBC` varchar(50) DEFAULT NULL,
  `OSDIAM` varchar(50) DEFAULT NULL,
  `OSBC` varchar(50) DEFAULT NULL,
  `RXCOMMENTS` text,
  `COMMENTS` text,
  PRIMARY KEY (`id`),
  UNIQUE KEY `pid` (`pid`,`encounter`,`id`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `form_eye_mag_prefs`
--

DROP TABLE IF EXISTS `form_eye_mag_prefs`;
CREATE TABLE `form_eye_mag_prefs` (
  `PEZONE` varchar(25) DEFAULT NULL,
  `LOCATION` varchar(25) DEFAULT NULL,
  `LOCATION_text` varchar(25) NOT NULL,
  `id` bigint(20) DEFAULT NULL,
  `selection` varchar(255) DEFAULT NULL,
  `ZONE_ORDER` int(11) DEFAULT NULL,
  `GOVALUE` varchar(10) DEFAULT '0',
  `ordering` smallint(6) DEFAULT NULL,
  `FILL_ACTION` varchar(10) NOT NULL DEFAULT 'ADD',
  `GORIGHT` varchar(50) NOT NULL,
  `GOLEFT` varchar(50) NOT NULL,
  `UNSPEC` varchar(50) NOT NULL,
  UNIQUE KEY `id` (`id`,`PEZONE`,`LOCATION`,`selection`)
) ENGINE=InnoDB;

--
-- Inserting data for table `form_eye_mag_prefs`
--

-- --------------------------------------------------------

--
-- Table structure for table `form_eye_mag_orders`
--

DROP TABLE IF EXISTS `form_eye_mag_orders`;
CREATE TABLE `form_eye_mag_orders` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `form_id` int(20) NOT NULL,
  `pid` bigint(20) NOT NULL,
  `ORDER_DETAILS` varchar(255) NOT NULL,
  `ORDER_STATUS` varchar(50) DEFAULT NULL,
  `ORDER_PRIORITY` varchar(50) DEFAULT NULL,
  `ORDER_DATE_PLACED` date NOT NULL,
  `ORDER_PLACED_BYWHOM` varchar(50) DEFAULT NULL,
  `ORDER_DATE_COMPLETED` date DEFAULT NULL,
  `ORDER_COMPLETED_BYWHOM` varchar(50) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `VISIT_ID` (`pid`,`ORDER_DETAILS`,`ORDER_DATE_PLACED`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `form_eye_mag_impplan`
--

DROP TABLE IF EXISTS `form_eye_mag_impplan`;
CREATE TABLE `form_eye_mag_impplan` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `form_id` bigint(20) NOT NULL,
  `pid` bigint(20) NOT NULL,
  `title` varchar(255) NOT NULL,
  `code` varchar(50) DEFAULT NULL,
  `codetype` varchar(50) DEFAULT NULL,
  `codedesc` varchar(255) DEFAULT NULL,
  `codetext` varchar(255) DEFAULT NULL,
  `plan` varchar(3000) DEFAULT NULL,
  `PMSFH_link` varchar(50) DEFAULT NULL,
  `IMPPLAN_order` tinyint(4) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `second_index` (`form_id`,`pid`,`title`,`plan`(20))
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `form_eye_mag_wearing`
--

DROP TABLE IF EXISTS `form_eye_mag_wearing`;
CREATE TABLE `form_eye_mag_wearing` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `ENCOUNTER` int(11) NOT NULL,
  `FORM_ID` smallint(6) NOT NULL,
  `PID` bigint(20) NOT NULL,
  `RX_NUMBER` int(11) NOT NULL,
  `ODSPH` varchar(10) DEFAULT NULL,
  `ODCYL` varchar(10) DEFAULT NULL,
  `ODAXIS` varchar(10) DEFAULT NULL,
  `OSSPH` varchar(10) DEFAULT NULL,
  `OSCYL` varchar(10) DEFAULT NULL,
  `OSAXIS` varchar(10) DEFAULT NULL,
  `ODMIDADD` varchar(10) DEFAULT NULL,
  `OSMIDADD` varchar(10) DEFAULT NULL,
  `ODADD` varchar(10) DEFAULT NULL,
  `OSADD` varchar(10) DEFAULT NULL,
  `ODVA` varchar(10) DEFAULT NULL,
  `OSVA` varchar(10) DEFAULT NULL,
  `ODNEARVA` varchar(10) DEFAULT NULL,
  `OSNEARVA` varchar(10) DEFAULT NULL,
  `ODHPD` varchar(20) DEFAULT NULL,
  `ODHBASE` varchar(20) DEFAULT NULL,
  `ODVPD` varchar(20) DEFAULT NULL,
  `ODVBASE` varchar(20) DEFAULT NULL,
  `ODSLABOFF` varchar(20) DEFAULT NULL,
  `ODVERTEXDIST` varchar(20) DEFAULT NULL,
  `OSHPD` varchar(20) DEFAULT NULL,
  `OSHBASE` varchar(20) DEFAULT NULL,
  `OSVPD` varchar(20) DEFAULT NULL,
  `OSVBASE` varchar(20) DEFAULT NULL,
  `OSSLABOFF` varchar(20) DEFAULT NULL,
  `OSVERTEXDIST` varchar(20) DEFAULT NULL,
  `ODMPDD` varchar(20) DEFAULT NULL,
  `ODMPDN` varchar(20) DEFAULT NULL,
  `OSMPDD` varchar(20) DEFAULT NULL,
  `OSMPDN` varchar(20) DEFAULT NULL,
  `BPDD` varchar(20) DEFAULT NULL,
  `BPDN` varchar(20) DEFAULT NULL,
  `LENS_MATERIAL` varchar(20) DEFAULT NULL,
  `LENS_TREATMENTS` varchar(100) DEFAULT NULL,
  `RX_TYPE` varchar(25) DEFAULT NULL,
  `COMMENTS` text,
  UNIQUE KEY `id` (`id`),
  UNIQUE KEY `FORM_ID` (`FORM_ID`,`ENCOUNTER`,`PID`,`RX_NUMBER`)
) ENGINE=InnoDB;

--
-- Table structure for table `form_taskman`
--

DROP TABLE IF EXISTS `form_taskman`;
CREATE TABLE `form_taskman` (
    `ID` bigint(20) NOT NULL AUTO_INCREMENT,
    `REQ_DATE` datetime NOT NULL,
    `FROM_ID` bigint(20) NOT NULL,
    `TO_ID` bigint(20) NOT NULL,
    `PATIENT_ID` bigint(20) NOT NULL, `DOC_TYPE` varchar(20) DEFAULT NULL,
    `DOC_ID` bigint(20) DEFAULT NULL,
    `ENC_ID` bigint(20) DEFAULT NULL,
    `METHOD` varchar(20) NOT NULL, `COMPLETED` varchar(1) DEFAULT NULL COMMENT '1 = completed',
    `COMPLETED_DATE` datetime DEFAULT NULL,
    `COMMENT` varchar(50) DEFAULT NULL,
    `USERFIELD_1` varchar(50) DEFAULT NULL,
    PRIMARY KEY (`ID`)
) ENGINE=INNODB;

-- -----------------------------------------------------
--
-- Table structure for table 'product_registration'
--

DROP TABLE IF EXISTS `product_registration`;
CREATE TABLE `product_registration` (
    `id` int(11) NOT NULL AUTO_INCREMENT,
    `email` VARCHAR(255) NULL,
    `opt_out` TINYINT(1) NULL,
    `auth_by_id` INT(11) NULL,
    `telemetry_disabled` TINYINT(1) NULL COMMENT '1 opted out, disabled. NULL ask. 0 use option scopes',
    `last_ask_date` DATETIME NULL,
    `last_ask_version`TINYTEXT,
    `options` TEXT COMMENT 'JSON array of scope options',
  PRIMARY KEY (id)
) ENGINE=InnoDB;

-- ---------------------------------------------------------

--
-- Table structure for table 'codes_history'
--

DROP TABLE IF EXISTS `codes_history`;
CREATE TABLE `codes_history` (
  `log_id` bigint(20) NOT NULL auto_increment,
  `date` datetime,
  `code` varchar(25),
  `modifier` varchar(12),
  `active` tinyint(1),
  `diagnosis_reporting` tinyint(1),
  `financial_reporting` tinyint(1),
  `category` varchar(255),
  `code_type_name` varchar(255),
  `code_text` text,
  `code_text_short` text,
  `prices` text,
  `action_type` varchar(25),
  `update_by` varchar(255),
   PRIMARY KEY (`log_id`)
) ENGINE=InnoDB;

-- ---------------------------------------------------------

--
-- Table structure for `therapy_groups`
--

DROP TABLE IF EXISTS `therapy_groups`;
CREATE TABLE `therapy_groups` (
  `group_id` int(11) NOT NULL auto_increment,
  `group_name` varchar(255) NOT NULL ,
  `group_start_date` date NOT NULL ,
  `group_end_date` date,
  `group_type` tinyint NOT NULL,
  `group_participation` tinyint NOT NULL,
  `group_status` int(11) NOT NULL,
  `group_notes` text,
  `group_guest_counselors` varchar(255),
  PRIMARY KEY  (`group_id`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for `therapy_groups_participants`
--

DROP TABLE IF EXISTS `therapy_groups_participants`;
CREATE TABLE `therapy_groups_participants` (
  `group_id` int(11) NOT NULL,
  `pid` bigint(20) NOT NULL,
  `group_patient_status` int(11) NOT NULL,
  `group_patient_start` date NOT NULL ,
  `group_patient_end` date,
  `group_patient_comment` text,
  PRIMARY KEY (`group_id`,`pid`)
) ENGINE=InnoDB;

-- -- ---------------------------------------------------------

--
-- Table structure for `therapy_groups_participant_attendance`
--

DROP TABLE IF EXISTS `therapy_groups_participant_attendance`;
CREATE TABLE `therapy_groups_participant_attendance` (
  `form_id` int(11) NOT NULL ,
  `pid` bigint(20) NOT NULL,
  `meeting_patient_comment` text ,
  `meeting_patient_status` varchar(15),
  PRIMARY KEY (`form_id`,`pid`)
) ENGINE=InnoDB;

-- -- ---------------------------------------------------------

--
-- Table structure for `therapy_groups_counselors`
--

DROP TABLE IF EXISTS `therapy_groups_counselors`;
CREATE TABLE `therapy_groups_counselors`(
    `group_id` int(11) NOT NULL,
    `user_id` int(11) NOT NULL,
    PRIMARY KEY (`group_id`,`user_id`)
) ENGINE=InnoDB;

-- -- ---------------------------------------------------------

--
-- Table structure for `form_groups_encounter`
--

DROP TABLE IF EXISTS `form_groups_encounter`;
CREATE TABLE `form_groups_encounter` (
  `id` bigint(20) NOT NULL auto_increment,
  `date` datetime default NULL,
  `reason` longtext,
  `facility` longtext,
  `facility_id` int(11) NOT NULL default '0',
  `group_id` bigint(20) default NULL,
  `encounter` bigint(20) default NULL,
  `onset_date` datetime default NULL,
  `sensitivity` varchar(30) default NULL,
  `billing_note` text,
  `pc_catid` int(11) NOT NULL default '5' COMMENT 'event category from openemr_postcalendar_categories',
  `last_level_billed` int  NOT NULL DEFAULT 0 COMMENT '0=none, 1=ins1, 2=ins2, etc',
  `last_level_closed` int  NOT NULL DEFAULT 0 COMMENT '0=none, 1=ins1, 2=ins2, etc',
  `last_stmt_date`    date DEFAULT NULL,
  `stmt_count`        int  NOT NULL DEFAULT 0,
  `provider_id` INT(11) DEFAULT '0' COMMENT 'default and main provider for this visit',
  `supervisor_id` INT(11) DEFAULT '0' COMMENT 'supervising provider, if any, for this visit',
  `invoice_refno` varchar(31) NOT NULL DEFAULT '',
  `referral_source` varchar(31) NOT NULL DEFAULT '',
  `billing_facility` INT(11) NOT NULL DEFAULT 0,
  `external_id` VARCHAR(20) DEFAULT NULL,
  `pos_code` tinyint(4) default NULL,
  `counselors` VARCHAR (255),
  `appt_id` INT(11) default NULL,
  PRIMARY KEY  (`id`),
  KEY `pid_encounter` (`group_id`, `encounter`),
  KEY `encounter_date` (`date`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

-- -- ---------------------------------------------------------

--
-- Table structure for `form_group_attendance`
--

DROP TABLE IF EXISTS `form_group_attendance`;
CREATE TABLE `form_group_attendance` (
  `id`	bigint(20) auto_increment,
  `date`	date,
  `group_id`	int(11),
  `user`	varchar(255),
  `groupname`	varchar(255),
  `authorized`	tinyint(4),
  `encounter_id`	int(11),
  activity	tinyint(4),
  PRIMARY KEY (`id`)
) ENGINE=InnoDB;

-- -- ---------------------------------------------------------

--
-- Table structure for `patient_birthday_alert`
--

DROP TABLE IF EXISTS `patient_birthday_alert`;
CREATE TABLE `patient_birthday_alert` (
  `pid` bigint(20) NOT NULL DEFAULT 0,
  `user_id` bigint(20) NOT NULL DEFAULT 0,
  `turned_off_on` date NOT NULL,
  PRIMARY KEY  (`pid`,`user_id`)
) ENGINE=InnoDB;

--
-- Table structure for table `medex_icons`
--
DROP TABLE IF EXISTS `medex_icons`;
CREATE TABLE `medex_icons` (
  `i_UID` int(11) NOT NULL AUTO_INCREMENT,
  `msg_type` varchar(50) NOT NULL,
  `msg_status` varchar(10) NOT NULL,
  `i_description` varchar(255),
  `i_html` text,
  `i_blob` longtext,
  PRIMARY KEY (`i_UID`)
) ENGINE=InnoDB;

--
-- Dumping data for table `medex_icons`
--


-- --------------------------------------------------------

--
-- Table structure for table `medex_outgoing`
DROP TABLE IF EXISTS `medex_outgoing`;
CREATE TABLE `medex_outgoing` (
  `msg_uid` int(11) NOT NULL AUTO_INCREMENT,
  `msg_pid` int(11) NOT NULL,
  `msg_pc_eid` varchar(11) NOT NULL,
  `campaign_uid` int(11) NOT NULL DEFAULT '0',
  `msg_date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `msg_type` varchar(50) NOT NULL,
  `msg_reply` varchar(50) DEFAULT NULL,
  `msg_extra_text` text,
  `medex_uid` int(11),
  PRIMARY KEY (`msg_uid`),
  UNIQUE KEY `msg_eid` (`msg_uid`,`msg_pc_eid`,`medex_uid`)
) ENGINE=InnoDB;

--
-- Dumping data for table `medex_outgoing`
--


-- --------------------------------------------------------

--
-- Table structure for table `medex_prefs`
--
DROP TABLE IF EXISTS `medex_prefs`;
CREATE TABLE `medex_prefs` (
  `MedEx_id` int(11) DEFAULT '0',
  `ME_username` varchar(100) DEFAULT NULL,
  `ME_api_key` text,
  `ME_facilities` varchar(50) DEFAULT NULL,
  `ME_providers` varchar(100) DEFAULT NULL,
  `ME_hipaa_default_override` varchar(3) DEFAULT NULL,
  `PHONE_country_code` int(4) NOT NULL DEFAULT '1',
  `MSGS_default_yes` varchar(3) DEFAULT NULL,
  `POSTCARDS_local` varchar(3) DEFAULT NULL,
  `POSTCARDS_remote` varchar(3) DEFAULT NULL,
  `LABELS_local` varchar(3) DEFAULT NULL,
  `LABELS_choice` varchar(50) DEFAULT NULL,
  `combine_time` tinyint(4) DEFAULT NULL,
  `postcard_top` varchar(255) DEFAULT NULL,
  `status` text,
  `MedEx_lastupdated` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY `ME_username` (`ME_username`)
) ENGINE=InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `medex_recalls`
--
DROP TABLE IF EXISTS `medex_recalls`;
CREATE TABLE `medex_recalls` (
  `r_ID` int(11) NOT NULL AUTO_INCREMENT,
  `r_PRACTID` int(11) NOT NULL,
  `r_pid` int(11) NOT NULL COMMENT 'PatientID from pat_data',
  `r_eventDate` date NOT NULL COMMENT 'Date of Appt or Recall',
  `r_facility` int(11) NOT NULL,
  `r_provider` int(11) NOT NULL,
  `r_reason` varchar(255) DEFAULT NULL,
  `r_created` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`r_ID`),
  UNIQUE KEY `r_PRACTID` (`r_PRACTID`,`r_pid`)
) ENGINE=InnoDB;


-- --------------------------------------------------------

--
-- Table structure for table `form_eye_base`
--
DROP TABLE IF EXISTS `form_eye_base`;
CREATE TABLE `form_eye_base` (
  `id`         bigint(20) NOT NULL AUTO_INCREMENT COMMENT 'Links to forms.form_id',
  `date`       datetime DEFAULT NULL,
  `pid`        bigint(20)   DEFAULT NULL,
  `user`       varchar(255) DEFAULT NULL,
  `groupname`  varchar(255) DEFAULT NULL,
  `authorized` tinyint(4)   DEFAULT NULL,
  `activity`   tinyint(4)   DEFAULT NULL,
  PRIMARY KEY `form_link` (`id`)
) ENGINE = InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `form_eye_hpi`
--

DROP TABLE IF EXISTS `form_eye_hpi`;
CREATE TABLE `form_eye_hpi` (
  `id`          bigint(20) NOT NULL COMMENT 'Links to forms.form_id',
  `pid`         bigint(20)   DEFAULT NULL,
  `CC1`         varchar(255) DEFAULT NULL,
  `HPI1`        text,
  `QUALITY1`    varchar(255) DEFAULT NULL,
  `TIMING1`     varchar(255) DEFAULT NULL,
  `DURATION1`   varchar(255) DEFAULT NULL,
  `CONTEXT1`    varchar(255) DEFAULT NULL,
  `SEVERITY1`   varchar(255) DEFAULT NULL,
  `MODIFY1`     varchar(255) DEFAULT NULL,
  `ASSOCIATED1` varchar(255) DEFAULT NULL,
  `LOCATION1`   varchar(255) DEFAULT NULL,
  `CHRONIC1`    varchar(255) DEFAULT NULL,
  `CHRONIC2`    varchar(255) DEFAULT NULL,
  `CHRONIC3`    varchar(255) DEFAULT NULL,
  `CC2`         text,
  `HPI2`        text,
  `QUALITY2`    text,
  `TIMING2`     text,
  `DURATION2`   text,
  `CONTEXT2`    text,
  `SEVERITY2`   text,
  `MODIFY2`     text,
  `ASSOCIATED2` text,
  `LOCATION2`   text,
  `CC3`         text,
  `HPI3`        text,
  `QUALITY3`    text,
  `TIMING3`     text,
  `DURATION3`   text,
  `CONTEXT3`    text,
  `SEVERITY3`   text,
  `MODIFY3`     text,
  `ASSOCIATED3` text,
  `LOCATION3`   text,
  PRIMARY KEY `hpi_link` (`id`),
  UNIQUE KEY `id_pid` (`id`,`pid`)
)  ENGINE = InnoDB;


-- --------------------------------------------------------

--
-- Table structure for table `form_eye_ros`
--
DROP TABLE IF EXISTS `form_eye_ros`;
CREATE TABLE `form_eye_ros` (
  `id`           bigint(20) NOT NULL COMMENT 'Links to forms.form_id',
  `pid`          bigint(20)   DEFAULT NULL,
  `ROSGENERAL`   text,
  `ROSHEENT`     text,
  `ROSCV`        text,
  `ROSPULM`      text,
  `ROSGI`        text,
  `ROSGU`        text,
  `ROSDERM`      text,
  `ROSNEURO`     text,
  `ROSPSYCH`     text,
  `ROSMUSCULO`   text,
  `ROSIMMUNO`    text,
  `ROSENDOCRINE` text,
  `ROSCOMMENTS`  text,
  PRIMARY KEY `ros_link` (`id`),
  UNIQUE KEY `id_pid` (`id`,`pid`)
  )
  ENGINE = InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `form_eye_vitals`
--

DROP TABLE IF EXISTS `form_eye_vitals`;
CREATE TABLE `form_eye_vitals` (
  `id`          bigint(20)  NOT NULL COMMENT 'Links to forms.form_id',
  `pid`         bigint(20)   DEFAULT NULL,
  `alert`       char(3)     DEFAULT 'yes',
  `oriented`    char(3)     DEFAULT 'TPP',
  `confused`    char(3)     DEFAULT 'nml',
  `ODIOPAP`     varchar(10) DEFAULT NULL,
  `OSIOPAP`     varchar(10) DEFAULT NULL,
  `ODIOPTPN`    varchar(10) DEFAULT NULL,
  `OSIOPTPN`    varchar(10) DEFAULT NULL,
  `ODIOPFTN`    varchar(10) DEFAULT NULL,
  `OSIOPFTN`    varchar(10) DEFAULT NULL,
  `IOPTIME`     time        NOT NULL,
  `ODIOPPOST`   varchar(10) NOT NULL,
  `OSIOPPOST`   varchar(10) NOT NULL,
  `IOPPOSTTIME` time        DEFAULT NULL,
  `ODIOPTARGET` varchar(10) NOT NULL,
  `OSIOPTARGET` varchar(10) NOT NULL,
  `AMSLEROD`    smallint(1) DEFAULT NULL,
  `AMSLEROS`    smallint(1) DEFAULT NULL,
  `ODVF1`       tinyint(1)  DEFAULT NULL,
  `ODVF2`       tinyint(1)  DEFAULT NULL,
  `ODVF3`       tinyint(1)  DEFAULT NULL,
  `ODVF4`       tinyint(1)  DEFAULT NULL,
  `OSVF1`       tinyint(1)  DEFAULT NULL,
  `OSVF2`       tinyint(1)  DEFAULT NULL,
  `OSVF3`       tinyint(1)  DEFAULT NULL,
  `OSVF4`       tinyint(1)  DEFAULT NULL,
  PRIMARY KEY `vitals_link` (`id`),
  UNIQUE KEY `id_pid` (`id`,`pid`)
  )
  ENGINE = InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `form_eye_acuity`
--

DROP TABLE IF EXISTS `form_eye_acuity`;
CREATE TABLE `form_eye_acuity` (
  `id`            bigint(20)  NOT NULL COMMENT 'Links to forms.form_id',
  `pid`           bigint(20)   DEFAULT NULL,
  `SCODVA`        varchar(25)  DEFAULT NULL,
  `SCOSVA`        varchar(25)  DEFAULT NULL,
  `PHODVA`        varchar(25)  DEFAULT NULL,
  `PHOSVA`        varchar(25)  DEFAULT NULL,
  `CTLODVA`       varchar(25)  DEFAULT NULL,
  `CTLOSVA`       varchar(25)  DEFAULT NULL,
  `MRODVA`        varchar(25)  DEFAULT NULL,
  `MROSVA`        varchar(25)  DEFAULT NULL,
  `SCNEARODVA`    varchar(25)  DEFAULT NULL,
  `SCNEAROSVA`    varchar(25)  DEFAULT NULL,
  `MRNEARODVA`    varchar(25)  DEFAULT NULL,
  `MRNEAROSVA`    varchar(25)  DEFAULT NULL,
  `GLAREODVA`     varchar(25)  DEFAULT NULL,
  `GLAREOSVA`     varchar(25)  DEFAULT NULL,
  `GLARECOMMENTS` varchar(255) DEFAULT NULL,
  `ARODVA`        varchar(25)  DEFAULT NULL,
  `AROSVA`        varchar(25)  DEFAULT NULL,
  `CRODVA`        varchar(25)  DEFAULT NULL,
  `CROSVA`        varchar(25)  DEFAULT NULL,
  `CTLODVA1`      varchar(25)  DEFAULT NULL,
  `CTLOSVA1`      varchar(25)  DEFAULT NULL,
  `PAMODVA`       varchar(25)  DEFAULT NULL,
  `PAMOSVA`       varchar(25)  DEFAULT NULL,
  `LIODVA`        varchar(25)  NOT NULL,
  `LIOSVA`        varchar(25)  NOT NULL,
  `WODVANEAR`     varchar(25)  DEFAULT NULL,
  `OSVANEARCC`    varchar(25)  DEFAULT NULL,
  `BINOCVA`       varchar(25)  DEFAULT NULL,
  PRIMARY KEY `acuity_link` (`id`),
  UNIQUE KEY `id_pid` (`id`,`pid`)
  )
  ENGINE = InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `form_eye_refraction`
--
DROP TABLE IF EXISTS `form_eye_refraction`;
CREATE TABLE `form_eye_refraction` (
  `id`                bigint(20) NOT NULL COMMENT 'Links to forms.form_id',
  `pid`               bigint(20)   DEFAULT NULL,
  `MRODSPH`           varchar(25)  DEFAULT NULL,
  `MRODCYL`           varchar(25)  DEFAULT NULL,
  `MRODAXIS`          varchar(25)  DEFAULT NULL,
  `MRODPRISM`         varchar(25)  DEFAULT NULL,
  `MRODBASE`          varchar(25)  DEFAULT NULL,
  `MRODADD`           varchar(25)  DEFAULT NULL,
  `MROSSPH`           varchar(25)  DEFAULT NULL,
  `MROSCYL`           varchar(25)  DEFAULT NULL,
  `MROSAXIS`          varchar(25)  DEFAULT NULL,
  `MROSPRISM`         varchar(50)  DEFAULT NULL,
  `MROSBASE`          varchar(50)  DEFAULT NULL,
  `MROSADD`           varchar(25)  DEFAULT NULL,
  `MRODNEARSPHERE`    varchar(25)  DEFAULT NULL,
  `MRODNEARCYL`       varchar(25)  DEFAULT NULL,
  `MRODNEARAXIS`      varchar(25)  DEFAULT NULL,
  `MRODPRISMNEAR`     varchar(50)  DEFAULT NULL,
  `MRODBASENEAR`      varchar(25)  DEFAULT NULL,
  `MROSNEARSHPERE`    varchar(25)  DEFAULT NULL,
  `MROSNEARCYL`       varchar(25)  DEFAULT NULL,
  `MROSNEARAXIS`      varchar(125) DEFAULT NULL,
  `MROSPRISMNEAR`     varchar(50)  DEFAULT NULL,
  `MROSBASENEAR`      varchar(25)  DEFAULT NULL,
  `CRODSPH`           varchar(25)  DEFAULT NULL,
  `CRODCYL`           varchar(25)  DEFAULT NULL,
  `CRODAXIS`          varchar(25)  DEFAULT NULL,
  `CROSSPH`           varchar(25)  DEFAULT NULL,
  `CROSCYL`           varchar(25)  DEFAULT NULL,
  `CROSAXIS`          varchar(25)  DEFAULT NULL,
  `CRCOMMENTS`        varchar(255) DEFAULT NULL,
  `BALANCED`          char(2)    NOT NULL,
  `ARODSPH`           varchar(25)  DEFAULT NULL,
  `ARODCYL`           varchar(25)  DEFAULT NULL,
  `ARODAXIS`          varchar(25)  DEFAULT NULL,
  `AROSSPH`           varchar(25)  DEFAULT NULL,
  `AROSCYL`           varchar(25)  DEFAULT NULL,
  `AROSAXIS`          varchar(25)  DEFAULT NULL,
  `ARODADD`           varchar(25)  DEFAULT NULL,
  `AROSADD`           varchar(25)  DEFAULT NULL,
  `ARNEARODVA`        varchar(25)  DEFAULT NULL,
  `ARNEAROSVA`        varchar(25)  DEFAULT NULL,
  `ARODPRISM`         varchar(50)  DEFAULT NULL,
  `AROSPRISM`         varchar(50)  DEFAULT NULL,
  `CTLODSPH`          varchar(25)  DEFAULT NULL,
  `CTLODCYL`          varchar(25)  DEFAULT NULL,
  `CTLODAXIS`         varchar(25)  DEFAULT NULL,
  `CTLODBC`           varchar(25)  DEFAULT NULL,
  `CTLODDIAM`         varchar(25)  DEFAULT NULL,
  `CTLOSSPH`          varchar(25)  DEFAULT NULL,
  `CTLOSCYL`          varchar(25)  DEFAULT NULL,
  `CTLOSAXIS`         varchar(25)  DEFAULT NULL,
  `CTLOSBC`           varchar(25)  DEFAULT NULL,
  `CTLOSDIAM`         varchar(25)  DEFAULT NULL,
  `CTL_COMMENTS`      text,
  `CTLMANUFACTUREROD` varchar(50)  DEFAULT NULL,
  `CTLSUPPLIEROD`     varchar(50)  DEFAULT NULL,
  `CTLBRANDOD`        varchar(50)  DEFAULT NULL,
  `CTLMANUFACTUREROS` varchar(50)  DEFAULT NULL,
  `CTLSUPPLIEROS`     varchar(50)  DEFAULT NULL,
  `CTLBRANDOS`        varchar(50)  DEFAULT NULL,
  `CTLODADD`          varchar(25)  DEFAULT NULL,
  `CTLOSADD`          varchar(25)  DEFAULT NULL,
  `NVOCHECKED`        varchar(25)  DEFAULT NULL,
  `ADDCHECKED`        varchar(25)  DEFAULT NULL,
  PRIMARY KEY `refraction_link` (`id`),
  UNIQUE KEY `id_pid` (`id`,`pid`)
  )
  ENGINE = InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `form_eye_biometrics`
--

DROP TABLE IF EXISTS `form_eye_biometrics`;
CREATE TABLE `form_eye_biometrics` (
  `id`            bigint (20) NOT NULL COMMENT 'Links to forms.form_id',
  `pid`           bigint(20)   DEFAULT NULL,
  `ODK1`          varchar (10) DEFAULT NULL,
  `ODK2`          varchar (10) DEFAULT NULL,
  `ODK2AXIS`      varchar (10) DEFAULT NULL,
  `OSK1`          varchar (10) DEFAULT NULL,
  `OSK2`          varchar (10) DEFAULT NULL,
  `OSK2AXIS`      varchar (10) DEFAULT NULL,
  `ODAXIALLENGTH` varchar (20) DEFAULT NULL,
  `OSAXIALLENGTH` varchar (20) DEFAULT NULL,
  `ODPDMeasured`  varchar (20) DEFAULT NULL,
  `OSPDMeasured`  varchar (20) DEFAULT NULL,
  `ODACD`         varchar (20) DEFAULT NULL,
  `OSACD`         varchar (20) DEFAULT NULL,
  `ODW2W`         varchar (20) DEFAULT NULL,
  `OSW2W`         varchar (20) DEFAULT NULL,
  `ODLT`          varchar (20) DEFAULT NULL,
  `OSLT`          varchar (20) DEFAULT NULL,
  PRIMARY KEY `biometrics_link` (`id`),
  UNIQUE KEY `id_pid` (`id`,`pid`)
)
  ENGINE = InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `form_eye_external`
--

DROP TABLE IF EXISTS `form_eye_external`;
CREATE TABLE `form_eye_external` (
  `id`           bigint(20)  NOT NULL COMMENT 'Links to forms.form_id',
  `pid`          bigint(20)  DEFAULT NULL,
  `RUL`          text,
  `LUL`          text,
  `RLL`          text,
  `LLL`          text,
  `RBROW`        text,
  `LBROW`        text,
  `RMCT`         text,
  `LMCT`         text,
  `RADNEXA`      text,
  `LADNEXA`      text,
  `RMRD`         varchar(25) DEFAULT NULL,
  `LMRD`         varchar(25) DEFAULT NULL,
  `RLF`          varchar(25) DEFAULT NULL,
  `LLF`          varchar(25) DEFAULT NULL,
  `RVFISSURE`    varchar(25) DEFAULT NULL,
  `LVFISSURE`    varchar(25) DEFAULT NULL,
  `ODHERTEL`     varchar(25) DEFAULT NULL,
  `OSHERTEL`     varchar(25) DEFAULT NULL,
  `HERTELBASE`   varchar(25) DEFAULT NULL,
  `RCAROTID`     text,
  `LCAROTID`     text,
  `RTEMPART`     text,
  `LTEMPART`     text,
  `RCNV`         text,
  `LCNV`         text,
  `RCNVII`       text,
  `LCNVII`       text,
  `EXT_COMMENTS` text,
  PRIMARY KEY `external_link` (`id`),
  UNIQUE KEY `id_pid` (`id`,`pid`)
)
  ENGINE = InnoDB;

-- --------------------------------------------------------

--
-- Table structure for table `form_eye_antseg`
--

DROP TABLE IF EXISTS `form_eye_antseg`;
CREATE TABLE `form_eye_antseg` (
  `id`                   bigint(20) NOT NULL COMMENT 'Links to forms.form_id',
  `pid`                  bigint(20)   DEFAULT NULL,
  `ODSCHIRMER1`          varchar(25) DEFAULT NULL,
  `OSSCHIRMER1`          varchar(25) DEFAULT NULL,
  `ODSCHIRMER2`          varchar(25) DEFAULT NULL,
  `OSSCHIRMER2`          varchar(25) DEFAULT NULL,
  `ODTBUT`               varchar(25) DEFAULT NULL,
  `OSTBUT`               varchar(25) DEFAULT NULL,
  `OSCONJ`               text,
  `ODCONJ`               text,
  `ODCORNEA`             text,
  `OSCORNEA`             text,
  `ODAC`                 text,
  `OSAC`                 text,
  `ODLENS`               text,
  `OSLENS`               text,
  `ODIRIS`               text,
  `OSIRIS`               text,
  `PUPIL_NORMAL`         varchar(2)  DEFAULT '1',
  `ODPUPILSIZE1`         varchar(25) DEFAULT NULL,
  `ODPUPILSIZE2`         varchar(25) DEFAULT NULL,
  `ODPUPILREACTIVITY`    char(25)    DEFAULT NULL,
  `ODAPD`                varchar(25) DEFAULT NULL,
  `OSPUPILSIZE1`         varchar(25) DEFAULT NULL,
  `OSPUPILSIZE2`         varchar(25) DEFAULT NULL,
  `OSPUPILREACTIVITY`    char(25)    DEFAULT NULL,
  `OSAPD`                varchar(25) DEFAULT NULL,
  `DIMODPUPILSIZE1`      varchar(25) DEFAULT NULL,
  `DIMODPUPILSIZE2`      varchar(25) DEFAULT NULL,
  `DIMODPUPILREACTIVITY` varchar(25) DEFAULT NULL,
  `DIMOSPUPILSIZE1`      varchar(25) DEFAULT NULL,
  `DIMOSPUPILSIZE2`      varchar(25) DEFAULT NULL,
  `DIMOSPUPILREACTIVITY` varchar(25) DEFAULT NULL,
  `PUPIL_COMMENTS`       text,
  `ODKTHICKNESS`         varchar(25) DEFAULT NULL,
  `OSKTHICKNESS`         varchar(25) DEFAULT NULL,
  `ODGONIO`              varchar(25) DEFAULT NULL,
  `OSGONIO`              varchar(25) DEFAULT NULL,
  `ANTSEG_COMMENTS`      text,
  PRIMARY KEY `antseg_link` (`id`),
  UNIQUE KEY `id_pid` (`id`,`pid`)
 )
  ENGINE = InnoDB;


-- --------------------------------------------------------

--
-- Table structure for table `form_eye_postseg`
--

DROP TABLE IF EXISTS `form_eye_postseg`;
CREATE TABLE `form_eye_postseg` (
  `id`              bigint(20)  NOT NULL COMMENT 'Links to forms.form_id',
  `pid`             bigint(20)   DEFAULT NULL,
  `ODDISC`          text,
  `OSDISC`          text,
  `ODCUP`           text,
  `OSCUP`           text,
  `ODMACULA`        text,
  `OSMACULA`        text,
  `ODVESSELS`       text,
  `OSVESSELS`       text,
  `ODVITREOUS`      text,
  `OSVITREOUS`      text,
  `ODPERIPH`        text,
  `OSPERIPH`        text,
  `ODCMT`           text,
  `OSCMT`           text,
  `RETINA_COMMENTS` text,
  `DIL_RISKS`       char(2)     NOT NULL DEFAULT 'on',
  `DIL_MEDS`        mediumtext,
  `WETTYPE`         varchar(10) NOT NULL,
  `ATROPINE`        varchar(25) NOT NULL,
  `CYCLOMYDRIL`     varchar(25) NOT NULL,
  `TROPICAMIDE`     varchar(25) NOT NULL,
  `CYCLOGYL`        varchar(25) NOT NULL,
  `NEO25`           varchar(25) NOT NULL,
  PRIMARY KEY `postseg_link` (`id`),
  UNIQUE KEY `id_pid` (`id`,`pid`)
 )
  ENGINE = InnoDB;


-- --------------------------------------------------------

--
-- Table structure for table `form_eye_neuro`
--

DROP TABLE IF EXISTS `form_eye_neuro`;
CREATE TABLE `form_eye_neuro` (
  `id`         bigint (20) NOT NULL COMMENT 'Links to forms.form_id',
  `pid`        bigint(20)   DEFAULT NULL,
  `ACT`        char (3) NOT NULL DEFAULT 'on',
  `ACT5CCDIST` text,
  `ACT1CCDIST` text,
  `ACT2CCDIST` text,
  `ACT3CCDIST` text,
  `ACT4CCDIST` text,
  `ACT6CCDIST` text,
  `ACT7CCDIST` text,
  `ACT8CCDIST` text,
  `ACT9CCDIST` text,
  `ACT10CCDIST` text,
  `ACT11CCDIST` text,
  `ACT1SCDIST` text,
  `ACT2SCDIST` text,
  `ACT3SCDIST` text,
  `ACT4SCDIST` text,
  `ACT5SCDIST` text,
  `ACT6SCDIST` text,
  `ACT7SCDIST` text,
  `ACT8SCDIST` text,
  `ACT9SCDIST` text,
  `ACT10SCDIST` text,
  `ACT11SCDIST` text,
  `ACT1SCNEAR` text,
  `ACT2SCNEAR` text,
  `ACT3SCNEAR` text,
  `ACT4SCNEAR` text,
  `ACT5CCNEAR` text,
  `ACT6CCNEAR` text,
  `ACT7CCNEAR` text,
  `ACT8CCNEAR` text,
  `ACT9CCNEAR` text,
  `ACT10CCNEAR` text,
  `ACT11CCNEAR` text,
  `ACT5SCNEAR` text,
  `ACT6SCNEAR` text,
  `ACT7SCNEAR` text,
  `ACT8SCNEAR` text,
  `ACT9SCNEAR` text,
  `ACT10SCNEAR` text,
  `ACT11SCNEAR` text,
  `ACT1CCNEAR` text,
  `ACT2CCNEAR` text,
  `ACT3CCNEAR` text,
  `ACT4CCNEAR`text,
  `MOTILITYNORMAL` char (3) NOT NULL DEFAULT 'on',
  `MOTILITY_RS` char (1) DEFAULT '0',
  `MOTILITY_RI` char (1) DEFAULT '0',
  `MOTILITY_RR` char (1) DEFAULT '0',
  `MOTILITY_RL` char (1) DEFAULT '0',
  `MOTILITY_LS` char (1) DEFAULT '0',
  `MOTILITY_LI` char (1) DEFAULT '0',
  `MOTILITY_LR` char (1) DEFAULT '0',
  `MOTILITY_LL` char (1) DEFAULT '0',
  `MOTILITY_RRSO` int (1) DEFAULT NULL,
  `MOTILITY_RLSO` int (1) DEFAULT NULL,
  `MOTILITY_RRIO` int (1) DEFAULT NULL,
  `MOTILITY_RLIO` int (1) DEFAULT NULL,
  `MOTILITY_LRSO` int (1) DEFAULT NULL,
  `MOTILITY_LLSO` int (1) DEFAULT NULL,
  `MOTILITY_LRIO` int (1) DEFAULT NULL,
  `MOTILITY_LLIO` int (1) DEFAULT NULL,
  `NEURO_COMMENTS` text,
  `STEREOPSIS` varchar (25) DEFAULT NULL,
  `ODNPA` text,
  `OSNPA` text,
  `VERTFUSAMPS` text,
  `DIVERGENCEAMPS` text,
  `NPC` varchar (10) DEFAULT NULL,
  `DACCDIST` varchar (20) DEFAULT NULL,
  `DACCNEAR` varchar (20) DEFAULT NULL,
  `CACCDIST` varchar (20) DEFAULT NULL,
  `CACCNEAR` varchar (20) DEFAULT NULL,
  `ODCOLOR` text,
  `OSCOLOR` text,
  `ODCOINS` text,
  `OSCOINS` text,
  `ODREDDESAT` varchar (20) DEFAULT NULL,
  `OSREDDESAT` varchar (20) DEFAULT NULL,
  PRIMARY KEY `neuro_link` (`id`),
  UNIQUE KEY `id_pid` (`id`,`pid`)
 )
  ENGINE = InnoDB;


-- --------------------------------------------------------

--
-- Table structure for table `form_eye_locking`
--

DROP TABLE IF EXISTS `form_eye_locking`;
CREATE TABLE `form_eye_locking` (
  `id`         bigint(20) NOT NULL COMMENT 'Links to forms.form_id',
  `pid`        bigint(20)          DEFAULT NULL,
  `IMP`        text,
  `PLAN`       text,
  `Resource`   varchar(50)         DEFAULT NULL,
  `Technician` varchar(50)         DEFAULT NULL,
  `LOCKED`     varchar(3)          DEFAULT NULL,
  `LOCKEDDATE` timestamp  NOT NULL DEFAULT CURRENT_TIMESTAMP
  ON UPDATE CURRENT_TIMESTAMP,
  `LOCKEDBY`   varchar(50)         DEFAULT NULL,
  PRIMARY KEY `locking_link` (`id`),
  UNIQUE KEY `id_pid` (`id`,`pid`)
  )
  ENGINE = InnoDB;

DROP TABLE IF EXISTS `login_mfa_registrations`;
CREATE TABLE `login_mfa_registrations` (
  `user_id`         bigint(20)     NOT NULL,
  `name`            varchar(30)    NOT NULL,
  `last_challenge`  datetime       DEFAULT NULL,
  `method`          varchar(31)    NOT NULL COMMENT 'Q&A, U2F, TOTP etc.',
  `var1`            varchar(4096)  NOT NULL DEFAULT '' COMMENT 'Question, U2F registration etc.',
  `var2`            varchar(256)   NOT NULL DEFAULT '' COMMENT 'Answer etc.',
  PRIMARY KEY (`user_id`, `name`)
) ENGINE=InnoDB;

DROP TABLE IF EXISTS `benefit_eligibility`;
CREATE TABLE `benefit_eligibility` (
    `response_id` bigint(20) NOT NULL,
    `verification_id` bigint(20) NOT NULL,
    `type` varchar(4) DEFAULT NULL,
    `benefit_type` varchar(255) DEFAULT NULL,
    `start_date` date DEFAULT NULL,
    `end_date` date DEFAULT NULL,
    `coverage_level` varchar(255) DEFAULT NULL,
    `coverage_type` varchar(512) DEFAULT NULL,
    `plan_type` varchar(255) DEFAULT NULL,
    `plan_description` varchar(255) DEFAULT NULL,
    `coverage_period` varchar(255) DEFAULT NULL,
    `amount` decimal(5,2) DEFAULT NULL,
    `percent` decimal(3,2) DEFAULT NULL,
    `network_ind` varchar(2) DEFAULT NULL,
    `message` varchar(512) DEFAULT NULL,
    `response_status` enum('A','D') DEFAULT 'A',
    `response_create_date` date DEFAULT NULL,
    `response_modify_date` date DEFAULT NULL
) ENGINE=InnoDB;

DROP TABLE IF EXISTS `oauth_clients`;
CREATE TABLE `oauth_clients` (
`client_id` varchar(80) NOT NULL,
`client_role` varchar(20) DEFAULT NULL,
`client_name` varchar(80) NOT NULL,
`client_secret` text,
`registration_token` varchar(80) DEFAULT NULL,
`registration_uri_path` varchar(40) DEFAULT NULL,
`register_date` datetime DEFAULT NULL,
`revoke_date` datetime DEFAULT NULL,
`contacts` text,
`redirect_uri` text,
`grant_types` varchar(80) DEFAULT NULL,
`scope` text,
`user_id` varchar(40) DEFAULT NULL,
`site_id` varchar(64) DEFAULT NULL,
`is_confidential` tinyint(1) NOT NULL DEFAULT '1',
`logout_redirect_uris` text,
`jwks_uri` text,
`jwks` text,
`initiate_login_uri` text,
`endorsements` text,
`policy_uri` text,
`tos_uri` text,
`is_enabled` tinyint(1) NOT NULL DEFAULT '0',
`skip_ehr_launch_authorization_flow` tinyint(1) NOT NULL DEFAULT '0',
`dsi_type` TINYINT UNSIGNED NOT NULL DEFAULT '1' COMMENT '0=none, 1=evidence-based,2=predictive',
PRIMARY KEY (`client_id`)
) ENGINE=InnoDB;

DROP TABLE IF EXISTS `oauth_trusted_user`;
CREATE TABLE `oauth_trusted_user` (
`id` bigint(20) NOT NULL AUTO_INCREMENT,
`user_id` varchar(80) DEFAULT NULL,
`client_id` varchar(80) DEFAULT NULL,
`scope` text,
`persist_login` tinyint(1) DEFAULT '0',
`time` timestamp NULL DEFAULT NULL,
`code` text,
`session_cache` text,
`grant_type` varchar(32) DEFAULT NULL,
PRIMARY KEY (`id`),
KEY `accounts_id` (`user_id`),
KEY `clients_id` (`client_id`)
) ENGINE=InnoDB;

DROP TABLE IF EXISTS `x12_remote_tracker`;
CREATE TABLE `x12_remote_tracker` (
`id` bigint(20) NOT NULL AUTO_INCREMENT,
`x12_partner_id` int(11) NOT NULL,
`x12_filename` varchar(255) NOT NULL,
`status` varchar(255) NOT NULL,
`claims` text,
`messages` text,
`created_at` datetime DEFAULT NULL,
`updated_at` datetime DEFAULT NULL,
PRIMARY KEY (`id`)
) ENGINE=InnoDB;

DROP TABLE IF EXISTS `export_job`;
CREATE TABLE `export_job` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `uuid` binary(16) DEFAULT NULL,
  `user_id` varchar(40) NOT NULL,
  `client_id` varchar(80) NOT NULL,
  `status` varchar(40) NOT NULL,
  `start_time` datetime DEFAULT NULL,
  `resource_include_time` datetime DEFAULT NULL,
  `output_format` varchar(128) NOT NULL,
  `request_uri` varchar(128) NOT NULL,
  `resources` text,
  `output` text,
  `errors` text,
  `access_token_id` text,
  UNIQUE (`uuid`),
  PRIMARY KEY  (`id`)
) ENGINE=InnoDB COMMENT='fhir export jobs';

DROP TABLE IF EXISTS `form_vital_details`;
CREATE TABLE `form_vital_details` (
`id` bigint(20) NOT NULL AUTO_INCREMENT,
`form_id` bigint(20) NOT NULL COMMENT 'FK to vital_forms.id',
`vitals_column` varchar(64) NOT NULL COMMENT 'Column name from form_vitals',
`interpretation_list_id` varchar(100) DEFAULT NULL COMMENT 'FK to list_options.list_id for observation_interpretation',
`interpretation_option_id` varchar(100) DEFAULT NULL COMMENT 'FK to list_options.option_id for observation_interpretation',
`interpretation_codes` varchar(255) DEFAULT NULL COMMENT 'Archived original codes value from list_options observation_interpretation',
`interpretation_title` varchar(255) DEFAULT NULL COMMENT 'Archived original title value from list_options observation_interpretation',
`reason_code` VARCHAR(31) NULL DEFAULT NULL COMMENT 'Medical code explaining reason of the vital observation value in form codesystem:codetype;...;',
`reason_description` TEXT COMMENT 'Human readable text description of the reason_code column',
`reason_status` VARCHAR(31) NULL DEFAULT NULL COMMENT 'The status of the reason ie completed, in progress, etc',
PRIMARY KEY (`id`),
KEY `fk_form_id` (`form_id`),
KEY `fk_list_options_id` (`interpretation_list_id`, `interpretation_option_id`)
) ENGINE=InnoDB COMMENT='Detailed information of each vital_forms observation column';

DROP TABLE IF EXISTS `form_vitals_calculation`;
CREATE TABLE `form_vitals_calculation` (
   `id` int NOT NULL AUTO_INCREMENT,
   `uuid` binary(16) DEFAULT NULL,
   `encounter` bigint(20) DEFAULT NULL COMMENT 'fk to form_encounter.id',
   `pid` bigint(20) NOT NULL COMMENT 'fk to patient_data.pid',
   `date_start` datetime DEFAULT NULL,
   `date_end` datetime DEFAULT NULL,
   `created_at` datetime DEFAULT NULL,
   `updated_at` datetime DEFAULT NULL,
   `created_by` bigint(20) DEFAULT NULL,
   `updated_by` bigint(20) DEFAULT NULL,
   `calculation_id` varchar(64) DEFAULT NULL COMMENT 'application identifier representing calculation e.g., bp-MeanLast5, bp-Mean3Day, bp-MeanEncounter',
   PRIMARY KEY (`id`),
   UNIQUE KEY `unq_uuid` (`uuid`),
   KEY `idx_pid` (`pid`),
   KEY `idx_encounter` (`encounter`),
   KEY `idx_calculation_id` (`calculation_id`)
) ENGINE=InnoDB COMMENT = 'Main calculation records - one per logical calculation (e.g., average BP)';

DROP TABLE IF EXISTS `form_vitals_calculation_components`;
CREATE TABLE `form_vitals_calculation_components` (
    `id` int NOT NULL AUTO_INCREMENT,
    `fvc_uuid` binary(16) NOT NULL COMMENT 'fk to form_vitals_calculation.uuid',
    `vitals_column` varchar(64) NOT NULL COMMENT 'Component type: bps, bpd, pulse, etc.',
    `value` DECIMAL(12,6) DEFAULT NULL COMMENT 'Calculated numeric component value',
    `value_string` varchar(255) DEFAULT NULL COMMENT 'Calculated non-numeric component value',
    `value_unit` varchar(16) DEFAULT NULL COMMENT 'Unit for this component value',
    `component_order` int NOT NULL DEFAULT 0 COMMENT 'Display order for components',
    PRIMARY KEY (`id`),
    UNIQUE KEY `unq_fvc_component` (`fvc_uuid`, `vitals_column`),
    KEY `idx_vitals_column` (`vitals_column`),
    KEY `idx_component_order` (`fvc_uuid`, `component_order`)
) ENGINE=InnoDB COMMENT = 'Component values for calculations (e.g., systolic=120, diastolic=80)';

DROP TABLE IF EXISTS `form_vitals_calculation_form_vitals`;
CREATE TABLE `form_vitals_calculation_form_vitals` (
   `fvc_uuid` binary(16) NOT NULL COMMENT 'fk to form_vitals_calculation.uuid',
   `vitals_id` bigint(20) NOT NULL COMMENT 'fk to form_vitals.id',
   PRIMARY KEY (`fvc_uuid`, `vitals_id`)
) ENGINE=InnoDB COMMENT = 'Join table between form_vitals_calculation and form_vitals table representing the derivative observation relationship between the calculation and the source records';

DROP TABLE IF EXISTS `jwt_grant_history`;
CREATE TABLE `jwt_grant_history` (
`id` INT NOT NULL AUTO_INCREMENT
 , `jti` VARCHAR(100) NOT NULL COMMENT 'Unique JWT id'
 , `client_id` VARCHAR(80) NOT NULL COMMENT 'FK oauth2_clients.client_id'
 , `jti_exp` TIMESTAMP NULL DEFAULT NULL COMMENT 'jwt exp claim when the jwt expires'
 , `creation_date` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'datetime the grant authorization was requested'
 , PRIMARY KEY (`id`)
 , KEY `jti` (`jti`)
) ENGINE = InnoDB COMMENT = 'Holds JWT authorization grant ids to prevent replay attacks';

DROP TABLE IF EXISTS `document_templates`;
CREATE TABLE `document_templates` (
  `id` bigint(21) UNSIGNED NOT NULL AUTO_INCREMENT,
  `pid` bigint(20) DEFAULT NULL,
  `provider` int(11) UNSIGNED DEFAULT NULL,
  `encounter` int(11) UNSIGNED DEFAULT NULL,
  `modified_date` datetime NOT NULL DEFAULT current_timestamp(),
  `profile` varchar(63) NOT NULL,
  `category` varchar(63) NOT NULL,
  `location` varchar(255) DEFAULT NULL,
  `template_name` varchar(255) DEFAULT NULL,
  `status` varchar(31) DEFAULT NULL,
  `send_date` datetime NOT NULL DEFAULT current_timestamp(),
  `end_date` datetime DEFAULT NULL,
  `size` int(11) NOT NULL DEFAULT 0,
  `template_content` mediumblob DEFAULT NULL,
  `mime` varchar(31) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `location` (`pid`,`profile`,`category`,`template_name`)
) ENGINE=InnoDB;

DROP TABLE IF EXISTS `document_template_profiles`;
CREATE TABLE `document_template_profiles` (
  `id` bigint(21) UNSIGNED NOT NULL AUTO_INCREMENT,
  `template_id` bigint(21) UNSIGNED NOT NULL,
  `profile` varchar(64) NOT NULL,
  `template_name` varchar(255) NOT NULL,
  `category` varchar(64) NOT NULL,
  `provider` int(11) UNSIGNED DEFAULT NULL,
  `modified_date` datetime NOT NULL DEFAULT current_timestamp(),
  `member_of` varchar(64) NOT NULL,
  `active` tinyint(1) NOT NULL DEFAULT 0,
  `recurring` tinyint(1) NOT NULL DEFAULT 1,
  `event_trigger` varchar(31) NOT NULL,
  `period` int(4) NOT NULL,
  `notify_trigger` varchar(31) NOT NULL,
  `notify_period` int(4) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `location` (`profile`,`template_id`,`member_of`)
) ENGINE=InnoDB;

DROP TABLE IF EXISTS `valueset_oid`;
CREATE TABLE `valueset_oid` (
  `nqf_code` varchar(255) NOT NULL DEFAULT '',
  `code` varchar(255) NOT NULL DEFAULT '',
  `code_system` varchar(255) NOT NULL DEFAULT '',
  `code_type` varchar(255) DEFAULT NULL,
  `valueset` varchar(255) NOT NULL DEFAULT '',
  `description` varchar(255) DEFAULT NULL,
  `valueset_name` varchar(500) DEFAULT NULL,
  PRIMARY KEY (`nqf_code`,`code`,`valueset`)
) ENGINE=InnoDB;

DROP TABLE IF EXISTS `questionnaire_repository`;
CREATE TABLE `questionnaire_repository` (
    `id` bigint(21) UNSIGNED NOT NULL AUTO_INCREMENT,
    `uuid` binary(16) DEFAULT NULL,
    `questionnaire_id` varchar(255) DEFAULT NULL,
    `provider` int(11) UNSIGNED DEFAULT NULL,
    `version` int(11) NOT NULL DEFAULT 1,
    `created_date` datetime DEFAULT current_timestamp(),
    `modified_date` datetime DEFAULT current_timestamp(),
    `name` varchar(255) DEFAULT NULL,
    `type` varchar(63) NOT NULL DEFAULT 'Questionnaire',
    `profile` varchar(255) DEFAULT NULL,
    `active` tinyint(1) NOT NULL DEFAULT 1,
    `status` varchar(31) DEFAULT NULL,
    `source_url` text,
    `code` varchar(255) DEFAULT NULL,
    `code_display` text,
    `questionnaire` longtext,
    `lform` longtext,
    `category` VARCHAR(64) DEFAULT NULL COMMENT 'Used for grouping and organizing ',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uuid` (`uuid`),
    KEY `search` (`name`,`questionnaire_id`)
) ENGINE=InnoDB;

DROP TABLE IF EXISTS `questionnaire_response`;
CREATE TABLE `questionnaire_response` (
  `id` bigint(21) NOT NULL AUTO_INCREMENT,
  `uuid` binary(16) DEFAULT NULL,
  `response_id` varchar(255) DEFAULT NULL COMMENT 'A globally unique id for answer set. String version of UUID',
  `questionnaire_foreign_id` bigint(21) DEFAULT NULL COMMENT 'questionnaire_repository id for subject questionnaire',
  `questionnaire_id` varchar(255) DEFAULT NULL COMMENT 'Id for questionnaire content. String version of UUID',
  `questionnaire_name` varchar(255) DEFAULT NULL,
  `patient_id` int(11) DEFAULT NULL,
  `encounter` int(11) DEFAULT NULL COMMENT 'May or may not be associated with an encounter',
  `audit_user_id` int(11) DEFAULT NULL,
  `creator_user_id` int(11) DEFAULT NULL COMMENT 'user id if answers are provider',
  `create_time` datetime DEFAULT current_timestamp(),
  `last_updated` datetime DEFAULT NULL,
  `version` int(11) NOT NULL DEFAULT 1,
  `status` varchar(63) DEFAULT NULL COMMENT 'form current status. completed,active,incomplete',
  `questionnaire` longtext COMMENT 'the subject questionnaire json',
  `questionnaire_response` longtext COMMENT 'questionnaire response json',
  `form_response` longtext COMMENT 'lform answers array json',
  `form_score` int(11) DEFAULT NULL COMMENT 'Arithmetic scoring of questionnaires',
  `tscore` double DEFAULT NULL COMMENT 'T-Score',
  `error` double DEFAULT NULL COMMENT 'Standard error for the T-Score',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uuid` (`uuid`),
  KEY `response_index` (`response_id`, `patient_id`, `questionnaire_id`, `questionnaire_name`)
) ENGINE=InnoDB;

DROP TABLE IF EXISTS `form_questionnaire_assessments`;
CREATE TABLE `form_questionnaire_assessments` (
  `id` bigint(21) NOT NULL AUTO_INCREMENT,
  `date` datetime DEFAULT current_timestamp(),
  `response_id` TEXT COMMENT 'The foreign id to the questionnaire_response repository',
  `pid` bigint(21) NOT NULL DEFAULT 0,
  `user` varchar(255) DEFAULT NULL,
  `groupname` varchar(255) DEFAULT NULL,
  `authorized` tinyint(4) NOT NULL DEFAULT 0,
  `activity` tinyint(4) NOT NULL DEFAULT 1,
  `copyright` text,
  `form_name` varchar(255) DEFAULT NULL,
  `response_meta` text COMMENT 'json meta data for the response resource',
  `questionnaire_id` TEXT COMMENT 'The foreign id to the questionnaire_repository',
  `questionnaire` longtext,
  `questionnaire_response` longtext,
  `lform` longtext,
  `lform_response` longtext,
  `category` VARCHAR(64) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB;

DROP TABLE IF EXISTS `onetime_auth`;
CREATE TABLE `onetime_auth` (
    `id` int(11) NOT NULL AUTO_INCREMENT,
    `pid` bigint(20) DEFAULT NULL,
    `create_user_id` bigint(20) DEFAULT NULL,
    `context` varchar(64) DEFAULT NULL,
    `access_count` int(11) NOT NULL DEFAULT 0,
    `remote_ip` varchar(32) DEFAULT NULL,
    `onetime_pin` varchar(10) DEFAULT NULL COMMENT 'Max 10 numeric. Default 6',
    `onetime_token` tinytext,
    `redirect_url` tinytext,
    `expires` int(11) DEFAULT NULL,
    `date_created` datetime DEFAULT current_timestamp(),
    `last_accessed` datetime DEFAULT NULL,
    `scope` tinytext COMMENT 'context scope for this token',
    `profile` tinytext COMMENT 'profile of scope for this token',
    `onetime_actions` text COMMENT 'JSON array of actions that can be performed with this token',
    PRIMARY KEY (`id`),
    KEY `pid` (`pid`,`onetime_token`(32))
) ENGINE=InnoDB;

DROP TABLE IF EXISTS `patient_settings`;
CREATE TABLE `patient_settings` (
     `setting_patient`  bigint(20)   NOT NULL DEFAULT 0,
     `setting_label` varchar(100)  NOT NULL,
     `setting_value` varchar(255) NOT NULL DEFAULT '',
     PRIMARY KEY (`setting_patient`, `setting_label`)
) ENGINE=InnoDB;

DROP TABLE IF EXISTS `recent_patients`;
CREATE TABLE `recent_patients` (
    `user_id` varchar(40) NOT NULL,
    `patients` TEXT,
    PRIMARY KEY (`user_id`)
) ENGINE=InnoDB;

DROP TABLE IF EXISTS `dsi_source_attributes`;
CREATE TABLE `dsi_source_attributes` (
 `id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
 `client_id` VARCHAR(80) NOT NULL,
 `list_id` VARCHAR(100) NOT NULL,
 `option_id` VARCHAR(100) NOT NULL,
 `clinical_rule_id` VARCHAR(31) DEFAULT NULL,
 `source_value` TEXT,
 `created_by` BIGINT(20) DEFAULT NULL,
 `last_updated_by` BIGINT(20) DEFAULT NULL,
 `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
 `last_updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
 PRIMARY KEY (`id`),
 UNIQUE (`list_id`, `option_id`, `client_id`)
) ENGINE=InnoDB COMMENT = 'Holds information about decision support intervention system source attributes';

-- Populate list with ONC default values
-- Populate list with ONC default values
-- -----------------------------------------------------
--
-- Table structure for table 'track_events'
--

DROP TABLE IF EXISTS `track_events`;
CREATE TABLE `track_events` (
    `id`                  INT(11) UNSIGNED NOT NULL AUTO_INCREMENT,
    `event_type`     TEXT,
    `event_label`    VARCHAR(255) DEFAULT NULL,
    `event_url`       TEXT,
    `event_target`  TEXT,
    `first_event`     DATETIME NULL,
    `last_event`     DATETIME NULL,
    `label_count`    INT UNSIGNED NOT NULL DEFAULT 1,
    PRIMARY KEY (`id`),
    UNIQUE KEY `unique_event_label_target` (`event_label`, `event_url`(255), `event_target`(255))
) ENGINE = InnoDB COMMENT = 'Telemetry Event Data';

-- -----------------------------------------------------
--
-- Table structure for table `care_teams`
--

DROP TABLE IF EXISTS `care_teams`;
CREATE TABLE `care_teams` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `uuid` binary(16) DEFAULT NULL,
  `pid` int(11) NOT NULL COMMENT 'fk to patient_data.pid',
  `status` varchar(100) DEFAULT 'active' COMMENT 'fk to list_options.option_id where list_id=Care_Team_Status',
  `team_name` varchar(255) DEFAULT NULL,
  `note` text,
  `date_created` datetime DEFAULT current_timestamp(),
  `date_updated` datetime DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `created_by` BIGINT(20) COMMENT 'fk to users.id for user who created this record',
  `updated_by` BIGINT(20) COMMENT 'fk to users.id for user who last updated this record',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uuid` (`uuid`)
) ENGINE=InnoDB;

DROP TABLE IF EXISTS `care_team_member`;
CREATE TABLE `care_team_member` (
    `id` int(11) NOT NULL AUTO_INCREMENT,
    `care_team_id` int(11) NOT NULL,
    `user_id` BIGINT(20) COMMENT 'fk to users.id represents a provider or staff member',
    `contact_id` BIGINT(20) COMMENT 'fk to contact.id which represents a contact person not in users or facility table',
    `role` varchar(50) NOT NULL COMMENT 'fk to list_options.option_id WHERE list_id=care_team_roles',
    `facility_id` BIGINT(20) COMMENT 'fk to facility.id represents an organization or location',
    `provider_since` date NULL,
    `status` varchar(100) DEFAULT 'active' COMMENT 'fk to list_options.option_id where list_id=Care_Team_Status',
    `date_created` datetime DEFAULT current_timestamp(),
    `date_updated` datetime DEFAULT current_timestamp() ON UPDATE current_timestamp(),
    `created_by` BIGINT(20) COMMENT 'fk to users.id and is the user that added this team member',
    `updated_by` BIGINT(20) COMMENT 'fk to users.id and is the user that last updated this team member',
    `note` text,
    PRIMARY KEY (`id`),
    UNIQUE KEY `care_team_member_unique` (`care_team_id`, `user_id`, `facility_id`, `contact_id`)
) ENGINE=InnoDB COMMENT='Stores members of a care team for a patient';
-- ----------------------------------------------------------------
--
-- Care Team Roles (based on HL7/USCDI v3)
--

-- ---------------------------------------------------------------------------------------------------------------------------------
--
-- 2023 Performance Period Measures

-- ---------------------------------------------------------------------------------------------------------------------------------
--
-- 2024 Performance Period Measures

-- ---------------------------------------------------------------------------------------------------------------------------------
--
-- 2025 Performance Period Measures

-- ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------
--
-- Periods List

-- --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
-- Social History SDOH

DROP TABLE IF EXISTS `form_history_sdoh`;
CREATE TABLE `form_history_sdoh`
(
    `id`                              bigint(21) UNSIGNED NOT NULL AUTO_INCREMENT,
    `uuid`                            binary(16)                   DEFAULT NULL,
    `pid`                             int(10) UNSIGNED    NOT NULL,
    `encounter`                       int(10) UNSIGNED             DEFAULT NULL,
    `created_at`                      datetime            NOT NULL DEFAULT current_timestamp(),
    `updated_at`                      datetime            NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
    `created_by`                      int(10) UNSIGNED             DEFAULT NULL COMMENT 'fk to users.id user that created this record',
    `updated_by`                      int(10) UNSIGNED             DEFAULT NULL COMMENT 'fk to users.id user that last modified this record',
    `assessment_date`                 date                         DEFAULT NULL,
    `screening_tool`                  varchar(255)                 DEFAULT NULL COMMENT 'fk to list_options.option_id WHERE list_id=sdoh_instruments represents the assessment tool used to administer this assessment',
    `assessor`                        varchar(255)                 DEFAULT NULL COMMENT 'fk to users.username the user that administered the assessment',
    `food_insecurity`                 varchar(50)                  DEFAULT NULL COMMENT 'fk to list_options.option_id WHERE list_id=sdoh_food_insecurity_risk',
    `food_insecurity_notes`           text,
    `housing_instability`             varchar(50)                  DEFAULT NULL COMMENT 'fk to list_options.option_id WHERE list_id=sdoh_housing_worry',
    `housing_instability_notes`       text,
    `transportation_insecurity`       varchar(50)                  DEFAULT NULL COMMENT 'fk to list_options.option_id WHERE list_id=sdoh_transportation_barrier',
    `transportation_insecurity_notes` text,
    `utilities_insecurity`            varchar(50)                  DEFAULT NULL COMMENT 'fk to list_options.option_id WHERE list_id=sdoh_utilities_shutoff',
    `utilities_insecurity_notes`      text,
    `interpersonal_safety`            varchar(50)                  DEFAULT NULL COMMENT 'fk to list_options.option_id WHERE list_id=sdoh_financial_strain',
    `interpersonal_safety_notes`      text,
    `financial_strain`                varchar(50)                  DEFAULT NULL COMMENT 'fk to list_options.option_id WHERE list_id=sdoh_financial_strain',
    `financial_strain_notes`          text,
    `social_isolation`                varchar(50)                  DEFAULT NULL COMMENT 'fk to list_options.option_id WHERE list_id=sdoh_social_isolation_freq',
    `social_isolation_notes`          text,
    `childcare_needs`                 varchar(50)                  DEFAULT NULL COMMENT 'fk to list_options.option_id WHERE list_id=sdoh_childcare_needs',
    `childcare_needs_notes`           text,
    `digital_access`                  varchar(50)                  DEFAULT NULL COMMENT 'fk to list_options.option_id WHERE list_id=sdoh_digital_access',
    `digital_access_notes`            text,
    `employment_status`               varchar(50)                  DEFAULT NULL COMMENT 'fk to list_options.option_id WHERE list_id=sdoh_food_insecurity_risk',
    `education_level`                 varchar(50)                  DEFAULT NULL COMMENT 'fk to list_options.option_id WHERE list_id=sdoh_education_level',
    `caregiver_status`                varchar(20)                  DEFAULT NULL COMMENT 'fk to list_options.option_id WHERE list_id=sdoh_food_insecurity_risk',
    `veteran_status`                  varchar(20)                  DEFAULT NULL COMMENT 'fk to list_options.option_id WHERE list_id=sdoh_food_insecurity_risk',
    `pregnancy_status`                varchar(20)                  DEFAULT NULL COMMENT 'fk to list_options.option_id WHERE list_id=pregnancy_status',
    `pregnancy_edd`                   date                         DEFAULT NULL COMMENT 'Estimated due date for pregnancy',
    `pregnancy_intent`                VARCHAR(32)                  DEFAULT NULL COMMENT 'fk to list_options.option_id WHERE list_id=pregnancy_intent Pregnancy Intent Over Next Year (codes from PregnancyIntent list)',
    `postpartum_status`               varchar(20)                  DEFAULT NULL COMMENT 'fk to list_options.option_id WHERE list_id=postpartum_status',
    `postpartum_end`                  date                         DEFAULT NULL COMMENT 'PostPartum end date',
    `goals`                           text,
    `interventions`                   text,
    `instrument_score`                INT                          DEFAULT NULL,
    `positive_domain_count`           INT                          DEFAULT NULL,
    `declined_flag`                   TINYINT(1)                   DEFAULT NULL,
    `disability_status`               VARCHAR(50)                  DEFAULT NULL COMMENT 'fk to list_options.option_id WHERE list_id=disability_status',
    `disability_status_notes`         TEXT,
    `disability_scale`                TEXT,
    `hunger_q1`                       VARCHAR(50)                  DEFAULT NULL COMMENT 'LOINC 88122-7 response' COMMENT 'fk to list_options.option_id WHERE list_id=vital_signs_answers',
    `hunger_q2`                       VARCHAR(50)                  DEFAULT NULL COMMENT 'LOINC 88123-5 response'COMMENT 'fk to list_options.option_id WHERE list_id=vital_signs_answers',
    `hunger_score`                    INT                          DEFAULT NULL COMMENT 'Calculated HVS score',
    PRIMARY KEY (`id`),
    KEY `uuid_idx` (`uuid`),
    KEY `pid_idx` (`pid`),
    KEY `assessment_idx` (`assessment_date`),
    KEY `encounter_idx` (`encounter`)
) ENGINE = InnoDB;

--
-- Table structure for linking clinical notes to documents
--
DROP TABLE IF EXISTS `clinical_notes_documents`;
CREATE TABLE  `clinical_notes_documents` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `clinical_note_id` bigint(20) NOT NULL COMMENT 'Foreign key to form_clinical_notes.id',
  `document_id` bigint(20) NOT NULL COMMENT 'Foreign key to documents.id',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'When the link was created',
  `created_by` varchar(255) DEFAULT NULL COMMENT 'Username who created the link',
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_note_document` (`clinical_note_id`, `document_id`),
  KEY `idx_clinical_note_id` (`clinical_note_id`),
  KEY `idx_document_id` (`document_id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB COMMENT='Links clinical notes to patient documents';

--
-- Table structure for linking clinical notes to procedure results
--
DROP TABLE IF EXISTS `clinical_notes_procedure_results`;
CREATE TABLE `clinical_notes_procedure_results` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `clinical_note_id` bigint(20) NOT NULL COMMENT 'Foreign key to form_clinical_notes.id',
  `procedure_result_id` bigint(20) NOT NULL COMMENT 'Foreign key to procedure_result.procedure_result_id',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'When the link was created',
  `created_by` varchar(255) DEFAULT NULL COMMENT 'Username who created the link',
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_note_result` (`clinical_note_id`, `procedure_result_id`),
  KEY `idx_clinical_note_id` (`clinical_note_id`),
  KEY `idx_procedure_result_id` (`procedure_result_id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB COMMENT='Links clinical notes to procedure results/lab values';

-- Patient Preferences Database Schema
-- Uses OpenEMR's list_options table for LOINC codes
-- Table for storing patient treatment intervention preferences
DROP TABLE IF EXISTS `patient_treatment_intervention_preferences`;
CREATE TABLE `patient_treatment_intervention_preferences` (
    `id` int(11) NOT NULL AUTO_INCREMENT,
    `uuid` binary(16) DEFAULT NULL,
    `patient_id` int(11) NOT NULL COMMENT 'fk to patient_data.pid',
    `observation_code` varchar(50) NOT NULL COMMENT 'LOINC code',
    `observation_code_text` varchar(255) DEFAULT NULL,
    `value_type` enum('coded','text','boolean') DEFAULT 'coded',
    `value_code` varchar(50) DEFAULT NULL COMMENT 'fk to preference_value_sets.answer_code',
    `value_code_system` varchar(255) DEFAULT NULL COMMENT 'fk to preference_value_sets.answer_system',
    `value_display` varchar(255) DEFAULT NULL COMMENT 'fk to preference_value_sets.answer_display',
    `value_text` text,
    `value_boolean` tinyint(1) DEFAULT NULL,
    `effective_datetime` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `status` varchar(20) DEFAULT 'final' COMMENT 'valid options are final,amended,preliminary',
    `note` text,
    PRIMARY KEY (`id`),
    UNIQUE KEY `unq_uuid` (`uuid`),
    KEY `patient_id` (`patient_id`),
    KEY `observation_code` (`observation_code`),
    KEY `status` (`status`)
    ) ENGINE=InnoDB;

    -- Table for storing patient care experience preferences
DROP TABLE IF EXISTS `patient_care_experience_preferences`;
CREATE TABLE `patient_care_experience_preferences` (
    `id` int(11) NOT NULL AUTO_INCREMENT,
    `uuid` binary(16) DEFAULT NULL,
    `patient_id` int(11) NOT NULL,
    `observation_code` varchar(50) NOT NULL COMMENT 'LOINC code',
    `observation_code_text` varchar(255) DEFAULT NULL,
    `value_type` enum('coded','text','boolean') DEFAULT 'coded',
    `value_code` varchar(50) DEFAULT NULL COMMENT 'fk to preference_value_sets.answer_code',
    `value_code_system` varchar(255) DEFAULT NULL COMMENT 'fk to preference_value_sets.answer_system',
    `value_display` varchar(255) DEFAULT NULL COMMENT 'fk to preference_value_sets.answer_display',
    `value_text` text,
    `value_boolean` tinyint(1) DEFAULT NULL,
    `effective_datetime` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `status` varchar(20) DEFAULT 'final' COMMENT 'valid options are final,amended,preliminary',
    `note` text,
    PRIMARY KEY (`id`),
    UNIQUE KEY `unq_uuid` (`uuid`),
    KEY `patient_id` (`patient_id`),
    KEY `observation_code` (`observation_code`),
    KEY `status` (`status`)
) ENGINE=InnoDB;

    -- ------------------------------------- Parent lists under `lists`--------------------------------------------------------------------
    INSERT INTO `list_options` (`list_id`,`option_id`,`title`,`seq`)
    VALUES  ('lists','treatment_intervention_preferences','Treatment Intervention Preferences',1);
    INSERT INTO `list_options` (`list_id`,`option_id`,`title`,`seq`,`notes`,`codes`,`activity`) VALUES
    ('treatment_intervention_preferences','81329-5','Thoughts on resuscitation (CPR)',10,'tip_resuscitation_answers','LOINC:81329-5',1),
    ('treatment_intervention_preferences','81330-3','Thoughts on intubation',20,'tip_intubation_answers','LOINC:81330-3',1),
    ('treatment_intervention_preferences','81331-1','Thoughts on tube feeding',30,'tip_tubefeeding_answers','LOINC:81331-1',1),
    ('treatment_intervention_preferences','81332-9','Thoughts on IV fluid and support',40,'tip_ivfluids_answers','LOINC:81332-9',1),
    ('treatment_intervention_preferences','81333-7','Thoughts on antibiotics',50,'tip_antibiotics_answers','LOINC:81333-7',1),
    ('treatment_intervention_preferences','75773-2','Goals, preferences, and priorities for medical treatment [Reported]',5,'tip_general_answers','LOINC:75773-2',1),
    ('treatment_intervention_preferences','81336-0','Patient''s thoughts on cardiopulmonary bypass',60,'tip_bypass_answers','LOINC:81336-0',1),
    ('treatment_intervention_preferences','81337-8','Patient''s thoughts on mechanical ventilation',70,'tip_ventilation_answers','LOINC:81337-8',1),
    ('treatment_intervention_preferences','81376-6','Upon death organ donation consent',80,'tip_organ_donation_answers','LOINC:81376-6',1),
    ('treatment_intervention_preferences','81378-2','Patient Healthcare goals',90,'tip_healthcare_goals_text','LOINC:81378-2',1);

    INSERT INTO `list_options` (`list_id`,`option_id`,`title`,`seq`)
    VALUES ('lists','care_experience_preferences','Care Experience Preferences',1);
    INSERT INTO `list_options` (`list_id`,`option_id`,`title`,`seq`,`notes`,`codes`,`activity`) VALUES
    ('care_experience_preferences','95541-9','Care experience preference',10,'cep_general_answers','LOINC:95541-9',1),
    ('care_experience_preferences','81364-2','Religious or cultural beliefs (reported)',20,'cep_religious_answers','LOINC:81364-2',1),
    ('care_experience_preferences','81365-9','Religious/cultural affiliation contact to notify (reported)',30,'cep_religious_contact_answers','LOINC:81365-9',1),
    ('care_experience_preferences','103980-9','Preferred pharmacy',40,'cep_pharmacy_answers','LOINC:103980-9',1),
    ('care_experience_preferences','81338-6','Patient goals, preferences & priorities for care experience',90,'cep_overall_narrative','LOINC:81338-6',1),
    ('care_experience_preferences','81342-8','Care experience preference under certain health conditions',50,'cep_conditional_answers','LOINC:81342-8',1),
    ('care_experience_preferences','81343-6','Care experience preference at end of life',60,'cep_endoflife_answers','LOINC:81343-6',1),
    ('care_experience_preferences','81362-6','Preferred location for healthcare',70,'cep_location_answers','LOINC:81362-6',1),
    ('care_experience_preferences','81363-4','Preferred healthcare professional',80,'cep_professional_answers','LOINC:81363-4',1);
 -- Value sets table for coded answers
DROP TABLE IF EXISTS `preference_value_sets`;
CREATE TABLE `preference_value_sets` (
    `id` int(11) NOT NULL AUTO_INCREMENT,
    `loinc_code` varchar(50) NOT NULL,
    `answer_code` varchar(100) NOT NULL,
    `answer_system` varchar(255) NOT NULL,
    `answer_display` varchar(255) NOT NULL,
    `answer_definition` text,
    `sort_order` int(11) DEFAULT 0,
    `active` tinyint(1) DEFAULT 1,
    PRIMARY KEY (`id`),
    KEY `loinc_code` (`loinc_code`)
    ) ENGINE=InnoDB COMMENT='Answer lists for preference codes';

    INSERT INTO `preference_value_sets`
    (`loinc_code`,`answer_code`,`answer_system`,`answer_display`,`sort_order`,`active`) VALUES
    ('81329-5','LA33470-8','http://loinc.org','Yes CPR',1,1),
    ('81329-5','LA33471-6','http://loinc.org','No CPR (Do Not Attempt Resuscitation)',2,1),
    ('81329-5','UNK','http://terminology.hl7.org/CodeSystem/v3-NullFlavor','Unknown',99,1),
    ('81329-5','OTH','http://terminology.hl7.org/CodeSystem/v3-NullFlavor','Other (see free text)',100,1),
    ('81330-3','373066001','http://snomed.info/sct','Yes',1,1),
    ('81330-3','373067005','http://snomed.info/sct','No',2,1),
    ('81330-3','UNK','http://terminology.hl7.org/CodeSystem/v3-NullFlavor','Unknown',99,1),
    ('81330-3','OTH','http://terminology.hl7.org/CodeSystem/v3-NullFlavor','Other (see free text)',100,1),
    ('81331-1','373066001','http://snomed.info/sct','Yes',1,1),
    ('81331-1','373067005','http://snomed.info/sct','No',2,1),
    ('81331-1','UNK','http://terminology.hl7.org/CodeSystem/v3-NullFlavor','Unknown',99,1),
    ('81331-1','OTH','http://terminology.hl7.org/CodeSystem/v3-NullFlavor','Other (see free text)',100,1),
    ('81332-9','373066001','http://snomed.info/sct','Yes',1,1),
    ('81332-9','373067005','http://snomed.info/sct','No',2,1),
    ('81332-9','UNK','http://terminology.hl7.org/CodeSystem/v3-NullFlavor','Unknown',99,1),
    ('81332-9','OTH','http://terminology.hl7.org/CodeSystem/v3-NullFlavor','Other (see free text)',100,1),
    ('81333-7','373066001','http://snomed.info/sct','Yes',1,1),
    ('81333-7','373067005','http://snomed.info/sct','No',2,1),
    ('81333-7','UNK','http://terminology.hl7.org/CodeSystem/v3-NullFlavor','Unknown',99,1),
    ('81333-7','OTH','http://terminology.hl7.org/CodeSystem/v3-NullFlavor','Other (see free text)',100,1),
    ('81364-2','160542002','http://snomed.info/sct','Muslim',1,1),
    ('81364-2','160540005','http://snomed.info/sct','Jewish',2,1),
    ('81364-2','160539006','http://snomed.info/sct','Christian',3,1),
    ('81364-2','160538003','http://snomed.info/sct','Hindu',4,1),
    ('81364-2','160543007','http://snomed.info/sct','Buddhist',5,1),
    ('81364-2','276119007','http://snomed.info/sct','No religion',6,1),
    ('81364-2','OTH','http://terminology.hl7.org/CodeSystem/v3-NullFlavor','Other',99,1),
    ('81365-9','373066001','http://snomed.info/sct','Yes',1,1),
    ('81365-9','373067005','http://snomed.info/sct','No',2,1),
    ('81365-9','UNK','http://terminology.hl7.org/CodeSystem/v3-NullFlavor','Unknown',99,1),
    ('81365-9','OTH','http://terminology.hl7.org/CodeSystem/v3-NullFlavor','Other (see free text)',100,1),
    ('103980-9','OTH','http://terminology.hl7.org/CodeSystem/v3-NullFlavor','Other (see free text)',100,1),
    ('95541-9','OTH','http://terminology.hl7.org/CodeSystem/v3-NullFlavor','Other (see free text)',100,1),
    ('81338-6','OTH','http://terminology.hl7.org/CodeSystem/v3-NullFlavor','Other (see free text)',100,1);
    INSERT INTO `preference_value_sets`
    (`loinc_code`,`answer_code`,`answer_system`,`answer_display`,`sort_order`,`active`) VALUES
    ('75773-2','385643006','http://snomed.info/sct','Prefers full resuscitation',1,1),
    ('75773-2','385644000','http://snomed.info/sct','Prefers limited resuscitation',2,1),
    ('75773-2','304253006','http://snomed.info/sct','Does not want resuscitation',3,1),
    ('75773-2','395092004','http://snomed.info/sct','Prefers aggressive treatment',4,1),
    ('75773-2','395093009','http://snomed.info/sct','Prefers comfort measures only',5,1),
    ('75773-2','OTH','http://terminology.hl7.org/CodeSystem/v3-NullFlavor','Other',99,1);
    -- Cardiopulmonary Bypass (81336-0)
    INSERT INTO `preference_value_sets`
    (`loinc_code`,`answer_code`,`answer_system`,`answer_display`,`sort_order`,`active`) VALUES
    ('81336-0','373066001','http://snomed.info/sct','Yes',1,1),
    ('81336-0','373067005','http://snomed.info/sct','No',2,1),
    ('81336-0','261665006','http://snomed.info/sct','Unknown',98,1),
    ('81336-0','UNK','http://terminology.hl7.org/CodeSystem/v3-NullFlavor','Unknown',99,1);
    -- Mechanical Ventilation (81337-8)
    INSERT INTO `preference_value_sets`
    (`loinc_code`,`answer_code`,`answer_system`,`answer_display`,`sort_order`,`active`) VALUES
    ('81337-8','LA33470-8','http://loinc.org','Yes ventilation',1,1),
    ('81337-8','LA33471-6','http://loinc.org','No ventilation',2,1),
    ('81337-8','LA32996-3','http://loinc.org','Trial period of ventilation',3,1),
    ('81337-8','UNK','http://terminology.hl7.org/CodeSystem/v3-NullFlavor','Unknown',99,1);
    -- Organ Donation (81376-6)
    INSERT INTO `preference_value_sets`
    (`loinc_code`,`answer_code`,`answer_system`,`answer_display`,`sort_order`,`active`) VALUES
    ('81376-6','LA33-6','http://loinc.org','Yes',1,1),
    ('81376-6','LA32-8','http://loinc.org','No',2,1),
    ('81376-6','LA32948-4','http://loinc.org','Yes, but only certain organs/tissues',3,1),
    ('81376-6','LA4489-6','http://loinc.org','Unknown',99,1);
    -- Care Under Certain Health Conditions (81342-8)
    INSERT INTO `preference_value_sets`
    (`loinc_code`,`answer_code`,`answer_system`,`answer_display`,`sort_order`,`active`) VALUES
    ('81342-8','LA33474-0','http://loinc.org','If mentally incapacitated',1,1),
    ('81342-8','LA33475-7','http://loinc.org','If terminally ill',2,1),
    ('81342-8','LA33476-5','http://loinc.org','If permanently unconscious',3,1),
    ('81342-8','LA33477-3','http://loinc.org','If severe chronic illness',4,1),
    ('81342-8','OTH','http://terminology.hl7.org/CodeSystem/v3-NullFlavor','Other condition',99,1);
    -- Care at End of Life (81343-6)
    INSERT INTO `preference_value_sets`
    (`loinc_code`,`answer_code`,`answer_system`,`answer_display`,`sort_order`,`active`) VALUES
    ('81343-6','395092004','http://snomed.info/sct','Prefers aggressive treatment',1,1),
    ('81343-6','395093009','http://snomed.info/sct','Prefers comfort measures only',2,1),
    ('81343-6','385644000','http://snomed.info/sct','Limited intervention',3,1),
    ('81343-6','225270000','http://snomed.info/sct','Hospice care',4,1),
    ('81343-6','385656005','http://snomed.info/sct','Home death preferred',5,1),
    ('81343-6','OTH','http://terminology.hl7.org/CodeSystem/v3-NullFlavor','Other',99,1);
    -- Preferred Location (81362-6)
    INSERT INTO `preference_value_sets`
    (`loinc_code`,`answer_code`,`answer_system`,`answer_display`,`sort_order`,`active`) VALUES
    ('81362-6','264362003','http://snomed.info/sct','Home',1,1),
    ('81362-6','22232009','http://snomed.info/sct','Hospital',2,1),
    ('81362-6','284546000','http://snomed.info/sct','Hospice',3,1),
    ('81362-6','42665001','http://snomed.info/sct','Nursing home',4,1),
    ('81362-6','413456002','http://snomed.info/sct','Adult day care center',5,1),
    ('81362-6','OTH','http://terminology.hl7.org/CodeSystem/v3-NullFlavor','Other location',99,1);
    -- Preferred Healthcare Professional (81363-4)
    INSERT INTO `preference_value_sets`
    (`loinc_code`,`answer_code`,`answer_system`,`answer_display`,`sort_order`,`active`) VALUES
    ('81363-4','309343006','http://snomed.info/sct','Physician',1,1),
    ('81363-4','106292003','http://snomed.info/sct','Professional nurse',2,1),
    ('81363-4','224571005','http://snomed.info/sct','Nurse practitioner',3,1),
    ('81363-4','449161006','http://snomed.info/sct','Physician assistant',4,1),
    ('81363-4','768730001','http://snomed.info/sct','Home health aide',5,1),
    ('81363-4','OTH','http://terminology.hl7.org/CodeSystem/v3-NullFlavor','Other provider',99,1);
    -- Add more religious/cultural options
    INSERT INTO `preference_value_sets`
    (`loinc_code`,`answer_code`,`answer_system`,`answer_display`,`sort_order`,`active`) VALUES
    ('81364-2','309884000','http://snomed.info/sct','Atheist',7,1),
    ('81364-2','160234004','http://snomed.info/sct','Agnostic',8,1),
    ('81364-2','428821008','http://snomed.info/sct','Latter Day Saints',9,1),
    ('81364-2','80587008','http://snomed.info/sct','Jehovah''s Witness',10,1),
    ('81364-2','309687009','http://snomed.info/sct','Baptist',11,1),
    ('81364-2','160540005','http://snomed.info/sct','Sikh',12,1),
    ('81364-2','LA14063-6','http://loinc.org','Prefer not to answer',98,1);
    -- General Preferences
    INSERT INTO preference_value_sets(`loinc_code`,`answer_code`,`answer_system`,`answer_display`,`sort_order`,`active`) VALUES
    ('95541-9', 314433002, 'http://snomed.info/sct', 'Preference for health professional (finding)', 1, 1);

-- Doctrine Migrations tracking
-- Their tooling will create this automatically, but having it here simplifies
-- schema comparisons.
DROP TABLE IF EXISTS `migrations`;
CREATE TABLE `migrations` (
    `version` varchar(191) NOT NULL,
    `executed_at` datetime DEFAULT NULL,
    `execution_duration_ms` int DEFAULT NULL,
    PRIMARY KEY (`version`)
) ENGINE=InnoDB;
