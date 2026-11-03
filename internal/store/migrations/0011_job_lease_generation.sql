-- Attempts may be decremented by throttling. Fencing therefore needs a
-- separate monotonically increasing generation that is never decremented.
ALTER TABLE jobs ADD COLUMN lease_generation bigint NOT NULL DEFAULT 0;
