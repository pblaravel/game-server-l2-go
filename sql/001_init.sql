-- PostgreSQL schema converted from l2-unity Java MariaDB installers.
-- Table/column names keep Java DAO compatibility.

CREATE SEQUENCE IF NOT EXISTS object_id_seq START WITH 268435456;

CREATE TABLE IF NOT EXISTS accounts (
    login        VARCHAR(45) PRIMARY KEY,
    password     VARCHAR(256) NOT NULL DEFAULT '',
    last_active  BIGINT NOT NULL DEFAULT 0,
    access_level INTEGER NOT NULL DEFAULT 0,
    last_server  INTEGER NOT NULL DEFAULT 1,
    last_ip      VARCHAR(128)
);

CREATE TABLE IF NOT EXISTS gameservers (
    server_id INTEGER PRIMARY KEY,
    hexid     VARCHAR(50) NOT NULL DEFAULT '',
    host      VARCHAR(50) NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS characters (
    account_name VARCHAR(45),
    obj_id       INTEGER PRIMARY KEY,
    char_name    VARCHAR(35) NOT NULL,
    level        SMALLINT DEFAULT 1,
    maxhp        INTEGER DEFAULT 80,
    curhp        DOUBLE PRECISION DEFAULT 80,
    maxcp        INTEGER DEFAULT 40,
    curcp        DOUBLE PRECISION DEFAULT 40,
    maxmp        INTEGER DEFAULT 30,
    curmp        DOUBLE PRECISION DEFAULT 30,
    face         SMALLINT DEFAULT 0,
    hairstyle    SMALLINT DEFAULT 0,
    haircolor    SMALLINT DEFAULT 0,
    sex          SMALLINT DEFAULT 0,
    heading      INTEGER DEFAULT 0,
    x            INTEGER DEFAULT 0,
    y            INTEGER DEFAULT 0,
    z            INTEGER DEFAULT 0,
    exp          BIGINT DEFAULT 0,
    expbeforedeath BIGINT DEFAULT 0,
    sp           INTEGER NOT NULL DEFAULT 0,
    karma        INTEGER DEFAULT 0,
    pvpkills     INTEGER DEFAULT 0,
    pkkills      INTEGER DEFAULT 0,
    clanid       INTEGER DEFAULT 0,
    race         SMALLINT DEFAULT 0,
    classid      SMALLINT DEFAULT 0,
    base_class   SMALLINT NOT NULL DEFAULT 0,
    deletetime   BIGINT DEFAULT 0,
    title        VARCHAR(16) DEFAULT '',
    rec_have     SMALLINT NOT NULL DEFAULT 0,
    rec_left     SMALLINT NOT NULL DEFAULT 0,
    accesslevel  INTEGER DEFAULT 0,
    online       SMALLINT DEFAULT 0,
    onlinetime   INTEGER DEFAULT 0,
    lastaccess   BIGINT DEFAULT 0,
    wantspeace   SMALLINT DEFAULT 0,
    isin7sdungeon SMALLINT NOT NULL DEFAULT 0,
    punish_level SMALLINT NOT NULL DEFAULT 0,
    punish_timer BIGINT NOT NULL DEFAULT 0,
    power_grade  SMALLINT DEFAULT 0,
    nobless      SMALLINT NOT NULL DEFAULT 0,
    hero         SMALLINT NOT NULL DEFAULT 0,
    subpledge    SMALLINT NOT NULL DEFAULT 0,
    lvl_joined_academy SMALLINT NOT NULL DEFAULT 0,
    apprentice   INTEGER NOT NULL DEFAULT 0,
    sponsor      INTEGER NOT NULL DEFAULT 0,
    varka_ketra_ally SMALLINT NOT NULL DEFAULT 0,
    clan_join_expiry_time BIGINT NOT NULL DEFAULT 0,
    clan_create_expiry_time BIGINT NOT NULL DEFAULT 0,
    death_penalty_level SMALLINT NOT NULL DEFAULT 0,
    str          INTEGER NOT NULL DEFAULT 40,
    dex          INTEGER NOT NULL DEFAULT 30,
    con          INTEGER NOT NULL DEFAULT 43,
    intel        INTEGER NOT NULL DEFAULT 21,
    wit          INTEGER NOT NULL DEFAULT 11,
    men          INTEGER NOT NULL DEFAULT 25
);
CREATE INDEX IF NOT EXISTS idx_characters_account ON characters(account_name);
CREATE INDEX IF NOT EXISTS idx_characters_clan ON characters(clanid);

CREATE TABLE IF NOT EXISTS items (
    owner_id      INTEGER,
    object_id     INTEGER PRIMARY KEY,
    item_id       INTEGER NOT NULL,
    count         INTEGER NOT NULL DEFAULT 0,
    enchant_level SMALLINT NOT NULL DEFAULT 0,
    loc           VARCHAR(16),
    loc_data      INTEGER,
    custom_type1  INTEGER NOT NULL DEFAULT 0,
    custom_type2  INTEGER NOT NULL DEFAULT 0,
    mana_left     INTEGER NOT NULL DEFAULT -1,
    time          BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS character_skills (
    char_obj_id INTEGER NOT NULL,
    skill_id    INTEGER NOT NULL,
    skill_level INTEGER NOT NULL,
    class_index INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (char_obj_id, skill_id, class_index)
);

CREATE TABLE IF NOT EXISTS character_shortcuts (
    char_obj_id INTEGER NOT NULL,
    slot        INTEGER NOT NULL,
    page        INTEGER NOT NULL,
    type        INTEGER,
    shortcut_id INTEGER,
    level       INTEGER,
    class_index INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (char_obj_id, slot, page, class_index)
);

CREATE TABLE IF NOT EXISTS character_quests (
    char_obj_id INTEGER NOT NULL,
    name        VARCHAR(40) NOT NULL,
    var         VARCHAR(20) NOT NULL,
    value       VARCHAR(255),
    class_index INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (char_obj_id, name, var, class_index)
);

CREATE TABLE IF NOT EXISTS character_subclasses (
    char_obj_id INTEGER NOT NULL,
    class_id    INTEGER NOT NULL,
    exp         BIGINT NOT NULL DEFAULT 0,
    sp          INTEGER NOT NULL DEFAULT 0,
    level       INTEGER NOT NULL DEFAULT 40,
    class_index INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (char_obj_id, class_id)
);

CREATE TABLE IF NOT EXISTS character_hennas (
    char_obj_id INTEGER NOT NULL,
    slot        INTEGER NOT NULL,
    symbol_id   INTEGER,
    class_index INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (char_obj_id, slot, class_index)
);

CREATE TABLE IF NOT EXISTS character_macroses (
    char_obj_id INTEGER NOT NULL,
    id          INTEGER NOT NULL,
    icon        INTEGER,
    name        VARCHAR(40),
    descr       VARCHAR(80),
    acronym     VARCHAR(4),
    commands    TEXT,
    PRIMARY KEY (char_obj_id, id)
);

CREATE TABLE IF NOT EXISTS character_recipebook (
    char_obj_id INTEGER NOT NULL,
    id          INTEGER NOT NULL,
    PRIMARY KEY (char_obj_id, id)
);

CREATE TABLE IF NOT EXISTS character_friends (
    char_id   INTEGER NOT NULL,
    friend_id INTEGER NOT NULL,
    PRIMARY KEY (char_id, friend_id)
);

CREATE TABLE IF NOT EXISTS character_recommends (
    char_id INTEGER NOT NULL,
    target_id INTEGER NOT NULL,
    PRIMARY KEY (char_id, target_id)
);

CREATE TABLE IF NOT EXISTS character_raid_points (
    char_id INTEGER NOT NULL,
    boss_id INTEGER NOT NULL,
    points  INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (char_id, boss_id)
);

CREATE TABLE IF NOT EXISTS character_memo (
    char_id INTEGER NOT NULL,
    var     VARCHAR(255) NOT NULL,
    val     TEXT,
    PRIMARY KEY (char_id, var)
);

CREATE TABLE IF NOT EXISTS character_skills_save (
    char_obj_id    INTEGER NOT NULL,
    skill_id       INTEGER NOT NULL,
    skill_level    INTEGER NOT NULL,
    effect_count   INTEGER NOT NULL,
    effect_cur_time INTEGER NOT NULL,
    reuse_delay    INTEGER NOT NULL,
    systime        BIGINT NOT NULL,
    restore_type   INTEGER NOT NULL,
    class_index    INTEGER NOT NULL,
    buff_index     INTEGER NOT NULL,
    PRIMARY KEY (char_obj_id, skill_id, class_index)
);

CREATE TABLE IF NOT EXISTS clan_data (
    clan_id                  INTEGER PRIMARY KEY,
    clan_name                VARCHAR(45),
    clan_level               INTEGER,
    reputation_score         INTEGER NOT NULL DEFAULT 0,
    hascastle                INTEGER,
    ally_id                  INTEGER,
    ally_name                VARCHAR(45),
    leader_id                INTEGER,
    crest_id                 INTEGER,
    crest_large_id           INTEGER,
    ally_crest_id            INTEGER,
    auction_bid_at           INTEGER NOT NULL DEFAULT 0,
    ally_penalty_expiry_time BIGINT NOT NULL DEFAULT 0,
    ally_penalty_type        SMALLINT NOT NULL DEFAULT 0,
    char_penalty_expiry_time BIGINT NOT NULL DEFAULT 0,
    dissolving_expiry_time   BIGINT NOT NULL DEFAULT 0,
    new_leader_id            INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS clan_privs (
    clan_id   INTEGER NOT NULL,
    rank      INTEGER NOT NULL,
    party     INTEGER NOT NULL,
    privs     INTEGER NOT NULL,
    PRIMARY KEY (clan_id, rank, party)
);

CREATE TABLE IF NOT EXISTS clan_skills (
    clan_id     INTEGER NOT NULL,
    skill_id    INTEGER NOT NULL,
    skill_level INTEGER NOT NULL,
    skill_name  VARCHAR(26),
    PRIMARY KEY (clan_id, skill_id)
);

CREATE TABLE IF NOT EXISTS clan_subpledges (
    clan_id     INTEGER NOT NULL,
    sub_pledge_id INTEGER NOT NULL,
    name        VARCHAR(45),
    leader_id   INTEGER,
    PRIMARY KEY (clan_id, sub_pledge_id)
);

CREATE TABLE IF NOT EXISTS clan_wars (
    clan1       INTEGER NOT NULL,
    clan2       INTEGER NOT NULL,
    expiry_time BIGINT,
    PRIMARY KEY (clan1, clan2)
);

CREATE TABLE IF NOT EXISTS castle (
    id            INTEGER PRIMARY KEY,
    name          VARCHAR(25) NOT NULL,
    tax_percent   INTEGER NOT NULL DEFAULT 0,
    treasury      BIGINT NOT NULL DEFAULT 0,
    siege_date    BIGINT NOT NULL DEFAULT 0,
    reg_time_end  BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS siege_clans (
    castle_id INTEGER NOT NULL,
    clan_id   INTEGER NOT NULL,
    type      INTEGER DEFAULT 0,
    PRIMARY KEY (castle_id, clan_id)
);

CREATE TABLE IF NOT EXISTS clanhall (
    id            INTEGER PRIMARY KEY,
    name          VARCHAR(40) NOT NULL,
    owner_id      INTEGER NOT NULL DEFAULT 0,
    lease         INTEGER NOT NULL DEFAULT 0,
    desc          TEXT,
    location      VARCHAR(80),
    paid_until    BIGINT NOT NULL DEFAULT 0,
    grade         INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS items_on_ground (
    object_id INTEGER PRIMARY KEY,
    item_id   INTEGER,
    count     INTEGER,
    enchant   INTEGER,
    x         INTEGER,
    y         INTEGER,
    z         INTEGER,
    drop_time BIGINT
);

CREATE TABLE IF NOT EXISTS pets (
    item_obj_id INTEGER PRIMARY KEY,
    name        VARCHAR(16),
    level       INTEGER,
    curhp       INTEGER,
    curmp       INTEGER,
    exp         BIGINT,
    sp          INTEGER,
    fed         INTEGER
);

CREATE TABLE IF NOT EXISTS augmentations (
    item_oid   INTEGER PRIMARY KEY,
    attributes INTEGER,
    skill      INTEGER,
    level      INTEGER
);

CREATE TABLE IF NOT EXISTS seven_signs (
    char_obj_id      INTEGER PRIMARY KEY,
    cabal            VARCHAR(4) NOT NULL,
    seal             INTEGER NOT NULL,
    red_stones       INTEGER NOT NULL DEFAULT 0,
    green_stones     INTEGER NOT NULL DEFAULT 0,
    blue_stones      INTEGER NOT NULL DEFAULT 0,
    ancient_adena_amount INTEGER NOT NULL DEFAULT 0,
    contribution_score INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS seven_signs_status (
    id                    INTEGER PRIMARY KEY,
    current_cycle         INTEGER NOT NULL,
    dawn_stone_score      INTEGER NOT NULL,
    dusk_stone_score      INTEGER NOT NULL,
    current_status        INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS olympiad_nobles (
    char_id          INTEGER PRIMARY KEY,
    class_id         INTEGER NOT NULL,
    olympiad_points  INTEGER NOT NULL DEFAULT 0,
    competitions_done INTEGER NOT NULL DEFAULT 0,
    competitions_won  INTEGER NOT NULL DEFAULT 0,
    competitions_lost INTEGER NOT NULL DEFAULT 0,
    competitions_drawn INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS heroes (
    char_id   INTEGER PRIMARY KEY,
    class_id  INTEGER NOT NULL,
    count     INTEGER NOT NULL DEFAULT 0,
    played    INTEGER NOT NULL DEFAULT 0,
    claimed   BOOLEAN NOT NULL DEFAULT FALSE,
    message   VARCHAR(300) NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS bookmarks (
    id      SERIAL PRIMARY KEY,
    obj_Id  INTEGER NOT NULL,
    name    VARCHAR(32),
    x       INTEGER,
    y       INTEGER,
    z       INTEGER
);

CREATE TABLE IF NOT EXISTS server_memo (
    var VARCHAR(255) PRIMARY KEY,
    val TEXT
);

CREATE TABLE IF NOT EXISTS spawn_data (
    id        SERIAL PRIMARY KEY,
    npc_id    INTEGER NOT NULL,
    x         INTEGER NOT NULL,
    y         INTEGER NOT NULL,
    z         INTEGER NOT NULL,
    heading   INTEGER NOT NULL DEFAULT 0,
    respawn   INTEGER NOT NULL DEFAULT 60
);
