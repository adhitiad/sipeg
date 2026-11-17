## Ringkasan

<!-- Jelaskan perubahan singkat dan motivasinya. -->

## Checklist

- [ ] `gofmt -l .` bersih (backend) / `bun run format:check` (frontend)
- [ ] `go vet ./...` lulus
- [ ] `go test ./...` lulus / `bun run test` lulus
- [ ] Tidak ada kredensial/PII di diff
- [ ] Dokumentasi (README/OpenAPI/RUNBOOK) diperbarui bila relevan
- [ ] Migrasi Goose berversi (bila ada perubahan skema), tidak memakai AutoMigrate
- [ ] Permission/grant baru tercatat di docs/PRD.md §11 dan backend/cmd/server/policy.go

## Bukti

<!-- Screenshot / log / hasil uji -->