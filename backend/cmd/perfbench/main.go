// Command perfbench menjalankan pengukuran performa SIMPEG terhadap
// dependensi nyata: PostgreSQL, Redis, QStash, dan SMTP.
//
// Alat ini adalah perkakas operator, bukan bagian dari jalur permintaan
// aplikasi. Seluruh kredensial dibaca dari environment sehingga tidak ada
// nilai rahasia yang ditulis di dalam repo.
package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	ctx := context.Background()
	var err error
	switch os.Args[1] {
	case "seed":
		err = runSeed(ctx, os.Args[2:])
	case "schema":
		err = runSchema(ctx, os.Args[2:])
	case "explain":
		err = runExplain(ctx, os.Args[2:])
	case "latency":
		err = runLatency(ctx, os.Args[2:])
	case "e2e":
		err = runE2E(ctx, os.Args[2:])
	case "deps":
		err = runDeps(ctx, os.Args[2:])
	case "slo":
		err = runSLO(ctx, os.Args[2:])
	case "schedule":
		err = runSchedule(ctx, os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "perfbench: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `perfbench - pengukuran performa SIMPEG

Pakai:
  perfbench seed     --employees=5000 --months=6 --audit=50000
  perfbench schema   [--verbose]
  perfbench explain  [--scenario=nama]
  perfbench latency  [--iterations=50]
  perfbench e2e      [--duration=60s] [--streams=16]
  perfbench deps     [--redis] [--qstash] [--smtp] [--count=100]
  perfbench slo      [--url=http://127.0.0.1:8080]
  perfbench schedule [--create --destination=URL --cron=...] [--delete=ID]

Kredensial dibaca dari environment: DATABASE_URL, UPSTASH_REDIS_REST_URL,
UPSTASH_REDIS_REST_TOKEN, QSTASH_URL, QSTASH_TOKEN, dan SMTP_*.
`)
}
