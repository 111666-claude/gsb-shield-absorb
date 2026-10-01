package shield

import "testing"

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

func TestPriorityCascade(t *testing.T) {
	absorber := NewAbsorber()
	absorber.Add("p", Shield{ID: "a", Amount: 10, ExpireMs: 1000, Priority: 2})
	absorber.Add("p", Shield{ID: "b", Amount: 50, ExpireMs: 1000, Priority: 1})
	absorbed, remaining := absorber.Apply("p", 30, 0)
	if absorbed != 30 || remaining != 0 {
		t.Fatalf("吸收量不守恒：absorbed=%v remaining=%v", absorbed, remaining)
	}
	if absorber.AmountOf("p", "a") != 10 || absorber.AmountOf("p", "b") != 20 {
		t.Fatalf("应该先吸优先级小的 b：a=%v b=%v", absorber.AmountOf("p", "a"), absorber.AmountOf("p", "b"))
	}
}

func TestOverflowConserved(t *testing.T) {
	absorber := NewAbsorber()
	absorber.Add("p", Shield{ID: "a", Amount: 30, ExpireMs: 1000, Priority: 2})
	absorber.Add("p", Shield{ID: "b", Amount: 30, ExpireMs: 1000, Priority: 1})
	absorbed, remaining := absorber.Apply("p", 100, 0)
	if absorbed != 60 || remaining != 40 {
		t.Fatalf("溢出伤害必须守恒下传：absorbed=%v remaining=%v", absorbed, remaining)
	}
	if absorber.AmountOf("p", "a") != 0 || absorber.AmountOf("p", "b") != 0 {
		t.Fatalf("护盾量不许为负：a=%v b=%v", absorber.AmountOf("p", "a"), absorber.AmountOf("p", "b"))
	}
}

func TestExpiredShieldInert(t *testing.T) {
	absorber := NewAbsorber()
	absorber.Add("p", Shield{ID: "a", Amount: 50, ExpireMs: 1000, Priority: 1})
	absorbed, remaining := absorber.Apply("p", 20, 5000)
	if absorbed != 0 || remaining != 20 {
		t.Fatalf("过期护盾不该吸收：absorbed=%v remaining=%v", absorbed, remaining)
	}
	if absorber.Remaining("p", 5000) != 0 {
		t.Fatalf("过期护盾不该计入剩余：%v", absorber.Remaining("p", 5000))
	}
	if absorber.AmountOf("p", "a") != 0 {
		t.Fatalf("过期护盾不该留量：%v", absorber.AmountOf("p", "a"))
	}
}

func TestHugeDamageNeverNegative(t *testing.T) {
	absorber := NewAbsorber()
	absorber.Add("p", Shield{ID: "a", Amount: 30, ExpireMs: 1000, Priority: 1})
	absorbed, remaining := absorber.Apply("p", 1000, 0)
	if absorbed != 30 || remaining != 970 {
		t.Fatalf("大额伤害也要守恒：absorbed=%v remaining=%v", absorbed, remaining)
	}
	if absorber.AmountOf("p", "a") != 0 {
		t.Fatalf("护盾量不许为负：%v", absorber.AmountOf("p", "a"))
	}
}

func TestSamePriorityByID(t *testing.T) {
	absorber := NewAbsorber()
	absorber.Add("p", Shield{ID: "b", Amount: 10, ExpireMs: 1000, Priority: 1})
	absorber.Add("p", Shield{ID: "a", Amount: 10, ExpireMs: 1000, Priority: 1})
	absorber.Apply("p", 15, 0)
	if absorber.AmountOf("p", "a") != 0 || absorber.AmountOf("p", "b") != 5 {
		t.Fatalf("同优先级应按 ID 升序吸收：a=%v b=%v", absorber.AmountOf("p", "a"), absorber.AmountOf("p", "b"))
	}
}

func TestScannedTracksTouchedLayersOnly(t *testing.T) {
	absorber := NewAbsorber()
	const layers = 20000
	for i := 0; i < layers; i++ {
		absorber.Add("p", Shield{ID: idOf(i), Amount: 100, ExpireMs: 1 << 60, Priority: i})
	}
	absorber.Add("q", Shield{ID: "other", Amount: 100, ExpireMs: 1 << 60})
	if absorber.Scanned() != 0 {
		t.Fatal("加护盾不该计入扫描")
	}
	absorber.Apply("p", 50, 0)
	if absorber.Scanned() != 1 {
		t.Fatalf("小伤害只该碰一层，不该随护盾总数增长：%v", absorber.Scanned())
	}
	absorber.Apply("p", 50, 0)
	if absorber.Scanned() != 2 {
		t.Fatalf("再次小伤害仍只该碰一层：%v", absorber.Scanned())
	}
	absorber.Apply("q", 10, 0)
	if absorber.Scanned() != 3 {
		t.Fatalf("别的玩家的护盾不该被扫：%v", absorber.Scanned())
	}
}

func TestScannedExpiredAmortized(t *testing.T) {
	absorber := NewAbsorber()
	for i := 0; i < 100; i++ {
		absorber.Add("p", Shield{ID: idOf(i), Amount: 10, ExpireMs: 100, Priority: i})
	}
	absorber.Add("p", Shield{ID: "live", Amount: 10, ExpireMs: 1 << 60, Priority: 1000})
	absorber.Apply("p", 5, 200)
	if absorber.Scanned() != 101 {
		t.Fatalf("过期层只该被扫一次：%v", absorber.Scanned())
	}
	absorber.Apply("p", 5, 200)
	if absorber.Scanned() != 102 {
		t.Fatalf("过期层摘除后不该重复扫：%v", absorber.Scanned())
	}
}

func TestDeterministicReplay(t *testing.T) {
	run := func() (float64, float64, float64) {
		absorber := NewAbsorber()
		absorber.Add("p", Shield{ID: "a", Amount: 10, ExpireMs: 100, Priority: 2})
		absorber.Add("p", Shield{ID: "b", Amount: 20, ExpireMs: 1000, Priority: 1})
		absorber.Add("p", Shield{ID: "c", Amount: 30, ExpireMs: 1000, Priority: 2})
		var absorbed, remaining float64
		for _, step := range []struct {
			damage float64
			now    int64
		}{{25, 0}, {15, 50}, {40, 500}} {
			absorbed, remaining = absorber.Apply("p", step.damage, step.now)
		}
		return absorbed, remaining, absorber.Remaining("p", 500)
	}
	x1, y1, z1 := run()
	x2, y2, z2 := run()
	if x1 != x2 || y1 != y2 || z1 != z2 {
		t.Fatalf("同一伤害序列重复执行结果必须相同：(%v,%v,%v) vs (%v,%v,%v)", x1, y1, z1, x2, y2, z2)
	}
}

func idOf(i int) string {
	return "s" + itoa(i)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}
