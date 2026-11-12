// Package jobs mendaftarkan pekerjaan asinkron yang dijalankan lewat callback
// QStash.
//
// QStash bertindak sebagai penjadwal: ia memanggil endpoint callback backend
// secara berkala sesuai jadwal cron, dan melakukan percobaan ulang bila
// callback gagal. Pekerjaan yang dijalankan harus idempoten karena QStash
// tidak menjamin pengiriman tepat satu kali.
package jobs

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"simpeg/backend/internal/qstash"
)

// CronAuditCleanup adalah jadwal harian pembersihan jejak audit.
//
// Waktu cron dihitung dalam UTC. "19:00 UTC" setara dengan 02:00 WIB.
const CronAuditCleanup = "0 19 * * *"

// CronPayrollNotify adalah jadwal harian pemberitahuan ketersediaan slip
// gaji H+1.
//
// Waktu cron dihitung dalam UTC. "19:05 UTC" setara dengan 02:05 WIB,
// sehingga slip dengan tanggal gajian kemarin selalu siap pada saat
// pemberitahuan berjalan.
const CronPayrollNotify = "5 19 * * *"

// CronSLOEvaluate adalah jadwal berkala pemeriksaan target SLO.
//
// Pemeriksaan berjalan tiap lima menit agar alert latensi dan laju
// kesalahan dikirim relatif cepat setelah target terlampaui. Jadwal memakai
// ekspresi cron yang didukung QStash, yaitu satu field menit dan satu field
// jam.
const CronSLOEvaluate = "*/5 * * * *"

// ErrScheduleUnavailable menandai jadwal tidak dapat didaftarkan.
var ErrScheduleUnavailable = errors.New("qstash schedule is unavailable")

// EnsureSchedule mendaftarkan jadwal berkala bila belum ada.
//
// Idempoten: bila jadwal dengan destination yang sama sudah terdaftar, id
// jadwal tersebut dikembalikan tanpa membuat jadwal baru. Ini mencegah
// penjadwalan ganda ketika backend dinyalakan ulang.
func EnsureSchedule(ctx context.Context, publisher *qstash.Publisher, request qstash.ScheduleRequest) (string, error) {
	if publisher == nil {
		return "", fmt.Errorf("%w: publisher tidak dikonfigurasi", ErrScheduleUnavailable)
	}
	schedules, err := publisher.ListSchedules(ctx)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrScheduleUnavailable, err.Error())
	}
	for _, schedule := range schedules {
		if schedule.Destination == request.Destination {
			return schedule.ID, nil
		}
	}
	id, err := publisher.CreateSchedule(ctx, request)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrScheduleUnavailable, err.Error())
	}
	return id, nil
}

// Destination menyusun URL callback untuk sebuah path internal.
func Destination(baseURL, path string) string {
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/")
}
