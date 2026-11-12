package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"simpeg/backend/internal/qstash"
)

// runSchedule menampilkan dan mengelola jadwal berkala pada QStash.
//
// Perkakas ini dipakai operator untuk memastikan jadwal benar-benar
// terdaftar setelah backend dijalankan. Kegagalan pendaftaran jadwal pada
// QStash hanya muncul di log, sehingga tanpa pemeriksaan ini operator tidak
// punya cara memastikan pekerjaan terjadwal berjalan.
func runSchedule(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("schedule", flag.ContinueOnError)
	create := flags.Bool("create", false, "buat jadwal pada tujuan yang diberikan")
	deleteID := flags.String("delete", "", "id jadwal yang dibatalkan")
	destination := flags.String("destination", "", "URL tujuan untuk jadwal baru")
	cron := flags.String("cron", "0 19 * * *", "ekspresi cron untuk jadwal baru")
	retries := flags.Int("retries", 3, "percobaan ulang untuk jadwal baru")
	body := flags.String("body", `{"kind":"audit.cleanup"}`, "isi pesan untuk jadwal baru")
	if err := flags.Parse(args); err != nil {
		return err
	}
	token, err := requireEnv("QSTASH_TOKEN")
	if err != nil {
		return err
	}
	publisher := &qstash.Publisher{
		BaseURL:      envOr("QSTASH_URL", ""),
		SchedulesURL: envOr("QSTASH_SCHEDULES_URL", ""),
		Token:        token,
	}

	if *deleteID != "" {
		if err := publisher.DeleteSchedule(ctx, *deleteID); err != nil {
			return fmt.Errorf("batalkan jadwal: %w", err)
		}
		fmt.Printf("jadwal %s dibatalkan\n", *deleteID)
		return nil
	}

	if *create {
		if strings.TrimSpace(*destination) == "" {
			return fmt.Errorf("tujuan jadwal wajib diisi")
		}
		id, err := publisher.CreateSchedule(ctx, qstash.ScheduleRequest{
			Destination: *destination,
			Body:        *body,
			Cron:        *cron,
			Retries:     *retries,
		})
		if err != nil {
			return fmt.Errorf("buat jadwal: %w", err)
		}
		fmt.Printf("jadwal dibuat: %s\n", id)
		return nil
	}

	schedules, err := publisher.ListSchedules(ctx)
	if err != nil {
		return fmt.Errorf("daftar jadwal: %w", err)
	}
	if len(schedules) == 0 {
		fmt.Println("tidak ada jadwal terdaftar")
		return nil
	}
	fmt.Printf("%d jadwal terdaftar:\n", len(schedules))
	for _, schedule := range schedules {
		fmt.Printf("  %s  cron=%-16s tujuan=%s\n", schedule.ID, schedule.Cron, schedule.Destination)
	}
	return nil
}
