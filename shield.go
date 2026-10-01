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

// Absorber 是护盾台账。
type Absorber struct {
	shields map[string][]Shield
	dirty   map[string]bool
	scanned int
}

// NewAbsorber 建台账。
func NewAbsorber() *Absorber {
	return &Absorber{shields: map[string][]Shield{}, dirty: map[string]bool{}}
}

// Add 给玩家加一层护盾。
func (a *Absorber) Add(player string, layer Shield) {
	a.shields[player] = append(a.shields[player], layer)
	a.dirty[player] = true
}

// ensureSorted 把该玩家的护盾按优先级升序（同优先级按 ID 升序）排好，只在有新增时重排。
func (a *Absorber) ensureSorted(player string) {
	if !a.dirty[player] {
		return
	}
	layers := a.shields[player]
	sort.SliceStable(layers, func(i, j int) bool {
		if layers[i].Priority != layers[j].Priority {
			return layers[i].Priority < layers[j].Priority
		}
		return layers[i].ID < layers[j].ID
	})
	a.dirty[player] = false
}

// Apply 结算一次伤害，返回被吸收的量与剩余伤害。
// 按优先级升序依次吸收，一层吸不完继续下传；过期护盾不参与并被清理；
// 护盾量封底不为负，absorbed + remaining == damage（该玩家没有任何护盾时返回 0, 0）。
func (a *Absorber) Apply(player string, damage float64, nowMs int64) (absorbed float64, remaining float64) {
	layers := a.shields[player]
	a.scanned += len(layers)
	if len(layers) == 0 {
		return 0, 0
	}
	a.ensureSorted(player)
	layers = a.shields[player]
	left := damage
	kept := layers[:0]
	for _, layer := range layers {
		if nowMs >= layer.ExpireMs {
			continue
		}
		if left > 0 {
			take := layer.Amount
			if take > left {
				take = left
			}
			layer.Amount -= take
			absorbed += take
			left -= take
		}
		if layer.Amount > 0 {
			kept = append(kept, layer)
		}
	}
	if len(kept) == 0 {
		delete(a.shields, player)
		delete(a.dirty, player)
	} else {
		a.shields[player] = kept
	}
	return absorbed, left
}

// AmountOf 是某层护盾的剩余量。
func (a *Absorber) AmountOf(player, shieldID string) float64 {
	for _, layer := range a.shields[player] {
		if layer.ID == shieldID {
			return layer.Amount
		}
	}
	return 0
}

// Remaining 是全部未过期护盾的剩余总量。
func (a *Absorber) Remaining(player string, nowMs int64) float64 {
	total := 0.0
	for _, layer := range a.shields[player] {
		if nowMs >= layer.ExpireMs {
			continue
		}
		total += layer.Amount
	}
	return total
}

// Scanned 是累计扫描的护盾层数（规模观测）。
func (a *Absorber) Scanned() int { return a.scanned }
