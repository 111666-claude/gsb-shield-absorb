// Package shield 是护盾吸收：按优先级吸收伤害、过期护盾失效、溢出伤害继续下传，吸收量守恒。
package shield

import "sort"

// Shield 是一层护盾。
type Shield struct {
	ID       string
	Amount   float64
	ExpireMs int64
	Priority int // 越小越先吸收
}

// slot 是台账里的一层护盾；dead 表示已耗尽或被触及时发现过期，惰性摘除不搬移。
type slot struct {
	shield Shield
	dead   bool
}

// playerState 是某个玩家的护盾台账。
// layers 始终按 (Priority, ID) 升序；head 之前都是已摘除的槽位；
// index 用稳定指针支持按 ID 直接查剩余量。
type playerState struct {
	layers []*slot
	head   int
	index  map[string]*slot
}

// Absorber 是护盾台账。
type Absorber struct {
	players map[string]*playerState
	scanned int
}

// NewAbsorber 建台账。
func NewAbsorber() *Absorber {
	return &Absorber{players: map[string]*playerState{}}
}

func (a *Absorber) state(player string) *playerState {
	st := a.players[player]
	if st == nil {
		st = &playerState{index: map[string]*slot{}}
		a.players[player] = st
	}
	return st
}

// lessShield 报告 x 是否应排在 y 前面：Priority 升序，同优先级按 ID 升序。
func lessShield(x, y Shield) bool {
	if x.Priority != y.Priority {
		return x.Priority < y.Priority
	}
	return x.ID < y.ID
}

// Add 给玩家加一层护盾，按 (Priority, ID) 插入有序位置。
func (a *Absorber) Add(player string, layer Shield) {
	st := a.state(player)
	s := &slot{shield: layer}
	pos := st.head + sort.Search(len(st.layers)-st.head, func(i int) bool {
		return !lessShield(st.layers[st.head+i].shield, layer)
	})
	st.layers = append(st.layers, nil)
	copy(st.layers[pos+1:], st.layers[pos:])
	st.layers[pos] = s
	st.index[layer.ID] = s
}

// drop 摘除一个槽位并回收 head 之前的连续死槽。
func (a *Absorber) drop(st *playerState, i int) {
	s := st.layers[i]
	s.dead = true
	delete(st.index, s.shield.ID)
	for st.head < len(st.layers) && st.layers[st.head].dead {
		st.head++
	}
	if st.head == len(st.layers) {
		st.layers = nil
		st.head = 0
	}
}

// Apply 结算一次伤害，返回被吸收的量与剩余伤害。
// 按 (Priority, ID) 升序逐层吸收，一层吸不完继续传下一层；
// now >= ExpireMs 的护盾在被触及时失效，不参与吸收；每层护盾量封底为 0。
// 伤害吸完即停，未触及的层不计入 Scanned；每个槽位最多被扫一次。
// 从未加过护盾的玩家返回 (0, 0)；加过护盾的玩家满足 absorbed+remaining == damage。
func (a *Absorber) Apply(player string, damage float64, nowMs int64) (float64, float64) {
	st := a.players[player]
	if st == nil {
		return 0, 0
	}
	absorbed := 0.0
	remaining := damage
	for i := st.head; i < len(st.layers) && remaining > 0; i++ {
		a.scanned++
		s := st.layers[i]
		if nowMs >= s.shield.ExpireMs || s.shield.Amount <= 0 {
			a.drop(st, i)
			continue
		}
		take := remaining
		if s.shield.Amount < take {
			take = s.shield.Amount
		}
		s.shield.Amount -= take
		absorbed += take
		remaining -= take
		if s.shield.Amount <= 0 {
			a.drop(st, i)
		}
	}
	return absorbed, damage - absorbed
}

// AmountOf 是某层护盾的剩余量；已摘除或不存在的层返回 0。
func (a *Absorber) AmountOf(player, shieldID string) float64 {
	st := a.players[player]
	if st == nil {
		return 0
	}
	if s, ok := st.index[shieldID]; ok && !s.dead {
		return s.shield.Amount
	}
	return 0
}

// Remaining 是全部未过期护盾的剩余总量；now >= ExpireMs 的护盾不计入。
func (a *Absorber) Remaining(player string, nowMs int64) float64 {
	st := a.players[player]
	if st == nil {
		return 0
	}
	total := 0.0
	for i := st.head; i < len(st.layers); i++ {
		s := st.layers[i]
		if !s.dead && nowMs < s.shield.ExpireMs {
			total += s.shield.Amount
		}
	}
	return total
}

// Scanned 是累计实际触及的护盾层数（规模观测）。
func (a *Absorber) Scanned() int { return a.scanned }
