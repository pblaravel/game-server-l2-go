package session

import "testing"

func TestCheckLoginPair(t *testing.T) {
	k := New(1, 2, 3, 4)
	if !k.CheckLoginPair(1, 2) {
		t.Fatal("pair")
	}
	if k.CheckLoginPair(1, 9) {
		t.Fatal("mismatch")
	}
}

func TestGameClientKeyConstructor(t *testing.T) {
	// Java gameserver record SessionKey(play, play, login, login)
	// called as new SessionKey(loginKey1, loginKey2, playKey1, playKey2)
	k := NewGameClientKey(11, 22, 33, 44)
	if k.PlayOkID1 != 11 || k.PlayOkID2 != 22 || k.LoginOkID1 != 33 || k.LoginOkID2 != 44 {
		t.Fatalf("%+v", k)
	}
}
