// Package shield 是护盾吸收：按优先级吸收伤害、过期护盾失效、溢出伤害继续下传，吸收量守恒。
// 缺陷：不看优先级（按加入顺序）、不看过期、溢出伤害被丢弃、护盾量能吸成负数、每次全扫护盾表。
package shield

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
	scanned int
}

// NewAbsorber 建台账。
func NewAbsorber() *Absorber {
	return &Absorber{shields: map[string][]Shield{}}
}

// Add 给玩家加一层护盾。
func (a *Absorber) Add(player string, layer Shield) {
	a.shields[player] = append(a.shields[player], layer)
}

// Apply 结算一次伤害，返回被吸收的量与剩余伤害。
// 缺陷：按加入顺序吸收、不看过期、只吸第一层就把剩余伤害丢掉、护盾量不封底、每次全扫护盾表。
func (a *Absorber) Apply(player string, damage float64, nowMs int64) (float64, float64) {
	layers := a.shields[player]
	a.scanned += len(layers)
	if len(layers) == 0 {
		return 0, 0
	}
	absorbed := damage
	if layers[0].Amount < damage {
		absorbed = layers[0].Amount
	}
	a.shields[player][0].Amount -= damage
	return absorbed, 0
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

// Remaining 是全部护盾的剩余总量。
func (a *Absorber) Remaining(player string, nowMs int64) float64 {
	total := 0.0
	for _, layer := range a.shields[player] {
		total += layer.Amount
	}
	return total
}

// Scanned 是累计扫描的护盾层数（规模观测）。
func (a *Absorber) Scanned() int { return a.scanned }
