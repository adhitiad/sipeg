// Package migrations mengekspos berkas migrasi SQL Goose sebagai embedded
// filesystem.
//
// Perintah migrate (cmd/migrate) memakai FS ini supaya container migrasi
// tidak perlu memasang direktori migrations sebagai volume dan berkas SQL
// tidak bisa tertinggal dari image.
package migrations

import "embed"

//go:embed *.sql
var Files embed.FS
