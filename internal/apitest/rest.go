package apitest

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// REST is an HTTP facade over the Java-compatible binary TCP protocol.
type REST struct {
	Target Target

	mu    sync.Mutex
	holds []*GSHold
}

func (api *REST) Close() {
	api.mu.Lock()
	defer api.mu.Unlock()
	for _, h := range api.holds {
		h.Close()
	}
	api.holds = nil
}

func (api *REST) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", api.health)
	mux.HandleFunc("GET /api/catalog", api.catalog)
	mux.HandleFunc("GET /api/login/init", api.loginInit)
	mux.HandleFunc("POST /api/login/ping", api.loginPing)
	mux.HandleFunc("POST /api/login/auth", api.loginAuth)
	mux.HandleFunc("POST /api/login/servers", api.loginServers)
	mux.HandleFunc("POST /api/login/play", api.loginPlay)
	mux.HandleFunc("GET /api/gsreg/init", api.gsInit)
	mux.HandleFunc("POST /api/gsreg/register", api.gsRegister)
	mux.HandleFunc("POST /api/game/protocol", api.gameProtocol)
	return mux
}

type authBody struct {
	Account      string `json:"account"`
	PasswordHash string `json:"passwordHash"`
	ServerID     int    `json:"serverId"`
	HexID        string `json:"hexid"`
	Version      int32  `json:"version"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func readBody(r *http.Request) (authBody, error) {
	var b authBody
	if r.Body == nil {
		return b, nil
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return b, err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return b, nil
	}
	if err := json.Unmarshal(raw, &b); err != nil {
		return b, err
	}
	return b, nil
}

func decodeHash(s string) []byte {
	if s == "" {
		return []byte{1, 2, 3, 4}
	}
	if b, err := base64.StdEncoding.DecodeString(s); err == nil && len(b) > 0 {
		return b
	}
	return []byte(s)
}

func decodeHexID(s string) []byte {
	if s == "" {
		return []byte{0x81, 0xa8, 0xba, 0x90, 0xdb, 0x0e, 0x77, 0xd3, 0x03, 0x39, 0x73, 0x88, 0xe2, 0x5e, 0xce, 0xfa}
	}
	if b, err := base64.StdEncoding.DecodeString(s); err == nil && len(b) > 0 {
		return b
	}
	return []byte(s)
}

func (api *REST) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"login": api.Target.Login,
		"gsreg": api.Target.GSReg,
		"game":  api.Target.Game,
	})
}

func (api *REST) catalog(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, Catalog())
}

func (api *REST) loginInit(w http.ResponseWriter, _ *http.Request) {
	out, err := LoginInit(api.Target.Login)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (api *REST) loginPing(w http.ResponseWriter, _ *http.Request) {
	out, err := LoginPing(api.Target.Login)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (api *REST) loginAuth(w http.ResponseWriter, r *http.Request) {
	b, err := readBody(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if b.Account == "" {
		b.Account = "apitest"
	}
	out, err := LoginAuth(api.Target.Login, b.Account, decodeHash(b.PasswordHash))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (api *REST) loginServers(w http.ResponseWriter, r *http.Request) {
	b, err := readBody(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if b.Account == "" {
		b.Account = "apitest"
	}
	out, err := LoginServers(api.Target.Login, b.Account, decodeHash(b.PasswordHash))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (api *REST) loginPlay(w http.ResponseWriter, r *http.Request) {
	b, err := readBody(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if b.Account == "" {
		b.Account = "apitest"
	}
	if b.ServerID == 0 {
		b.ServerID = 1
	}
	out, err := LoginPlay(api.Target.Login, b.Account, decodeHash(b.PasswordHash), b.ServerID)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (api *REST) gsInit(w http.ResponseWriter, _ *http.Request) {
	out, err := GSRegInit(api.Target.GSReg)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (api *REST) gsRegister(w http.ResponseWriter, r *http.Request) {
	b, err := readBody(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if b.ServerID == 0 {
		b.ServerID = 1
	}
	hold, err := OpenGSReg(api.Target.GSReg, b.ServerID, decodeHexID(b.HexID))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	api.mu.Lock()
	api.holds = append(api.holds, hold)
	api.mu.Unlock()
	writeJSON(w, http.StatusOK, hold.Result)
}

func (api *REST) gameProtocol(w http.ResponseWriter, r *http.Request) {
	b, err := readBody(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if b.Version == 0 {
		if q := r.URL.Query().Get("version"); q != "" {
			if n, err := strconv.Atoi(q); err == nil {
				b.Version = int32(n)
			}
		}
	}
	if b.Version == 0 {
		b.Version = 740
	}
	out, err := GameProtocol(api.Target.Game, b.Version)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
