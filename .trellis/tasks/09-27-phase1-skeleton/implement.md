# Implement — Phase 1 执行清单

顺序执行，每步可编译：

1. [ ] go.mod + config + logger + shared(errors/ids/pagination/tx/respond) + 单测
2. [ ] migrations 001-008 全部 SQL + migrate runner + migrate_test（临时库）
3. [ ] database.go（连接池参数：MaxOpen=20, MaxIdle=5, Lifetime=30m）
4. [ ] middleware(requestid/logging/recover) + router(/healthz) + main.go 装配 + config.example.yaml
5. [ ] 8 个模块目录：AGENTS.md + api.go + module.go 空挂载
6. [ ] 验证：`go build ./... && go vet ./... && go test ./...`；对 anmo 库跑 migrate 幂等两次；curl /healthz
7. [ ] check 自查 → spec 更新评估 → commit → archive → journal PHASE RESULT

回滚点：每步一个 commit；migration 只增不改。
