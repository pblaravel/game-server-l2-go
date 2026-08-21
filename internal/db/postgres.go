package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pblaravel/game-server-l2-go/internal/gameserver"
	"github.com/pblaravel/game-server-l2-go/internal/loginserver"
)

type Pool struct {
	p *pgxpool.Pool
}

func Connect(ctx context.Context, url string) (*Pool, error) {
	p, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := p.Ping(ctx); err != nil {
		p.Close()
		return nil, err
	}
	return &Pool{p: p}, nil
}

// ConnectWithRetry pings Postgres until wait elapses or ctx is cancelled.
func ConnectWithRetry(ctx context.Context, url string, wait time.Duration) (*Pool, error) {
	if url == "" {
		return nil, errors.New("empty database url")
	}
	deadline := time.Now().Add(wait)
	var last error
	for {
		attempt, cancel := context.WithTimeout(ctx, 3*time.Second)
		p, err := Connect(attempt, url)
		cancel()
		if err == nil {
			return p, nil
		}
		last = err
		if ctx.Err() != nil {
			return nil, last
		}
		if time.Now().After(deadline) {
			return nil, last
		}
		select {
		case <-ctx.Done():
			return nil, last
		case <-time.After(time.Second):
		}
	}
}

func (p *Pool) Close() {
	if p != nil && p.p != nil {
		p.p.Close()
	}
}

func (p *Pool) Ping(ctx context.Context) error { return p.p.Ping(ctx) }

type AccountRepo struct{ p *Pool }

func NewAccountRepo(p *Pool) *AccountRepo { return &AccountRepo{p: p} }

func (r *AccountRepo) GetAccountInfo(ctx context.Context, login string) (*loginserver.AccountInfo, error) {
	row := r.p.p.QueryRow(ctx, `SELECT login, password, access_level, last_server, COALESCE(last_ip,''), last_active FROM accounts WHERE login=$1`, login)
	var a loginserver.AccountInfo
	if err := row.Scan(&a.Login, &a.PassHash, &a.AccessLevel, &a.LastServer, &a.LastIP, &a.LastActive); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *AccountRepo) CreateAccount(ctx context.Context, info loginserver.AccountInfo) error {
	_, err := r.p.p.Exec(ctx, `INSERT INTO accounts (login, password, access_level, last_server, last_ip, last_active) VALUES ($1,$2,$3,$4,$5,$6)`,
		info.Login, info.PassHash, info.AccessLevel, info.LastServer, info.LastIP, info.LastActive)
	return err
}

func (r *AccountRepo) UpdateAccount(ctx context.Context, info loginserver.AccountInfo) error {
	_, err := r.p.p.Exec(ctx, `UPDATE accounts SET password=$2, access_level=$3, last_server=$4, last_ip=$5, last_active=$6 WHERE login=$1`,
		info.Login, info.PassHash, info.AccessLevel, info.LastServer, info.LastIP, info.LastActive)
	return err
}

func (r *AccountRepo) UpdateAccountLastServer(ctx context.Context, account string, serverID int) error {
	_, err := r.p.p.Exec(ctx, `UPDATE accounts SET last_server=$2 WHERE login=$1`, account, serverID)
	return err
}

type GameServerRepo struct{ p *Pool }

func NewGameServerRepo(p *Pool) *GameServerRepo { return &GameServerRepo{p: p} }

