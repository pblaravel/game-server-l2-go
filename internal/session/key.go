package session

import "fmt"

// Key is the 4×int32 session token shared by login and game servers.
// Login-server constructor order matches Java SessionKey(loginOK1, loginOK2, playOK1, playOK2).
type Key struct {
	LoginOkID1 int32
	LoginOkID2 int32
	PlayOkID1  int32
	PlayOkID2  int32
}

func New(loginOK1, loginOK2, playOK1, playOK2 int32) Key {
	return Key{
		LoginOkID1: loginOK1,
		LoginOkID2: loginOK2,
		PlayOkID1:  playOK1,
		PlayOkID2:  playOK2,
	}
}

// NewGameClientKey matches the gameserver record constructor
// SessionKey(playOkId1, playOkId2, loginOkId1, loginOkId2) as used by addClient.
func NewGameClientKey(playOkId1, playOkId2, loginOkId1, loginOkId2 int32) Key {
	return Key{
		PlayOkID1:  playOkId1,
		PlayOkID2:  playOkId2,
		LoginOkID1: loginOkId1,
		LoginOkID2: loginOkId2,
	}
}

func (k Key) CheckLoginPair(loginOk1, loginOk2 int32) bool {
	return k.LoginOkID1 == loginOk1 && k.LoginOkID2 == loginOk2
}

func (k Key) Equals(other Key, showLicense bool) bool {
	if showLicense {
		return k.PlayOkID1 == other.PlayOkID1 &&
			k.LoginOkID1 == other.LoginOkID1 &&
			k.PlayOkID2 == other.PlayOkID2 &&
			k.LoginOkID2 == other.LoginOkID2
	}
	return k.PlayOkID1 == other.PlayOkID1 && k.PlayOkID2 == other.PlayOkID2
}

func (k Key) String() string {
	return fmt.Sprintf("PlayOk: %d %d LoginOk:%d %d", k.PlayOkID1, k.PlayOkID2, k.LoginOkID1, k.LoginOkID2)
}
