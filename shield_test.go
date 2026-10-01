package shield

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"
)

func TestNoShieldTakesNoDamage(t *testing.T) {
	absorbed, remaining := NewAbsorber().Apply("p", 50, 0)
	if absorbed != 0 || remaining != 0 {
		t.Fatalf("没有护盾时不该吸收：%v %v", absorbed, remaining)
	}
}

func TestSingleShieldAbsorbs(t *testing.T) {
	absorber := NewAbsorber()
	absorber.Add("p", Shield{ID: "s1", Amount: 100, ExpireMs: 1000})
	if absorbed, _ := absorber.Apply("p", 40, 0); absorbed != 40 {
		t.Fatalf("一层护盾应该吸满这 40 点：%v", absorbed)
	}
}

func TestRemainingCountsShield(t *testing.T) {
	absorber := NewAbsorber()
	absorber.Add("p", Shield{ID: "s1", Amount: 30, ExpireMs: 1000})
	if absorber.Remaining("p", 0) != 30 {
		t.Fatalf("剩余护盾量不对：%v", absorber.Remaining("p", 0))
	}
}

func TestScannedStartsAtZero(t *testing.T) {
	if NewAbsorber().Scanned() != 0 {
		t.Fatal("初始扫描数应该是 0")
	}
}

func TestPriorityOrderAndSamePriorityByID(t *testing.T) {
	absorber := NewAbsorber()
	absorber.Add("p", Shield{ID: "z", Amount: 10, ExpireMs: 1000, Priority: 2})
	absorber.Add("p", Shield{ID: "b", Amount: 10, ExpireMs: 1000, Priority: 1})
	absorber.Add("p", Shield{ID: "a", Amount: 10, ExpireMs: 1000, Priority: 1})
	absorbed, remaining := absorber.Apply("p", 15, 0)
	if absorbed != 15 || remaining != 0 {
		t.Fatalf("吸收守恒被破坏：absorbed=%v remaining=%v", absorbed, remaining)
	}
	if got := absorber.AmountOf("p", "a"); got != 0 {
		t.Fatalf("同优先级时 ID 小的先吸，a 应被吸干：%v", got)
	}
	if got := absorber.AmountOf("p", "b"); got != 5 {
		t.Fatalf("a 吸完后轮到 b：%v", got)
	}
	if got := absorber.AmountOf("p", "z"); got != 10 {
		t.Fatalf("优先级高的 z 不该被动：%v", got)
	}
}

func TestOverflowConservation(t *testing.T) {
	absorber := NewAbsorber()
	absorber.Add("p", Shield{ID: "a", Amount: 30, ExpireMs: 1000, Priority: 2})
	absorber.Add("p", Shield{ID: "b", Amount: 30, ExpireMs: 1000, Priority: 1})
	absorbed, remaining := absorber.Apply("p", 100, 0)
	if absorbed != 60 || remaining != 40 {
		t.Fatalf("溢出伤害应继续下传并守恒：absorbed=%v remaining=%v", absorbed, remaining)
	}
	if absorbed+remaining != 100 {
		t.Fatal("absorbed + remaining 必须等于 damage")
	}
	if absorber.AmountOf("p", "a") != 0 || absorber.AmountOf("p", "b") != 0 {
		t.Fatal("两层护盾都应被吸干且不为负")
	}
}

func TestExpiredShieldDoesNotAbsorb(t *testing.T) {
	absorber := NewAbsorber()
	absorber.Add("p", Shield{ID: "old", Amount: 50, ExpireMs: 1000, Priority: 1})
	absorber.Add("p", Shield{ID: "live", Amount: 20, ExpireMs: 5000, Priority: 2})
	if got := absorber.Remaining("p", 500); got != 70 {
		t.Fatalf("未过期护盾都应计入 Remaining：%v", got)
	}
	if got := absorber.Remaining("p", 1000); got != 20 {
		t.Fatalf("now >= ExpireMs 的护盾不该计入 Remaining：%v", got)
	}
	absorbed, remaining := absorber.Apply("p", 30, 1000)
	if absorbed != 20 || remaining != 10 {
		t.Fatalf("过期护盾不该参与吸收：absorbed=%v remaining=%v", absorbed, remaining)
	}
	if got := absorber.Remaining("p", 1000); got != 0 {
		t.Fatalf("结算后不该有剩余护盾：%v", got)
	}
}

func TestShieldNeverNegative(t *testing.T) {
	absorber := NewAbsorber()
	absorber.Add("p", Shield{ID: "a", Amount: 30, ExpireMs: 1000, Priority: 1})
	absorbed, remaining := absorber.Apply("p", 1000, 0)
	if absorbed != 30 || remaining != 970 {
		t.Fatalf("大额伤害应吸干护盾并守恒：absorbed=%v remaining=%v", absorbed, remaining)
	}
	if got := absorber.AmountOf("p", "a"); got < 0 {
		t.Fatalf("护盾量不许为负：%v", got)
	}
}

