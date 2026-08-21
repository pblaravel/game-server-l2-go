-- Seed data required for a blank L2 Unity server.
-- Values match Java INSERT IGNORE from gameserver/db/sql/*.sql
-- plus Interlude newbie-zone datapack so EnterWorld is not empty.

INSERT INTO castle (id, name, current_tax_percent, next_tax_percent, treasury, tax_revenue, seed_income, siege_date, reg_time_over, certificates) VALUES
(1, 'Gludio',   15, 15, 0, 0, 0, 0, TRUE, 300),
(2, 'Dion',     15, 15, 0, 0, 0, 0, TRUE, 300),
(3, 'Giran',    15, 15, 0, 0, 0, 0, TRUE, 300),
(4, 'Oren',     15, 15, 0, 0, 0, 0, TRUE, 300),
(5, 'Aden',     15, 15, 0, 0, 0, 0, TRUE, 300),
(6, 'Innadril', 15, 15, 0, 0, 0, 0, TRUE, 300),
(7, 'Goddard',  15, 15, 0, 0, 0, 0, TRUE, 300),
(8, 'Rune',     15, 15, 0, 0, 0, 0, TRUE, 300),
(9, 'Schuttgart', 15, 15, 0, 0, 0, 0, TRUE, 300)
ON CONFLICT (id) DO NOTHING;

INSERT INTO clanhall (id, name, owner_id, paid_until, paid, seller_bid, seller_name, seller_clan_name, end_date, location, grade) VALUES
(21, 'Fortress of Resistance', 0, 0, 0, 0, '', '', 0, 'Dion', 3),
(22, 'Moonstone Hall', 0, 0, 0, 0, '', '', 0, 'Gludio', 2),
(23, 'Onyx Hall', 0, 0, 0, 0, '', '', 0, 'Gludio', 2),
(24, 'Topaz Hall', 0, 0, 0, 0, '', '', 0, 'Gludio', 2),
(25, 'Ruby Hall', 0, 0, 0, 0, '', '', 0, 'Gludio', 2),
(26, 'Crystal Hall', 0, 0, 0, 0, '', '', 0, 'Gludin', 2),
(27, 'Onyx Hall', 0, 0, 0, 0, '', '', 0, 'Gludin', 2),
(28, 'Sapphire Hall', 0, 0, 0, 0, '', '', 0, 'Gludin', 2),
(29, 'Moonstone Hall', 0, 0, 0, 0, '', '', 0, 'Gludin', 2),
(30, 'Emerald Hall', 0, 0, 0, 0, '', '', 0, 'Gludin', 2),
(31, 'The Atramental Barracks', 0, 0, 0, 0, '', '', 0, 'Dion', 2),
(32, 'The Scarlet Barracks', 0, 0, 0, 0, '', '', 0, 'Dion', 2),
(33, 'The Viridian Barracks', 0, 0, 0, 0, '', '', 0, 'Dion', 2),
(34, 'Devastated Castle', 0, 0, 0, 0, '', '', 0, 'Aden', 3),
(35, 'Bandit Stronghold', 0, 0, 0, 0, '', '', 0, 'Oren', 3),
(36, 'The Golden Chamber', 0, 0, 0, 0, '', '', 0, 'Aden', 2),
(37, 'The Silver Chamber', 0, 0, 0, 0, '', '', 0, 'Aden', 2),
(38, 'The Mithril Chamber', 0, 0, 0, 0, '', '', 0, 'Aden', 2),
(39, 'Silver Manor', 0, 0, 0, 0, '', '', 0, 'Aden', 2),
(40, 'Gold Manor', 0, 0, 0, 0, '', '', 0, 'Aden', 2),
(41, 'The Bronze Chamber', 0, 0, 0, 0, '', '', 0, 'Aden', 2),
(42, 'The Golden Chamber', 0, 0, 0, 0, '', '', 0, 'Giran', 2),
(43, 'The Silver Chamber', 0, 0, 0, 0, '', '', 0, 'Giran', 2),
(44, 'The Mithril Chamber', 0, 0, 0, 0, '', '', 0, 'Giran', 2),
(45, 'The Bronze Chamber', 0, 0, 0, 0, '', '', 0, 'Giran', 2),
(46, 'Silver Manor', 0, 0, 0, 0, '', '', 0, 'Giran', 2),
(47, 'Moonstone Hall', 0, 0, 0, 0, '', '', 0, 'Goddard', 2),
(48, 'Onyx Hall', 0, 0, 0, 0, '', '', 0, 'Goddard', 2),
(49, 'Emerald Hall', 0, 0, 0, 0, '', '', 0, 'Goddard', 2),
(50, 'Sapphire Hall', 0, 0, 0, 0, '', '', 0, 'Goddard', 2),
(51, 'Mont Chamber', 0, 0, 0, 0, '', '', 0, 'Rune', 2),
(52, 'Astaire Chamber', 0, 0, 0, 0, '', '', 0, 'Rune', 2),
(53, 'Aria Chamber', 0, 0, 0, 0, '', '', 0, 'Rune', 2),
(54, 'Yiana Chamber', 0, 0, 0, 0, '', '', 0, 'Rune', 2),
(55, 'Roien Chamber', 0, 0, 0, 0, '', '', 0, 'Rune', 2),
(56, 'Luna Chamber', 0, 0, 0, 0, '', '', 0, 'Rune', 2),
(57, 'Traban Chamber', 0, 0, 0, 0, '', '', 0, 'Rune', 2),
(58, 'Eisen Hall', 0, 0, 0, 0, '', '', 0, 'Schuttgart', 2),
(59, 'Heavy Metal Hall', 0, 0, 0, 0, '', '', 0, 'Schuttgart', 2),
(60, 'Molten Ore Hall', 0, 0, 0, 0, '', '', 0, 'Schuttgart', 2),
(61, 'Titan Hall', 0, 0, 0, 0, '', '', 0, 'Schuttgart', 2),
(62, 'Rainbow Springs', 0, 0, 0, 0, '', '', 0, 'Goddard', 3),
(63, 'Beast Farm', 0, 0, 0, 0, '', '', 0, 'Rune', 3),
(64, 'Fortress of the Dead', 0, 0, 0, 0, '', '', 0, 'Rune', 3)
ON CONFLICT (id) DO NOTHING;

INSERT INTO seven_signs_status (id, current_cycle, festival_cycle, active_period, date, previous_winner,
    dawn_stone_score, dawn_festival_score, dusk_stone_score, dusk_festival_score,
    avarice_owner, gnosis_owner, strife_owner,
    avarice_dawn_score, gnosis_dawn_score, strife_dawn_score,
    avarice_dusk_score, gnosis_dusk_score, strife_dusk_score,
    accumulated_bonus0, accumulated_bonus1, accumulated_bonus2, accumulated_bonus3, accumulated_bonus4)
VALUES (0, 1, 1, 'COMPETITION', 0, 'NORMAL', 0, 0, 0, 0, 'NORMAL', 'NORMAL', 'NORMAL', 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
ON CONFLICT (id) DO NOTHING;

INSERT INTO seven_signs_festival (festival_id, cabal, cycle, date, score, members) VALUES
(0, 'DAWN', 1, 0, 0, ''),
(1, 'DAWN', 1, 0, 0, ''),
(2, 'DAWN', 1, 0, 0, ''),
(3, 'DAWN', 1, 0, 0, ''),
(4, 'DAWN', 1, 0, 0, ''),
(0, 'DUSK', 1, 0, 0, ''),
(1, 'DUSK', 1, 0, 0, ''),
(2, 'DUSK', 1, 0, 0, ''),
(3, 'DUSK', 1, 0, 0, ''),
(4, 'DUSK', 1, 0, 0, '')
ON CONFLICT (festival_id, cabal, cycle) DO NOTHING;

INSERT INTO mdt_bets (lane_id, bet) VALUES
(1, 0), (2, 0), (3, 0), (4, 0), (5, 0), (6, 0), (7, 0), (8, 0)
ON CONFLICT (lane_id) DO NOTHING;

INSERT INTO server_memo (var, val) VALUES
('server_crash', 'false')
ON CONFLICT (var) DO NOTHING;

INSERT INTO class_templates (class_id, name, race, is_mage) VALUES
(0,  'Human Fighter', 0, FALSE),
(10, 'Human Mystic', 0, TRUE),
(18, 'Elven Fighter', 1, FALSE),
(25, 'Elven Mystic', 1, TRUE),
(31, 'Dark Fighter', 2, FALSE),
(38, 'Dark Mystic', 2, TRUE),
(44, 'Orc Fighter', 3, FALSE),
(49, 'Orc Mystic', 3, TRUE),
(53, 'Dwarven Fighter', 4, FALSE)
ON CONFLICT (class_id) DO NOTHING;

INSERT INTO npc_templates (npc_id, name, title, level, maxhp, maxmp, is_attackable) VALUES
(30006, 'Roxxy', 'Gatekeeper', 70, 10000, 0, FALSE),
(30001, 'Lector', 'Weapon Merchant', 70, 8000, 0, FALSE),
(30002, 'Jackson', 'Armor Merchant', 70, 8000, 0, FALSE),
(30003, 'Silvia', 'Accessory Merchant', 70, 8000, 0, FALSE),
(30008, 'Miner', 'Newbie Helper', 70, 8000, 0, FALSE),
(30009, 'Newbie Guide', '', 70, 8000, 0, FALSE),
(30031, 'Captain Bathis', 'Guard Captain', 70, 12000, 0, FALSE),
(30039, 'Guard Gilbert', 'Guard', 70, 9000, 0, FALSE),
(30040, 'Guard Leon', 'Guard', 70, 9000, 0, FALSE),
(30041, 'Guard Arnold', 'Guard', 70, 9000, 0, FALSE),
(30042, 'Guard Abellos', 'Guard', 70, 9000, 0, FALSE),
(30048, 'Darin', '', 70, 5000, 0, FALSE),
(30049, 'Bonnie', '', 70, 5000, 0, FALSE),
(30050, 'Elias', 'Warehouse Keeper', 70, 8000, 0, FALSE),
(30146, 'Mirabel', 'Gatekeeper', 70, 10000, 0, FALSE),
(30134, 'Jasmine', 'Gatekeeper', 70, 10000, 0, FALSE),
(30576, 'Tataru Zu Hestui', 'Gatekeeper', 70, 10000, 0, FALSE),
(30540, 'Wirphy', 'Gatekeeper', 70, 10000, 0, FALSE),
(20120, 'Wolf', '', 4, 80, 0, TRUE),
(20481, 'Elder Keltir', '', 3, 60, 0, TRUE),
(20006, 'Orc Archer', '', 8, 160, 0, TRUE),
(20001, 'Gremlin', '', 1, 40, 0, TRUE),
(20003, 'Goblin', '', 5, 90, 0, TRUE)
ON CONFLICT (npc_id) DO NOTHING;

INSERT INTO player_levels (level, exp) VALUES
(1, 0),
(2, 68), (3, 363), (4, 1168), (5, 2884), (6, 6038), (7, 11287), (8, 19423), (9, 31378), (10, 48229),
(11, 71201), (12, 101676), (13, 141192), (14, 191452), (15, 254327), (16, 331864), (17, 426284), (18, 539995), (19, 675590), (20, 835854),
(21, 1023775), (22, 1242536), (23, 1495531), (24, 1786365), (25, 2118860), (26, 2497059), (27, 2925229), (28, 3407873), (29, 3949727), (30, 4555766),
(31, 5231213), (32, 5981539), (33, 6812472), (34, 7729999), (35, 8740372), (36, 9850111), (37, 11066012), (38, 12395149), (39, 13844879), (40, 15422851),
(41, 17137002), (42, 18995573), (43, 21007103), (44, 23180442), (45, 25524751), (46, 28049509), (47, 30764519), (48, 33679907), (49, 36806133), (50, 40153995),
(51, 45524865), (52, 51262204), (53, 57383682), (54, 63907585), (55, 70852742), (56, 80700339), (57, 91162131), (58, 102265326), (59, 114038008), (60, 126509030),
(61, 146307211), (62, 167243291), (63, 189363788), (64, 212716741), (65, 237351413), (66, 271973532), (67, 308441375), (68, 346825235), (69, 387197529), (70, 429632402),
(71, 474205751), (72, 532692055), (73, 606319094), (74, 696376867), (75, 804219972), (76, 931275828), (77, 1151275834), (78, 1511275834), (79, 2099275834), (80, 4200000000),
(81, 6299994999)
ON CONFLICT (level) DO NOTHING;

INSERT INTO npc_spawns (npc_id, x, y, z, heading, respawn) VALUES
(30006, -71338, 258271, -3104, 0, 60),
(30001, -71424, 258191, -3104, 8192, 60),
(30002, -71280, 258191, -3104, 24576, 60),
(30003, -71338, 258080, -3104, 16384, 60),
(30008, -84057, 243220, -3728, 0, 60),
(30009, -71380, 258400, -3104, 0, 60),
(30031, -72224, 257788, -3120, 0, 60),
(30039, -71880, 257988, -3120, 0, 60),
(30040, -72580, 257988, -3120, 32768, 60),
(30050, -71080, 258271, -3104, 0, 60),
(30146, 45873, 49688, -3056, 0, 60),
(30134, 9690, 15537, -4570, 0, 60),
(30576, -45251, -112400, -240, 0, 60),
(30540, 115072, -178176, -880, 0, 60),
(20120, -74720, 245280, -3616, 0, 20),
(20120, -75100, 244900, -3616, 8192, 20),
(20120, -74300, 245600, -3616, 16384, 20),
(20481, -75600, 246200, -3648, 0, 20),
(20481, -76000, 245800, -3648, 24576, 20),
(20006, -77000, 247000, -3760, 0, 30),
(20001, -73500, 244500, -3552, 0, 15),
(20003, -74000, 246800, -3680, 0, 20);
