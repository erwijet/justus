-- Create "listens" table
CREATE TABLE "public"."listens" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "user_id" uuid NULL,
  "song_id" uuid NULL,
  "played_at" timestamptz NULL,
  PRIMARY KEY ("id")
);
-- Create index "idx_listens_deleted_at" to table: "listens"
CREATE INDEX "idx_listens_deleted_at" ON "public"."listens" ("deleted_at");
-- Create index "idx_user_song_played" to table: "listens"
CREATE UNIQUE INDEX "idx_user_song_played" ON "public"."listens" ("user_id", "song_id", "played_at");
-- Create "songs" table
CREATE TABLE "public"."songs" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "spotify_id" text NULL,
  "name" text NULL,
  "artist_name" text NULL,
  PRIMARY KEY ("id")
);
-- Create index "idx_songs_deleted_at" to table: "songs"
CREATE INDEX "idx_songs_deleted_at" ON "public"."songs" ("deleted_at");
-- Create index "idx_songs_spotify_id" to table: "songs"
CREATE UNIQUE INDEX "idx_songs_spotify_id" ON "public"."songs" ("spotify_id");
-- Create "users" table
CREATE TABLE "public"."users" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "spotify_id" text NULL,
  "display_name" text NULL,
  "access_token" text NULL,
  "refresh_token" text NULL,
  "token_expiry" timestamptz NULL,
  PRIMARY KEY ("id")
);
-- Create index "idx_users_deleted_at" to table: "users"
CREATE INDEX "idx_users_deleted_at" ON "public"."users" ("deleted_at");
-- Create index "idx_users_spotify_id" to table: "users"
CREATE UNIQUE INDEX "idx_users_spotify_id" ON "public"."users" ("spotify_id");
