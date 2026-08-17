-- Skill / class-tree tables for databases that already applied 001_init.
-- Data rows are upserted from data/xml at gameserver start (Java SkillTable + PlayerData).

CREATE TABLE IF NOT EXISTS skill_templates (
    skill_id      INTEGER NOT NULL,
    skill_level   INTEGER NOT NULL,
    name          VARCHAR(80) NOT NULL DEFAULT '',
    operate_type  VARCHAR(16) NOT NULL DEFAULT 'ACTIVE',
    skill_type    VARCHAR(32) NOT NULL DEFAULT '',
    is_magic      BOOLEAN NOT NULL DEFAULT FALSE,
    mp_consume    INTEGER NOT NULL DEFAULT 0,
    hit_time      INTEGER NOT NULL DEFAULT 0,
    reuse_delay   INTEGER NOT NULL DEFAULT 0,
    cool_time     INTEGER NOT NULL DEFAULT 0,
    power         DOUBLE PRECISION NOT NULL DEFAULT 0,
    magic_lvl     INTEGER NOT NULL DEFAULT 0,
    target_type   VARCHAR(32) NOT NULL DEFAULT '',
    cast_range    INTEGER NOT NULL DEFAULT 0,
    effect_range  INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (skill_id, skill_level)
);

CREATE TABLE IF NOT EXISTS class_skills (
    class_id    INTEGER NOT NULL,
    skill_id    INTEGER NOT NULL,
    skill_level INTEGER NOT NULL,
    cost        INTEGER NOT NULL DEFAULT 0,
    min_lvl     INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY (class_id, skill_id, skill_level)
);
