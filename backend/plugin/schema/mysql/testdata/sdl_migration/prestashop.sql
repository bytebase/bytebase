SET
  SESSION sql_mode='';
SET
  NAMES 'utf8mb4';

CREATE TABLE `ps_accessory` (
  `id_product_1` int(10) unsigned NOT NULL,
  `id_product_2` int(10) unsigned NOT NULL,
  PRIMARY KEY `accessory_product` (`id_product_1`, `id_product_2`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Address info associated with a user */
CREATE TABLE `ps_address` (
  `id_address` int(10) unsigned NOT NULL auto_increment,
  `id_country` int(10) unsigned NOT NULL,
  `id_state` int(10) unsigned DEFAULT NULL,
  `id_customer` int(10) unsigned NOT NULL DEFAULT '0',
  `id_manufacturer` int(10) unsigned NOT NULL DEFAULT '0',
  `id_supplier` int(10) unsigned NOT NULL DEFAULT '0',
  `id_warehouse` int(10) unsigned NOT NULL DEFAULT '0',
  `alias` varchar(32) NOT NULL,
  `company` varchar(255) DEFAULT NULL,
  `lastname` varchar(255) NOT NULL,
  `firstname` varchar(255) NOT NULL,
  `address1` varchar(128) NOT NULL,
  `address2` varchar(128) DEFAULT NULL,
  `postcode` varchar(12) DEFAULT NULL,
  `city` varchar(64) NOT NULL,
  `other` MEDIUMTEXT,
  `phone` varchar(32) DEFAULT NULL,
  `phone_mobile` varchar(32) DEFAULT NULL,
  `vat_number` varchar(32) DEFAULT NULL,
  `dni` varchar(16) DEFAULT NULL,
  `date_add` datetime NOT NULL,
  `date_upd` datetime NOT NULL,
  `active` tinyint(1) unsigned NOT NULL DEFAULT '1',
  `deleted` tinyint(1) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id_address`),
  KEY `address_customer` (`id_customer`),
  KEY `id_country` (`id_country`),
  KEY `id_state` (`id_state`),
  KEY `id_manufacturer` (`id_manufacturer`),
  KEY `id_supplier` (`id_supplier`),
  KEY `id_warehouse` (`id_warehouse`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Used for search, if a search string is present inside the table, search the alias as well */
CREATE TABLE `ps_alias` (
  `id_alias` int(10) unsigned NOT NULL auto_increment,
  `alias` varchar(191) NOT NULL,
  `search` varchar(255) NOT NULL,
  `active` tinyint(1) NOT NULL DEFAULT '1',
  PRIMARY KEY (`id_alias`),
  UNIQUE KEY `alias` (`alias`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Contains all virtual products (attachements, like images, files, ...) */
CREATE TABLE `ps_attachment` (
  `id_attachment` int(10) unsigned NOT NULL auto_increment,
  `file` varchar(40) NOT NULL,
  `file_name` varchar(255) NOT NULL,
  `file_size` bigint(10) unsigned NOT NULL DEFAULT '0',
  `mime` varchar(128) NOT NULL,
  PRIMARY KEY (`id_attachment`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Name / Description linked to an attachment, localised */
CREATE TABLE `ps_attachment_lang` (
  `id_attachment` int(10) unsigned NOT NULL auto_increment,
  `id_lang` int(10) unsigned NOT NULL,
  `name` varchar(255) DEFAULT NULL,
  `description` MEDIUMTEXT,
  PRIMARY KEY (`id_attachment`, `id_lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Relationship between a product and an attachment */
CREATE TABLE `ps_product_attachment` (
  `id_product` int(10) unsigned NOT NULL,
  `id_attachment` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id_product`, `id_attachment`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Describe the carrier informations */
CREATE TABLE `ps_carrier` (
  `id_carrier` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `id_reference` int(10) unsigned NOT NULL,
  `name` varchar(64) NOT NULL,
  `url` varchar(255) DEFAULT NULL,
  `active` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `deleted` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `shipping_handling` tinyint(1) unsigned NOT NULL DEFAULT '1',
  `range_behavior` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `is_module` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `is_free` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `shipping_external` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `need_range` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `external_module_name` varchar(64) DEFAULT NULL,
  `shipping_method` int(2) NOT NULL DEFAULT '0',
  `position` int(10) unsigned NOT NULL DEFAULT '0',
  `max_width` int(10) DEFAULT '0',
  `max_height` int(10) DEFAULT '0',
  `max_depth` int(10) DEFAULT '0',
  `max_weight` DECIMAL(20, 6) DEFAULT '0',
  `grade` int(10) DEFAULT '0',
  PRIMARY KEY (`id_carrier`),
  KEY `deleted` (`deleted`, `active`),
  KEY `reference` (
    `id_reference`, `deleted`, `active`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Localization carrier infos */
CREATE TABLE `ps_carrier_lang` (
  `id_carrier` int(10) unsigned NOT NULL,
  `id_shop` int(11) unsigned NOT NULL DEFAULT '1',
  `id_lang` int(10) unsigned NOT NULL,
  `delay` varchar(512) DEFAULT NULL,
  PRIMARY KEY (
    `id_lang`, `id_shop`, `id_carrier`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Association between a zone and a carrier */
CREATE TABLE `ps_carrier_zone` (
  `id_carrier` int(10) unsigned NOT NULL,
  `id_zone` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id_carrier`, `id_zone`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Describe the metadata associated with the carts */
CREATE TABLE `ps_cart` (
  `id_cart` int(10) unsigned NOT NULL auto_increment,
  `id_shop_group` INT(11) UNSIGNED NOT NULL DEFAULT '1',
  `id_shop` INT(11) UNSIGNED NOT NULL DEFAULT '1',
  `id_carrier` int(10) unsigned NOT NULL,
  `delivery_option` MEDIUMTEXT NOT NULL,
  `id_lang` int(10) unsigned NOT NULL,
  `id_address_delivery` int(10) unsigned NOT NULL,
  `id_address_invoice` int(10) unsigned NOT NULL,
  `id_currency` int(10) unsigned NOT NULL,
  `id_customer` int(10) unsigned NOT NULL,
  `id_guest` int(10) unsigned NOT NULL,
  `secure_key` varchar(32) NOT NULL DEFAULT '-1',
  `recyclable` tinyint(1) unsigned NOT NULL DEFAULT '1',
  `gift` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `gift_message` MEDIUMTEXT,
  `mobile_theme` tinyint(1) NOT NULL DEFAULT '0',
  `allow_seperated_package` TINYINT(1) UNSIGNED NOT NULL DEFAULT '0',
  `date_add` datetime NOT NULL,
  `date_upd` datetime NOT NULL,
  `checkout_session_data` MEDIUMTEXT NULL,
  PRIMARY KEY (`id_cart`),
  KEY `cart_customer` (`id_customer`),
  KEY `id_address_delivery` (`id_address_delivery`),
  KEY `id_address_invoice` (`id_address_invoice`),
  KEY `id_carrier` (`id_carrier`),
  KEY `id_lang` (`id_lang`),
  KEY `id_currency` (`id_currency`),
  KEY `id_guest` (`id_guest`),
  KEY `id_shop_group` (`id_shop_group`),
  KEY `id_shop_2` (`id_shop`, `date_upd`),
  KEY `id_shop` (`id_shop`, `date_add`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Contains all the promo code rules */
CREATE TABLE `ps_cart_rule` (
  `id_cart_rule` int(10) unsigned NOT NULL auto_increment,
  `id_customer` int unsigned NOT NULL DEFAULT '0',
  `date_from` datetime NOT NULL,
  `date_to` datetime DEFAULT NULL,
  `description` MEDIUMTEXT,
  `quantity` int(10) unsigned DEFAULT '0',
  `quantity_per_user` int(10) unsigned DEFAULT '0',
  `priority` int(10) unsigned NOT NULL DEFAULT 1,
  `partial_use` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `code` varchar(254) NOT NULL,
  `minimum_amount` decimal(20, 6) NOT NULL DEFAULT '0.000000',
  `minimum_amount_tax` tinyint(1) NOT NULL DEFAULT '0',
  `minimum_amount_currency` int unsigned NOT NULL DEFAULT '0',
  `minimum_amount_shipping` tinyint(1) NOT NULL DEFAULT '0',
  `country_restriction` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `carrier_restriction` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `group_restriction` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `cart_rule_restriction` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `product_restriction` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `shop_restriction` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `free_shipping` tinyint(1) NOT NULL DEFAULT '0',
  `reduction_percent` decimal(5, 2) NOT NULL DEFAULT '0.00',
  `reduction_amount` decimal(20, 6) NOT NULL DEFAULT '0.000000',
  `reduction_tax` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `reduction_currency` int(10) unsigned NOT NULL DEFAULT '0',
  `reduction_product` int(10) NOT NULL DEFAULT '0',
  `reduction_exclude_special` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `gift_product` int(10) unsigned NOT NULL DEFAULT '0',
  `gift_product_attribute` int(10) unsigned NOT NULL DEFAULT '0',
  `highlight` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `active` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `date_add` datetime NOT NULL,
  `date_upd` datetime NOT NULL,
  `id_cart_rule_type` int(10) unsigned DEFAULT NULL,
  `minimum_product_quantity` int(10) unsigned NOT NULL DEFAULT 0,
  `total_quantity` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id_cart_rule`),
  KEY `id_customer` (
    `id_customer`, `active`, `date_to`
  ),
  KEY `group_restriction` (
    `group_restriction`, `active`, `date_to`
  ),
  KEY `id_customer_2` (
    `id_customer`, `active`, `highlight`,
    `date_to`
  ),
  KEY `group_restriction_2` (
    `group_restriction`, `active`, `highlight`,
    `date_to`
  ),
  KEY `date_from` (`date_from`),
  KEY `date_to` (`date_to`),
  KEY `id_cart_rule_type` (`id_cart_rule_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Localized name assocatied with a promo code */
CREATE TABLE `ps_cart_rule_lang` (
  `id_cart_rule` int(10) unsigned NOT NULL,
  `id_lang` int(10) unsigned NOT NULL,
  `name` varchar(254) NOT NULL,
  PRIMARY KEY (`id_cart_rule`, `id_lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Country associated with a promo code */
CREATE TABLE `ps_cart_rule_country` (
  `id_cart_rule` int(10) unsigned NOT NULL,
  `id_country` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id_cart_rule`, `id_country`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* User group associated with a promo code */
CREATE TABLE `ps_cart_rule_group` (
  `id_cart_rule` int(10) unsigned NOT NULL,
  `id_group` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id_cart_rule`, `id_group`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Carrier associated with a promo code */
CREATE TABLE `ps_cart_rule_carrier` (
  `id_cart_rule` int(10) unsigned NOT NULL,
  `id_carrier` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id_cart_rule`, `id_carrier`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Allowed combination of promo code */
CREATE TABLE `ps_cart_rule_combination` (
  `id_cart_rule_1` int(10) unsigned NOT NULL,
  `id_cart_rule_2` int(10) unsigned NOT NULL,
  PRIMARY KEY (
    `id_cart_rule_1`, `id_cart_rule_2`
  ),
  KEY `id_cart_rule_1` (`id_cart_rule_1`),
  KEY `id_cart_rule_2` (`id_cart_rule_2`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* @TODO : check checkProductRestrictionsFromCart() to understand the code */
CREATE TABLE `ps_cart_rule_product_rule_group` (
  `id_product_rule_group` int(10) unsigned NOT NULL auto_increment,
  `id_cart_rule` int(10) unsigned NOT NULL,
  `quantity` int(10) unsigned NOT NULL DEFAULT 1,
  `type` ENUM(
    'at_least_one_product_rule', 'all_product_rules'
  ) NOT NULL DEFAULT 'at_least_one_product_rule',
  PRIMARY KEY (`id_product_rule_group`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* @TODO : check checkProductRestrictionsFromCart() to understand the code */
CREATE TABLE `ps_cart_rule_product_rule` (
  `id_product_rule` int(10) unsigned NOT NULL auto_increment,
  `id_product_rule_group` int(10) unsigned NOT NULL,
  `type` ENUM(
    'products', 'categories', 'attributes',
    'manufacturers', 'suppliers', 'combinations', 'features'
  ) NOT NULL,
  PRIMARY KEY (`id_product_rule`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* @TODO : check checkProductRestrictionsFromCart() to understand the code */
CREATE TABLE `ps_cart_rule_product_rule_value` (
  `id_product_rule` int(10) unsigned NOT NULL,
  `id_item` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id_product_rule`, `id_item`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Association between a cart and a promo code */
CREATE TABLE `ps_cart_cart_rule` (
  `id_cart` int(10) unsigned NOT NULL,
  `id_cart_rule` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id_cart`, `id_cart_rule`),
  KEY (`id_cart_rule`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Association between a shop and a promo code */
CREATE TABLE `ps_cart_rule_shop` (
  `id_cart_rule` int(10) unsigned NOT NULL,
  `id_shop` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id_cart_rule`, `id_shop`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Discount types for compatibility */
CREATE TABLE `ps_cart_rule_type` (
  `id_cart_rule_type` int(10) unsigned NOT NULL auto_increment,
  `discount_type` varchar(128) NOT NULL,
  `is_core` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `active` tinyint(1) unsigned NOT NULL DEFAULT '1',
  `date_add` datetime NOT NULL,
  `date_upd` datetime NOT NULL,
  PRIMARY KEY (`id_cart_rule_type`),
  UNIQUE KEY `discount_type` (`discount_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Localized names for cart rule types */
CREATE TABLE `ps_cart_rule_type_lang` (
  `id_cart_rule_type` int(10) unsigned NOT NULL,
  `id_lang` int(10) unsigned NOT NULL,
  `name` varchar(254) NOT NULL,
  `description` TEXT,
  PRIMARY KEY (`id_cart_rule_type`, `id_lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Cart rule compatibility table */
CREATE TABLE `ps_cart_rule_compatible_types` (
  `id_cart_rule` int(10) unsigned NOT NULL,
  `id_cart_rule_type` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id_cart_rule`, `id_cart_rule_type`),
  KEY `id_cart_rule` (`id_cart_rule`),
  KEY `id_cart_rule_type` (`id_cart_rule_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* List of products inside a cart */
CREATE TABLE `ps_cart_product` (
  `id_cart` int(10) unsigned NOT NULL,
  `id_product` int(10) unsigned NOT NULL,
  `id_address_delivery` int(10) unsigned NOT NULL DEFAULT '0',
  `id_shop` int(10) unsigned NOT NULL DEFAULT '1',
  `id_product_attribute` int(10) unsigned NOT NULL DEFAULT '0',
  `id_customization` int(10) unsigned NOT NULL DEFAULT '0',
  `quantity` int(10) unsigned NOT NULL DEFAULT '0',
  `date_add` datetime NOT NULL,
  PRIMARY KEY (
    `id_cart`, `id_product`, `id_product_attribute`,
    `id_customization`, `id_address_delivery`
  ),
  KEY `id_product_attribute` (`id_product_attribute`),
  KEY `id_cart_order` (
    `id_cart`, `date_add`, `id_product`,
    `id_product_attribute`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* List of product categories */
CREATE TABLE `ps_category` (
  `id_category` int(10) unsigned NOT NULL auto_increment,
  `id_parent` int(10) unsigned NOT NULL,
  `id_shop_default` int(10) unsigned NOT NULL DEFAULT 1,
  `level_depth` tinyint(3) unsigned NOT NULL DEFAULT '0',
  `nleft` int(10) unsigned NOT NULL DEFAULT '0',
  `nright` int(10) unsigned NOT NULL DEFAULT '0',
  `active` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `date_add` datetime NOT NULL,
  `date_upd` datetime NOT NULL,
  `redirect_type` ENUM(
    '404', '410',
    '301', '302'
    ) NOT NULL DEFAULT '301',
  `id_type_redirected` int(10) unsigned NOT NULL DEFAULT '0',
  `position` int(10) unsigned NOT NULL DEFAULT '0',
  `is_root_category` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id_category`),
  KEY `category_parent` (`id_parent`),
  KEY `nleftrightactive` (`nleft`, `nright`, `active`),
  KEY `level_depth` (`level_depth`),
  KEY `nright` (`nright`),
  KEY `activenleft` (`active`, `nleft`),
  KEY `activenright` (`active`, `nright`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Association between a product category and a group of customer */
CREATE TABLE `ps_category_group` (
  `id_category` int(10) unsigned NOT NULL,
  `id_group` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id_category`, `id_group`),
  KEY `id_category` (`id_category`),
  KEY `id_group` (`id_group`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Localized product category infos */
CREATE TABLE `ps_category_lang` (
  `id_category` int(10) unsigned NOT NULL,
  `id_shop` INT(11) UNSIGNED NOT NULL DEFAULT '1',
  `id_lang` int(10) unsigned NOT NULL,
  `name` varchar(128) NOT NULL,
  `description` MEDIUMTEXT,
  `additional_description` MEDIUMTEXT,
  `link_rewrite` varchar(128) NOT NULL,
  `meta_title` varchar(255) DEFAULT NULL,
  `meta_description` varchar(512) DEFAULT NULL,
  PRIMARY KEY (
    `id_category`, `id_shop`, `id_lang`
  ),
  KEY `category_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Association between a product category and a product */
CREATE TABLE `ps_category_product` (
  `id_category` int(10) unsigned NOT NULL,
  `id_product` int(10) unsigned NOT NULL,
  `position` int(10) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id_category`, `id_product`),
  INDEX (`id_product`),
  INDEX (`id_category`, `position`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Information on content block position and category */
CREATE TABLE `ps_cms` (
  `id_cms` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `id_cms_category` int(10) unsigned NOT NULL,
  `position` int(10) unsigned NOT NULL DEFAULT '0',
  `active` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `indexation` tinyint(1) unsigned NOT NULL DEFAULT '1',
  PRIMARY KEY (`id_cms`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Localized CMS infos */
CREATE TABLE `ps_cms_lang` (
  `id_cms` int(10) unsigned NOT NULL,
  `id_lang` int(10) unsigned NOT NULL,
  `id_shop` int(10) unsigned NOT NULL DEFAULT '1',
  `meta_title` varchar(255) NOT NULL,
  `head_seo_title` varchar(255) DEFAULT NULL,
  `meta_description` varchar(512) DEFAULT NULL,
  `content` longtext,
  `link_rewrite` varchar(128) NOT NULL,
  PRIMARY KEY (`id_cms`, `id_shop`, `id_lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* CMS category informations */
CREATE TABLE `ps_cms_category` (
  `id_cms_category` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `id_parent` int(10) unsigned NOT NULL,
  `level_depth` tinyint(3) unsigned NOT NULL DEFAULT '0',
  `active` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `date_add` datetime NOT NULL,
  `date_upd` datetime NOT NULL,
  `position` int(10) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id_cms_category`),
  KEY `category_parent` (`id_parent`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Localized CMS category info */
CREATE TABLE `ps_cms_category_lang` (
  `id_cms_category` int(10) unsigned NOT NULL,
  `id_lang` int(10) unsigned NOT NULL,
  `id_shop` int(10) unsigned NOT NULL DEFAULT '1',
  `name` varchar(128) NOT NULL,
  `description` MEDIUMTEXT,
  `link_rewrite` varchar(128) NOT NULL,
  `meta_title` varchar(255) DEFAULT NULL,
  `meta_description` varchar(512) DEFAULT NULL,
  PRIMARY KEY (
    `id_cms_category`, `id_shop`, `id_lang`
  ),
  KEY `category_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Association between a CMS category and a shop */
CREATE TABLE `ps_cms_category_shop` (
  `id_cms_category` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `id_shop` INT(11) UNSIGNED NOT NULL,
  PRIMARY KEY (`id_cms_category`, `id_shop`),
  KEY `id_shop` (`id_shop`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Store the configuration, depending on the shop & the group. See configuration.xml to have the list of
existing variables */
CREATE TABLE `ps_configuration` (
  `id_configuration` int(10) unsigned NOT NULL auto_increment,
  `id_shop_group` INT(11) UNSIGNED DEFAULT NULL,
  `id_shop` INT(11) UNSIGNED DEFAULT NULL,
  `name` varchar(254) NOT NULL,
  `value` MEDIUMTEXT,
  `date_add` datetime NOT NULL,
  `date_upd` datetime NOT NULL,
  PRIMARY KEY (`id_configuration`),
  KEY `name` (`name`),
  KEY `id_shop` (`id_shop`),
  KEY `id_shop_group` (`id_shop_group`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Localized configuration info */
CREATE TABLE `ps_configuration_lang` (
  `id_configuration` int(10) unsigned NOT NULL,
  `id_lang` int(10) unsigned NOT NULL,
  `value` MEDIUMTEXT,
  `date_upd` datetime DEFAULT NULL,
  PRIMARY KEY (`id_configuration`, `id_lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Store the KPI configuration variables (dashboard) */
CREATE TABLE `ps_configuration_kpi` (
  `id_configuration_kpi` int(10) unsigned NOT NULL auto_increment,
  `id_shop_group` INT(11) UNSIGNED DEFAULT NULL,
  `id_shop` INT(11) UNSIGNED DEFAULT NULL,
  `name` varchar(64) NOT NULL,
  `value` MEDIUMTEXT,
  `date_add` datetime NOT NULL,
  `date_upd` datetime NOT NULL,
  PRIMARY KEY (`id_configuration_kpi`),
  KEY `name` (`name`),
  KEY `id_shop` (`id_shop`),
  KEY `id_shop_group` (`id_shop_group`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Localized KPI configuration label */
CREATE TABLE `ps_configuration_kpi_lang` (
  `id_configuration_kpi` int(10) unsigned NOT NULL,
  `id_lang` int(10) unsigned NOT NULL,
  `value` MEDIUMTEXT,
  `date_upd` datetime DEFAULT NULL,
  PRIMARY KEY (
    `id_configuration_kpi`, `id_lang`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* User connections log. See PS_STATSDATA_PAGESVIEWS variable */
CREATE TABLE `ps_connections` (
  `id_connections` int(10) unsigned NOT NULL auto_increment,
  `id_shop_group` INT(11) UNSIGNED NOT NULL DEFAULT '1',
  `id_shop` INT(11) UNSIGNED NOT NULL DEFAULT '1',
  `id_guest` int(10) unsigned NOT NULL,
  `id_page` int(10) unsigned NOT NULL,
  `ip_address` BIGINT NULL DEFAULT NULL,
  `date_add` datetime NOT NULL,
  `http_referer` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`id_connections`),
  KEY `id_guest` (`id_guest`),
  KEY `date_add` (`date_add`),
  KEY `id_page` (`id_page`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* User connection pages log. See PS_STATSDATA_CUSTOMER_PAGESVIEWS variable */
CREATE TABLE `ps_connections_page` (
  `id_connections` int(10) unsigned NOT NULL,
  `id_page` int(10) unsigned NOT NULL,
  `time_start` datetime NOT NULL,
  `time_end` datetime DEFAULT NULL,
  PRIMARY KEY (
    `id_connections`, `id_page`, `time_start`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* User connection source log. */
CREATE TABLE `ps_connections_source` (
  `id_connections_source` int(10) unsigned NOT NULL auto_increment,
  `id_connections` int(10) unsigned NOT NULL,
  `http_referer` varchar(255) DEFAULT NULL,
  `request_uri` varchar(255) DEFAULT NULL,
  `keywords` varchar(255) DEFAULT NULL,
  `date_add` datetime NOT NULL,
  PRIMARY KEY (`id_connections_source`),
  KEY `connections` (`id_connections`),
  KEY `orderby` (`date_add`),
  KEY `http_referer` (`http_referer`),
  KEY `request_uri` (`request_uri`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Store technical contact informations */
CREATE TABLE `ps_contact` (
  `id_contact` int(10) unsigned NOT NULL auto_increment,
  `email` varchar(255) NOT NULL,
  `customer_service` tinyint(1) NOT NULL DEFAULT '0',
  `position` tinyint(2) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id_contact`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Localized technical contact infos */
CREATE TABLE `ps_contact_lang` (
  `id_contact` int(10) unsigned NOT NULL,
  `id_lang` int(10) unsigned NOT NULL,
  `name` varchar(255) NOT NULL,
  `description` MEDIUMTEXT,
  PRIMARY KEY (`id_contact`, `id_lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Country specific data */
CREATE TABLE `ps_country` (
  `id_country` int(10) unsigned NOT NULL auto_increment,
  `id_zone` int(10) unsigned NOT NULL,
  `id_currency` int(10) unsigned NOT NULL DEFAULT '0',
  `iso_code` varchar(3) NOT NULL,
  `call_prefix` int(10) NOT NULL DEFAULT '0',
  `active` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `contains_states` tinyint(1) NOT NULL DEFAULT '0',
  `need_identification_number` tinyint(1) NOT NULL DEFAULT '0',
  `need_zip_code` tinyint(1) NOT NULL DEFAULT '1',
  `zip_code_format` varchar(12) NOT NULL DEFAULT '',
  `display_tax_label` BOOLEAN NOT NULL,
  PRIMARY KEY (`id_country`),
  KEY `country_iso_code` (`iso_code`),
  KEY `country_` (`id_zone`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Localized country information */
CREATE TABLE `ps_country_lang` (
  `id_country` int(10) unsigned NOT NULL,
  `id_lang` int(10) unsigned NOT NULL,
  `name` varchar(64) NOT NULL,
  PRIMARY KEY (`id_country`, `id_lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Currency specification */
CREATE TABLE `ps_currency` (
  `id_currency` int(10) unsigned NOT NULL auto_increment,
  `name` varchar(64) NOT NULL, /* Deprecated since 1.7.5.0. Use ps_currency_lang.name instead. */
  `iso_code` varchar(3) NOT NULL DEFAULT '0',
  `numeric_iso_code` varchar(3),
  `precision` int(2) NOT NULL DEFAULT 6,
  `conversion_rate` decimal(13,6) NOT NULL,
  `deleted` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `active` tinyint(1) unsigned NOT NULL DEFAULT '1',
  `unofficial` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `modified` tinyint(1) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id_currency`),
  KEY `currency_iso_code` (`iso_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Localized currency information */
CREATE TABLE `ps_currency_lang` (
  `id_currency` int(10) unsigned NOT NULL,
  `id_lang` int(10) unsigned NOT NULL,
  `name` varchar(255) NOT NULL,
  `symbol` varchar(255) NOT NULL,
  `pattern` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`id_currency`,`id_lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Customer info */
CREATE TABLE `ps_customer` (
  `id_customer` int(10) unsigned NOT NULL auto_increment,
  `id_shop_group` INT(11) UNSIGNED NOT NULL DEFAULT '1',
  `id_shop` INT(11) UNSIGNED NOT NULL DEFAULT '1',
  `id_gender` int(10) unsigned NOT NULL,
  `id_default_group` int(10) unsigned NOT NULL DEFAULT '1',
  `id_lang` int(10) unsigned NULL,
  `id_risk` int(10) unsigned NOT NULL DEFAULT '1',
  `company` varchar(255),
  `siret` varchar(14),
  `ape` varchar(6),
  `firstname` varchar(255) NOT NULL,
  `lastname` varchar(255) NOT NULL,
  `email` varchar(255) NOT NULL,
  `passwd` varchar(255) NOT NULL,
  `last_passwd_gen` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `birthday` date DEFAULT NULL,
  `newsletter` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `ip_registration_newsletter` varchar(15) DEFAULT NULL,
  `newsletter_date_add` datetime DEFAULT NULL,
  `optin` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `website` varchar(128),
  `outstanding_allow_amount` DECIMAL(20, 6) NOT NULL DEFAULT '0.00',
  `show_public_prices` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `max_payment_days` int(10) unsigned NOT NULL DEFAULT '60',
  `secure_key` varchar(32) NOT NULL DEFAULT '-1',
  `note` MEDIUMTEXT,
  `active` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `is_guest` tinyint(1) NOT NULL DEFAULT '0',
  `deleted` tinyint(1) NOT NULL DEFAULT '0',
  `date_add` datetime NOT NULL,
  `date_upd` datetime NOT NULL,
  `reset_password_token` varchar(40) DEFAULT NULL,
  `reset_password_validity` datetime DEFAULT NULL,
  PRIMARY KEY (`id_customer`),
  KEY `customer_email` (`email`),
  KEY `customer_login` (`email`, `passwd`),
  KEY `id_customer_passwd` (`id_customer`, `passwd`),
  KEY `id_gender` (`id_gender`),
  KEY `id_shop_group` (`id_shop_group`),
  KEY `id_shop` (`id_shop`, `date_add`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Customer group association */
CREATE TABLE `ps_customer_group` (
  `id_customer` int(10) unsigned NOT NULL,
  `id_group` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id_customer`, `id_group`),
  INDEX customer_login(id_group),
  KEY `id_customer` (`id_customer`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Customer support private messaging */
CREATE TABLE `ps_customer_message` (
  `id_customer_message` int(10) unsigned NOT NULL auto_increment,
  `id_customer_thread` int(11) DEFAULT NULL,
  `id_employee` int(10) unsigned DEFAULT NULL,
  `id_product` int(10) unsigned DEFAULT NULL,
  `message` MEDIUMTEXT NOT NULL,
  `file_name` varchar(18) DEFAULT NULL,
  `ip_address` varchar(16) DEFAULT NULL,
  `user_agent` varchar(255) DEFAULT NULL,
  `date_add` datetime NOT NULL,
  `date_upd` datetime NOT NULL,
  `private` TINYINT NOT NULL DEFAULT '0',
  `read` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id_customer_message`),
  KEY `id_customer_thread` (`id_customer_thread`),
  KEY `id_employee` (`id_employee`),
  KEY `id_product` (`id_product`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* store the header of already fetched emails from imap support messaging */
CREATE TABLE `ps_customer_message_sync_imap` (
  `md5_header` varbinary(32) NOT NULL,
  KEY `md5_header_index` (
    `md5_header`(4)
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Customer support private messaging */
CREATE TABLE `ps_customer_thread` (
  `id_customer_thread` int(11) unsigned NOT NULL auto_increment,
  `id_shop` INT(11) UNSIGNED NOT NULL DEFAULT '1',
  `id_lang` int(10) unsigned NOT NULL,
  `id_contact` int(10) unsigned NOT NULL,
  `id_customer` int(10) unsigned DEFAULT NULL,
  `id_order` int(10) unsigned DEFAULT NULL,
  `id_product` int(10) unsigned DEFAULT NULL,
  `status` enum(
    'open', 'closed', 'pending1', 'pending2'
  ) NOT NULL DEFAULT 'open',
  `email` varchar(255) NOT NULL,
  `token` varchar(12) DEFAULT NULL,
  `date_add` datetime NOT NULL,
  `date_upd` datetime NOT NULL,
  PRIMARY KEY (`id_customer_thread`),
  KEY `id_shop` (`id_shop`),
  KEY `id_lang` (`id_lang`),
  KEY `id_contact` (`id_contact`),
  KEY `id_customer` (`id_customer`),
  KEY `id_order` (`id_order`),
  KEY `id_product` (`id_product`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Customization associated with a purchase (engraving...) */
CREATE TABLE `ps_customization` (
  `id_customization` int(10) unsigned NOT NULL auto_increment,
  `id_product_attribute` int(10) unsigned NOT NULL DEFAULT '0',
  `id_address_delivery` int(10) UNSIGNED NOT NULL DEFAULT '0',
  `id_cart` int(10) unsigned NOT NULL,
  `id_product` int(10) NOT NULL,
  `quantity` int(10) NOT NULL,
  `quantity_refunded` INT NOT NULL DEFAULT '0',
  `quantity_returned` INT NOT NULL DEFAULT '0',
  `in_cart` TINYINT(1) UNSIGNED NOT NULL DEFAULT '0',
  PRIMARY KEY (
    `id_customization`, `id_cart`, `id_product`,
    `id_address_delivery`
  ),
  KEY `id_product_attribute` (`id_product_attribute`),
  KEY `id_cart_product` (
    `id_cart`, `id_product`, `id_product_attribute`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Customization possibility for a product */
CREATE TABLE `ps_customization_field` (
  `id_customization_field` int(10) unsigned NOT NULL auto_increment,
  `id_product` int(10) unsigned NOT NULL,
  `type` tinyint(1) NOT NULL,
  `required` tinyint(1) NOT NULL,
  `is_module` TINYINT(1) NOT NULL DEFAULT '0',
  `is_deleted` TINYINT(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id_customization_field`),
  KEY `id_product` (`id_product`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Localized customization fields */
CREATE TABLE `ps_customization_field_lang` (
  `id_customization_field` int(10) unsigned NOT NULL,
  `id_lang` int(10) unsigned NOT NULL,
  `id_shop` int(10) UNSIGNED NOT NULL DEFAULT '1',
  `name` varchar(255) NOT NULL,
  PRIMARY KEY (
    `id_customization_field`, `id_lang`,
    `id_shop`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Customization content associated with a purchase (e.g. : text to engrave) */
CREATE TABLE `ps_customized_data` (
  `id_customization` int(10) unsigned NOT NULL,
  `type` tinyint(1) NOT NULL,
  `index` int(3) NOT NULL,
  `value` varchar(1024) NOT NULL,
  `id_module` int(10) NOT NULL DEFAULT '0',
  `price` decimal(20, 6) NOT NULL DEFAULT '0',
  `weight` decimal(20, 6) NOT NULL DEFAULT '0',
  PRIMARY KEY (
    `id_customization`, `type`, `index`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Date range info (used in PS_STATSDATA_PAGESVIEWS mode) */
CREATE TABLE `ps_date_range` (
  `id_date_range` int(10) unsigned NOT NULL auto_increment,
  `time_start` datetime NOT NULL,
  `time_end` datetime NOT NULL,
  PRIMARY KEY (`id_date_range`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Delivery info associated with a carrier and a shop */
CREATE TABLE `ps_delivery` (
  `id_delivery` int(10) unsigned NOT NULL auto_increment,
  `id_shop` INT UNSIGNED NULL DEFAULT NULL,
  `id_shop_group` INT UNSIGNED NULL DEFAULT NULL,
  `id_carrier` int(10) unsigned NOT NULL,
  `id_range_price` int(10) unsigned DEFAULT NULL,
  `id_range_weight` int(10) unsigned DEFAULT NULL,
  `id_zone` int(10) unsigned NOT NULL,
  `price` decimal(20, 6) NOT NULL,
  PRIMARY KEY (`id_delivery`),
  KEY `id_zone` (`id_zone`),
  KEY `id_carrier` (`id_carrier`, `id_zone`),
  KEY `id_range_price` (`id_range_price`),
  KEY `id_range_weight` (`id_range_weight`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Admin users */
CREATE TABLE `ps_employee` (
  `id_employee` int(10) unsigned NOT NULL auto_increment,
  `id_profile` int(10) unsigned NOT NULL,
  `id_lang` int(10) unsigned NOT NULL DEFAULT '0',
  `lastname` varchar(255) NOT NULL,
  `firstname` varchar(255) NOT NULL,
  `email` varchar(255) NOT NULL,
  `passwd` varchar(255) NOT NULL,
  `last_passwd_gen` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `stats_date_from` date DEFAULT NULL,
  `stats_date_to` date DEFAULT NULL,
  `stats_compare_from` date DEFAULT NULL,
  `stats_compare_to` date DEFAULT NULL,
  `stats_compare_option` int(1) unsigned NOT NULL DEFAULT 1,
  `preselect_date_range` varchar(32) DEFAULT NULL,
  `bo_color` varchar(32) DEFAULT NULL,
  `bo_theme` varchar(32) DEFAULT NULL,
  `bo_css` varchar(64) DEFAULT NULL,
  `default_tab` int(10) unsigned NOT NULL DEFAULT '0',
  `bo_width` int(10) unsigned NOT NULL DEFAULT '0',
  `bo_menu` tinyint(1) NOT NULL DEFAULT '1',
  `active` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `optin` tinyint(1) unsigned DEFAULT NULL,
  `id_last_order` int(10) unsigned NOT NULL DEFAULT '0',
  `id_last_customer_message` int(10) unsigned NOT NULL DEFAULT '0',
  `id_last_customer` int(10) unsigned NOT NULL DEFAULT '0',
  `last_connection_date` date DEFAULT NULL,
  `reset_password_token` varchar(40) DEFAULT NULL,
  `reset_password_validity` datetime DEFAULT NULL,
  `has_enabled_gravatar` TINYINT UNSIGNED DEFAULT 0 NOT NULL,
  PRIMARY KEY (`id_employee`),
  KEY `employee_login` (`email`, `passwd`),
  KEY `id_employee_passwd` (`id_employee`, `passwd`),
  KEY `id_profile` (`id_profile`),
  KEY `IDX_1D8DF9EBBA299860` (`id_lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Admin users shop */
CREATE TABLE `ps_employee_shop` (
  `id_employee` INT(11) UNSIGNED NOT NULL,
  `id_shop` INT(11) UNSIGNED NOT NULL,
  PRIMARY KEY (`id_employee`, `id_shop`),
  KEY `id_shop` (`id_shop`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Registry for all declared extra property definitions */
CREATE TABLE `ps_extra_property_definition` (
  `id_extra_property_definition` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `entity_name` varchar(64) NOT NULL,
  `module_name` varchar(64) DEFAULT NULL,
  `property_name` varchar(64) NOT NULL,
  `type` ENUM ('int','bool','string','float','date','html','json','choice') NOT NULL DEFAULT 'string',
  `scope` ENUM ('common','lang','shop') NOT NULL DEFAULT 'common',
  `sql_index` ENUM ('none','key','unique') NOT NULL DEFAULT 'none',
  `size` smallint(5) unsigned DEFAULT NULL,
  `default_value` varchar(255) DEFAULT NULL,
  `required` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `constraints` longtext DEFAULT NULL,
  `display_front` tinyint(1) unsigned NOT NULL DEFAULT 1,
  `associated_apis` text DEFAULT NULL,
  `associated_grids` text DEFAULT NULL,
  `associated_forms` text DEFAULT NULL,
  `form_field_type` varchar(255) DEFAULT NULL,
  `form_options` text DEFAULT NULL,
  `label_wording` varchar(191) DEFAULT NULL,
  `label_domain` varchar(255) DEFAULT NULL,
  `description_wording` varchar(191) DEFAULT NULL,
  `description_domain` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`id_extra_property_definition`),
  UNIQUE KEY `extra_property_definition_unique` (`entity_name`, `module_name`, `property_name`),
  KEY `entity_name` (`entity_name`, `scope`),
  KEY `module_name` (`module_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Position of each feature */
CREATE TABLE `ps_feature` (
  `id_feature` int(10) unsigned NOT NULL auto_increment,
  `position` int(10) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id_feature`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Localized feature info */
CREATE TABLE `ps_feature_lang` (
  `id_feature` int(10) unsigned NOT NULL,
  `id_lang` int(10) unsigned NOT NULL,
  `name` varchar(128) DEFAULT NULL,
  PRIMARY KEY (`id_feature`, `id_lang`),
  KEY (`id_lang`, `name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Association between a feature and a product */
CREATE TABLE `ps_feature_product` (
  `id_feature` int(10) unsigned NOT NULL,
  `id_product` int(10) unsigned NOT NULL,
  `id_feature_value` int(10) unsigned NOT NULL,
  PRIMARY KEY (
    `id_feature`, `id_product`, `id_feature_value`
  ),
  KEY `id_feature_value` (`id_feature_value`),
  KEY `id_product` (`id_product`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Various choice associated with a feature */
CREATE TABLE `ps_feature_value` (
  `id_feature_value` int(10) unsigned NOT NULL auto_increment,
  `id_feature` int(10) unsigned NOT NULL,
  `custom` tinyint(3) unsigned DEFAULT NULL,
  `position` int(10) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id_feature_value`),
  KEY `feature` (`id_feature`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Localized feature choice */
CREATE TABLE `ps_feature_value_lang` (
  `id_feature_value` int(10) unsigned NOT NULL,
  `id_lang` int(10) unsigned NOT NULL,
  `value` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`id_feature_value`, `id_lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* User titles (e.g. : Mr, Mrs...) */
CREATE TABLE IF NOT EXISTS `ps_gender` (
  `id_gender` int(11) NOT NULL AUTO_INCREMENT,
  `type` tinyint(1) NOT NULL,
  PRIMARY KEY (`id_gender`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Localized user title */
CREATE TABLE IF NOT EXISTS `ps_gender_lang` (
  `id_gender` int(10) unsigned NOT NULL,
  `id_lang` int(10) unsigned NOT NULL,
  `name` varchar(20) NOT NULL,
  PRIMARY KEY (`id_gender`, `id_lang`),
  KEY `id_gender` (`id_gender`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Group special price rules */
CREATE TABLE `ps_group` (
  `id_group` int(10) unsigned NOT NULL auto_increment,
  `reduction` decimal(5, 2) NOT NULL DEFAULT '0.00',
  `price_display_method` TINYINT NOT NULL DEFAULT '0',
  `show_prices` tinyint(1) unsigned NOT NULL DEFAULT '1',
  `date_add` datetime NOT NULL,
  `date_upd` datetime NOT NULL,
  PRIMARY KEY (`id_group`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Localized group info */
CREATE TABLE `ps_group_lang` (
  `id_group` int(10) unsigned NOT NULL,
  `id_lang` int(10) unsigned NOT NULL,
  `name` varchar(32) NOT NULL,
  PRIMARY KEY (`id_group`, `id_lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Category specific reduction */
CREATE TABLE `ps_group_reduction` (
  `id_group_reduction` MEDIUMINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `id_group` INT(10) UNSIGNED NOT NULL,
  `id_category` INT(10) UNSIGNED NOT NULL,
  `reduction` DECIMAL(5, 4) NOT NULL,
  PRIMARY KEY (`id_group_reduction`),
  UNIQUE KEY(`id_group`, `id_category`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Cache which store product price after reduction */
CREATE TABLE `ps_product_group_reduction_cache` (
  `id_product` INT UNSIGNED NOT NULL,
  `id_group` INT UNSIGNED NOT NULL,
  `reduction` DECIMAL(5, 4) NOT NULL,
  PRIMARY KEY (`id_product`, `id_group`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Specify a carrier for a given product */
CREATE TABLE `ps_product_carrier` (
  `id_product` int(10) unsigned NOT NULL,
  `id_carrier_reference` int(10) unsigned NOT NULL,
  `id_shop` int(10) unsigned NOT NULL,
  PRIMARY KEY (
    `id_product`, `id_carrier_reference`,
    `id_shop`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Stats from guest user */
CREATE TABLE `ps_guest` (
  `id_guest` int(10) unsigned NOT NULL auto_increment,
  `id_operating_system` int(10) unsigned DEFAULT NULL,
  `id_web_browser` int(10) unsigned DEFAULT NULL,
  `id_customer` int(10) unsigned DEFAULT NULL,
  `javascript` tinyint(1) DEFAULT '0',
  `screen_resolution_x` smallint(5) unsigned DEFAULT NULL,
  `screen_resolution_y` smallint(5) unsigned DEFAULT NULL,
  `screen_color` tinyint(3) unsigned DEFAULT NULL,
  `sun_java` tinyint(1) DEFAULT NULL,
  `adobe_flash` tinyint(1) DEFAULT NULL,
  `adobe_director` tinyint(1) DEFAULT NULL,
  `apple_quicktime` tinyint(1) DEFAULT NULL,
  `real_player` tinyint(1) DEFAULT NULL,
  `windows_media` tinyint(1) DEFAULT NULL,
  `accept_language` varchar(8) DEFAULT NULL,
  `mobile_theme` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id_guest`),
  KEY `id_customer` (`id_customer`),
  KEY `id_operating_system` (`id_operating_system`),
  KEY `id_web_browser` (`id_web_browser`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Store hook description */
CREATE TABLE `ps_hook` (
  `id_hook` int(10) unsigned NOT NULL auto_increment,
  `name` varchar(191) NOT NULL,
  `title` varchar(255) NOT NULL,
  `description` MEDIUMTEXT,
  `active` tinyint(1) unsigned NOT NULL DEFAULT '1',
  `position` tinyint(1) NOT NULL DEFAULT '1',
  PRIMARY KEY (`id_hook`),
  UNIQUE KEY `hook_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Hook alias name */
CREATE TABLE `ps_hook_alias` (
  `id_hook_alias` int(10) unsigned NOT NULL auto_increment,
  `alias` varchar(191) NOT NULL,
  `name` varchar(191) NOT NULL,
  PRIMARY KEY (`id_hook_alias`),
  UNIQUE KEY `alias` (`alias`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Define registered hook module */
CREATE TABLE `ps_hook_module` (
  `id_module` int(10) unsigned NOT NULL,
  `id_shop` INT(11) UNSIGNED NOT NULL DEFAULT '1',
  `id_hook` int(10) unsigned NOT NULL,
  `position` tinyint(2) unsigned NOT NULL,
  PRIMARY KEY (
    `id_module`, `id_hook`, `id_shop`
  ),
  KEY `id_hook` (`id_hook`),
  KEY `id_module` (`id_module`),
  KEY `position` (`id_shop`, `position`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* List of page type where the hook is not loaded */
CREATE TABLE `ps_hook_module_exceptions` (
  `id_hook_module_exceptions` int(10) unsigned NOT NULL auto_increment,
  `id_shop` INT(11) UNSIGNED NOT NULL DEFAULT '1',
  `id_module` int(10) unsigned NOT NULL,
  `id_hook` int(10) unsigned NOT NULL,
  `file_name` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`id_hook_module_exceptions`),
  KEY `id_module` (`id_module`),
  KEY `id_hook` (`id_hook`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Product image info */
CREATE TABLE `ps_image` (
  `id_image` int(10) unsigned NOT NULL auto_increment,
  `id_product` int(10) unsigned NOT NULL,
  `position` smallint(2) unsigned NOT NULL DEFAULT '0',
  `cover` tinyint(1) unsigned NULL DEFAULT NULL,
  PRIMARY KEY (`id_image`),
  KEY `image_product` (`id_product`),
  UNIQUE KEY `id_product_cover` (`id_product`, `cover`),
  UNIQUE KEY `idx_product_image` (
    `id_image`, `id_product`, `cover`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Localized product image */
CREATE TABLE `ps_image_lang` (
  `id_image` int(10) unsigned NOT NULL,
  `id_lang` int(10) unsigned NOT NULL,
  `legend` varchar(128) DEFAULT NULL,
  PRIMARY KEY (`id_image`, `id_lang`),
  KEY `id_image` (`id_image`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Manufacturer info */
CREATE TABLE `ps_manufacturer` (
  `id_manufacturer` int(10) unsigned NOT NULL auto_increment,
  `name` varchar(64) NOT NULL,
  `date_add` datetime NOT NULL,
  `date_upd` datetime NOT NULL,
  `active` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id_manufacturer`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* localized manufacturer info */
CREATE TABLE `ps_manufacturer_lang` (
  `id_manufacturer` int(10) unsigned NOT NULL,
  `id_lang` int(10) unsigned NOT NULL,
  `description` MEDIUMTEXT,
  `short_description` MEDIUMTEXT,
  `meta_title` varchar(255) DEFAULT NULL,
  `meta_description` varchar(512) DEFAULT NULL,
  PRIMARY KEY (`id_manufacturer`, `id_lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Private messaging */
CREATE TABLE `ps_message` (
  `id_message` int(10) unsigned NOT NULL auto_increment,
  `id_cart` int(10) unsigned DEFAULT NULL,
  `id_customer` int(10) unsigned NOT NULL,
  `id_employee` int(10) unsigned DEFAULT NULL,
  `id_order` int(10) unsigned NOT NULL,
  `message` MEDIUMTEXT NOT NULL,
  `private` tinyint(1) unsigned NOT NULL DEFAULT '1',
  `date_add` datetime NOT NULL,
  PRIMARY KEY (`id_message`),
  KEY `message_order` (`id_order`),
  KEY `id_cart` (`id_cart`),
  KEY `id_customer` (`id_customer`),
  KEY `id_employee` (`id_employee`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Private messaging read flag */
CREATE TABLE `ps_message_readed` (
  `id_message` int(10) unsigned NOT NULL,
  `id_employee` int(10) unsigned NOT NULL,
  `date_add` datetime NOT NULL,
  PRIMARY KEY (`id_message`, `id_employee`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* List of route type that can be localized */
CREATE TABLE `ps_meta` (
  `id_meta` int(10) unsigned NOT NULL auto_increment,
  `page` varchar(64) NOT NULL,
  `configurable` TINYINT(1) UNSIGNED NOT NULL DEFAULT '1',
  PRIMARY KEY (`id_meta`),
  UNIQUE KEY `page` (`page`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Localized routes */
CREATE TABLE `ps_meta_lang` (
  `id_meta` int(10) unsigned NOT NULL,
  `id_shop` INT(11) UNSIGNED NOT NULL DEFAULT '1',
  `id_lang` int(10) unsigned NOT NULL,
  `title` varchar(128) DEFAULT NULL,
  `description` varchar(255) DEFAULT NULL,
  `url_rewrite` varchar(255) NOT NULL,
  PRIMARY KEY (`id_meta`, `id_shop`, `id_lang`),
  KEY `id_shop` (`id_shop`),
  KEY `id_lang` (`id_lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Installed module list */
CREATE TABLE `ps_module` (
  `id_module` int(10) unsigned NOT NULL auto_increment,
  `name` varchar(64) NOT NULL,
  `active` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `version` VARCHAR(8) NOT NULL,
  PRIMARY KEY (`id_module`),
  UNIQUE KEY `name_UNIQUE` (`name`),
  KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Module / class authorization_role */
CREATE TABLE `ps_authorization_role` (
  `id_authorization_role` int(10) unsigned NOT NULL auto_increment,
  `slug` VARCHAR(191) NOT NULL,
  PRIMARY KEY (`id_authorization_role`),
  UNIQUE KEY (`slug`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Association between a profile and a module authorization_role (can be 'CREATE', 'READ', 'UPDATE' or 'DELETE') */
CREATE TABLE `ps_module_access` (
  `id_profile` int(10) unsigned NOT NULL,
  `id_authorization_role` int(10) unsigned NOT NULL,
  PRIMARY KEY (
    `id_profile`, `id_authorization_role`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* countries allowed for each module (e.g. : countries supported for a payment module) */
CREATE TABLE `ps_module_country` (
  `id_module` int(10) unsigned NOT NULL,
  `id_shop` INT(11) UNSIGNED NOT NULL DEFAULT '1',
  `id_country` int(10) unsigned NOT NULL,
  PRIMARY KEY (
    `id_module`, `id_shop`, `id_country`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* currencies allowed for each module */
CREATE TABLE `ps_module_currency` (
  `id_module` int(10) unsigned NOT NULL,
  `id_shop` INT(11) UNSIGNED NOT NULL DEFAULT '1',
  `id_currency` int(11) NOT NULL,
  PRIMARY KEY (
    `id_module`, `id_shop`, `id_currency`
  ),
  KEY `id_module` (`id_module`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* groups allowed for each module */
CREATE TABLE `ps_module_group` (
  `id_module` int(10) unsigned NOT NULL,
  `id_shop` INT(11) UNSIGNED NOT NULL DEFAULT '1',
  `id_group` int(11) unsigned NOT NULL,
  PRIMARY KEY (
    `id_module`, `id_shop`, `id_group`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* carriers allowed for each module */
CREATE TABLE `ps_module_carrier` (
  `id_module` INT(10) unsigned NOT NULL,
  `id_shop` INT(11) unsigned NOT NULL DEFAULT '1',
  `id_reference` INT(11) NOT NULL,
  PRIMARY KEY (
    `id_module`, `id_shop`, `id_reference`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* List of OS (used in guest stats) */
CREATE TABLE `ps_operating_system` (
  `id_operating_system` int(10) unsigned NOT NULL auto_increment,
  `name` varchar(64) DEFAULT NULL,
  PRIMARY KEY (`id_operating_system`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* List of orders */
CREATE TABLE `ps_orders` (
  `id_order` int(10) unsigned NOT NULL auto_increment,
  `reference` VARCHAR(255),
  `id_shop_group` INT(11) UNSIGNED NOT NULL DEFAULT '1',
  `id_shop` INT(11) UNSIGNED NOT NULL DEFAULT '1',
  `id_carrier` int(10) unsigned NOT NULL,
  `id_lang` int(10) unsigned NOT NULL,
  `id_customer` int(10) unsigned NOT NULL,
  `id_cart` int(10) unsigned NOT NULL,
  `id_currency` int(10) unsigned NOT NULL,
  `id_address_delivery` int(10) unsigned NOT NULL,
  `id_address_invoice` int(10) unsigned NOT NULL,
  `current_state` int(10) unsigned NOT NULL,
  `secure_key` varchar(32) NOT NULL DEFAULT '-1',
  `payment` varchar(255) NOT NULL,
  `conversion_rate` decimal(13, 6) NOT NULL DEFAULT 1,
  `module` varchar(255) DEFAULT NULL,
  `recyclable` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `gift` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `gift_message` MEDIUMTEXT,
  `mobile_theme` tinyint(1) NOT NULL DEFAULT '0',
  `total_discounts` decimal(20, 6) NOT NULL DEFAULT '0.00',
  `total_discounts_tax_incl` decimal(20, 6) NOT NULL DEFAULT '0.00',
  `total_discounts_tax_excl` decimal(20, 6) NOT NULL DEFAULT '0.00',
  `total_paid` decimal(20, 6) NOT NULL DEFAULT '0.00',
  `total_paid_tax_incl` decimal(20, 6) NOT NULL DEFAULT '0.00',
  `total_paid_tax_excl` decimal(20, 6) NOT NULL DEFAULT '0.00',
  `total_paid_real` decimal(20, 6) NOT NULL DEFAULT '0.00',
  `total_products` decimal(20, 6) NOT NULL DEFAULT '0.00',
  `total_products_wt` decimal(20, 6) NOT NULL DEFAULT '0.00',
  `total_shipping` decimal(20, 6) NOT NULL DEFAULT '0.00',
  `total_shipping_tax_incl` decimal(20, 6) NOT NULL DEFAULT '0.00',
  `total_shipping_tax_excl` decimal(20, 6) NOT NULL DEFAULT '0.00',
  `carrier_tax_rate` DECIMAL(10, 3) NOT NULL DEFAULT '0.00',
  `total_wrapping` decimal(20, 6) NOT NULL DEFAULT '0.00',
  `total_wrapping_tax_incl` decimal(20, 6) NOT NULL DEFAULT '0.00',
  `total_wrapping_tax_excl` decimal(20, 6) NOT NULL DEFAULT '0.00',
  `round_mode` tinyint(1) NOT NULL DEFAULT '2',
  `round_type` tinyint(1) NOT NULL DEFAULT '1',
  `invoice_number` int(10) unsigned NOT NULL DEFAULT '0',
  `delivery_number` int(10) unsigned NOT NULL DEFAULT '0',
  `invoice_date` datetime NOT NULL,
  `delivery_date` datetime NOT NULL,
  `valid` int(1) unsigned NOT NULL DEFAULT '0',
  `date_add` datetime NOT NULL,
  `date_upd` datetime NOT NULL,
  `note` MEDIUMTEXT,
  PRIMARY KEY (`id_order`),
  KEY `reference` (`reference`),
  KEY `id_customer` (`id_customer`),
  KEY `id_cart` (`id_cart`),
  KEY `invoice_number` (`invoice_number`),
  KEY `id_carrier` (`id_carrier`),
  KEY `id_lang` (`id_lang`),
  KEY `id_currency` (`id_currency`),
  KEY `id_address_delivery` (`id_address_delivery`),
  KEY `id_address_invoice` (`id_address_invoice`),
  KEY `id_shop_group` (`id_shop_group`),
  KEY (`current_state`),
  KEY `id_shop` (`id_shop`),
  INDEX `date_add`(`date_add`),
  INDEX `invoice_date`(`invoice_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Order tax detail */
CREATE TABLE `ps_order_detail_tax` (
  `id_order_detail` int(11) NOT NULL,
  `id_tax` int(11) NOT NULL,
  `unit_amount` DECIMAL(16, 6) NOT NULL DEFAULT '0.00',
  `total_amount` DECIMAL(16, 6) NOT NULL DEFAULT '0.00',
  KEY (`id_order_detail`),
  KEY `id_tax` (`id_tax`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* list of invoice */
CREATE TABLE `ps_order_invoice` (
  `id_order_invoice` int(11) UNSIGNED NOT NULL AUTO_INCREMENT,
  `id_order` int(11) NOT NULL,
  `number` int(11) NOT NULL,
  `delivery_number` int(11) NOT NULL,
  `delivery_date` datetime,
  `total_discount_tax_excl` decimal(20, 6) NOT NULL DEFAULT '0.00',
  `total_discount_tax_incl` decimal(20, 6) NOT NULL DEFAULT '0.00',
  `total_paid_tax_excl` decimal(20, 6) NOT NULL DEFAULT '0.00',
  `total_paid_tax_incl` decimal(20, 6) NOT NULL DEFAULT '0.00',
  `total_products` decimal(20, 6) NOT NULL DEFAULT '0.00',
  `total_products_wt` decimal(20, 6) NOT NULL DEFAULT '0.00',
  `total_shipping_tax_excl` decimal(20, 6) NOT NULL DEFAULT '0.00',
  `total_shipping_tax_incl` decimal(20, 6) NOT NULL DEFAULT '0.00',
  `shipping_tax_computation_method` int(10) unsigned NOT NULL,
  `total_wrapping_tax_excl` decimal(20, 6) NOT NULL DEFAULT '0.00',
  `total_wrapping_tax_incl` decimal(20, 6) NOT NULL DEFAULT '0.00',
  `shop_address` MEDIUMTEXT DEFAULT NULL,
  `note` MEDIUMTEXT,
  `date_add` datetime NOT NULL,
  PRIMARY KEY (`id_order_invoice`),
  KEY `id_order` (`id_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* global invoice tax */
CREATE TABLE IF NOT EXISTS `ps_order_invoice_tax` (
  `id_order_invoice` int(11) NOT NULL,
  `type` varchar(15) NOT NULL,
  `id_tax` int(11) NOT NULL,
  `amount` decimal(10, 6) NOT NULL DEFAULT '0.000000',
  KEY `id_tax` (`id_tax`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* order detail (every product inside an order) */
CREATE TABLE `ps_order_detail` (
  `id_order_detail` int(10) unsigned NOT NULL auto_increment,
  `id_order` int(10) unsigned NOT NULL,
  `id_order_invoice` int(11) DEFAULT NULL,
  `id_warehouse` int(10) unsigned DEFAULT '0',
  `id_shop` int(11) unsigned NOT NULL,
  `product_id` int(10) unsigned NOT NULL,
  `product_attribute_id` int(10) unsigned DEFAULT NULL,
  `id_customization` int(10) unsigned DEFAULT 0,
  `product_name` MEDIUMTEXT NOT NULL,
  `product_quantity` int(10) unsigned NOT NULL DEFAULT '0',
  `product_quantity_in_stock` int(10) NOT NULL DEFAULT '0',
  `product_quantity_refunded` int(10) unsigned NOT NULL DEFAULT '0',
  `product_quantity_return` int(10) unsigned NOT NULL DEFAULT '0',
  `product_quantity_reinjected` int(10) unsigned NOT NULL DEFAULT '0',
  `product_price` decimal(20, 6) NOT NULL DEFAULT '0.000000',
  `reduction_percent` DECIMAL(5, 2) NOT NULL DEFAULT '0.00',
  `reduction_amount` DECIMAL(20, 6) NOT NULL DEFAULT '0.000000',
  `reduction_amount_tax_incl` DECIMAL(20, 6) NOT NULL DEFAULT '0.000000',
  `reduction_amount_tax_excl` DECIMAL(20, 6) NOT NULL DEFAULT '0.000000',
  `group_reduction` DECIMAL(5, 2) NOT NULL DEFAULT '0.00',
  `product_quantity_discount` decimal(20, 6) NOT NULL DEFAULT '0.000000',
  `product_ean13` varchar(20) DEFAULT NULL,
  `product_isbn` varchar(32) DEFAULT NULL,
  `product_upc` varchar(12) DEFAULT NULL,
  `product_mpn` varchar(40) DEFAULT NULL,
  `product_reference` varchar(64) DEFAULT NULL,
  `product_supplier_reference` varchar(64) DEFAULT NULL,
  `product_weight` DECIMAL(20, 6) NOT NULL,
  `id_tax_rules_group` INT(11) UNSIGNED DEFAULT '0',
  `tax_computation_method` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `tax_name` varchar(16) NOT NULL,
  `tax_rate` DECIMAL(10, 3) NOT NULL DEFAULT '0.000',
  `ecotax` decimal(17, 6) NOT NULL DEFAULT '0.000000',
  `ecotax_tax_rate` DECIMAL(5, 3) NOT NULL DEFAULT '0.000',
  `discount_quantity_applied` TINYINT(1) NOT NULL DEFAULT '0',
  `download_hash` varchar(255) DEFAULT NULL,
  `download_nb` int(10) unsigned DEFAULT '0',
  `download_deadline` datetime DEFAULT NULL,
  `total_price_tax_incl` DECIMAL(20, 6) NOT NULL DEFAULT '0.000000',
  `total_price_tax_excl` DECIMAL(20, 6) NOT NULL DEFAULT '0.000000',
  `unit_price_tax_incl` DECIMAL(20, 6) NOT NULL DEFAULT '0.000000',
  `unit_price_tax_excl` DECIMAL(20, 6) NOT NULL DEFAULT '0.000000',
  `total_shipping_price_tax_incl` DECIMAL(20, 6) NOT NULL DEFAULT '0.000000',
  `total_shipping_price_tax_excl` DECIMAL(20, 6) NOT NULL DEFAULT '0.000000',
  `purchase_supplier_price` DECIMAL(20, 6) NOT NULL DEFAULT '0.000000',
  `original_product_price` DECIMAL(20, 6) NOT NULL DEFAULT '0.000000',
  `original_wholesale_price` DECIMAL(20, 6) NOT NULL DEFAULT '0.000000',
  `total_refunded_tax_excl` DECIMAL(20, 6) NOT NULL DEFAULT '0.000000',
  `total_refunded_tax_incl` DECIMAL(20, 6) NOT NULL DEFAULT '0.000000',
  PRIMARY KEY (`id_order_detail`),
  KEY `order_detail_order` (`id_order`),
  KEY `product_id` (
    `product_id`, `product_attribute_id`
  ),
  KEY `product_attribute_id` (`product_attribute_id`),
  KEY `id_tax_rules_group` (`id_tax_rules_group`),
  KEY `id_order_id_order_detail` (`id_order`, `id_order_detail`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Promo code used in the order */
CREATE TABLE `ps_order_cart_rule` (
  `id_order_cart_rule` int(10) unsigned NOT NULL auto_increment,
  `id_order` int(10) unsigned NOT NULL,
  `id_cart_rule` int(10) unsigned NOT NULL,
  `id_order_invoice` int(10) unsigned DEFAULT '0',
  `name` varchar(254) NOT NULL,
  `value` decimal(20, 6) NOT NULL DEFAULT '0.000000',
  `value_tax_excl` decimal(20, 6) NOT NULL DEFAULT '0.000000',
  `free_shipping` tinyint(1) NOT NULL DEFAULT '0',
  `deleted` tinyint(1) UNSIGNED NOT NULL DEFAULT '0',
  PRIMARY KEY (`id_order_cart_rule`),
  KEY `id_order` (`id_order`),
  KEY `id_cart_rule` (`id_cart_rule`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* order transactional information */
CREATE TABLE `ps_order_history` (
  `id_order_history` int(10) unsigned NOT NULL auto_increment,
  `id_employee` int(10) unsigned NOT NULL,
  `id_order` int(10) unsigned NOT NULL,
  `id_order_state` int(10) unsigned NOT NULL,
  `date_add` datetime NOT NULL,
  PRIMARY KEY (`id_order_history`),
  KEY `order_history_order` (`id_order`),
  KEY `id_employee` (`id_employee`),
  KEY `id_order_state` (`id_order_state`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Type of predefined message that can be inserted to an order */
CREATE TABLE `ps_order_message` (
  `id_order_message` int(10) unsigned NOT NULL auto_increment,
  `date_add` datetime NOT NULL,
  PRIMARY KEY (`id_order_message`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Localized predefined order message */
CREATE TABLE `ps_order_message_lang` (
  `id_order_message` int(10) unsigned NOT NULL,
  `id_lang` int(10) unsigned NOT NULL,
  `name` varchar(128) NOT NULL,
  `message` MEDIUMTEXT NOT NULL,
  PRIMARY KEY (`id_order_message`, `id_lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Return state associated with an order */
CREATE TABLE `ps_order_return` (
  `id_order_return` int(10) unsigned NOT NULL auto_increment,
  `id_customer` int(10) unsigned NOT NULL,
  `id_order` int(10) unsigned NOT NULL,
  `state` tinyint(1) unsigned NOT NULL DEFAULT '1',
  `question` MEDIUMTEXT NOT NULL,
  `date_add` datetime NOT NULL,
  `date_upd` datetime NOT NULL,
  PRIMARY KEY (`id_order_return`),
  KEY `order_return_customer` (`id_customer`),
  KEY `id_order` (`id_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Return detail for each product inside an order */
CREATE TABLE `ps_order_return_detail` (
  `id_order_return` int(10) unsigned NOT NULL,
  `id_order_detail` int(10) unsigned NOT NULL,
  `id_customization` int(10) unsigned NOT NULL DEFAULT '0',
  `product_quantity` int(10) unsigned NOT NULL DEFAULT '0',
  `cancelled` tinyint(1) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (
    `id_order_return`, `id_order_detail`,
    `id_customization`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* List of possible return states color */
CREATE TABLE `ps_order_return_state` (
  `id_order_return_state` int(10) unsigned NOT NULL auto_increment,
  `color` varchar(32) DEFAULT NULL,
  `is_cancelling_return` tinyint(1) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id_order_return_state`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Localized return states name */
CREATE TABLE `ps_order_return_state_lang` (
  `id_order_return_state` int(10) unsigned NOT NULL,
  `id_lang` int(10) unsigned NOT NULL,
  `name` varchar(64) NOT NULL,
  PRIMARY KEY (
    `id_order_return_state`, `id_lang`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Order slip info */
CREATE TABLE `ps_order_slip` (
  `id_order_slip` int(10) unsigned NOT NULL auto_increment,
  `conversion_rate` decimal(13, 6) NOT NULL DEFAULT 1,
  `id_customer` int(10) unsigned NOT NULL,
  `id_order` int(10) unsigned NOT NULL,
  `total_products_tax_excl` DECIMAL(20, 6) NULL,
  `total_products_tax_incl` DECIMAL(20, 6) NULL,
  `total_shipping_tax_excl` DECIMAL(20, 6) NULL,
  `total_shipping_tax_incl` DECIMAL(20, 6) NULL,
  `shipping_cost` tinyint(3) unsigned NOT NULL DEFAULT '0',
  `amount` DECIMAL(20, 6) NOT NULL DEFAULT '0.000000',
  `shipping_cost_amount` DECIMAL(20, 6) NOT NULL DEFAULT '0.000000',
  `partial` TINYINT(1) NOT NULL,
  `order_slip_type` TINYINT(1) unsigned NOT NULL DEFAULT '0',
  `date_add` datetime NOT NULL,
  `date_upd` datetime NOT NULL,
  PRIMARY KEY (`id_order_slip`),
  KEY `order_slip_customer` (`id_customer`),
  KEY `id_order` (`id_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Detail of the order slip (every product) */
CREATE TABLE `ps_order_slip_detail` (
  `id_order_slip` int(10) unsigned NOT NULL,
  `id_order_detail` int(10) unsigned NOT NULL,
  `product_quantity` int(10) unsigned NOT NULL DEFAULT '0',
  `unit_price_tax_excl` DECIMAL(20, 6) NULL,
  `unit_price_tax_incl` DECIMAL(20, 6) NULL,
  `total_price_tax_excl` DECIMAL(20, 6) NULL,
  `total_price_tax_incl` DECIMAL(20, 6),
  `amount_tax_excl` DECIMAL(20, 6) DEFAULT NULL,
  `amount_tax_incl` DECIMAL(20, 6) DEFAULT NULL,
  PRIMARY KEY (
    `id_order_slip`, `id_order_detail`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* List of available order states */
CREATE TABLE `ps_order_state` (
  `id_order_state` int(10) UNSIGNED NOT NULL auto_increment,
  `invoice` tinyint(1) UNSIGNED DEFAULT '0',
  `send_email` tinyint(1) UNSIGNED NOT NULL DEFAULT '0',
  `module_name` VARCHAR(255) NULL DEFAULT NULL,
  `color` varchar(32) DEFAULT NULL,
  `unremovable` tinyint(1) UNSIGNED NOT NULL,
  `hidden` tinyint(1) UNSIGNED NOT NULL DEFAULT '0',
  `logable` tinyint(1) NOT NULL DEFAULT '0',
  `delivery` tinyint(1) UNSIGNED NOT NULL DEFAULT '0',
  `shipped` tinyint(1) UNSIGNED NOT NULL DEFAULT '0',
  `paid` tinyint(1) UNSIGNED NOT NULL DEFAULT '0',
  `pdf_invoice` tinyint(1) UNSIGNED NOT NULL default '0',
  `pdf_delivery` tinyint(1) UNSIGNED NOT NULL default '0',
  `deleted` tinyint(1) UNSIGNED NOT NULL DEFAULT '0',
  PRIMARY KEY (`id_order_state`),
  KEY `module_name` (`module_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Localized order state */
CREATE TABLE `ps_order_state_lang` (
  `id_order_state` int(10) unsigned NOT NULL,
  `id_lang` int(10) unsigned NOT NULL,
  `name` varchar(64) NOT NULL,
  `template` varchar(64) NOT NULL,
  PRIMARY KEY (`id_order_state`, `id_lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Define which products / quantities define a pack. A product could be a pack */
CREATE TABLE `ps_pack` (
  `id_product_pack` int(10) unsigned NOT NULL,
  `id_product_item` int(10) unsigned NOT NULL,
  `id_product_attribute_item` int(10) unsigned NOT NULL,
  `quantity` int(10) unsigned NOT NULL DEFAULT 1,
  PRIMARY KEY (
    `id_product_pack`, `id_product_item`,
    `id_product_attribute_item`
  ),
  KEY `product_item` (
    `id_product_item`, `id_product_attribute_item`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* page stats (PS_STATSDATA_CUSTOMER_PAGESVIEWS) */
CREATE TABLE `ps_page` (
  `id_page` int(10) unsigned NOT NULL auto_increment,
  `id_page_type` int(10) unsigned NOT NULL,
  `id_object` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id_page`),
  KEY `id_page_type` (`id_page_type`),
  KEY `id_object` (`id_object`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* List of page type (stats) */
CREATE TABLE `ps_page_type` (
  `id_page_type` int(10) unsigned NOT NULL auto_increment,
  `name` varchar(255) NOT NULL,
  PRIMARY KEY (`id_page_type`),
  KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Page viewed (stats) */
CREATE TABLE `ps_page_viewed` (
  `id_page` int(10) unsigned NOT NULL,
  `id_shop_group` INT UNSIGNED NOT NULL DEFAULT '1',
  `id_shop` INT UNSIGNED NOT NULL DEFAULT '1',
  `id_date_range` int(10) unsigned NOT NULL,
  `counter` int(10) unsigned NOT NULL,
  PRIMARY KEY (
    `id_page`, `id_date_range`, `id_shop`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Payment info (see payment_invoice) */
CREATE TABLE `ps_order_payment` (
  `id_order_payment` INT NOT NULL auto_increment,
  `order_reference` VARCHAR(255),
  `id_currency` INT UNSIGNED NOT NULL,
  `amount` DECIMAL(20, 6) NOT NULL,
  `payment_method` varchar(255) NOT NULL,
  `conversion_rate` decimal(13, 6) NOT NULL DEFAULT 1,
  `transaction_id` VARCHAR(254) NULL,
  `card_number` VARCHAR(254) NULL,
  `card_brand` VARCHAR(254) NULL,
  `card_expiration` CHAR(7) NULL,
  `card_holder` VARCHAR(254) NULL,
  `date_add` DATETIME NOT NULL,
  `id_employee` INT NULL,
  PRIMARY KEY (`id_order_payment`),
  KEY `order_reference`(`order_reference`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* list of products */
CREATE TABLE `ps_product` (
  `id_product` int(10) unsigned NOT NULL auto_increment,
  `id_supplier` int(10) unsigned DEFAULT NULL,
  `id_manufacturer` int(10) unsigned DEFAULT NULL,
  `id_category_default` int(10) unsigned DEFAULT NULL,
  `id_shop_default` int(10) unsigned NOT NULL DEFAULT 1,
  `id_tax_rules_group` INT(11) UNSIGNED NOT NULL,
  `on_sale` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `online_only` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `ean13` varchar(20) DEFAULT NULL,
  `isbn` varchar(32) DEFAULT NULL,
  `upc` varchar(12) DEFAULT NULL,
  `mpn` varchar(40) DEFAULT NULL,
  `ecotax` decimal(17, 6) NOT NULL DEFAULT '0.00',
  `quantity` int(10) NOT NULL DEFAULT '0',
  `minimal_quantity` int(10) unsigned NOT NULL DEFAULT '1',
  `low_stock_threshold` int(10) NULL DEFAULT NULL,
  `low_stock_alert` TINYINT(1) NOT NULL DEFAULT 0,
  `price` decimal(20, 6) NOT NULL DEFAULT '0.000000',
  `wholesale_price` decimal(20, 6) NOT NULL DEFAULT '0.000000',
  `unity` varchar(255) DEFAULT NULL,
  `unit_price` decimal(20, 6) NOT NULL DEFAULT '0.000000',
  `unit_price_ratio` decimal(20, 6) NOT NULL DEFAULT '0.000000',
  `additional_shipping_cost` decimal(20, 6) NOT NULL DEFAULT '0.000000',
  `reference` varchar(64) DEFAULT NULL,
  `supplier_reference` varchar(64) DEFAULT NULL,
  `location` varchar(255) NOT NULL DEFAULT '',
  `width` DECIMAL(20, 6) NOT NULL DEFAULT '0',
  `height` DECIMAL(20, 6) NOT NULL DEFAULT '0',
  `depth` DECIMAL(20, 6) NOT NULL DEFAULT '0',
  `weight` DECIMAL(20, 6) NOT NULL DEFAULT '0',
  `out_of_stock` int(10) unsigned NOT NULL DEFAULT '2',
  `additional_delivery_times` tinyint(1) unsigned NOT NULL DEFAULT '1',
  `quantity_discount` tinyint(1) DEFAULT '0',
  `customizable` tinyint(2) NOT NULL DEFAULT '0',
  `uploadable_files` tinyint(4) NOT NULL DEFAULT '0',
  `text_fields` tinyint(4) NOT NULL DEFAULT '0',
  `active` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `redirect_type` ENUM(
    '404', '410', '301-product', '302-product',
    '301-category', '302-category', '200-displayed',
    '404-displayed', '410-displayed', 'default'
  ) NOT NULL DEFAULT 'default',
  `id_type_redirected` int(10) unsigned NOT NULL DEFAULT '0',
  `available_for_order` tinyint(1) NOT NULL DEFAULT '1',
  `available_date` date DEFAULT NULL,
  `show_condition` tinyint(1) NOT NULL DEFAULT '0',
  `condition` ENUM('new', 'used', 'refurbished', 'open_box', 'damaged', 'new_with_defects') NOT NULL DEFAULT 'new',
  `show_price` tinyint(1) NOT NULL DEFAULT '1',
  `indexed` tinyint(1) NOT NULL DEFAULT '0',
  `visibility` ENUM(
    'both', 'catalog', 'search', 'none'
  ) NOT NULL DEFAULT 'both',
  `cache_is_pack` tinyint(1) NOT NULL DEFAULT '0',
  `cache_has_attachments` tinyint(1) NOT NULL DEFAULT '0',
  `is_virtual` tinyint(1) NOT NULL DEFAULT '0',
  `cache_default_attribute` int(10) unsigned DEFAULT NULL,
  `date_add` datetime NOT NULL,
  `date_upd` datetime NOT NULL,
  `advanced_stock_management` tinyint(1) DEFAULT '0' NOT NULL,
  `pack_stock_type` int(11) unsigned DEFAULT '3' NOT NULL,
  `state` int(11) unsigned NOT NULL DEFAULT '1',
  `product_type` ENUM(
    'standard', 'pack', 'virtual', 'combinations', ''
  ) NOT NULL DEFAULT '',
  PRIMARY KEY (`id_product`),
  INDEX reference_idx(`reference`),
  INDEX supplier_reference_idx(`supplier_reference`),
  KEY `product_supplier` (`id_supplier`),
  KEY `product_manufacturer` (`id_manufacturer`, `id_product`),
  KEY `id_category_default` (`id_category_default`),
  KEY `indexed` (`indexed`),
  KEY `date_add` (`date_add`),
  KEY `state` (`state`, `date_upd`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* shop specific product info */
CREATE TABLE IF NOT EXISTS `ps_product_shop` (
  `id_product` int(10) unsigned NOT NULL,
  `id_shop` int(10) unsigned NOT NULL,
  `id_category_default` int(10) unsigned DEFAULT NULL,
  `id_tax_rules_group` INT(11) UNSIGNED NOT NULL,
  `on_sale` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `online_only` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `ecotax` decimal(17, 6) NOT NULL DEFAULT '0.000000',
  `minimal_quantity` int(10) unsigned NOT NULL DEFAULT '1',
  `low_stock_threshold` int(10) NULL DEFAULT NULL,
  `low_stock_alert` TINYINT(1) NOT NULL DEFAULT 0,
  `price` decimal(20, 6) NOT NULL DEFAULT '0.000000',
  `wholesale_price` decimal(20, 6) NOT NULL DEFAULT '0.000000',
  `unity` varchar(255) DEFAULT NULL,
  `unit_price` decimal(20, 6) NOT NULL DEFAULT '0.000000',
  `unit_price_ratio` decimal(20, 6) NOT NULL DEFAULT '0.000000',
  `additional_shipping_cost` decimal(20, 6) NOT NULL DEFAULT '0.000000',
  `customizable` tinyint(2) NOT NULL DEFAULT '0',
  `uploadable_files` tinyint(4) NOT NULL DEFAULT '0',
  `text_fields` tinyint(4) NOT NULL DEFAULT '0',
  `active` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `redirect_type` ENUM(
    '404', '410', '301-product', '302-product',
    '301-category', '302-category', '200-displayed',
    '404-displayed', '410-displayed', 'default'
  ) NOT NULL DEFAULT 'default',
  `id_type_redirected` int(10) unsigned NOT NULL DEFAULT '0',
  `available_for_order` tinyint(1) NOT NULL DEFAULT '1',
  `available_date` date DEFAULT NULL,
  `show_condition` tinyint(1) NOT NULL DEFAULT '1',
  `condition` enum('new', 'used', 'refurbished', 'open_box', 'damaged', 'new_with_defects') NOT NULL DEFAULT 'new',
  `show_price` tinyint(1) NOT NULL DEFAULT '1',
  `indexed` tinyint(1) NOT NULL DEFAULT '0',
  `visibility` enum(
    'both', 'catalog', 'search', 'none'
  ) NOT NULL DEFAULT 'both',
  `cache_default_attribute` int(10) unsigned DEFAULT NULL,
  `advanced_stock_management` tinyint(1) DEFAULT '0' NOT NULL,
  `date_add` datetime NOT NULL,
  `date_upd` datetime NOT NULL,
  `pack_stock_type` int(11) unsigned DEFAULT '3' NOT NULL,
  PRIMARY KEY (`id_product`, `id_shop`),
  KEY `id_category_default` (`id_category_default`),
  KEY `date_add` (
    `date_add`, `active`, `visibility`
  ),
  KEY `indexed` (
    `indexed`, `active`, `id_product`
  ),
  INDEX `shop_tax` (`id_shop`, `id_tax_rules_group`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* list of product attributes (E.g. : color) */
CREATE TABLE `ps_product_attribute` (
  `id_product_attribute` int(10) unsigned NOT NULL auto_increment,
  `id_product` int(10) unsigned NOT NULL,
  `reference` varchar(64) DEFAULT NULL,
  `supplier_reference` varchar(64) DEFAULT NULL,
  `ean13` varchar(20) DEFAULT NULL,
  `isbn` varchar(32) DEFAULT NULL,
  `upc` varchar(12) DEFAULT NULL,
  `mpn` varchar(40) DEFAULT NULL,
  `wholesale_price` decimal(20, 6) NOT NULL DEFAULT '0.000000',
  `price` decimal(20, 6) NOT NULL DEFAULT '0.000000',
  `ecotax` decimal(17, 6) NOT NULL DEFAULT '0.00',
  `weight` DECIMAL(20, 6) NOT NULL DEFAULT '0',
  `unit_price_impact` DECIMAL(20, 6) NOT NULL DEFAULT '0.00',
  `default_on` tinyint(1) unsigned NULL DEFAULT NULL,
  `minimal_quantity` int(10) unsigned NOT NULL DEFAULT '1',
  `low_stock_threshold` int(10) NULL DEFAULT NULL,
  `low_stock_alert` TINYINT(1) NOT NULL DEFAULT 0,
  `available_date` date DEFAULT NULL,
  PRIMARY KEY (`id_product_attribute`),
  KEY `product_attribute_product` (`id_product`),
  KEY `reference` (`reference`),
  KEY `supplier_reference` (`supplier_reference`),
  UNIQUE KEY `product_default` (`id_product`, `default_on`),
  KEY `id_product_id_product_attribute` (
    `id_product_attribute`, `id_product`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Localized combination information */
CREATE TABLE `ps_product_attribute_lang` (
  `id_product_attribute` int(10) unsigned NOT NULL,
  `id_lang` int(10) unsigned NOT NULL,
  `available_now` varchar(255) DEFAULT NULL,
  `available_later` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`id_product_attribute`, `id_lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* shop specific attribute info */
CREATE TABLE `ps_product_attribute_shop` (
  `id_product` int(10) unsigned NOT NULL,
  `id_product_attribute` int(10) unsigned NOT NULL,
  `id_shop` int(10) unsigned NOT NULL,
  `wholesale_price` decimal(20, 6) NOT NULL DEFAULT '0.000000',
  `price` decimal(20, 6) NOT NULL DEFAULT '0.000000',
  `ecotax` decimal(17, 6) NOT NULL DEFAULT '0.00',
  `weight` DECIMAL(20, 6) NOT NULL DEFAULT '0',
  `unit_price_impact` DECIMAL(20, 6) NOT NULL DEFAULT '0.00',
  `default_on` tinyint(1) unsigned NULL DEFAULT NULL,
  `minimal_quantity` int(10) unsigned NOT NULL DEFAULT '1',
  `low_stock_threshold` int(10) NULL DEFAULT NULL,
  `low_stock_alert` TINYINT(1) NOT NULL DEFAULT 0,
  `available_date` date DEFAULT NULL,
  PRIMARY KEY (
    `id_product_attribute`, `id_shop`
  ),
  UNIQUE KEY `id_product` (
    `id_product`, `id_shop`, `default_on`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* association between attribute and combination */
CREATE TABLE `ps_product_attribute_combination` (
  `id_attribute` int(10) unsigned NOT NULL,
  `id_product_attribute` int(10) unsigned NOT NULL,
  PRIMARY KEY (
    `id_attribute`, `id_product_attribute`
  ),
  KEY `id_product_attribute` (`id_product_attribute`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* image associated with an attribute */
CREATE TABLE `ps_product_attribute_image` (
  `id_product_attribute` int(10) unsigned NOT NULL,
  `id_image` int(10) unsigned NOT NULL,
  PRIMARY KEY (
    `id_product_attribute`, `id_image`
  ),
  KEY `id_image` (`id_image`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Virtual product download info */
CREATE TABLE `ps_product_download` (
  `id_product_download` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `id_product` int(10) unsigned NOT NULL,
  `display_filename` varchar(255) DEFAULT NULL,
  `filename` varchar(255) DEFAULT NULL,
  `date_add` datetime NOT NULL,
  `date_expiration` datetime DEFAULT NULL,
  `nb_days_accessible` int(10) unsigned DEFAULT NULL,
  `nb_downloadable` int(10) unsigned DEFAULT '1',
  `active` tinyint(1) unsigned NOT NULL DEFAULT '1',
  `is_shareable` tinyint(1) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id_product_download`),
  KEY `product_active` (`id_product`, `active`),
  UNIQUE KEY `id_product` (`id_product`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Localized product info */
CREATE TABLE `ps_product_lang` (
  `id_product` int(10) unsigned NOT NULL,
  `id_shop` INT(11) UNSIGNED NOT NULL DEFAULT '1',
  `id_lang` int(10) unsigned NOT NULL,
  `description` MEDIUMTEXT,
  `description_short` MEDIUMTEXT,
  `link_rewrite` varchar(128) NOT NULL,
  `meta_description` varchar(512) DEFAULT NULL,
  `meta_title` varchar(128) DEFAULT NULL,
  `name` varchar(128) NOT NULL,
  `available_now` varchar(255) DEFAULT NULL,
  `available_later` varchar(255) DEFAULT NULL,
  `delivery_in_stock` varchar(255) DEFAULT NULL,
  `delivery_out_stock` varchar(255) DEFAULT NULL,
  PRIMARY KEY (
    `id_product`, `id_shop`, `id_lang`
  ),
  KEY `id_lang` (`id_lang`),
  KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* info about number of products sold */
CREATE TABLE `ps_product_sale` (
  `id_product` int(10) unsigned NOT NULL,
  `quantity` int(10) unsigned NOT NULL DEFAULT '0',
  `sale_nbr` int(10) unsigned NOT NULL DEFAULT '0',
  `date_upd` date DEFAULT NULL,
  PRIMARY KEY (`id_product`),
  KEY `quantity` (`quantity`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* tags associated with a product */
CREATE TABLE `ps_product_tag` (
  `id_product` int(10) unsigned NOT NULL,
  `id_tag` int(10) unsigned NOT NULL,
  `id_lang` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id_product`, `id_tag`),
  KEY `id_tag` (`id_tag`),
  KEY `id_lang` (`id_lang`, `id_tag`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* List of profile (admin, superadmin, etc...) */
CREATE TABLE `ps_profile` (
  `id_profile` int(10) unsigned NOT NULL auto_increment,
  PRIMARY KEY (`id_profile`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Localized profile names */
CREATE TABLE `ps_profile_lang` (
  `id_lang` int(10) unsigned NOT NULL,
  `id_profile` int(10) unsigned NOT NULL,
  `name` varchar(128) NOT NULL,
  PRIMARY KEY (`id_profile`, `id_lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* List of quick access link used in the admin */
CREATE TABLE `ps_quick_access` (
  `id_quick_access` int(10) unsigned NOT NULL auto_increment,
  `new_window` tinyint(1) NOT NULL DEFAULT '0',
  `link` varchar(255) NOT NULL,
  PRIMARY KEY (`id_quick_access`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Localized quick access names */
CREATE TABLE `ps_quick_access_lang` (
  `id_quick_access` int(10) unsigned NOT NULL,
  `id_lang` int(10) unsigned NOT NULL,
  `name` varchar(32) NOT NULL,
  PRIMARY KEY (`id_quick_access`, `id_lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* price ranges used for delivery */
CREATE TABLE `ps_range_price` (
  `id_range_price` int(10) unsigned NOT NULL auto_increment,
  `id_carrier` int(10) unsigned NOT NULL,
  `delimiter1` decimal(20, 6) NOT NULL,
  `delimiter2` decimal(20, 6) NOT NULL,
  PRIMARY KEY (`id_range_price`),
  UNIQUE KEY `id_carrier` (
    `id_carrier`, `delimiter1`, `delimiter2`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Weight ranges used for delivery */
CREATE TABLE `ps_range_weight` (
  `id_range_weight` int(10) unsigned NOT NULL auto_increment,
  `id_carrier` int(10) unsigned NOT NULL,
  `delimiter1` decimal(20, 6) NOT NULL,
  `delimiter2` decimal(20, 6) NOT NULL,
  PRIMARY KEY (`id_range_weight`),
  UNIQUE KEY `id_carrier` (
    `id_carrier`, `delimiter1`, `delimiter2`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* List of custom SQL request saved on the admin (used to generate exports) */
CREATE TABLE IF NOT EXISTS `ps_request_sql` (
  `id_request_sql` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(200) NOT NULL,
  `sql` MEDIUMTEXT NOT NULL,
  PRIMARY KEY (`id_request_sql`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* List of search engine + query string (used by SEO module) */
CREATE TABLE `ps_search_engine` (
  `id_search_engine` int(10) unsigned NOT NULL auto_increment,
  `server` varchar(64) NOT NULL,
  `getvar` varchar(16) NOT NULL,
  PRIMARY KEY (`id_search_engine`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Index constructed by the search engine */
CREATE TABLE `ps_search_index` (
  `id_product` int(11) unsigned NOT NULL,
  `id_word` int(11) unsigned NOT NULL,
  `weight` smallint(4) unsigned NOT NULL DEFAULT 1,
  PRIMARY KEY (`id_word`, `id_product`),
  KEY `id_product` (`id_product`, `weight`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* List of words available for a given shop & lang */
CREATE TABLE `ps_search_word` (
  `id_word` int(10) unsigned NOT NULL auto_increment,
  `id_shop` int(11) unsigned NOT NULL DEFAULT 1,
  `id_lang` int(10) unsigned NOT NULL,
  `word` varchar(30) NOT NULL,
  PRIMARY KEY (`id_word`),
  UNIQUE KEY `id_lang` (`id_lang`, `id_shop`, `word`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* List of price reduction depending on given conditions */
CREATE TABLE `ps_specific_price` (
  `id_specific_price` INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `id_specific_price_rule` INT(11) UNSIGNED NOT NULL,
  `id_cart` INT(11) UNSIGNED NOT NULL,
  `id_product` INT UNSIGNED NOT NULL,
  `id_shop` INT(11) UNSIGNED NOT NULL DEFAULT '1',
  `id_shop_group` INT(11) UNSIGNED NOT NULL,
  `id_currency` INT UNSIGNED NOT NULL,
  `id_country` INT UNSIGNED NOT NULL,
  `id_group` INT UNSIGNED NOT NULL,
  `id_customer` INT UNSIGNED NOT NULL,
  `id_product_attribute` INT UNSIGNED NOT NULL,
  `price` DECIMAL(20, 6) NOT NULL,
  `from_quantity` mediumint(8) UNSIGNED NOT NULL,
  `reduction` DECIMAL(20, 6) NOT NULL,
  `reduction_tax` tinyint(1) NOT NULL DEFAULT 1,
  `reduction_type` ENUM('amount', 'percentage') NOT NULL,
  `from` DATETIME NOT NULL,
  `to` DATETIME NOT NULL,
  PRIMARY KEY (`id_specific_price`),
  KEY (
    `id_product`, `id_shop`, `id_currency`,
    `id_country`, `id_group`, `id_customer`,
    `from_quantity`, `from`, `to`
  ),
  KEY `from_quantity` (`from_quantity`),
  KEY (`id_specific_price_rule`),
  KEY (`id_cart`),
  KEY `id_product_attribute` (`id_product_attribute`),
  KEY `id_shop` (`id_shop`),
  KEY `id_customer` (`id_customer`),
  KEY `from` (`from`),
  KEY `to` (`to`),
  UNIQUE KEY `id_product_2` (
    `id_product`, `id_product_attribute`,
    `id_customer`, `id_cart`, `from`,
    `to`, `id_shop`, `id_shop_group`,
    `id_currency`, `id_country`, `id_group`,
    `from_quantity`, `id_specific_price_rule`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* State localization info */
CREATE TABLE `ps_state` (
  `id_state` int(10) unsigned NOT NULL auto_increment,
  `id_country` int(11) unsigned NOT NULL,
  `id_zone` int(11) unsigned NOT NULL,
  `name` varchar(80) NOT NULL,
  `iso_code` varchar(7) NOT NULL,
  `tax_behavior` smallint(1) NOT NULL DEFAULT '0',
  `active` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id_state`),
  KEY `id_country` (`id_country`),
  KEY `name` (`name`),
  KEY `id_zone` (`id_zone`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* List of suppliers */
CREATE TABLE `ps_supplier` (
  `id_supplier` int(10) unsigned NOT NULL auto_increment,
  `name` varchar(64) NOT NULL,
  `date_add` datetime NOT NULL,
  `date_upd` datetime NOT NULL,
  `active` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id_supplier`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Localized supplier data */
CREATE TABLE `ps_supplier_lang` (
  `id_supplier` int(10) unsigned NOT NULL,
  `id_lang` int(10) unsigned NOT NULL,
  `description` MEDIUMTEXT,
  `meta_title` varchar(255) DEFAULT NULL,
  `meta_description` varchar(512) DEFAULT NULL,
  PRIMARY KEY (`id_supplier`, `id_lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* List of tags */
CREATE TABLE `ps_tag` (
  `id_tag` int(10) unsigned NOT NULL auto_increment,
  `id_lang` int(10) unsigned NOT NULL,
  `name` varchar(32) NOT NULL,
  PRIMARY KEY (`id_tag`),
  KEY `tag_name` (`name`),
  KEY `id_lang` (`id_lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Count info associated with each tag depending on lang, group & shop (cloud tags) */
CREATE TABLE `ps_tag_count` (
  `id_group` int(10) unsigned NOT NULL DEFAULT 0,
  `id_tag` int(10) unsigned NOT NULL DEFAULT 0,
  `id_lang` int(10) unsigned NOT NULL DEFAULT 0,
  `id_shop` int(11) unsigned NOT NULL DEFAULT 0,
  `counter` int(10) unsigned NOT NULL DEFAULT 0,
  PRIMARY KEY (`id_group`, `id_tag`),
  KEY (
    `id_group`, `id_lang`, `id_shop`,
    `counter`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* List of taxes */
CREATE TABLE `ps_tax` (
  `id_tax` int(10) unsigned NOT NULL auto_increment,
  `rate` DECIMAL(10, 3) NOT NULL,
  `active` tinyint(1) unsigned NOT NULL DEFAULT '1',
  `deleted` tinyint(1) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id_tax`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Localized tax names */
CREATE TABLE `ps_tax_lang` (
  `id_tax` int(10) unsigned NOT NULL,
  `id_lang` int(10) unsigned NOT NULL,
  `name` varchar(32) NOT NULL,
  PRIMARY KEY (`id_tax`, `id_lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* List of timezone */
CREATE TABLE `ps_timezone` (
  id_timezone int(10) unsigned NOT NULL auto_increment,
  name VARCHAR(32) NOT NULL,
  PRIMARY KEY (`id_timezone`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* List of web browsers */
CREATE TABLE `ps_web_browser` (
  `id_web_browser` int(10) unsigned NOT NULL auto_increment,
  `name` varchar(64) DEFAULT NULL,
  PRIMARY KEY (`id_web_browser`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* List of geographic zones */
CREATE TABLE `ps_zone` (
  `id_zone` int(10) unsigned NOT NULL auto_increment,
  `name` varchar(64) NOT NULL,
  `active` tinyint(1) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id_zone`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Carrier available for a specific group */
CREATE TABLE `ps_carrier_group` (
  `id_carrier` int(10) unsigned NOT NULL,
  `id_group` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id_carrier`, `id_group`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* List of stores */
CREATE TABLE `ps_store` (
  `id_store` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `id_country` int(10) unsigned NOT NULL,
  `id_state` int(10) unsigned DEFAULT NULL,
  `city` varchar(64) NOT NULL,
  `postcode` varchar(12) NOT NULL,
  `latitude` decimal(13, 8) DEFAULT NULL,
  `longitude` decimal(13, 8) DEFAULT NULL,
  `phone` varchar(16) DEFAULT NULL,
  `fax` varchar(16) DEFAULT NULL,
  `email` varchar(255) DEFAULT NULL,
  `active` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `date_add` datetime NOT NULL,
  `date_upd` datetime NOT NULL,
  PRIMARY KEY (`id_store`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `ps_store_lang` (
  `id_store` int(11) unsigned NOT NULL,
  `id_lang` int(11) unsigned NOT NULL,
  `name` varchar(255) NOT NULL,
  `address1` varchar(255) NOT NULL,
  `address2` varchar(255) DEFAULT NULL,
  `hours` MEDIUMTEXT,
  `note` MEDIUMTEXT,
  PRIMARY KEY (`id_store`, `id_lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Webservice account infos */
CREATE TABLE `ps_webservice_account` (
  `id_webservice_account` int(11) NOT NULL AUTO_INCREMENT,
  `key` varchar(32) NOT NULL,
  `description` MEDIUMTEXT NULL,
  `class_name` VARCHAR(50) NOT NULL DEFAULT 'WebserviceRequest',
  `is_module` TINYINT(2) NOT NULL DEFAULT '0',
  `module_name` VARCHAR(50) NULL DEFAULT NULL,
  `active` tinyint(2) NOT NULL,
  PRIMARY KEY (`id_webservice_account`),
  KEY `key` (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Permissions associated with a webservice account */
CREATE TABLE `ps_webservice_permission` (
  `id_webservice_permission` int(11) NOT NULL AUTO_INCREMENT,
  `resource` varchar(50) NOT NULL,
  `method` enum(
    'GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD'
  ) NOT NULL,
  `id_webservice_account` int(11) NOT NULL,
  PRIMARY KEY (`id_webservice_permission`),
  UNIQUE KEY `resource_2` (
    `resource`, `method`, `id_webservice_account`
  ),
  KEY `resource` (`resource`),
  KEY `method` (`method`),
  KEY `id_webservice_account` (`id_webservice_account`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_required_field` (
  `id_required_field` int(11) NOT NULL AUTO_INCREMENT,
  `object_name` varchar(32) NOT NULL,
  `field_name` varchar(32) NOT NULL,
  PRIMARY KEY (`id_required_field`),
  KEY `object_name` (`object_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_memcached_servers` (
  `id_memcached_server` INT(11) UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `ip` VARCHAR(254) NOT NULL,
  `port` INT(11) UNSIGNED NOT NULL,
  `weight` INT(11) UNSIGNED NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_product_country_tax` (
  `id_product` int(11) NOT NULL,
  `id_country` int(11) NOT NULL,
  `id_tax` int(11) NOT NULL,
  PRIMARY KEY (`id_product`, `id_country`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_tax_rule` (
  `id_tax_rule` int(11) NOT NULL AUTO_INCREMENT,
  `id_tax_rules_group` int(11) NOT NULL,
  `id_country` int(11) NOT NULL,
  `id_state` int(11) NOT NULL,
  `zipcode_from` VARCHAR(12) NOT NULL,
  `zipcode_to` VARCHAR(12) NOT NULL,
  `id_tax` int(11) NOT NULL,
  `behavior` int(11) NOT NULL,
  `description` VARCHAR(100) NOT NULL,
  PRIMARY KEY (`id_tax_rule`),
  KEY `id_tax_rules_group` (`id_tax_rules_group`),
  KEY `id_tax` (`id_tax`),
  KEY `category_getproducts` (
    `id_tax_rules_group`, `id_country`,
    `id_state`, `zipcode_from`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_tax_rules_group` (
  `id_tax_rules_group` INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `name` VARCHAR(64) NOT NULL,
  `active` INT NOT NULL,
  `deleted` TINYINT(1) UNSIGNED NOT NULL,
  `date_add` DATETIME NOT NULL,
  `date_upd` DATETIME NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_specific_price_priority` (
  `id_specific_price_priority` INT NOT NULL AUTO_INCREMENT,
  `id_product` INT NOT NULL,
  `priority` VARCHAR(80) NOT NULL,
  PRIMARY KEY (
    `id_specific_price_priority`, `id_product`
  ),
  UNIQUE KEY `id_product` (`id_product`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_log` (
  `id_log` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `severity` tinyint(1) NOT NULL,
  `error_code` int(11) DEFAULT NULL,
  `message` MEDIUMTEXT NOT NULL,
  `object_type` varchar(32) DEFAULT NULL,
  `object_id` int(10) unsigned DEFAULT NULL,
  `id_shop` int(10) unsigned DEFAULT NULL,
  `id_shop_group` int(10) unsigned DEFAULT NULL,
  `id_lang` int(10) unsigned DEFAULT NULL,
  `in_all_shops` tinyint(1) unsigned NOT NULL DEFAULT '0',
  `id_employee` int(10) unsigned DEFAULT NULL,
  `date_add` datetime NOT NULL,
  `date_upd` datetime NOT NULL,
  PRIMARY KEY (`id_log`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_import_match` (
  `id_import_match` int(10) NOT NULL AUTO_INCREMENT,
  `name` varchar(32) NOT NULL,
  `match` MEDIUMTEXT NOT NULL,
  `skip` int(2) NOT NULL,
  PRIMARY KEY (`id_import_match`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_country_shop` (
  `id_country` INT(11) UNSIGNED NOT NULL,
  `id_shop` INT(11) UNSIGNED NOT NULL,
  PRIMARY KEY (`id_country`, `id_shop`),
  KEY `id_shop` (`id_shop`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_carrier_shop` (
  `id_carrier` INT(11) UNSIGNED NOT NULL,
  `id_shop` INT(11) UNSIGNED NOT NULL,
  PRIMARY KEY (`id_carrier`, `id_shop`),
  KEY `id_shop` (`id_shop`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_address_format` (
  `id_country` int(10) unsigned NOT NULL,
  `format` varchar(255) NOT NULL DEFAULT '',
  PRIMARY KEY (`id_country`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_cms_shop` (
  `id_cms` INT(11) UNSIGNED NOT NULL,
  `id_shop` INT(11) UNSIGNED NOT NULL,
  PRIMARY KEY (`id_cms`, `id_shop`),
  KEY `id_shop` (`id_shop`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_currency_shop` (
  `id_currency` INT(11) UNSIGNED NOT NULL,
  `id_shop` INT(11) UNSIGNED NOT NULL,
  `conversion_rate` decimal(13, 6) NOT NULL,
  PRIMARY KEY (`id_currency`, `id_shop`),
  KEY `id_shop` (`id_shop`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_contact_shop` (
  `id_contact` INT(11) UNSIGNED NOT NULL,
  `id_shop` INT(11) UNSIGNED NOT NULL,
  PRIMARY KEY (`id_contact`, `id_shop`),
  KEY `id_shop` (`id_shop`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_image_shop` (
  `id_product` int(10) unsigned NOT NULL,
  `id_image` INT(11) UNSIGNED NOT NULL,
  `id_shop` INT(11) UNSIGNED NOT NULL,
  `cover` tinyint(1) UNSIGNED NULL DEFAULT NULL,
  PRIMARY KEY (`id_image`, `id_shop`),
  UNIQUE KEY `id_product` (`id_product`, `id_shop`, `cover`),
  KEY `id_shop` (`id_shop`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_feature_shop` (
  `id_feature` INT(11) UNSIGNED NOT NULL,
  `id_shop` INT(11) UNSIGNED NOT NULL,
  PRIMARY KEY (`id_feature`, `id_shop`),
  KEY `id_shop` (`id_shop`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_group_shop` (
  `id_group` INT(11) UNSIGNED NOT NULL,
  `id_shop` INT(11) UNSIGNED NOT NULL,
  PRIMARY KEY (`id_group`, `id_shop`),
  KEY `id_shop` (`id_shop`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_tax_rules_group_shop` (
  `id_tax_rules_group` INT(11) UNSIGNED NOT NULL,
  `id_shop` INT(11) UNSIGNED NOT NULL,
  PRIMARY KEY (`id_tax_rules_group`, `id_shop`),
  KEY `id_shop` (`id_shop`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_zone_shop` (
  `id_zone` INT(11) UNSIGNED NOT NULL,
  `id_shop` INT(11) UNSIGNED NOT NULL,
  PRIMARY KEY (`id_zone`, `id_shop`),
  KEY `id_shop` (`id_shop`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_manufacturer_shop` (
  `id_manufacturer` INT(11) UNSIGNED NOT NULL,
  `id_shop` INT(11) UNSIGNED NOT NULL,
  PRIMARY KEY (`id_manufacturer`, `id_shop`),
  KEY `id_shop` (`id_shop`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_supplier_shop` (
  `id_supplier` INT(11) UNSIGNED NOT NULL,
  `id_shop` INT(11) UNSIGNED NOT NULL,
  PRIMARY KEY (`id_supplier`, `id_shop`),
  KEY `id_shop` (`id_shop`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_store_shop` (
  `id_store` INT(11) UNSIGNED NOT NULL,
  `id_shop` INT(11) UNSIGNED NOT NULL,
  PRIMARY KEY (`id_store`, `id_shop`),
  KEY `id_shop` (`id_shop`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_module_shop` (
  `id_module` INT(11) UNSIGNED NOT NULL,
  `id_shop` INT(11) UNSIGNED NOT NULL,
  `enable_device` TINYINT(1) NOT NULL DEFAULT '7',
  PRIMARY KEY (`id_module`, `id_shop`),
  KEY `id_shop` (`id_shop`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_webservice_account_shop` (
  `id_webservice_account` INT(11) UNSIGNED NOT NULL,
  `id_shop` INT(11) UNSIGNED NOT NULL,
  PRIMARY KEY (
    `id_webservice_account`, `id_shop`
  ),
  KEY `id_shop` (`id_shop`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_stock_mvt_reason` (
  `id_stock_mvt_reason` INT(11) UNSIGNED NOT NULL AUTO_INCREMENT,
  `sign` tinyint(1) NOT NULL DEFAULT 1,
  `date_add` datetime NOT NULL,
  `date_upd` datetime NOT NULL,
  `deleted` tinyint(1) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id_stock_mvt_reason`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_stock_mvt_reason_lang` (
  `id_stock_mvt_reason` INT(11) UNSIGNED NOT NULL,
  `id_lang` INT(11) UNSIGNED NOT NULL,
  `name` VARCHAR(255) CHARACTER SET utf8 NOT NULL,
  PRIMARY KEY (
    `id_stock_mvt_reason`, `id_lang`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_stock` (
  `id_stock` INT(11) UNSIGNED NOT NULL AUTO_INCREMENT,
  `id_warehouse` INT(11) UNSIGNED NOT NULL,
  `id_product` INT(11) UNSIGNED NOT NULL,
  `id_product_attribute` INT(11) UNSIGNED NOT NULL,
  `reference` VARCHAR(64) NOT NULL,
  `ean13` VARCHAR(20) DEFAULT NULL,
  `isbn` VARCHAR(32) DEFAULT NULL,
  `upc` VARCHAR(12) DEFAULT NULL,
  `mpn` VARCHAR(40) DEFAULT NULL,
  `physical_quantity` INT(11) UNSIGNED NOT NULL,
  `usable_quantity` INT(11) UNSIGNED NOT NULL,
  `price_te` DECIMAL(20, 6) DEFAULT '0.000000',
  PRIMARY KEY (`id_stock`),
  KEY `id_warehouse` (`id_warehouse`),
  KEY `id_product` (`id_product`),
  KEY `id_product_attribute` (`id_product_attribute`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_warehouse` (
  `id_warehouse` INT(11) UNSIGNED NOT NULL AUTO_INCREMENT,
  `id_currency` INT(11) UNSIGNED NOT NULL,
  `id_address` INT(11) UNSIGNED NOT NULL,
  `id_employee` INT(11) UNSIGNED NOT NULL,
  `reference` VARCHAR(64) DEFAULT NULL,
  `name` VARCHAR(45) NOT NULL,
  `management_type` ENUM('WA', 'FIFO', 'LIFO') NOT NULL DEFAULT 'WA',
  `deleted` tinyint(1) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id_warehouse`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_warehouse_product_location` (
  `id_warehouse_product_location` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `id_product` int(11) unsigned NOT NULL,
  `id_product_attribute` int(11) unsigned NOT NULL,
  `id_warehouse` int(11) unsigned NOT NULL,
  `location` varchar(64) DEFAULT NULL,
  PRIMARY KEY (
    `id_warehouse_product_location`
  ),
  UNIQUE KEY `id_product` (
    `id_product`, `id_product_attribute`,
    `id_warehouse`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_warehouse_shop` (
  `id_shop` INT(11) UNSIGNED NOT NULL,
  `id_warehouse` INT(11) UNSIGNED NOT NULL,
  PRIMARY KEY (`id_warehouse`, `id_shop`),
  KEY `id_warehouse` (`id_warehouse`),
  KEY `id_shop` (`id_shop`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_warehouse_carrier` (
  `id_carrier` INT(11) UNSIGNED NOT NULL,
  `id_warehouse` INT(11) UNSIGNED NOT NULL,
  PRIMARY KEY (`id_warehouse`, `id_carrier`),
  KEY `id_warehouse` (`id_warehouse`),
  KEY `id_carrier` (`id_carrier`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_stock_available` (
  `id_stock_available` INT(11) UNSIGNED NOT NULL AUTO_INCREMENT,
  `id_product` INT(11) UNSIGNED NOT NULL,
  `id_product_attribute` INT(11) UNSIGNED NOT NULL,
  `id_shop` INT(11) UNSIGNED NOT NULL,
  `id_shop_group` INT(11) UNSIGNED NOT NULL,
  `quantity` INT(10) NOT NULL DEFAULT '0',
  `physical_quantity` INT(11) NOT NULL DEFAULT '0',
  `reserved_quantity` INT(11) NOT NULL DEFAULT '0',
  `depends_on_stock` TINYINT(1) UNSIGNED NOT NULL DEFAULT '0',
  `out_of_stock` TINYINT(1) UNSIGNED NOT NULL DEFAULT '0',
  `location` VARCHAR(255) NOT NULL DEFAULT '',
  PRIMARY KEY (`id_stock_available`),
  KEY `id_shop` (`id_shop`),
  KEY `id_shop_group` (`id_shop_group`),
  KEY `id_product` (`id_product`),
  KEY `id_product_attribute` (`id_product_attribute`),
  UNIQUE `product_sqlstock` (
    `id_product`, `id_product_attribute`,
    `id_shop`, `id_shop_group`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_supply_order` (
  `id_supply_order` INT(11) UNSIGNED NOT NULL AUTO_INCREMENT,
  `id_supplier` INT(11) UNSIGNED NOT NULL,
  `supplier_name` VARCHAR(64) NOT NULL,
  `id_lang` INT(11) UNSIGNED NOT NULL,
  `id_warehouse` INT(11) UNSIGNED NOT NULL,
  `id_supply_order_state` INT(11) UNSIGNED NOT NULL,
  `id_currency` INT(11) UNSIGNED NOT NULL,
  `id_ref_currency` INT(11) UNSIGNED NOT NULL,
  `reference` VARCHAR(64) NOT NULL,
  `date_add` DATETIME NOT NULL,
  `date_upd` DATETIME NOT NULL,
  `date_delivery_expected` DATETIME DEFAULT NULL,
  `total_te` DECIMAL(20, 6) DEFAULT '0.000000',
  `total_with_discount_te` DECIMAL(20, 6) DEFAULT '0.000000',
  `total_tax` DECIMAL(20, 6) DEFAULT '0.000000',
  `total_ti` DECIMAL(20, 6) DEFAULT '0.000000',
  `discount_rate` DECIMAL(20, 6) DEFAULT '0.000000',
  `discount_value_te` DECIMAL(20, 6) DEFAULT '0.000000',
  `is_template` tinyint(1) DEFAULT '0',
  PRIMARY KEY (`id_supply_order`),
  KEY `id_supplier` (`id_supplier`),
  KEY `id_warehouse` (`id_warehouse`),
  KEY `reference` (`reference`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_supply_order_detail` (
  `id_supply_order_detail` INT(11) UNSIGNED NOT NULL AUTO_INCREMENT,
  `id_supply_order` INT(11) UNSIGNED NOT NULL,
  `id_currency` INT(11) UNSIGNED NOT NULL,
  `id_product` INT(11) UNSIGNED NOT NULL,
  `id_product_attribute` INT(11) UNSIGNED NOT NULL,
  `reference` VARCHAR(64) NOT NULL,
  `supplier_reference` VARCHAR(64) NOT NULL,
  `name` varchar(128) NOT NULL,
  `ean13` VARCHAR(20) DEFAULT NULL,
  `isbn` VARCHAR(32) DEFAULT NULL,
  `upc` VARCHAR(12) DEFAULT NULL,
  `mpn` VARCHAR(40) DEFAULT NULL,
  `exchange_rate` DECIMAL(20, 6) DEFAULT '0.000000',
  `unit_price_te` DECIMAL(20, 6) DEFAULT '0.000000',
  `quantity_expected` INT(11) UNSIGNED NOT NULL,
  `quantity_received` INT(11) UNSIGNED NOT NULL,
  `price_te` DECIMAL(20, 6) DEFAULT '0.000000',
  `discount_rate` DECIMAL(20, 6) DEFAULT '0.000000',
  `discount_value_te` DECIMAL(20, 6) DEFAULT '0.000000',
  `price_with_discount_te` DECIMAL(20, 6) DEFAULT '0.000000',
  `tax_rate` DECIMAL(20, 6) DEFAULT '0.000000',
  `tax_value` DECIMAL(20, 6) DEFAULT '0.000000',
  `price_ti` DECIMAL(20, 6) DEFAULT '0.000000',
  `tax_value_with_order_discount` DECIMAL(20, 6) DEFAULT '0.000000',
  `price_with_order_discount_te` DECIMAL(20, 6) DEFAULT '0.000000',
  PRIMARY KEY (`id_supply_order_detail`),
  KEY `id_supply_order` (`id_supply_order`, `id_product`),
  KEY `id_product_attribute` (`id_product_attribute`),
  KEY `id_product_product_attribute` (
    `id_product`, `id_product_attribute`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_supply_order_history` (
  `id_supply_order_history` INT(11) UNSIGNED NOT NULL AUTO_INCREMENT,
  `id_supply_order` INT(11) UNSIGNED NOT NULL,
  `id_employee` INT(11) UNSIGNED NOT NULL,
  `employee_lastname` varchar(255) DEFAULT '',
  `employee_firstname` varchar(255) DEFAULT '',
  `id_state` INT(11) UNSIGNED NOT NULL,
  `date_add` DATETIME NOT NULL,
  PRIMARY KEY (`id_supply_order_history`),
  KEY `id_supply_order` (`id_supply_order`),
  KEY `id_employee` (`id_employee`),
  KEY `id_state` (`id_state`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_supply_order_state` (
  `id_supply_order_state` INT(11) UNSIGNED NOT NULL AUTO_INCREMENT,
  `delivery_note` tinyint(1) NOT NULL DEFAULT '0',
  `editable` tinyint(1) NOT NULL DEFAULT '0',
  `receipt_state` tinyint(1) NOT NULL DEFAULT '0',
  `pending_receipt` tinyint(1) NOT NULL DEFAULT '0',
  `enclosed` tinyint(1) NOT NULL DEFAULT '0',
  `color` VARCHAR(32) DEFAULT NULL,
  PRIMARY KEY (`id_supply_order_state`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_supply_order_state_lang` (
  `id_supply_order_state` INT(11) UNSIGNED NOT NULL,
  `id_lang` INT(11) UNSIGNED NOT NULL,
  `name` VARCHAR(128) DEFAULT NULL,
  PRIMARY KEY (
    `id_supply_order_state`, `id_lang`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_supply_order_receipt_history` (
  `id_supply_order_receipt_history` INT(11) UNSIGNED NOT NULL AUTO_INCREMENT,
  `id_supply_order_detail` INT(11) UNSIGNED NOT NULL,
  `id_employee` INT(11) UNSIGNED NOT NULL,
  `employee_lastname` varchar(255) DEFAULT '',
  `employee_firstname` varchar(255) DEFAULT '',
  `id_supply_order_state` INT(11) UNSIGNED NOT NULL,
  `quantity` INT(11) UNSIGNED NOT NULL,
  `date_add` DATETIME NOT NULL,
  PRIMARY KEY (
    `id_supply_order_receipt_history`
  ),
  KEY `id_supply_order_detail` (`id_supply_order_detail`),
  KEY `id_supply_order_state` (`id_supply_order_state`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_product_supplier` (
  `id_product_supplier` int(11) UNSIGNED NOT NULL AUTO_INCREMENT,
  `id_product` int(11) UNSIGNED NOT NULL,
  `id_product_attribute` int(11) UNSIGNED NOT NULL DEFAULT '0',
  `id_supplier` int(11) UNSIGNED NOT NULL,
  `product_supplier_reference` varchar(64) DEFAULT NULL,
  `product_supplier_price_te` decimal(20, 6) NOT NULL DEFAULT '0.000000',
  `id_currency` int(11) unsigned NOT NULL,
  PRIMARY KEY (`id_product_supplier`),
  UNIQUE KEY `id_product` (
    `id_product`, `id_product_attribute`,
    `id_supplier`
  ),
  KEY `id_supplier` (`id_supplier`, `id_product`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_order_carrier` (
  `id_order_carrier` int(11) NOT NULL AUTO_INCREMENT,
  `id_order` int(11) unsigned NOT NULL,
  `id_carrier` int(11) unsigned NOT NULL,
  `id_order_invoice` int(11) unsigned DEFAULT NULL,
  `weight` DECIMAL(20, 6) DEFAULT NULL,
  `shipping_cost_tax_excl` decimal(20, 6) DEFAULT NULL,
  `shipping_cost_tax_incl` decimal(20, 6) DEFAULT NULL,
  `tracking_number` varchar(64) DEFAULT NULL,
  `date_add` datetime NOT NULL,
  PRIMARY KEY (`id_order_carrier`),
  KEY `id_order` (`id_order`),
  KEY `id_carrier` (`id_carrier`),
  KEY `id_order_invoice` (`id_order_invoice`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `ps_specific_price_rule` (
  `id_specific_price_rule` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(255) NOT NULL,
  `id_shop` int(11) unsigned NOT NULL DEFAULT '1',
  `id_currency` int(10) unsigned NOT NULL,
  `id_country` int(10) unsigned NOT NULL,
  `id_group` int(10) unsigned NOT NULL,
  `from_quantity` mediumint(8) unsigned NOT NULL,
  `price` DECIMAL(20, 6),
  `reduction` decimal(20, 6) NOT NULL,
  `reduction_tax` tinyint(1) NOT NULL DEFAULT 1,
  `reduction_type` enum('amount', 'percentage') NOT NULL,
  `from` datetime NOT NULL,
  `to` datetime NOT NULL,
  PRIMARY KEY (`id_specific_price_rule`),
  KEY `id_product` (
    `id_shop`, `id_currency`, `id_country`,
    `id_group`, `from_quantity`, `from`,
    `to`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_specific_price_rule_condition_group` (
  `id_specific_price_rule_condition_group` INT(11) UNSIGNED NOT NULL AUTO_INCREMENT,
  `id_specific_price_rule` INT(11) UNSIGNED NOT NULL,
  PRIMARY KEY (
    `id_specific_price_rule_condition_group`,
    `id_specific_price_rule`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_specific_price_rule_condition` (
  `id_specific_price_rule_condition` INT(11) UNSIGNED NOT NULL AUTO_INCREMENT,
  `id_specific_price_rule_condition_group` INT(11) UNSIGNED NOT NULL,
  `type` VARCHAR(255) NOT NULL,
  `value` VARCHAR(255) NOT NULL,
  PRIMARY KEY (
    `id_specific_price_rule_condition`
  ),
  INDEX (
    `id_specific_price_rule_condition_group`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `ps_risk` (
  `id_risk` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `percent` tinyint(3) NOT NULL,
  `color` varchar(32) NULL,
  PRIMARY KEY (`id_risk`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `ps_risk_lang` (
  `id_risk` int(10) unsigned NOT NULL,
  `id_lang` int(10) unsigned NOT NULL,
  `name` varchar(20) NOT NULL,
  PRIMARY KEY (`id_risk`, `id_lang`),
  KEY `id_risk` (`id_risk`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_category_shop` (
  `id_category` int(11) NOT NULL,
  `id_shop` int(11) NOT NULL,
  `position` int(10) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id_category`, `id_shop`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_module_preference` (
  `id_module_preference` int(11) NOT NULL auto_increment,
  `id_employee` int(11) NOT NULL,
  `module` varchar(191) NOT NULL,
  `interest` tinyint(1) DEFAULT NULL,
  `favorite` tinyint(1) DEFAULT NULL,
  PRIMARY KEY (`id_module_preference`),
  UNIQUE KEY `employee_module` (`id_employee`, `module`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_tab_module_preference` (
  `id_tab_module_preference` int(11) NOT NULL auto_increment,
  `id_employee` int(11) NOT NULL,
  `id_tab` int(11) NOT NULL,
  `module` varchar(191) NOT NULL,
  PRIMARY KEY (`id_tab_module_preference`),
  UNIQUE KEY `employee_module` (
    `id_employee`, `id_tab`, `module`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_carrier_tax_rules_group_shop` (
  `id_carrier` int(11) unsigned NOT NULL,
  `id_tax_rules_group` int(11) unsigned NOT NULL,
  `id_shop` int(11) unsigned NOT NULL,
  PRIMARY KEY (
    `id_carrier`, `id_tax_rules_group`,
    `id_shop`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_order_invoice_payment` (
  `id_order_invoice` int(11) unsigned NOT NULL,
  `id_order_payment` int(11) unsigned NOT NULL,
  `id_order` int(11) unsigned NOT NULL,
  PRIMARY KEY (
    `id_order_invoice`, `id_order_payment`
  ),
  KEY `order_payment` (`id_order_payment`),
  KEY `id_order` (`id_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_smarty_cache` (
  `id_smarty_cache` char(40) NOT NULL,
  `name` char(40) NOT NULL,
  `cache_id` varchar(254) DEFAULT NULL,
  `modified` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `content` longtext NOT NULL,
  PRIMARY KEY (`id_smarty_cache`),
  KEY `name` (`name`),
  KEY `cache_id` (`cache_id`),
  KEY `modified` (`modified`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `ps_mail` (
  `id_mail` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `recipient` varchar(255) NOT NULL,
  `template` varchar(62) NOT NULL,
  `subject` varchar(255) NOT NULL,
  `id_lang` int(11) unsigned NOT NULL,
  `date_add` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id_mail`),
  KEY `recipient` (
    `recipient`(10)
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_smarty_lazy_cache` (
  `template_hash` varchar(32) NOT NULL DEFAULT '',
  `cache_id` varchar(191) NOT NULL DEFAULT '',
  `compile_id` varchar(32) NOT NULL DEFAULT '',
  `filepath` varchar(255) NOT NULL DEFAULT '',
  `last_update` datetime NOT NULL DEFAULT '0000-00-00 00:00:00',
  PRIMARY KEY (
    `template_hash`, `cache_id`, `compile_id`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_smarty_last_flush` (
  `type` ENUM('compile', 'template'),
  `last_flush` datetime NOT NULL DEFAULT '0000-00-00 00:00:00',
  PRIMARY KEY (`type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `ps_cms_role` (
  `id_cms_role` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(50) NOT NULL,
  `id_cms` int(11) unsigned NOT NULL,
  PRIMARY KEY (`id_cms_role`, `id_cms`),
  UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `ps_cms_role_lang` (
  `id_cms_role` int(11) unsigned NOT NULL,
  `id_lang` int(11) unsigned NOT NULL,
  `id_shop` int(11) unsigned NOT NULL,
  `name` varchar(128) DEFAULT NULL,
  PRIMARY KEY (
    `id_cms_role`, `id_lang`, id_shop
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_employee_session` (
  `id_employee_session` int(11) unsigned NOT NULL auto_increment,
  `id_employee` int(10) unsigned DEFAULT NULL,
  `token` varchar(40) DEFAULT NULL,
  `date_add` datetime NOT NULL,
  `date_upd` datetime NOT NULL,
  PRIMARY KEY `id_employee_session` (`id_employee_session`),
  KEY `IDX_B10E26A1D449934` (`id_employee`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_customer_session` (
  `id_customer_session` int(11) unsigned NOT NULL auto_increment,
  `id_customer` int(10) unsigned DEFAULT NULL,
  `token` varchar(40) DEFAULT NULL,
  `date_add` datetime NOT NULL,
  `date_upd` datetime NOT NULL,
  PRIMARY KEY `id_customer_session` (`id_customer_session`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Generated by Doctrine command */
CREATE TABLE `ps_translation` (
  `id_translation` INT AUTO_INCREMENT NOT NULL,
  `id_lang`        INT         NOT NULL,
  `key`          TEXT        NOT NULL,
  `translation`    TEXT        NOT NULL,
  `domain`         VARCHAR(80) NOT NULL,
  `theme`          VARCHAR(32) DEFAULT NULL,
  KEY          `IDX_ADEBEB36BA299860` (`id_lang`),
  KEY          `key` (`domain`),
  PRIMARY KEY (`id_translation`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_lang` (
  `id_lang`          INT AUTO_INCREMENT NOT NULL,
  `name`             VARCHAR(32) NOT NULL,
  `active`           TINYINT(1) NOT NULL,
  `iso_code`         VARCHAR(2)  NOT NULL,
  `language_code`    VARCHAR(5)  NOT NULL,
  `locale`           VARCHAR(5)  NOT NULL,
  `date_format_lite` VARCHAR(32) NOT NULL,
  `date_format_full` VARCHAR(32) NOT NULL,
  `is_rtl`           TINYINT(1) NOT NULL,
  PRIMARY KEY (`id_lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE `ps_lang_shop` (
  `id_lang` INT NOT NULL,
  `id_shop` INT NOT NULL,
  KEY   `IDX_2F43BFC7BA299860` (`id_lang`),
  KEY   `IDX_2F43BFC7274A50A0` (`id_shop`),
  PRIMARY KEY (`id_lang`, `id_shop`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_mutation` (
  `id_mutation`        INT UNSIGNED AUTO_INCREMENT NOT NULL,
  `mutation_table`     VARCHAR(255) NOT NULL,
  `mutation_row_id`    BIGINT       NOT NULL,
  `mutation_action`    ENUM('create', 'update', 'delete'),
  `mutator_type`       ENUM('employee', 'api_client', 'module'),
  `mutator_identifier` VARCHAR(255) NOT NULL,
  `mutation_details`   VARCHAR(255) DEFAULT NULL,
  `date_add`           DATETIME     NOT NULL,
  PRIMARY KEY (`id_mutation`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_image_type` (
  `id_image_type` INT UNSIGNED AUTO_INCREMENT NOT NULL,
  `name`          VARCHAR(64) NOT NULL,
  `width`         INT UNSIGNED NOT NULL,
  `height`        INT UNSIGNED NOT NULL,
  `products`      TINYINT(1) DEFAULT 1 NOT NULL,
  `categories`    TINYINT(1) DEFAULT 1 NOT NULL,
  `manufacturers` TINYINT(1) DEFAULT 1 NOT NULL,
  `suppliers`     TINYINT(1) DEFAULT 1 NOT NULL,
  `stores`        TINYINT(1) DEFAULT 1 NOT NULL,
  UNIQUE KEY `UNIQ_907C95215E237E06` (`name`),
  PRIMARY KEY (`id_image_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_attribute_group` (
  `id_attribute_group` INT AUTO_INCREMENT NOT NULL,
  `is_color_group`     TINYINT(1) NOT NULL,
  `group_type`         VARCHAR(255) NOT NULL,
  `position`           INT          NOT NULL,
  PRIMARY KEY (`id_attribute_group`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE `ps_attribute_group_lang` (
  `id_attribute_group` INT          NOT NULL,
  `id_lang`            INT          NOT NULL,
  `name`               VARCHAR(128) NOT NULL,
  `public_name`        VARCHAR(64)  NOT NULL,
  KEY              `IDX_4653726C67A664FB` (`id_attribute_group`),
  KEY              `IDX_4653726CBA299860` (`id_lang`),
  PRIMARY KEY (`id_attribute_group`, `id_lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE `ps_attribute_group_shop` (
  `id_attribute_group` INT NOT NULL,
  `id_shop`            INT NOT NULL,
  KEY              `IDX_DB30BAAC67A664FB` (`id_attribute_group`),
  KEY              `IDX_DB30BAAC274A50A0` (`id_shop`),
  PRIMARY KEY (`id_attribute_group`, `id_shop`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_attribute` (
  `id_attribute`       INT AUTO_INCREMENT NOT NULL,
  `id_attribute_group` INT         NOT NULL,
  `color`              VARCHAR(32) NOT NULL,
  `position`           INT         NOT NULL,
  KEY              `attribute_group` (`id_attribute_group`),
  PRIMARY KEY (`id_attribute`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE `ps_attribute_lang` (
  `id_attribute` INT          NOT NULL,
  `id_lang`      INT          NOT NULL,
  `name`         VARCHAR(128) NOT NULL,
  KEY        `IDX_3ABE46A77A4F53DC` (`id_attribute`),
  KEY        `IDX_3ABE46A7BA299860` (`id_lang`),
  PRIMARY KEY (`id_attribute`, `id_lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE `ps_attribute_shop` (
  `id_attribute` INT NOT NULL,
  `id_shop`      INT NOT NULL,
  KEY        `IDX_A7DD8E677A4F53DC` (`id_attribute`),
  KEY        `IDX_A7DD8E67274A50A0` (`id_shop`),
  PRIMARY KEY (`id_attribute`, `id_shop`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_tab` (
  `id_tab`         INT AUTO_INCREMENT NOT NULL,
  `id_parent`      INT         NOT NULL,
  `position`       INT         NOT NULL,
  `module`         VARCHAR(64)  DEFAULT NULL,
  `class_name`     VARCHAR(64) NOT NULL,
  `route_name`     VARCHAR(256) DEFAULT NULL,
  `active`         TINYINT(1) NOT NULL,
  `enabled`        TINYINT(1) NOT NULL,
  `icon`           VARCHAR(32)  DEFAULT NULL,
  `wording`        VARCHAR(255) DEFAULT NULL,
  `wording_domain` VARCHAR(255) DEFAULT NULL,
  PRIMARY KEY (`id_tab`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE `ps_tab_lang` (
  `id_tab`  INT          NOT NULL,
  `id_lang` INT          NOT NULL,
  `name`    VARCHAR(128) NOT NULL,
  KEY   `IDX_CFD9262DED47AB56` (`id_tab`),
  KEY   `IDX_CFD9262DBA299860` (`id_lang`),
  PRIMARY KEY (`id_tab`, `id_lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE `ps_admin_filter` (
  `id`         INT AUTO_INCREMENT NOT NULL,
  `employee`   INT          NOT NULL,
  `shop`       INT          NOT NULL,
  `controller` VARCHAR(60)  NOT NULL,
  `action`     VARCHAR(100) NOT NULL,
  `filter`     LONGTEXT     NOT NULL,
  `filter_id`  VARCHAR(191) NOT NULL,
  UNIQUE KEY `admin_filter_search_id_idx` (`employee`, `shop`, `controller`, `action`, `filter_id`),
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_shop` (
  `id_shop`       INT AUTO_INCREMENT NOT NULL,
  `id_shop_group` INT          NOT NULL,
  `name`          VARCHAR(64)  NOT NULL,
  `color`         VARCHAR(50)  NOT NULL,
  `id_category`   INT          NOT NULL,
  `theme_name`    VARCHAR(255) NOT NULL,
  `active`        TINYINT(1) NOT NULL,
  `deleted`       TINYINT(1) NOT NULL,
  KEY             `IDX_CBDFBB9EF5C9E40` (`id_shop_group`),
  PRIMARY KEY (`id_shop`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_shop_url` (
  `id_shop_url`  int(11) unsigned NOT NULL AUTO_INCREMENT,
  `id_shop`      int(11) unsigned NOT NULL,
  `domain`       varchar(255) NOT NULL,
  `domain_ssl`   varchar(255) NOT NULL,
  `physical_uri` varchar(64)  NOT NULL,
  `virtual_uri`  varchar(64)  NOT NULL,
  `main`         TINYINT(1) NOT NULL,
  `active`       TINYINT(1) NOT NULL,
  PRIMARY KEY (`id_shop_url`),
  KEY          `id_shop` (`id_shop`, `main`),
  UNIQUE KEY `full_shop_url` (`domain`, `physical_uri`, `virtual_uri`),
  UNIQUE KEY `full_shop_url_ssl` (`domain_ssl`, `physical_uri`, `virtual_uri`),
  KEY `IDX_279F19DA274A50A0` (`id_shop`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_shop_group` (
  `id_shop_group`  INT AUTO_INCREMENT NOT NULL,
  `name`           VARCHAR(64) NOT NULL,
  `color`          VARCHAR(50) NOT NULL,
  `share_customer` TINYINT(1) NOT NULL,
  `share_order`    TINYINT(1) NOT NULL,
  `share_stock`    TINYINT(1) NOT NULL,
  `active`         TINYINT(1) NOT NULL,
  `deleted`        TINYINT(1) NOT NULL,
  PRIMARY KEY (`id_shop_group`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_module_history` (
  `id`          INT AUTO_INCREMENT NOT NULL,
  `id_employee` INT      NOT NULL,
  `id_module`   INT      NOT NULL,
  `date_add`    DATETIME NOT NULL,
  `date_upd`    DATETIME NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_feature_flag` (
  `id_feature_flag`     INT UNSIGNED AUTO_INCREMENT NOT NULL,
  `name`                VARCHAR(191)                         NOT NULL,
  `type`                VARCHAR(64)  DEFAULT 'env,dotenv,db' NOT NULL,
  `state`               TINYINT(1) DEFAULT 0 NOT NULL,
  `label_wording`       VARCHAR(191) DEFAULT ''              NOT NULL,
  `label_domain`        VARCHAR(255) DEFAULT ''              NOT NULL,
  `description_wording` VARCHAR(191) DEFAULT ''              NOT NULL,
  `description_domain`  VARCHAR(255) DEFAULT ''              NOT NULL,
  `stability`           VARCHAR(64)  DEFAULT 'beta'          NOT NULL,
  UNIQUE KEY `UNIQ_91700F175E237E06` (`name`),
  PRIMARY KEY (`id_feature_flag`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_api_client` (
  `id_api_client`   INT UNSIGNED AUTO_INCREMENT NOT NULL,
  `client_id`       VARCHAR(255)              NOT NULL,
  `client_name`     VARCHAR(255)              NOT NULL,
  `client_secret`   VARCHAR(255) DEFAULT NULL,
  `enabled`         TINYINT(1) NOT NULL,
  `scopes`          JSON                      NOT NULL,
  `description`     VARCHAR(255) DEFAULT ''   NOT NULL,
  `external_issuer` VARCHAR(255) DEFAULT NULL,
  `lifetime`        INT          DEFAULT 3600 NOT NULL,
  UNIQUE INDEX `api_client_client_id_idx` (`client_id`, `external_issuer`),
  UNIQUE INDEX `api_client_client_name_idx` (`client_name`, `external_issuer`),
  PRIMARY KEY (`id_api_client`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_stock_mvt` (
  `id_stock_mvt`        BIGINT AUTO_INCREMENT NOT NULL,
  `id_stock`            INT                      NOT NULL,
  `id_order`            INT            DEFAULT NULL,
  `id_supply_order`     INT            DEFAULT 0,
  `id_stock_mvt_reason` INT                      NOT NULL,
  `id_employee`         INT                      NOT NULL,
  `employee_lastname`   VARCHAR(255)   DEFAULT NULL,
  `employee_firstname`  VARCHAR(255)   DEFAULT NULL,
  `physical_quantity`   INT UNSIGNED NOT NULL,
  `date_add`            DATETIME                 NOT NULL,
  `sign`                SMALLINT       DEFAULT 1 NOT NULL,
  `price_te`            NUMERIC(20, 6) DEFAULT '0.000000',
  `last_wa`             NUMERIC(20, 6) DEFAULT '0.000000',
  `current_wa`          NUMERIC(20, 6) DEFAULT '0.000000',
  `referer`             BIGINT         DEFAULT NULL,
  INDEX                 `id_stock` (`id_stock`),
  INDEX                 `id_stock_mvt_reason` (`id_stock_mvt_reason`),
  PRIMARY KEY (`id_stock_mvt`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* Association between a profile and a tab authorization_role (can be 'CREATE', 'READ', 'UPDATE' or 'DELETE') */
CREATE TABLE `ps_access` (
  `id_profile` int(10) unsigned NOT NULL,
  `id_authorization_role` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id_profile`, `id_authorization_role`),
  KEY `IDX_564352A15FCA037F` (`id_profile`),
  KEY `IDX_564352A18C6DE0E5` (`id_authorization_role`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_shipment` (
  `id_shipment` int(10) AUTO_INCREMENT NOT NULL,
  `id_order` int(10) NOT NULL,
  `id_carrier` int(10) NOT NULL,
  `id_delivery_address` int(10) DEFAULT NULL,
  `shipping_cost_tax_excl` NUMERIC(20, 6) DEFAULT '0.000000',
  `shipping_cost_tax_incl` NUMERIC(20, 6) DEFAULT '0.000000',
  `packed_at` datetime DEFAULT NULL,
  `shipped_at` datetime DEFAULT NULL,
  `delivered_at` datetime DEFAULT NULL,
  `cancelled_at` DATETIME DEFAULT NULL,
  `tracking_number` varchar(255) DEFAULT NULL,
  `deleted` tinyint(1) NOT NULL DEFAULT 0,
  `date_add` datetime NOT NULL,
  `date_upd` datetime NOT NULL,
  PRIMARY KEY (`id_shipment`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_shipment_product` (
  `id_shipment_product` INT AUTO_INCREMENT NOT NULL,
  `id_shipment` int(10) NOT NULL,
  `id_order_detail` int(10) NOT NULL,
  `quantity` int(10) DEFAULT NULL,
  PRIMARY KEY (id_shipment_product)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE `ps_business_entity`
(
  `id_business_entity`  INT UNSIGNED AUTO_INCREMENT                     NOT NULL,
  `id_shop`             INT UNSIGNED                                    NOT NULL,
  `id_customer_group`   INT UNSIGNED                                    NOT NULL,
  `external_ref`        VARCHAR(255) DEFAULT NULL,
  `name`                VARCHAR(255)                                    NOT NULL,
  `legal_name`          VARCHAR(255) DEFAULT NULL,
  `delivery_authorized` TINYINT(1)                                      NOT NULL DEFAULT 0,
  `status`              ENUM ('pending','active','inactive','rejected') NOT NULL DEFAULT 'pending',
  `deleted`             TINYINT(1)                                      NOT NULL DEFAULT 0,
  `created_at`          DATETIME                                        NOT NULL,
  `updated_at`          DATETIME                                        NOT NULL,
  INDEX                 `business_entity_shop_idx` (`id_shop`),
  INDEX                 `business_entity_customer_group_idx` (`id_customer_group`),
  INDEX                 `business_entity_external_ref_idx` (`external_ref`),
  INDEX                 `business_entity_deleted_idx` (`deleted`),
  PRIMARY KEY (`id_business_entity`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE `ps_customer_b2b`
(
  `id_customer_b2b` INT UNSIGNED AUTO_INCREMENT          NOT NULL,
  `id_customer`     INT UNSIGNED                         NOT NULL,
  `status`          ENUM ('pending','active','rejected') NOT NULL DEFAULT 'pending',
  `external_ref`    VARCHAR(255) DEFAULT NULL,
  `created_at`      DATETIME NOT NULL,
  `updated_at`      DATETIME NOT NULL,
  UNIQUE INDEX `uniq_customer_b2b_customer` (`id_customer`),
  PRIMARY KEY (`id_customer_b2b`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE `ps_business_entity_customer_b2b`
(
  `id_business_entity_customer_b2b` INT UNSIGNED AUTO_INCREMENT NOT NULL,
  `id_business_entity`              INT UNSIGNED                NOT NULL,
  `id_customer_b2b`                 INT UNSIGNED                NOT NULL,
  `id_role`                         INT UNSIGNED                NOT NULL,
  `is_default`                      TINYINT(1)                  NOT NULL DEFAULT 0,
  `created_at`                      DATETIME                    NOT NULL,
  `updated_at`                      DATETIME                    NOT NULL,
  UNIQUE INDEX `uniq_be_customer` (`id_business_entity`, `id_customer_b2b`),
  INDEX                             `business_entity_customer_b2b_be_idx` (`id_business_entity`),
  INDEX                             `business_entity_customer_b2b_customer_idx` (`id_customer_b2b`),
  INDEX                             `business_entity_customer_b2b_role_idx` (`id_role`),
  PRIMARY KEY (`id_business_entity_customer_b2b`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE `ps_business_entity_identifier`
(
  `id_identifier`          INT UNSIGNED AUTO_INCREMENT NOT NULL,
  `id_business_entity`     INT UNSIGNED                NOT NULL,
  `id_business_identifier` INT UNSIGNED                NOT NULL,
  `value`                  VARCHAR(255)                NOT NULL,
  `created_at`             DATETIME                    NOT NULL,
  `updated_at`             DATETIME                    NOT NULL,
  UNIQUE INDEX `uniq_business_entity_identifier` (`id_business_entity`, `id_business_identifier`),
  INDEX                    `business_entity_identifier_id_business_entity_idx` (`id_business_entity`),
  INDEX                    `business_entity_identifier_id_business_identifier_idx` (`id_business_identifier`),
  INDEX                    `business_entity_identifier_value_idx` (`value`),
  PRIMARY KEY (`id_identifier`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE `ps_business_identifier`
(
  `id_business_identifier` INT UNSIGNED AUTO_INCREMENT NOT NULL,
  `label`                  VARCHAR(255)                NOT NULL,
  `unremovable`            TINYINT(1)                  NOT NULL DEFAULT 0,
  `id_zone`                INT UNSIGNED                DEFAULT NULL,
  `deleted`                TINYINT(1)                  NOT NULL DEFAULT 0,
  `created_at`             DATETIME                    NOT NULL,
  `updated_at`             DATETIME                    NOT NULL,
  INDEX                    `business_identifier_zone_idx` (`id_zone`),
  PRIMARY KEY (`id_business_identifier`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE `ps_business_entity_address`
(
  `id_business_entity_address` INT UNSIGNED AUTO_INCREMENT         NOT NULL,
  `id_business_entity`         INT UNSIGNED                        NOT NULL,
  `id_address`                 INT UNSIGNED                        NOT NULL,
  `address_type`               ENUM ('both','invoice','delivery')  NOT NULL DEFAULT 'both',
  `is_default`                 TINYINT(1)                          NOT NULL DEFAULT 0,
  `created_at`                 DATETIME                            NOT NULL,
  `updated_at`                 DATETIME                            NOT NULL,
  UNIQUE INDEX `uniq_be_address` (`id_business_entity`, `id_address`, `address_type`),
  INDEX                        `business_entity_address_be_idx` (`id_business_entity`),
  INDEX                        `business_entity_address_address_idx` (`id_address`),
  PRIMARY KEY (`id_business_entity_address`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE `ps_b2b_role`
(
  `id_role` INT UNSIGNED AUTO_INCREMENT NOT NULL,
  `role`    VARCHAR(64) NOT NULL,
  UNIQUE INDEX `uniq_b2b_role` (`role`),
  PRIMARY KEY (`id_role`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE `ps_b2b_role_authorization_role`
(
  `id_role`               INT UNSIGNED NOT NULL,
  `id_authorization_role` INT UNSIGNED NOT NULL,
  PRIMARY KEY (`id_role`, `id_authorization_role`),
  INDEX                   `b2b_role_authorization_role_role_idx` (`id_role`),
  INDEX                   `b2b_role_authorization_role_auth_role_idx` (`id_authorization_role`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;
