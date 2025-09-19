ALTER TABLE "public"."documents" ADD COLUMN "checksum" bytea;
ALTER TABLE "public"."documents" ADD COLUMN "file_info" jsonb DEFAULT '{}';
ALTER TABLE "public"."documents" ADD COLUMN "updated_at" timestamp(6) without time zone;
ALTER TABLE "public"."documents" ALTER COLUMN "metadata" TYPE jsonb;
CREATE INDEX idx_documents_metadata ON public.documents USING gin (metadata);

CREATE TABLE "public"."files" (
    "file_id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "filename" character varying(255) NOT NULL,
    "content" bytea NOT NULL,
    "mime_type" character varying(100),
    "metadata" jsonb DEFAULT '{}',
    "size_bytes" integer,
    "upload_time" timestamp(6) without time zone DEFAULT CURRENT_TIMESTAMP
);
ALTER TABLE "public"."files" ADD CONSTRAINT "files_pkey" PRIMARY KEY (file_id);
CREATE INDEX idx_files_metadata ON public.files USING gin (metadata);
CREATE INDEX idx_files_mime_type ON public.files USING btree (mime_type);

ALTER TABLE "public"."sessions" ADD COLUMN "last_accessed" timestamp(6) without time zone DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE "public"."sessions" ADD COLUMN "settings" jsonb DEFAULT '{"theme": "default"}'::jsonb;
ALTER TABLE "public"."sessions" ALTER COLUMN "session_id" SET DEFAULT gen_random_uuid();
CREATE INDEX idx_sessions_data ON public.sessions USING gin (data);
CREATE INDEX idx_sessions_settings ON public.sessions USING gin (settings);

