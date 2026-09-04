CREATE TABLE "ORDERS" (
    "ID" NUMBER DEFAULT "TESTSCHEMA"."ISEQ$$_73541".nextval NOT NULL,
    "CUSTOMER_ID" NUMBER NOT NULL,
    "PRODUCT_ID" NUMBER NOT NULL,
    "ORDER_DATE" DATE DEFAULT SYSDATE,
    "QUANTITY" NUMBER NOT NULL,
    "TOTAL_AMOUNT" NUMBER(12,2),
    "STATUS" VARCHAR2(20 BYTE) DEFAULT 'PENDING'
);
ALTER TABLE "ORDERS" ADD CONSTRAINT "SYS_C008901" PRIMARY KEY (ID);
COMMENT ON TABLE "ORDERS" IS 'Customer orders linking products and customers';
COMMENT ON COLUMN "ORDERS"."ID" IS 'Unique order identifier';
COMMENT ON COLUMN "ORDERS"."CUSTOMER_ID" IS 'Reference to customer who placed the order';
COMMENT ON COLUMN "ORDERS"."PRODUCT_ID" IS 'Reference to ordered product';
COMMENT ON COLUMN "ORDERS"."ORDER_DATE" IS 'Date when the order was placed';
COMMENT ON COLUMN "ORDERS"."QUANTITY" IS 'Number of items ordered';
COMMENT ON COLUMN "ORDERS"."TOTAL_AMOUNT" IS 'Total order amount including taxes';
COMMENT ON COLUMN "ORDERS"."STATUS" IS 'Current order status (PENDING, SHIPPED, DELIVERED, CANCELLED)';

ALTER TABLE "ORDERS" ADD CONSTRAINT "FK_ORDER_CUSTOMER" FOREIGN KEY ("CUSTOMER_ID") REFERENCES "CUSTOMERS" ("ID");
ALTER TABLE "ORDERS" ADD CONSTRAINT "FK_ORDER_PRODUCT" FOREIGN KEY ("PRODUCT_ID") REFERENCES "PRODUCTS" ("ID");
COMMENT ON TABLE "CUSTOMERS" IS 'Customer information and contact details';
COMMENT ON COLUMN "CUSTOMERS"."ID" IS 'Primary key for customer records';
COMMENT ON COLUMN "CUSTOMERS"."NAME" IS 'Full name of the customer';
COMMENT ON COLUMN "CUSTOMERS"."EMAIL" IS 'Customer email address (unique)';
COMMENT ON COLUMN "CUSTOMERS"."PHONE" IS 'Contact phone number';
COMMENT ON COLUMN "CUSTOMERS"."ADDRESS" IS 'Full mailing address with multiple lines';
COMMENT ON COLUMN "CUSTOMERS"."ID" IS 'Primary key for customer records';
COMMENT ON COLUMN "CUSTOMERS"."NAME" IS 'Full name of the customer';
COMMENT ON COLUMN "CUSTOMERS"."EMAIL" IS 'Customer email address (unique)';
COMMENT ON COLUMN "CUSTOMERS"."PHONE" IS 'Contact phone number';
COMMENT ON COLUMN "CUSTOMERS"."ADDRESS" IS 'Full mailing address with multiple lines';
COMMENT ON TABLE "PRODUCTS" IS 'Product catalog table containing all product information';
COMMENT ON COLUMN "PRODUCTS"."ID" IS 'Unique product identifier (auto-generated)';
COMMENT ON COLUMN "PRODUCTS"."NAME" IS 'Product name - must be unique and descriptive';
COMMENT ON COLUMN "PRODUCTS"."PRICE" IS 'Product price in USD with 2 decimal places';
COMMENT ON COLUMN "PRODUCTS"."DESCRIPTION" IS 'Detailed product description with HTML formatting support';
COMMENT ON COLUMN "PRODUCTS"."CATEGORY" IS 'Product category for filtering and organization';
COMMENT ON COLUMN "PRODUCTS"."CREATED_AT" IS 'Timestamp when the product was added to catalog';
COMMENT ON COLUMN "PRODUCTS"."ID" IS 'Unique product identifier (auto-generated)';
COMMENT ON COLUMN "PRODUCTS"."NAME" IS 'Product name - must be unique and descriptive';
COMMENT ON COLUMN "PRODUCTS"."PRICE" IS 'Product price in USD with 2 decimal places';
COMMENT ON COLUMN "PRODUCTS"."DESCRIPTION" IS 'Detailed product description with HTML formatting support';
COMMENT ON COLUMN "PRODUCTS"."CATEGORY" IS 'Product category for filtering and organization';
COMMENT ON COLUMN "PRODUCTS"."CREATED_AT" IS 'Timestamp when the product was added to catalog';
