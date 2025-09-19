CREATE SEQUENCE "public"."analytics_analysis_id_seq" START WITH 1 INCREMENT BY 1;
CREATE TABLE "public"."analytics" (
    "analysis_id" integer NOT NULL DEFAULT nextval('public.analytics_analysis_id_seq'::regclass),
    "data_points" _numeric NOT NULL,
    "categories" _text NOT NULL,
    "timestamps" _timestamptz NOT NULL,
    "user_ids" _uuid,
    "scores" _float4 DEFAULT ARRAY[]::REAL[],
    "flags" _bool DEFAULT ARRAY[]::BOOLEAN[]
);
ALTER TABLE "public"."analytics" ADD CONSTRAINT "analytics_pkey" PRIMARY KEY (analysis_id);
CREATE INDEX idx_analytics_categories ON public.analytics USING gin (categories);

ALTER TABLE "public"."events" ADD COLUMN "capacities" _int4;
ALTER TABLE "public"."events" ADD COLUMN "locations" _text DEFAULT ARRAY[]::TEXT[] NOT NULL;
ALTER TABLE "public"."events" ADD COLUMN "metadata" _jsonb DEFAULT ARRAY[]::JSONB[];
ALTER TABLE "public"."events" ADD COLUMN "schedules" _timestamp;
ALTER TABLE "public"."events" ALTER COLUMN "event_dates" TYPE _timestamptz;
CREATE INDEX idx_events_locations ON public.events USING gin (locations);
CREATE INDEX idx_events_participants ON public.events USING gin (participant_ids);

ALTER TABLE "public"."products" ADD COLUMN "features" _jsonb;
ALTER TABLE "public"."products" ADD COLUMN "ratings" _int4 DEFAULT ARRAY[0,0,0,0,0];
ALTER TABLE "public"."products" ADD COLUMN "variants" _text;
ALTER TABLE "public"."products" ALTER COLUMN "category_ids" TYPE _int8;
ALTER TABLE "public"."products" ALTER COLUMN "tags" SET DEFAULT ARRAY[]::TEXT[];
CREATE INDEX idx_products_categories ON public.products USING gin (category_ids);
CREATE INDEX idx_products_tags ON public.products USING gin (tags);

