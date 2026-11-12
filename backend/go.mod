module simpeg/backend

go 1.27.4

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.13.0
	github.com/pressly/goose/v3 v3.30.0
	github.com/rabbitmq/amqp091-go v1.16.0
	github.com/richardlehane/mscfb v1.0.9
	github.com/robfig/cron/v3 v3.0.1
	github.com/shakinm/xlsReader v0.9.12
	github.com/xuri/excelize/v2 v2.11.1
	gorm.io/driver/postgres v1.6.3
	gorm.io/gorm v1.31.2
)

// Repositori github.com/metakeule/fmtdate memuat go.mod yang masih
// menyatakan jalur lama gitlab.com/metakeule/fmtdate. Replace menunjuk
// langsung ke jalur yang sah agar modul dapat diunduh.
replace github.com/metakeule/fmtdate => gitlab.com/metakeule/fmtdate v1.2.2

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.3 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/metakeule/fmtdate v1.2.2 // indirect
	github.com/mfridman/interpolate v0.0.2 // indirect
	github.com/richardlehane/msoleps v1.0.6 // indirect
	github.com/sethvargo/go-retry v0.5.0 // indirect
	github.com/tiendc/go-deepcopy v1.7.2 // indirect
	github.com/xuri/efp v0.0.3-0.20260924042702-217b81952dec // indirect
	github.com/xuri/nfp v0.0.2-0.20250530014748-2ddeb826f9a9 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.59.0 // indirect
	golang.org/x/net v0.62.0 // indirect
	golang.org/x/sync v0.25.0 // indirect
	golang.org/x/text v0.44.0 // indirect
)
