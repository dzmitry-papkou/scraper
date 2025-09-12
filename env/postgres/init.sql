CREATE TABLE IF NOT EXISTS posts (
    id SERIAL PRIMARY KEY,
    hn_id INTEGER UNIQUE NOT NULL,
    title TEXT NOT NULL,
    url TEXT,
    author VARCHAR(255) NOT NULL,
    points INTEGER DEFAULT 0,
    comments_count INTEGER DEFAULT 0,
    post_time TIMESTAMP NOT NULL,
    scraped_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS post_history (
    id SERIAL PRIMARY KEY,
    post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    points INTEGER DEFAULT 0,
    comments_count INTEGER DEFAULT 0,
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS scraping_jobs (
    id SERIAL PRIMARY KEY,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    status VARCHAR(50) DEFAULT 'running',
    posts_scraped INTEGER DEFAULT 0,
    error_message TEXT
);

CREATE TABLE IF NOT EXISTS analysis_results (
    id SERIAL PRIMARY KEY,
    analysis_type VARCHAR(100) NOT NULL,
    analysis_date DATE NOT NULL,
    results TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- indexes
CREATE INDEX IF NOT EXISTS idx_posts_hn_id ON posts(hn_id);
CREATE INDEX IF NOT EXISTS idx_posts_post_time ON posts(post_time DESC);
CREATE INDEX IF NOT EXISTS idx_posts_author ON posts(author);
CREATE INDEX IF NOT EXISTS idx_posts_points ON posts(points DESC);
CREATE INDEX IF NOT EXISTS idx_posts_scraped_at ON posts(scraped_at DESC);
CREATE INDEX IF NOT EXISTS idx_posts_updated_at ON posts(updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_post_history_post_id ON post_history(post_id);
CREATE INDEX IF NOT EXISTS idx_post_history_recorded_at ON post_history(recorded_at DESC);

CREATE INDEX IF NOT EXISTS idx_scraping_jobs_status ON scraping_jobs(status);
CREATE INDEX IF NOT EXISTS idx_scraping_jobs_started_at ON scraping_jobs(started_at DESC);

CREATE INDEX IF NOT EXISTS idx_analysis_results_type ON analysis_results(analysis_type);
CREATE INDEX IF NOT EXISTS idx_analysis_results_date ON analysis_results(analysis_date DESC);


-- functions and triggers

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_posts_updated_at ON posts;
CREATE TRIGGER update_posts_updated_at 
    BEFORE UPDATE ON posts
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();


CREATE OR REPLACE VIEW recent_posts_with_history AS
SELECT 
    p.*,
    ph.points as latest_history_points,
    ph.comments_count as latest_history_comments,
    ph.recorded_at as latest_history_time
FROM posts p
LEFT JOIN LATERAL (
    SELECT points, comments_count, recorded_at
    FROM post_history
    WHERE post_id = p.id
    ORDER BY recorded_at DESC
    LIMIT 1
) ph ON true
ORDER BY p.scraped_at DESC
LIMIT 100;

CREATE OR REPLACE VIEW daily_statistics AS
SELECT 
    DATE(post_time) as date,
    COUNT(*) as total_posts,
    COALESCE(AVG(points), 0) as avg_points,
    COALESCE(AVG(comments_count), 0) as avg_comments,
    COALESCE(MAX(points), 0) as max_points,
    COALESCE(MAX(comments_count), 0) as max_comments,
    COUNT(DISTINCT author) as unique_authors
FROM posts
GROUP BY DATE(post_time)
ORDER BY date DESC;

CREATE OR REPLACE VIEW author_statistics AS
SELECT 
    author,
    COUNT(*) as post_count,
    COALESCE(AVG(points), 0) as avg_points,
    COALESCE(MAX(points), 0) as max_points,
    COALESCE(SUM(points), 0) as total_points,
    MIN(post_time) as first_post,
    MAX(post_time) as last_post
FROM posts
GROUP BY author
HAVING COUNT(*) > 1
ORDER BY avg_points DESC;


CREATE OR REPLACE FUNCTION get_basic_stats()
RETURNS TABLE(
    total_posts BIGINT,
    unique_authors BIGINT,
    avg_points NUMERIC,
    avg_comments NUMERIC,
    max_points INTEGER,
    max_comments INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*)::BIGINT as total_posts,
        COUNT(DISTINCT author)::BIGINT as unique_authors,
        COALESCE(AVG(points), 0)::NUMERIC as avg_points,
        COALESCE(AVG(comments_count), 0)::NUMERIC as avg_comments,
        COALESCE(MAX(points), 0) as max_points,
        COALESCE(MAX(comments_count), 0) as max_comments
    FROM posts;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION get_posts_in_range(
    start_date TIMESTAMP,
    end_date TIMESTAMP
)
RETURNS SETOF posts AS $$
BEGIN
    RETURN QUERY
    SELECT * FROM posts
    WHERE post_time BETWEEN start_date AND end_date
    ORDER BY post_time DESC;
END;
$$ LANGUAGE plpgsql;


-- initial scraping job for testing
INSERT INTO scraping_jobs (status, posts_scraped, completed_at) 
VALUES ('initialized', 0, CURRENT_TIMESTAMP)
ON CONFLICT DO NOTHING;