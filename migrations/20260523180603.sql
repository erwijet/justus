-- Drop index "idx_user_song_played" from table: "listens"
DROP INDEX "public"."idx_user_song_played";
-- Create index "idx_user_played_at" to table: "listens"
CREATE UNIQUE INDEX "idx_user_played_at" ON "public"."listens" ("user_id", "played_at");
