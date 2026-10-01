# shield-absorb

护盾吸收：按优先级吸收伤害、过期护盾失效、溢出伤害继续下传，吸收量守恒，只用 Go 标准库。

```
go test ./...
go run ./cmd/shield-absorb --sample overflow
go run ./cmd/shield-absorb --sample expire
go run ./cmd/shield-absorb --sample priority
```

## 口径（README 为准）

- **按优先级下传**：按 `Priority` 升序（同优先级按 `ID` 升序）依次吸收，
  一层吸不完的伤害继续传给下一层，最后剩下多少就返回多少。
- **过期失效**：`now >= ExpireMs` 的护盾不参与吸收，也不计入 `Remaining`。
- **守恒**：`absorbed + remaining == damage`；每层护盾的量不许为负。
- **查询代价**：`Apply` 只碰该玩家的护盾，`Scanned` 不随护盾总数线性增长。
- **规模**：单场 10 万层护盾、每秒 20 万次结算，单次 O(护盾层数) 摊还，内存 O(护盾层数)。

## 输出契约（不改格式）

```
absorbed=60 remaining=40
absorbed=0 remaining=20
a=10 b=20
a=0
```

## 目录

```
shield.go               护盾吸收
cmd/shield-absorb       命令行入口
shield_test.go          go test 用例
```
