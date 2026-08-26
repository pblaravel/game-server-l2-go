package loginserver

import "sync"

// GameServerInfo matches Java com.shnok.javaserver.model.GameServerInfo.
type GameServerInfo struct {
	mu         sync.Mutex
	id         int
	hexID      []byte
	authed     bool
	thread     *GameServerThread
	status     int
	hostname   string
	port       int
	maxPlayers int
}

func NewGameServerInfo(id int, hexID []byte, thread *GameServerThread) *GameServerInfo {
	return &GameServerInfo{
		id:     id,
		hexID:  append([]byte(nil), hexID...),
		thread: thread,
		status: StatusDown,
	}
}

func (g *GameServerInfo) ID() int              { return g.id }
func (g *GameServerInfo) SetID(id int)         { g.id = id }
func (g *GameServerInfo) HexID() []byte        { return g.hexID }
func (g *GameServerInfo) Name() string         { return ServerName(g.id) }
func (g *GameServerInfo) Hostname() string     { return g.hostname }
func (g *GameServerInfo) SetHostname(h string) { g.hostname = h }
func (g *GameServerInfo) Port() int            { return g.port }
func (g *GameServerInfo) SetPort(p int)        { g.port = p }
func (g *GameServerInfo) MaxPlayers() int      { return g.maxPlayers }
func (g *GameServerInfo) SetMaxPlayers(n int)  { g.maxPlayers = n }
func (g *GameServerInfo) Status() int          { return g.status }
func (g *GameServerInfo) SetStatus(v int)      { g.status = v }
func (g *GameServerInfo) IsAuthed() bool       { return g.authed }
func (g *GameServerInfo) SetAuthed(v bool)     { g.authed = v }
func (g *GameServerInfo) Thread() *GameServerThread {
	return g.thread
}
func (g *GameServerInfo) SetThread(t *GameServerThread) { g.thread = t }

func (g *GameServerInfo) CurrentPlayerCount() int {
	if g.thread == nil {
		return 0
	}
	return g.thread.GetPlayerCount()
}

func (g *GameServerInfo) SetDown() {
	g.authed = false
	g.port = 0
	g.thread = nil
	g.status = StatusDown
}

func (g *GameServerInfo) Lock()   { g.mu.Lock() }
func (g *GameServerInfo) Unlock() { g.mu.Unlock() }
