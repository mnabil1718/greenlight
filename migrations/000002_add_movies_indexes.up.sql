CREATE INDEX IF NOT EXISTS movies_title_index ON movies USING GIN (to_tsvector('simple', title));
CREATE INDEX IF NOT EXISTS movies_genres_index ON movies USING GIN (genres);