func (r *GameServerRepo) GetAllGameServers(ctx context.Context) ([]loginserver.GameServerRow, error) {
	rows, err := r.p.p.Query(ctx, `SELECT server_id, hexid, host FROM gameservers`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []loginserver.GameServerRow
	for rows.Next() {
		var g loginserver.GameServerRow
		if err := rows.Scan(&g.ServerID, &g.HexID, &g.Host); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *GameServerRepo) AddGameServer(ctx context.Context, gs loginserver.GameServerRow) error {
	_, err := r.p.p.Exec(ctx, `INSERT INTO gameservers (server_id, hexid, host) VALUES ($1,$2,$3)
		ON CONFLICT (server_id) DO UPDATE SET hexid=EXCLUDED.hexid, host=EXCLUDED.host`, gs.ServerID, gs.HexID, gs.Host)
	return err
}

type CharacterRepo struct{ p *Pool }

func NewCharacterRepo(p *Pool) *CharacterRepo { return &CharacterRepo{p: p} }

func (r *CharacterRepo) ListByAccount(ctx context.Context, account string) ([]*gameserver.Character, error) {
	rows, err := r.p.p.Query(ctx, `SELECT obj_id, account_name, char_name, COALESCE(title,''), level, maxhp, curhp, maxmp, curmp, maxcp, curcp,
		face, hairstyle, haircolor, sex, heading, x, y, z, exp, sp, karma, pvpkills, pkkills, COALESCE(clanid,0), race, classid, base_class,
		COALESCE(deletetime,0), accesslevel, COALESCE(lastaccess,0), str, dex, con, intel, wit, men
		FROM characters WHERE account_name=$1`, account)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*gameserver.Character
	for rows.Next() {
		ch := &gameserver.Character{}
		if err := scanCharacter(rows, ch); err != nil {
			return nil, err
		}
		out = append(out, ch)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, ch := range out {
		if err := r.loadOwned(ctx, ch); err != nil {
			return nil, err
		}
		gameserver.RestoreCharacter(ch)
	}
	return out, nil
}

func (r *CharacterRepo) GetByObjectID(ctx context.Context, id int32) (*gameserver.Character, error) {
	row := r.p.p.QueryRow(ctx, `SELECT obj_id, account_name, char_name, COALESCE(title,''), level, maxhp, curhp, maxmp, curmp, maxcp, curcp,
		face, hairstyle, haircolor, sex, heading, x, y, z, exp, sp, karma, pvpkills, pkkills, COALESCE(clanid,0), race, classid, base_class,
		COALESCE(deletetime,0), accesslevel, COALESCE(lastaccess,0), str, dex, con, intel, wit, men
		FROM characters WHERE obj_id=$1`, id)
	ch := &gameserver.Character{}
	if err := scanCharacter(row, ch); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if err := r.loadOwned(ctx, ch); err != nil {
		return nil, err
	}
	return gameserver.RestoreCharacter(ch), nil
}

func (r *CharacterRepo) GetObjectIDByName(ctx context.Context, name string) (int32, error) {
	var id int32
	err := r.p.p.QueryRow(ctx, `SELECT obj_id FROM characters WHERE char_name=$1`, name).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return id, nil
}

func (r *CharacterRepo) CountByAccount(ctx context.Context, account string) (int, error) {
	var n int
	err := r.p.p.QueryRow(ctx, `SELECT COUNT(*) FROM characters WHERE account_name=$1`, account).Scan(&n)
	return n, err
}

func (r *CharacterRepo) Create(ctx context.Context, ch *gameserver.Character) error {
	tx, err := r.p.p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `INSERT INTO characters (
		obj_id, account_name, char_name, title, level, maxhp, curhp, maxmp, curmp, maxcp, curcp,
		face, hairstyle, haircolor, sex, heading, x, y, z, exp, sp, karma, pvpkills, pkkills, clanid, race, classid, base_class,
		deletetime, accesslevel, lastaccess, str, dex, con, intel, wit, men)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37)`,
		ch.ObjectID, ch.Account, ch.Name, ch.Title, ch.Level, ch.MaxHP, ch.CurHP, ch.MaxMP, ch.CurMP, ch.MaxCP, ch.CurCP,
		ch.Face, ch.HairStyle, ch.HairColor, ch.Sex, ch.Heading, ch.X, ch.Y, ch.Z, ch.Exp, ch.SP, ch.Karma, ch.PvPKills, ch.PKKills, ch.ClanID, ch.Race, ch.ClassID, ch.BaseClass,
		ch.DeleteTime, ch.AccessLevel, ch.LastAccess, ch.STR, ch.DEX, ch.CON, ch.INT, ch.WIT, ch.MEN)
	if err != nil {
		return err
	}
	if err := saveOwned(ctx, tx, ch); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *CharacterRepo) Update(ctx context.Context, ch *gameserver.Character) error {
	tx, err := r.p.p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `UPDATE characters SET title=$2, level=$3, maxhp=$4, curhp=$5, maxmp=$6, curmp=$7, maxcp=$8, curcp=$9,
		heading=$10, x=$11, y=$12, z=$13, exp=$14, sp=$15, karma=$16, pvpkills=$17, pkkills=$18, lastaccess=$19, classid=$20
		WHERE obj_id=$1`,
		ch.ObjectID, ch.Title, ch.Level, ch.MaxHP, ch.CurHP, ch.MaxMP, ch.CurMP, ch.MaxCP, ch.CurCP,
		ch.Heading, ch.X, ch.Y, ch.Z, ch.Exp, ch.SP, ch.Karma, ch.PvPKills, ch.PKKills, ch.LastAccess, ch.ClassID)
	if err != nil {
		return err
	}
	if err := saveOwned(ctx, tx, ch); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *CharacterRepo) Delete(ctx context.Context, id int32) error {
	tx, err := r.p.p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM items WHERE owner_id=$1`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM character_skills WHERE char_obj_id=$1`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM character_shortcuts WHERE char_obj_id=$1`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM character_friends WHERE char_id=$1`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM characters WHERE obj_id=$1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *CharacterRepo) NextObjectID(ctx context.Context) (int32, error) {
	var id int32
	err := r.p.p.QueryRow(ctx, `SELECT nextval('object_id_seq')`).Scan(&id)
	return id, err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanCharacter(s scanner, ch *gameserver.Character) error {
	return s.Scan(&ch.ObjectID, &ch.Account, &ch.Name, &ch.Title, &ch.Level, &ch.MaxHP, &ch.CurHP, &ch.MaxMP, &ch.CurMP, &ch.MaxCP, &ch.CurCP,
		&ch.Face, &ch.HairStyle, &ch.HairColor, &ch.Sex, &ch.Heading, &ch.X, &ch.Y, &ch.Z, &ch.Exp, &ch.SP, &ch.Karma, &ch.PvPKills, &ch.PKKills, &ch.ClanID, &ch.Race, &ch.ClassID, &ch.BaseClass,
		&ch.DeleteTime, &ch.AccessLevel, &ch.LastAccess, &ch.STR, &ch.DEX, &ch.CON, &ch.INT, &ch.WIT, &ch.MEN)
}

type queryExecer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (r *CharacterRepo) loadOwned(ctx context.Context, ch *gameserver.Character) error {
	if err := loadItems(ctx, r.p.p, ch); err != nil {
		return err
	}
	if err := loadSkills(ctx, r.p.p, ch); err != nil {
		return err
	}
	if err := loadShortcuts(ctx, r.p.p, ch); err != nil {
		return err
	}
	return loadFriends(ctx, r.p.p, ch)
}

func saveOwned(ctx context.Context, q queryExecer, ch *gameserver.Character) error {
	if _, err := q.Exec(ctx, `DELETE FROM items WHERE owner_id=$1`, ch.ObjectID); err != nil {
		return err
	}
	if _, err := q.Exec(ctx, `DELETE FROM character_skills WHERE char_obj_id=$1`, ch.ObjectID); err != nil {
		return err
	}
	if _, err := q.Exec(ctx, `DELETE FROM character_shortcuts WHERE char_obj_id=$1`, ch.ObjectID); err != nil {
		return err
	}
	if _, err := q.Exec(ctx, `DELETE FROM character_friends WHERE char_id=$1`, ch.ObjectID); err != nil {
		return err
	}
	saveItem := func(it gameserver.Item, fallbackLoc string) error {
		loc := it.Loc
		if loc == "" {
			loc = fallbackLoc
		}
		if it.Equipped {
			loc = "PAPERDOLL"
		}
		_, err := q.Exec(ctx, `INSERT INTO items (owner_id, object_id, item_id, count, enchant_level, loc, loc_data, custom_type1, custom_type2, mana_left)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			ch.ObjectID, it.ObjectID, it.ItemID, it.Count, it.Enchant, loc, it.Slot, it.Custom1, it.Custom2, it.ManaLeft)
		return err
	}
	for _, it := range ch.Items {
		if err := saveItem(it, "INVENTORY"); err != nil {
			return err
		}
	}
	for _, it := range ch.Warehouse {
		it.Loc = "WAREHOUSE"
		if err := saveItem(it, "WAREHOUSE"); err != nil {
			return err
		}
	}
	for _, sk := range ch.Skills {
		if _, err := q.Exec(ctx, `INSERT INTO character_skills (char_obj_id, skill_id, skill_level, class_index)
			VALUES ($1,$2,$3,0)`, ch.ObjectID, sk.ID, sk.Level); err != nil {
			return err
		}
	}
	for _, sc := range ch.Shortcuts {
		if _, err := q.Exec(ctx, `INSERT INTO character_shortcuts (char_obj_id, slot, page, type, shortcut_id, level, class_index)
			VALUES ($1,$2,$3,$4,$5,$6,0)`, ch.ObjectID, sc.Slot, sc.Page, sc.Type, sc.ID, sc.Level); err != nil {
			return err
		}
	}
	for _, f := range ch.Friends {
		if _, err := q.Exec(ctx, `INSERT INTO character_friends (char_id, friend_id) VALUES ($1,$2)`,
			ch.ObjectID, f.ObjectID); err != nil {
			return err
		}
	}
	return nil
}

func loadItems(ctx context.Context, q queryExecer, ch *gameserver.Character) error {
	rows, err := q.Query(ctx, `SELECT object_id, item_id, count, enchant_level, COALESCE(loc,''), COALESCE(loc_data,0), custom_type1, custom_type2, mana_left
		FROM items WHERE owner_id=$1 ORDER BY loc_data, object_id`, ch.ObjectID)
	if err != nil {
		return err
	}
	defer rows.Close()
	ch.Items = ch.Items[:0]
	ch.Warehouse = ch.Warehouse[:0]
	for rows.Next() {
		var it gameserver.Item
		if err := rows.Scan(&it.ObjectID, &it.ItemID, &it.Count, &it.Enchant, &it.Loc, &it.Slot, &it.Custom1, &it.Custom2, &it.ManaLeft); err != nil {
			return err
		}
		it.Equipped = it.Loc == "PAPERDOLL"
		it.BodyPart = gameserver.BodyPartForItem(it.ItemID)
		if it.Loc == "WAREHOUSE" {
			ch.Warehouse = append(ch.Warehouse, it)
			continue
		}
		if it.Equipped {
			gameserver.EquipPaperdoll(ch, it.BodyPart, it.ItemID, it.ObjectID)
		}
		ch.Items = append(ch.Items, it)
	}
	return rows.Err()
}

func loadFriends(ctx context.Context, q queryExecer, ch *gameserver.Character) error {
	rows, err := q.Query(ctx, `SELECT f.friend_id, COALESCE(c.char_name, '')
		FROM character_friends f
		LEFT JOIN characters c ON c.obj_id = f.friend_id
		WHERE f.char_id=$1`, ch.ObjectID)
	if err != nil {
		return err
	}
	defer rows.Close()
	ch.Friends = ch.Friends[:0]
	for rows.Next() {
		var f gameserver.Friend
		if err := rows.Scan(&f.ObjectID, &f.Name); err != nil {
			return err
		}
		ch.Friends = append(ch.Friends, f)
	}
	return rows.Err()
}

func loadSkills(ctx context.Context, q queryExecer, ch *gameserver.Character) error {
	rows, err := q.Query(ctx, `SELECT skill_id, skill_level FROM character_skills WHERE char_obj_id=$1`, ch.ObjectID)
	if err != nil {
		return err
	}
	defer rows.Close()
	ch.Skills = ch.Skills[:0]
	for rows.Next() {
		var sk gameserver.Skill
		if err := rows.Scan(&sk.ID, &sk.Level); err != nil {
			return err
		}
		sk.Passive = sk.ID == 194 || sk.ID == 1320 || sk.ID == 1321 || sk.ID == 226
		if tpl := gameserver.GetSkill(sk.ID, sk.Level); tpl != nil {
			sk.Passive = tpl.IsPassive()
		}
		ch.Skills = append(ch.Skills, sk)
	}
	return rows.Err()
}

func loadShortcuts(ctx context.Context, q queryExecer, ch *gameserver.Character) error {
	rows, err := q.Query(ctx, `SELECT slot, page, type, shortcut_id, COALESCE(level,0) FROM character_shortcuts WHERE char_obj_id=$1`, ch.ObjectID)
	if err != nil {
		return err
	}
	defer rows.Close()
	ch.Shortcuts = ch.Shortcuts[:0]
	for rows.Next() {
		var sc gameserver.Shortcut
		if err := rows.Scan(&sc.Slot, &sc.Page, &sc.Type, &sc.ID, &sc.Level); err != nil {
			return err
		}
		sc.CharacterType = 1
		ch.Shortcuts = append(ch.Shortcuts, sc)
	}
	return rows.Err()
}

type NpcRepo struct{ p *Pool }

func NewNpcRepo(p *Pool) *NpcRepo { return &NpcRepo{p: p} }

func (r *NpcRepo) ListSpawns(ctx context.Context) ([]gameserver.NPC, error) {
	rows, err := r.p.p.Query(ctx, `
		SELECT s.npc_id, COALESCE(t.name, ''), COALESCE(t.title, ''), s.x, s.y, s.z, s.heading,
		       COALESCE(t.level, 1), COALESCE(t.maxhp, 100), COALESCE(t.maxmp, 0), COALESCE(t.is_attackable, false)
		FROM npc_spawns s
		LEFT JOIN npc_templates t ON t.npc_id = s.npc_id
		ORDER BY s.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []gameserver.NPC
	for rows.Next() {
		var n gameserver.NPC
		if err := rows.Scan(&n.NPCID, &n.Name, &n.Title, &n.X, &n.Y, &n.Z, &n.Heading, &n.Level, &n.MaxHP, &n.MaxMP, &n.IsAttackable); err != nil {
			return nil, err
		}
		n.CurHP = n.MaxHP
		n.CurMP = n.MaxMP
		out = append(out, n)
	}
	return out, rows.Err()
}

// PersistDatapack writes Java SkillTable + class skill trees into PostgreSQL.
func (p *Pool) PersistDatapack(ctx context.Context) error {
	tx, err := p.p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `TRUNCATE skill_templates, class_skills`); err != nil {
		return err
	}
	skills := gameserver.AllSkills()
	const batch = 200
	for i := 0; i < len(skills); i += batch {
		end := i + batch
		if end > len(skills) {
			end = len(skills)
		}
		for _, t := range skills[i:end] {
			if _, err := tx.Exec(ctx, `INSERT INTO skill_templates (
				skill_id, skill_level, name, operate_type, skill_type, is_magic,
				mp_consume, hit_time, reuse_delay, cool_time, power, magic_lvl, target_type, cast_range, effect_range)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
				t.ID, t.Level, t.Name, t.OperateType, t.SkillType, t.IsMagic,
				t.MPConsume, t.HitTime, t.ReuseDelay, t.CoolTime, t.Power, t.MagicLvl, t.TargetType, t.CastRange, t.EffectRange); err != nil {
				return err
			}
		}
	}
	for _, cls := range gameserver.AllClassTemplates() {
		for _, s := range cls.Skills {
			if _, err := tx.Exec(ctx, `INSERT INTO class_skills (class_id, skill_id, skill_level, cost, min_lvl)
				VALUES ($1,$2,$3,$4,$5)`, cls.ID, s.ID, s.Level, s.Cost, s.MinLvl); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

func MustConnect(ctx context.Context, url string) *Pool {
	p, err := Connect(ctx, url)
	if err != nil {
		panic(fmt.Errorf("postgres: %w", err))
	}
	return p
}
