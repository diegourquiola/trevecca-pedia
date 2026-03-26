-- Migration: Add moderator role to the roles table
-- This migration adds the 'moderator' role to enable future moderation features

INSERT INTO roles (name) VALUES ('moderator')
ON CONFLICT (name) DO NOTHING;
