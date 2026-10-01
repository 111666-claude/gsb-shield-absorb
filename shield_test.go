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
