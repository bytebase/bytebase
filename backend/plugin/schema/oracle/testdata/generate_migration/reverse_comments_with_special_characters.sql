CREATE TABLE "COMPLEX_TYPES" (
    "ID" NUMBER NOT NULL,
    "XML_DATA" XMLTYPE,
    "TIMESTAMP_TZ" TIMESTAMP(6) WITH TIME ZONE DEFAULT SYSTIMESTAMP,
    "INTERVAL_DS" INTERVAL DAY(2) TO SECOND(6),
    "RAW_DATA" RAW(100),
    "LONG_RAW_DATA" LONG RAW
);
ALTER TABLE "COMPLEX_TYPES" ADD CONSTRAINT "SYS_C008744" PRIMARY KEY (ID);
COMMENT ON TABLE "COMPLEX_TYPES" IS 'Advanced data types testing table
Supports: XMLType, Timestamps with timezone, Intervals, RAW data
Performance: Optimized for OLTP workloads @ 10K+ TPS
Security: Row-level security enabled, audit trails maintained';
COMMENT ON COLUMN "COMPLEX_TYPES"."XML_DATA" IS 'XMLType column for structured XML storage
Example: <?xml version="1.0" encoding="UTF-8"?>
<root><item id="1" name="test">Content</item></root>';
COMMENT ON COLUMN "COMPLEX_TYPES"."TIMESTAMP_TZ" IS 'Timestamp with timezone: YYYY-MM-DD HH24:MI:SS.FF TZR
Default: SYSTIMESTAMP (system timestamp with session timezone)';
COMMENT ON COLUMN "COMPLEX_TYPES"."INTERVAL_DS" IS 'Interval Day to Second: +/-DD HH:MI:SS.FFFFFF
Range: +/-999999999 days, precision up to nanoseconds';

COMMENT ON TABLE "MULTILANG_CONTENT" IS 'Multilingual content: English, Español, 中文, العربية, Русский, 日本語';
COMMENT ON COLUMN "MULTILANG_CONTENT"."TITLE_EN" IS 'English title field';
COMMENT ON COLUMN "MULTILANG_CONTENT"."TITLE_ES" IS 'Título en español';
COMMENT ON COLUMN "MULTILANG_CONTENT"."TITLE_ZH" IS '中文标题字段 - 支持简体和繁体中文';
COMMENT ON COLUMN "MULTILANG_CONTENT"."CONTENT_TEXT" IS 'Content field supporting:
1. HTML tags: <b>bold</b>, <i>italic</i>, <a href="http://example.com">links</a>
2. Markdown: **bold**, *italic*, [link](url), backtick-code-backtick
3. Special symbols: © ® ™ § ¶ † ‡ • … ‰ ′ ″
4. Emojis: 😀 😃 😄 😁 🚀 ⭐ 🎉 💯
5. Mathematical: ∀x∈ℝ: x² ≥ 0, ∑(i=1 to n) i = n(n+1)/2';
COMMENT ON COLUMN "MULTILANG_CONTENT"."TAGS" IS 'XML tags: <tag type="category" value="news"/><tag type="priority" value="high"/>End of XML';
COMMENT ON COLUMN "MULTILANG_CONTENT"."TITLE_EN" IS 'English title field';
COMMENT ON COLUMN "MULTILANG_CONTENT"."TITLE_ES" IS 'Título en español';
COMMENT ON COLUMN "MULTILANG_CONTENT"."TITLE_ZH" IS '中文标题字段 - 支持简体和繁体中文';
COMMENT ON COLUMN "MULTILANG_CONTENT"."CONTENT_TEXT" IS 'Content field supporting:
1. HTML tags: <b>bold</b>, <i>italic</i>, <a href="http://example.com">links</a>
2. Markdown: **bold**, *italic*, [link](url), backtick-code-backtick
3. Special symbols: © ® ™ § ¶ † ‡ • … ‰ ′ ″
4. Emojis: 😀 😃 😄 😁 🚀 ⭐ 🎉 💯
5. Mathematical: ∀x∈ℝ: x² ≥ 0, ∑(i=1 to n) i = n(n+1)/2';
COMMENT ON COLUMN "MULTILANG_CONTENT"."TAGS" IS 'XML tags: <tag type="category" value="news"/><tag type="priority" value="high"/>End of XML';
COMMENT ON TABLE "SPECIAL_DATA" IS 'Table containing "special" data with various formats & types';
COMMENT ON COLUMN "SPECIAL_DATA"."DATA_VALUE" IS 'Data value field - supports NULL, empty strings, and ''quoted'' content';
COMMENT ON COLUMN "SPECIAL_DATA"."METADATA_JSON" IS 'JSON metadata: {"key": "value", "array": [1,2,3]}';
COMMENT ON COLUMN "SPECIAL_DATA"."UNICODE_TEXT" IS 'Unicode text support:
Line 1: Basic ASCII text
Line 2: Special chars !@#$%^&*()_+-={}[]|;:,.<>?
Line 3: Math symbols ≤ ≥ ± × ÷ ∑ ∏ ∆ π
Line 4: Currencies $ € £ ¥ ₹';
COMMENT ON COLUMN "SPECIAL_DATA"."BINARY_DATA" IS 'Binary data storage\nSupports: images, documents, archives\tTab-separated info\r\nWindows line endings';
COMMENT ON COLUMN "SPECIAL_DATA"."CATEGORY" IS 'Category field: use % for wildcard searches, _ for single char matching';
COMMENT ON COLUMN "SPECIAL_DATA"."STATUS" IS 'Status field - values: ''ACTIVE'', ''INACTIVE'', ''PENDING''; DROP TABLE users; --';
COMMENT ON COLUMN "SPECIAL_DATA"."DATA_VALUE" IS 'Data value field - supports NULL, empty strings, and ''quoted'' content';
COMMENT ON COLUMN "SPECIAL_DATA"."METADATA_JSON" IS 'JSON metadata: {"key": "value", "array": [1,2,3]}';
COMMENT ON COLUMN "SPECIAL_DATA"."UNICODE_TEXT" IS 'Unicode text support:
Line 1: Basic ASCII text
Line 2: Special chars !@#$%^&*()_+-={}[]|;:,.<>?
Line 3: Math symbols ≤ ≥ ± × ÷ ∑ ∏ ∆ π
Line 4: Currencies $ € £ ¥ ₹';
COMMENT ON COLUMN "SPECIAL_DATA"."BINARY_DATA" IS 'Binary data storage\nSupports: images, documents, archives\tTab-separated info\r\nWindows line endings';
COMMENT ON COLUMN "SPECIAL_DATA"."CATEGORY" IS 'Category field: use % for wildcard searches, _ for single char matching';
COMMENT ON COLUMN "SPECIAL_DATA"."STATUS" IS 'Status field - values: ''ACTIVE'', ''INACTIVE'', ''PENDING''; DROP TABLE users; --';
