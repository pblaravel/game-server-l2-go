package gameserver

import (
	"context"
	"sync"
)

type CharacterStore interface {
	ListByAccount(ctx context.Context, account string) ([]*Character, error)
	GetByObjectID(ctx context.Context, id int32) (*Character, error)
	GetObjectIDByName(ctx context.Context, name string) (int32, error)
	CountByAccount(ctx context.Context, account string) (int, error)
	Create(ctx context.Context, ch *Character) error
	Update(ctx context.Context, ch *Character) error
	Delete(ctx context.Context, id int32) error
	NextObjectID(ctx context.Context) (int32, error)
}

type MemoryCharacterStore struct {
	mu     sync.Mutex
	chars  map[int32]*Character
	nextID int32
}

func NewMemoryCharacterStore() *MemoryCharacterStore {
	return &MemoryCharacterStore{chars: make(map[int32]*Character), nextID: 300000000}
}

// RestoreCharacter is Java Player.restore: after the row is read the derived
// state (stance, collision, stats from the class template and buffs) is rebuilt.
func RestoreCharacter(ch *Character) *Character {
	if ch == nil {
		return nil
	}
	ch.ApplyRuntimeDefaults()
	RecalcStats(ch)
	return ch
}

func cloneCharacter(ch *Character) *Character {
	if ch == nil {
		return nil
	}
	cp := *ch
	if ch.Items != nil {
		cp.Items = append([]Item(nil), ch.Items...)
	}
	if ch.Skills != nil {
		cp.Skills = append([]Skill(nil), ch.Skills...)
	}
	if ch.Shortcuts != nil {
		cp.Shortcuts = append([]Shortcut(nil), ch.Shortcuts...)
	}
	if ch.Warehouse != nil {
		cp.Warehouse = append([]Item(nil), ch.Warehouse...)
	}
	if ch.Friends != nil {
		cp.Friends = append([]Friend(nil), ch.Friends...)
	}
	return &cp
}

func (s *MemoryCharacterStore) ListByAccount(_ context.Context, account string) ([]*Character, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*Character
	for _, c := range s.chars {
		if c.Account == account {
			out = append(out, RestoreCharacter(cloneCharacter(c)))
		}
	}
	return out, nil
}

func (s *MemoryCharacterStore) GetByObjectID(_ context.Context, id int32) (*Character, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.chars[id]
	if !ok {
		return nil, nil
	}
	return RestoreCharacter(cloneCharacter(c)), nil
}

func (s *MemoryCharacterStore) GetObjectIDByName(_ context.Context, name string) (int32, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.chars {
		if c.Name == name {
			return c.ObjectID, nil
		}
	}
	return 0, nil
}

func (s *MemoryCharacterStore) CountByAccount(_ context.Context, account string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, c := range s.chars {
		if c.Account == account {
			n++
		}
	}
	return n, nil
}

func (s *MemoryCharacterStore) Create(_ context.Context, ch *Character) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chars[ch.ObjectID] = cloneCharacter(ch)
	return nil
}

func (s *MemoryCharacterStore) Update(_ context.Context, ch *Character) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chars[ch.ObjectID] = cloneCharacter(ch)
	return nil
}

func (s *MemoryCharacterStore) Delete(_ context.Context, id int32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.chars, id)
	return nil
}

func (s *MemoryCharacterStore) NextObjectID(_ context.Context) (int32, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	return s.nextID, nil
}
