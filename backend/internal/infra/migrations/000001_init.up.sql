-- 000001_init: baseline schema (generated from GORM AutoMigrate + production indexes)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: delivery_attempts; Type: TABLE
--

CREATE TABLE delivery_attempts (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    post_id bigint NOT NULL,
    channel_id bigint NOT NULL,
    status smallint DEFAULT 0 NOT NULL,
    attempts bigint DEFAULT 0 NOT NULL,
    next_at bigint NOT NULL,
    last_error text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE SEQUENCE delivery_attempts_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE delivery_attempts_id_seq OWNED BY delivery_attempts.id;
ALTER TABLE ONLY delivery_attempts ALTER COLUMN id SET DEFAULT nextval('delivery_attempts_id_seq'::regclass);

--
-- Name: delivery_channels; Type: TABLE
--

CREATE TABLE delivery_channels (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    kind character varying(32) NOT NULL,
    name text DEFAULT ''::text NOT NULL,
    enabled boolean DEFAULT true NOT NULL,
    configuration text DEFAULT '{}'::text NOT NULL,
    keywords text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE SEQUENCE delivery_channels_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE delivery_channels_id_seq OWNED BY delivery_channels.id;
ALTER TABLE ONLY delivery_channels ALTER COLUMN id SET DEFAULT nextval('delivery_channels_id_seq'::regclass);

--
-- Name: delivery_history; Type: TABLE
--

CREATE TABLE delivery_history (
    id bigint NOT NULL,
    user_id bigint,
    post_id bigint,
    channel_id bigint,
    status smallint NOT NULL,
    last_error text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone
);

CREATE SEQUENCE delivery_history_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE delivery_history_id_seq OWNED BY delivery_history.id;
ALTER TABLE ONLY delivery_history ALTER COLUMN id SET DEFAULT nextval('delivery_history_id_seq'::regclass);

--
-- Name: posts; Type: TABLE
--

CREATE TABLE posts (
    id bigint NOT NULL,
    qid text NOT NULL,
    title text NOT NULL,
    body text NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    user_id bigint NOT NULL
);

CREATE SEQUENCE posts_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE posts_id_seq OWNED BY posts.id;
ALTER TABLE ONLY posts ALTER COLUMN id SET DEFAULT nextval('posts_id_seq'::regclass);

--
-- Name: refresh_tokens; Type: TABLE
--

CREATE TABLE refresh_tokens (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    token_hash text NOT NULL,
    revoked boolean DEFAULT false NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone
);

CREATE SEQUENCE refresh_tokens_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE refresh_tokens_id_seq OWNED BY refresh_tokens.id;
ALTER TABLE ONLY refresh_tokens ALTER COLUMN id SET DEFAULT nextval('refresh_tokens_id_seq'::regclass);

--
-- Name: token_blacklist; Type: TABLE
--

CREATE TABLE token_blacklist (
    id bigint NOT NULL,
    token_hash text NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone
);

CREATE SEQUENCE token_blacklist_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE token_blacklist_id_seq OWNED BY token_blacklist.id;
ALTER TABLE ONLY token_blacklist ALTER COLUMN id SET DEFAULT nextval('token_blacklist_id_seq'::regclass);

--
-- Name: users; Type: TABLE
--

CREATE TABLE users (
    id bigint NOT NULL,
    email text DEFAULT ''::text NOT NULL,
    username text NOT NULL,
    name text,
    password_hash text,
    avatar_url text,
    post_key text NOT NULL,
    github_id bigint,
    role text DEFAULT 'user'::text NOT NULL,
    is_active boolean DEFAULT true,
    is_email_verified boolean DEFAULT false,
    last_login_at timestamp with time zone,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE SEQUENCE users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE users_id_seq OWNED BY users.id;
ALTER TABLE ONLY users ALTER COLUMN id SET DEFAULT nextval('users_id_seq'::regclass);

-- Primary keys

ALTER TABLE ONLY delivery_attempts ADD CONSTRAINT delivery_attempts_pkey PRIMARY KEY (id);
ALTER TABLE ONLY delivery_channels ADD CONSTRAINT delivery_channels_pkey PRIMARY KEY (id);
ALTER TABLE ONLY delivery_history ADD CONSTRAINT delivery_history_pkey PRIMARY KEY (id);
ALTER TABLE ONLY posts ADD CONSTRAINT posts_pkey PRIMARY KEY (id);
ALTER TABLE ONLY refresh_tokens ADD CONSTRAINT refresh_tokens_pkey PRIMARY KEY (id);
ALTER TABLE ONLY token_blacklist ADD CONSTRAINT token_blacklist_pkey PRIMARY KEY (id);
ALTER TABLE ONLY users ADD CONSTRAINT users_pkey PRIMARY KEY (id);

-- Unique constraints

ALTER TABLE ONLY posts ADD CONSTRAINT uni_posts_qid UNIQUE (qid);
ALTER TABLE ONLY refresh_tokens ADD CONSTRAINT uni_refresh_tokens_token_hash UNIQUE (token_hash);
ALTER TABLE ONLY token_blacklist ADD CONSTRAINT uni_token_blacklist_token_hash UNIQUE (token_hash);
ALTER TABLE ONLY users ADD CONSTRAINT uni_users_email UNIQUE (email);
ALTER TABLE ONLY users ADD CONSTRAINT uni_users_github_id UNIQUE (github_id);
ALTER TABLE ONLY users ADD CONSTRAINT uni_users_post_key UNIQUE (post_key);
ALTER TABLE ONLY users ADD CONSTRAINT uni_users_username UNIQUE (username);

-- GORM auto-generated indexes

CREATE INDEX idx_delivery_attempts_channel_id ON delivery_attempts USING btree (channel_id);
CREATE INDEX idx_delivery_attempts_post_id ON delivery_attempts USING btree (post_id);
CREATE INDEX idx_delivery_attempts_user_id ON delivery_attempts USING btree (user_id);
CREATE INDEX idx_delivery_channels_user_id ON delivery_channels USING btree (user_id);
CREATE INDEX idx_delivery_history_channel_id ON delivery_history USING btree (channel_id);
CREATE INDEX idx_delivery_history_post_id ON delivery_history USING btree (post_id);
CREATE INDEX idx_delivery_history_user_id ON delivery_history USING btree (user_id);
CREATE INDEX idx_posts_user_id ON posts USING btree (user_id);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens USING btree (user_id);
CREATE INDEX idx_token_blacklist_expires_at ON token_blacklist USING btree (expires_at);
CREATE INDEX idx_token_blacklist_token_hash ON token_blacklist USING btree (token_hash);

-- Production-specific: partial index for delivery claim query (replaces runtime migrateDeliveryIndexesAndOptions)
CREATE INDEX idx_da_pending ON delivery_attempts (next_at) WHERE status = 0;
CREATE INDEX idx_dh_user_channel_created ON delivery_history (user_id, channel_id, created_at DESC);
CREATE INDEX idx_dh_created ON delivery_history (created_at DESC);

-- Production-specific: storage reloptions (replaces runtime migrateDeliveryIndexesAndOptions)
ALTER TABLE delivery_attempts SET (
    fillfactor = 90,
    autovacuum_vacuum_scale_factor = 0.05,
    autovacuum_vacuum_threshold = 1000,
    autovacuum_analyze_scale_factor = 0.02,
    autovacuum_analyze_threshold = 1000
);
ALTER TABLE delivery_history SET (fillfactor = 100);

-- Production-specific: lz4 TOAST compression for posts.body (replaces runtime migratePostBodyCompressionLZ4)
ALTER TABLE posts ALTER COLUMN body SET COMPRESSION lz4;

-- Foreign keys

ALTER TABLE ONLY delivery_attempts
    ADD CONSTRAINT fk_delivery_attempts_channel FOREIGN KEY (channel_id) REFERENCES delivery_channels(id) ON DELETE CASCADE;
ALTER TABLE ONLY delivery_attempts
    ADD CONSTRAINT fk_delivery_attempts_post FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE;
ALTER TABLE ONLY delivery_attempts
    ADD CONSTRAINT fk_delivery_attempts_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE ONLY delivery_channels
    ADD CONSTRAINT fk_delivery_channels_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE ONLY delivery_history
    ADD CONSTRAINT fk_delivery_history_channel FOREIGN KEY (channel_id) REFERENCES delivery_channels(id) ON DELETE SET NULL;
ALTER TABLE ONLY delivery_history
    ADD CONSTRAINT fk_delivery_history_post FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE SET NULL;
ALTER TABLE ONLY delivery_history
    ADD CONSTRAINT fk_delivery_history_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE ONLY posts
    ADD CONSTRAINT fk_posts_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