func TestScannedOnlyTouchesTargetPlayer(t *testing.T) {
	absorber := NewAbsorber()
	for i := 0; i < 100000; i++ {
		absorber.Add("busy", Shield{ID: fmt.Sprintf("s%d", i), Amount: 1, ExpireMs: 1000, Priority: i})
	}
	absorber.Add("p", Shield{ID: "s", Amount: 10, ExpireMs: 1000})
	absorber.Apply("p", 5, 0)
	if absorber.Scanned() != 1 {
		t.Fatalf("Apply 只该碰该玩家的护盾，Scanned 不随总数增长：%v", absorber.Scanned())
	}
}

func TestScannedDoesNotGrowAfterShieldsConsumed(t *testing.T) {
	absorber := NewAbsorber()
	for i := 0; i < 100000; i++ {
		absorber.Add("p", Shield{ID: fmt.Sprintf("s%d", i), Amount: 1, ExpireMs: 1000, Priority: i})
	}
	if absorbed, _ := absorber.Apply("p", 100000, 0); absorbed != 100000 {
		t.Fatalf("10 万层应全部被吸干：%v", absorbed)
	}
	before := absorber.Scanned()
	for i := 0; i < 1000; i++ {
		absorber.Apply("p", 1, 0)
	}
	if absorber.Scanned() != before {
		t.Fatalf("护盾清空后结算不该再扫描：before=%v after=%v", before, absorber.Scanned())
	}
}

func TestDeterministicReplay(t *testing.T) {
	run := func() [][2]float64 {
		absorber := NewAbsorber()
		absorber.Add("p", Shield{ID: "a", Amount: 40, ExpireMs: 500, Priority: 2})
		absorber.Add("p", Shield{ID: "b", Amount: 40, ExpireMs: 1000, Priority: 1})
		absorber.Add("p", Shield{ID: "c", Amount: 40, ExpireMs: 2000, Priority: 1})
		var out [][2]float64
		for _, now := range []int64{0, 600, 1500} {
			absorbed, remaining := absorber.Apply("p", 50, now)
			out = append(out, [2]float64{absorbed, remaining})
		}
		return out
	}
	first, second := run(), run()
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("同一伤害序列重放结果必须相同：%v vs %v", first, second)
		}
	}
}

// refApply 是参考实现：每次独立排序、过滤过期、按序下传，语义直接翻译 README 口径。
func refApply(layers []Shield, damage float64, nowMs int64) (float64, float64, []Shield) {
	if len(layers) == 0 {
		return 0, 0, nil
	}
	sorted := append([]Shield(nil), layers...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Priority != sorted[j].Priority {
			return sorted[i].Priority < sorted[j].Priority
		}
		return sorted[i].ID < sorted[j].ID
	})
	absorbed := 0.0
	left := damage
	kept := make([]Shield, 0, len(sorted))
	for _, layer := range sorted {
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
	return absorbed, left, kept
}

func TestRandomMatchesReference(t *testing.T) {
	rng := rand.New(rand.NewSource(20261002))
	for trial := 0; trial < 200; trial++ {
		absorber := NewAbsorber()
		var ref []Shield
		var ids []string
		for i, n := 0, rng.Intn(9); i < n; i++ {
			layer := Shield{
				ID:       fmt.Sprintf("s%d", i),
				Amount:   float64(rng.Intn(100)),
				ExpireMs: int64(rng.Intn(2000)),
				Priority: rng.Intn(4),
			}
			absorber.Add("p", layer)
			ref = append(ref, layer)
			ids = append(ids, layer.ID)
		}
		for step, steps := 0, rng.Intn(7); step < steps; step++ {
			damage := float64(rng.Intn(150))
			now := int64(rng.Intn(2500))
			hadLayers := len(absorber.shields["p"]) > 0
			gotAbsorbed, gotRemaining := absorber.Apply("p", damage, now)
			wantAbsorbed, wantRemaining, kept := refApply(ref, damage, now)
			ref = kept
			if gotAbsorbed != wantAbsorbed || gotRemaining != wantRemaining {
				t.Fatalf("trial %d step %d：absorbed/remaining=%v/%v，参考为 %v/%v",
					trial, step, gotAbsorbed, gotRemaining, wantAbsorbed, wantRemaining)
			}
			if hadLayers && gotAbsorbed+gotRemaining != damage {
				t.Fatalf("trial %d step %d：守恒被破坏 %v+%v != %v",
					trial, step, gotAbsorbed, gotRemaining, damage)
			}
			for _, layer := range absorber.shields["p"] {
				if layer.Amount < 0 {
					t.Fatalf("trial %d step %d：护盾量为负 %v", trial, step, layer.Amount)
				}
			}
			wantTotal := 0.0
			for _, layer := range ref {
				if now < layer.ExpireMs {
					wantTotal += layer.Amount
				}
			}
			if got := absorber.Remaining("p", now); got != wantTotal {
				t.Fatalf("trial %d step %d：Remaining=%v，参考为 %v", trial, step, got, wantTotal)
			}
		}
		for _, id := range ids {
			want := 0.0
			for _, layer := range ref {
				if layer.ID == id {
					want = layer.Amount
				}
			}
			if got := absorber.AmountOf("p", id); got != want {
				t.Fatalf("trial %d：层 %s 剩余 %v，参考为 %v", trial, id, got, want)
			}
		}
	}
}
