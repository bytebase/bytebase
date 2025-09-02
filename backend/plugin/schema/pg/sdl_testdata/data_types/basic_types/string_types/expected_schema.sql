-- Expanded table with comprehensive string types
CREATE TABLE public.basic_strings (
    id SERIAL PRIMARY KEY,
    short_name VARCHAR(20) NOT NULL,
    description TEXT
);

CREATE TABLE public.string_types_test (
    id SERIAL PRIMARY KEY,
    
    -- Fixed length character types
    char_col CHAR(10),
    char_default CHAR(1) DEFAULT 'A',
    
    -- Variable length character types
    varchar_short VARCHAR(50) NOT NULL,
    varchar_long VARCHAR(500),
    varchar_no_limit VARCHAR,
    
    -- Unlimited text
    text_col TEXT,
    text_with_default TEXT DEFAULT 'default text',
    
    -- Constraints and indexes
    UNIQUE (varchar_short),
    CHECK (LENGTH(text_col) > 0 OR text_col IS NULL)
);

-- Create indexes on string columns
CREATE INDEX idx_string_varchar_short ON public.string_types_test(varchar_short);
CREATE INDEX idx_string_text_prefix ON public.string_types_test(LEFT(text_col, 50)) WHERE text_col IS NOT NULL;
CREATE INDEX idx_string_char_col ON public.string_types_test(char_col) WHERE char_col IS NOT NULL;