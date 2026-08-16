package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
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
	return out, rows.Err()
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
	return ch, nil
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
	_, err := r.p.p.Exec(ctx, `INSERT INTO characters (
		obj_id, account_name, char_name, title, level, maxhp, curhp, maxmp, curmp, maxcp, curcp,
		face, hairstyle, haircolor, sex, heading, x, y, z, exp, sp, karma, pvpkills, pkkills, clanid, race, classid, base_class,
		deletetime, accesslevel, lastaccess, str, dex, con, intel, wit, men)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37)`,
		ch.ObjectID, ch.Account, ch.Name, ch.Title, ch.Level, ch.MaxHP, ch.CurHP, ch.MaxMP, ch.CurMP, ch.MaxCP, ch.CurCP,
		ch.Face, ch.HairStyle, ch.HairColor, ch.Sex, ch.Heading, ch.X, ch.Y, ch.Z, ch.Exp, ch.SP, ch.Karma, ch.PvPKills, ch.PKKills, ch.ClanID, ch.Race, ch.ClassID, ch.BaseClass,
		ch.DeleteTime, ch.AccessLevel, ch.LastAccess, ch.STR, ch.DEX, ch.CON, ch.INT, ch.WIT, ch.MEN)
	return err
}

func (r *CharacterRepo) Update(ctx context.Context, ch *gameserver.Character) error {
	_, err := r.p.p.Exec(ctx, `UPDATE characters SET title=$2, level=$3, maxhp=$4, curhp=$5, maxmp=$6, curmp=$7, maxcp=$8, curcp=$9,
		heading=$10, x=$11, y=$12, z=$13, exp=$14, sp=$15, karma=$16, pvpkills=$17, pkkills=$18, lastaccess=$19, classid=$20
		WHERE obj_id=$1`,
		ch.ObjectID, ch.Title, ch.Level, ch.MaxHP, ch.CurHP, ch.MaxMP, ch.CurMP, ch.MaxCP, ch.CurCP,
		ch.Heading, ch.X, ch.Y, ch.Z, ch.Exp, ch.SP, ch.Karma, ch.PvPKills, ch.PKKills, ch.LastAccess, ch.ClassID)
	return err
}

func (r *CharacterRepo) Delete(ctx context.Context, id int32) error {
	_, err := r.p.p.Exec(ctx, `DELETE FROM characters WHERE obj_id=$1`, id)
	return err
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

func MustConnect(ctx context.Context, url string) *Pool {
	p, err := Connect(ctx, url)
	if err != nil {
		panic(fmt.Errorf("postgres: %w", err))
	}
	return p
}
