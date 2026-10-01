package shield

import (
	"math"
	"math/rand"
	"sort"
	"testing"
)

// refShield 是参考实现用的护盾状态。
type refShield struct {
	id       string
	amount   float64
	expireMs int64
	priority int
}

// refAbsorber 严格按「优先级下传 + 过期失效 + 守恒」写的朴素参考实现。
type refAbsorber struct {
	layers map[string][]refShield
}

func newRefAbsorber() *refAbsorber {
	return &refAbsorber{layers: map[string][]refShield{}}
}

func (r *refAbsorber) add(player string, s refShield) {
	r.layers[player] = append(r.layers[player], s)
	sort.SliceStable(r.layers[player], func(i, j int) bool {
		x, y := r.layers[player][i], r.layers[player][j]
		if x.priority != y.priority {
			return x.priority < y.priority
		}
		return x.id < y.id
	})
}

func (r *refAbsorber) apply(player string, damage float64, nowMs int64) (float64, float64) {
	layers, ok := r.layers[player]
	if !ok {
		return 0, 0
	}
	absorbed := 0.0
	remaining := damage
	kept := make([]refShield, 0, len(layers))
	i := 0
	for ; i < len(layers) && remaining > 0; i++ {
		s := layers[i]
		if nowMs >= s.expireMs || s.amount <= 0 {
			continue // 失效层只在被触及时摘除
		}
		take := math.Min(remaining, s.amount)
		s.amount -= take
		absorbed += take
		remaining -= take
		if s.amount > 0 {
			kept = append(kept, s)
		}
	}
	kept = append(kept, layers[i:]...) // 伤害吸完即停，未触及的层原样保留
	r.layers[player] = kept
	return absorbed, damage - absorbed
}

func (r *refAbsorber) amountOf(player, id string) float64 {
	for _, s := range r.layers[player] {
		if s.id == id {
			return s.amount
		}
	}
	return 0
}

func (r *refAbsorber) remaining(player string, nowMs int64) float64 {
	total := 0.0
	for _, s := range r.layers[player] {
		if nowMs < s.expireMs {
			total += s.amount
		}
	}
	return total
}

func TestRandomDifferential(t *testing.T) {
	rng := rand.New(rand.NewSource(20261002))
	const cases = 200
	for tc := 0; tc < cases; tc++ {
		got := NewAbsorber()
		ref := newRefAbsorber()
		players := []string{"p", "q"}
		idsByPlayer := map[string][]string{}
		hasShields := map[string]bool{}
		for _, player := range players {
			n := rng.Intn(13)
			used := map[string]bool{}
			for i := 0; i < n; i++ {
				var id string
				for {
					id = "id" + itoa(rng.Intn(40))
					if !used[id] {
						used[id] = true
						break
					}
				}
				s := Shield{
					ID:       id,
					Amount:   float64(rng.Intn(7)) * 10,
					ExpireMs: int64(rng.Intn(1501)),
					Priority: rng.Intn(5),
				}
				got.Add(player, s)
				ref.add(player, refShield{id, s.Amount, s.ExpireMs, s.Priority})
				idsByPlayer[player] = append(idsByPlayer[player], id)
				hasShields[player] = true
			}
		}
		hits := rng.Intn(9)
		for h := 0; h < hits; h++ {
			player := players[rng.Intn(len(players))]
			damage := float64(rng.Intn(13)) * 10
			now := int64(rng.Intn(2001))
			gAbs, gRem := got.Apply(player, damage, now)
			rAbs, rRem := ref.apply(player, damage, now)
			if !near(gAbs, rAbs) || !near(gRem, rRem) {
				t.Fatalf("case %d hit %d player %s damage %v now %d: absorbed (%v vs %v) remaining (%v vs %v)",
					tc, h, player, damage, now, gAbs, rAbs, gRem, rRem)
			}
			if !near(got.Remaining(player, now), ref.remaining(player, now)) {
				t.Fatalf("case %d hit %d: Remaining (%v vs %v)", tc, h,
					got.Remaining(player, now), ref.remaining(player, now))
			}
			for _, id := range idsByPlayer[player] {
				if !near(got.AmountOf(player, id), ref.amountOf(player, id)) {
					t.Fatalf("case %d hit %d layer %s: amount (%v vs %v)",
						tc, h, id, got.AmountOf(player, id), ref.amountOf(player, id))
				}
				if got.AmountOf(player, id) < -1e-9 {
					t.Fatalf("case %d hit %d layer %s: 护盾量为负 %v", tc, h, id, got.AmountOf(player, id))
				}
			}
			if hasShields[player] && !near(gAbs+gRem, damage) {
				t.Fatalf("case %d hit %d: 不守恒 absorbed=%v remaining=%v damage=%v", tc, h, gAbs, gRem, damage)
			}
		}
	}
}

func near(x, y float64) bool {
	return math.Abs(x-y) < 1e-9
}
