// Package domain memuat aturan Service Level Objective: definisi target,
// agregasi metrik per rute, dan evaluasi pelanggaran target.
//
// Paket ini tidak melakukan I/O. Aturan di sini dipakai bersama oleh
// pengoleki metrik, lapisan aplikasi, dan lapisan repository.
package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// BucketBoundaries adalah batas atas tiap bucket latensi dalam milidetik.
//
// Bucket disimpan sebagai hitungan kumulatif sehingga persentil dapat
// dihitung ulang dari satu baris metrik. Ambang 800 ms sengaja ikut
// disertakan karena angka itu target SLO bawaan: dengan ambang tersebut,
// keputusan pelanggaran pada target 800 ms tidak pernah meleset ke bawah.
//
// Nilai terakhir adalah batas atas yang masih terukur; sampel yang lebih
// lambat masuk ke bucket luapan.
var BucketBoundaries = [...]time.Duration{
	5 * time.Millisecond,
	10 * time.Millisecond,
	25 * time.Millisecond,
	50 * time.Millisecond,
	100 * time.Millisecond,
	200 * time.Millisecond,
	400 * time.Millisecond,
	800 * time.Millisecond,
	1200 * time.Millisecond,
	1600 * time.Millisecond,
	2400 * time.Millisecond,
	3200 * time.Millisecond,
	4800 * time.Millisecond,
	6400 * time.Millisecond,
}

// BucketOverflow adalah indeks bucket yang menampung sampel di atas batas
// terakhir.
const BucketOverflow = len(BucketBoundaries)

// Metric adalah agregasi metrik untuk satu rute pada satu jendela waktu.
type Metric struct {
	// Route adalah pola rute, bukan path mentah.
	Route string
	// Method adalah metode HTTP.
	Method string
	// Requests adalah jumlah permintaan pada jendela.
	Requests int64
	// Errors adalah jumlah permintaan yang berakhir dengan galat 5xx.
	Errors int64
	// LatencySum adalah jumlah latensi seluruh sampel.
	LatencySum time.Duration
	// LatencyMax adalah latensi tertinggi.
	LatencyMax time.Duration
	// Buckets menyimpan hitungan kumulatif permintaan dengan latensi tidak
	// lebih dari BucketBoundaries, diakhiri bucket luapan pada indeks
	// BucketOverflow.
	Buckets []int64
}

// Target adalah batas SLO yang harus dipenuhi.
type Target struct {
	// P95 adalah latensi persentil 95 maksimum yang boleh terlampaui.
	P95 time.Duration
	// ErrorRate adalah laju kesalahan maksimum yang boleh terlampaui,
	// dinyatakan sebagai pecahan 0 sampai 1.
	ErrorRate float64
	// MinRequests adalah jumlah permintaan minimum pada jendela sebelum
	// hasilnya dianggap bermakna. Jendela dengan permintaan lebih sedikit
	// dianggap tidak melanggar target.
	MinRequests int64
}

// Window adalah rentang waktu agregasi metrik.
type Window struct {
	Start time.Time
	End   time.Time
}

// RouteStat adalah statistik satu rute pada jendela yang dievaluasi.
type RouteStat struct {
	Route     string
	Method    string
	Requests  int64
	Errors    int64
	ErrorRate float64
	P95       time.Duration
	P99       time.Duration
}

// BreachKind menandai jenis pelanggaran target.
type BreachKind string

const (
	// BreachLatency menandai latensi persentil 95 melewati target.
	BreachLatency BreachKind = "latency"
	// BreachErrorRate menandai laju kesalahan melewati target.
	BreachErrorRate BreachKind = "error_rate"
)

// Breach adalah satu pelanggaran target pada sebuah rute.
type Breach struct {
	Route    string
	Method   string
	Kind     BreachKind
	Observed string
	Target   string
}

// Evaluation adalah hasil evaluasi target pada satu jendela.
type Evaluation struct {
	Window    Window
	Requests  int64
	Errors    int64
	ErrorRate float64
	// P95 adalah perkiraan persentil 95 hasil interpolasi.
	P95 time.Duration
	// P99 adalah perkiraan persentil 99 hasil interpolasi.
	P99 time.Duration
	// P95Bound adalah batas atas bucket yang memuat persentil 95. Nilai ini
	// tidak pernah lebih kecil daripada persentil 95 yang sebenarnya dan
	// dipakai untuk memutuskan pelanggaran target.
	P95Bound time.Duration
	Routes   []RouteStat
	Breaches []Breach
}

// OK melapor apakah target terpenuhi pada jendela ini.
func (e Evaluation) OK() bool { return len(e.Breaches) == 0 }

// NewMetric membuat metrik kosong untuk satu rute.
func NewMetric(route, method string) Metric {
	return Metric{
		Route:   route,
		Method:  method,
		Buckets: make([]int64, BucketOverflow+1),
	}
}

// Observe menambahkan satu sampel ke metrik.
//
// failed menandai permintaan yang berakhir dengan galat server. Sampel
// dengan latensi di atas batas terakhir tetap dihitung pada jumlah
// permintaan dan bucket luapan.
func (m *Metric) Observe(latency time.Duration, failed bool) {
	m.Requests++
	if failed {
		m.Errors++
	}
	m.LatencySum += latency
	if latency > m.LatencyMax {
		m.LatencyMax = latency
	}
	index := BucketOverflow
	for position, boundary := range BucketBoundaries {
		if latency <= boundary {
			index = position
			break
		}
	}
	for position := index; position < len(m.Buckets); position++ {
		m.Buckets[position]++
	}
}

// ErrorRate menghitung laju kesalahan metrik.
func (m Metric) ErrorRate() float64 {
	if m.Requests == 0 {
		return 0
	}
	return float64(m.Errors) / float64(m.Requests)
}

// Quantile menghitung perkiraan persentil dengan interpolasi linier di
// dalam bucket.
//
// Perkiraan ini dapat lebih kecil daripada nilai sebenarnya karena
// asumsi sebaran merata di dalam bucket. Untuk keputusan pelanggaran target
// gunakanlah QuantileUpper.
func (m Metric) Quantile(fraction float64) time.Duration {
	if m.Requests == 0 {
		return 0
	}
	if len(m.Buckets) == 0 {
		return m.LatencyMax
	}
	rank := fraction * float64(m.Requests)
	for index, cumulative := range m.Buckets {
		if float64(cumulative) < rank {
			continue
		}
		if index >= BucketOverflow {
			return BucketBoundaries[BucketOverflow-1]
		}
		lower := time.Duration(0)
		if index > 0 {
			lower = BucketBoundaries[index-1]
		}
		upper := BucketBoundaries[index]
		below := int64(0)
		if index > 0 {
			below = m.Buckets[index-1]
		}
		if cumulative == below {
			return upper
		}
		weight := (rank - float64(below)) / float64(cumulative-below)
		return lower + time.Duration(weight*float64(upper-lower))
	}
	return m.LatencyMax
}

// QuantileUpper mengembalikan batas atas bucket yang memuat persentil.
//
// Nilai ini tidak pernah lebih kecil daripada persentil sebenarnya, sehingga
// keputusan pelanggaran target tidak pernah lolos karena perkiraan yang
// terlalu kecil.
func (m Metric) QuantileUpper(fraction float64) time.Duration {
	if m.Requests == 0 {
		return 0
	}
	if len(m.Buckets) == 0 {
		return m.LatencyMax
	}
	rank := fraction * float64(m.Requests)
	for index, cumulative := range m.Buckets {
		if float64(cumulative) < rank {
			continue
		}
		if index >= BucketOverflow {
			return BucketBoundaries[BucketOverflow-1]
		}
		return BucketBoundaries[index]
	}
	return m.LatencyMax
}

// Merge menggabungkan beberapa metrik menjadi satu metrik gabungan.
//
// Metrik gabungan dipakai untuk menilai target SLO tingkat layanan secara
// keseluruhan, bukan per rute.
func Merge(metrics []Metric) Metric {
	merged := NewMetric("", "")
	for _, metric := range metrics {
		merged.Requests += metric.Requests
		merged.Errors += metric.Errors
		merged.LatencySum += metric.LatencySum
		if metric.LatencyMax > merged.LatencyMax {
			merged.LatencyMax = metric.LatencyMax
		}
		for index := range merged.Buckets {
			if index < len(metric.Buckets) {
				merged.Buckets[index] += metric.Buckets[index]
			}
		}
	}
	return merged
}

// Evaluate menilai metrik sebuah jendela terhadap target.
//
// Pelanggaran dilaporkan bila target terlampaui dan jendela cukup banyak
// permintaan untuk dianggap bermakna. Keputusan latensi memakai batas atas
// bucket sehingga tidak pernah lolos karena perkiraan yang terlalu kecil.
func Evaluate(window Window, metrics []Metric, target Target) Evaluation {
	merged := Merge(metrics)
	evaluation := Evaluation{
		Window:    window,
		Requests:  merged.Requests,
		Errors:    merged.Errors,
		ErrorRate: merged.ErrorRate(),
		P95:       merged.Quantile(0.95),
		P99:       merged.Quantile(0.99),
		P95Bound:  merged.QuantileUpper(0.95),
	}
	if merged.Requests < target.MinRequests {
		return evaluation
	}

	if evaluation.P95Bound > target.P95 {
		evaluation.Breaches = append(evaluation.Breaches, Breach{
			Route:    "*",
			Kind:     BreachLatency,
			Observed: FormatDuration(evaluation.P95Bound),
			Target:   FormatDuration(target.P95),
		})
	}
	if evaluation.ErrorRate > target.ErrorRate {
		evaluation.Breaches = append(evaluation.Breaches, Breach{
			Route:    "*",
			Kind:     BreachErrorRate,
			Observed: FormatPercent(evaluation.ErrorRate),
			Target:   FormatPercent(target.ErrorRate),
		})
	}

	routes := make([]RouteStat, 0, len(metrics))
	for _, metric := range metrics {
		routes = append(routes, RouteStat{
			Route:     metric.Route,
			Method:    metric.Method,
			Requests:  metric.Requests,
			Errors:    metric.Errors,
			ErrorRate: metric.ErrorRate(),
			P95:       metric.Quantile(0.95),
			P99:       metric.Quantile(0.99),
		})
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].P95 != routes[j].P95 {
			return routes[i].P95 > routes[j].P95
		}
		return routes[i].Route < routes[j].Route
	})
	evaluation.Routes = routes
	return evaluation
}

// FormatDuration merumuskan durasi dalam satuan yang mudah dibaca.
func FormatDuration(value time.Duration) string {
	switch {
	case value < time.Millisecond:
		return fmt.Sprintf("%.2f ms", float64(value.Microseconds())/1000)
	case value < time.Second:
		return fmt.Sprintf("%.0f ms", float64(value.Microseconds())/1000)
	default:
		return fmt.Sprintf("%.2f s", value.Seconds())
	}
}

// FormatPercent merumuskan pecahan sebagai persentase.
func FormatPercent(value float64) string {
	return fmt.Sprintf("%.2f%%", value*100)
}

// Describe merumuskan ringkasan evaluasi untuk ditampilkan pada alert.
func (e Evaluation) Describe() string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "Jendela %s sampai %s\n",
		e.Window.Start.UTC().Format(time.RFC3339),
		e.Window.End.UTC().Format(time.RFC3339))
	fmt.Fprintf(&builder, "Permintaan %d, galat %d (%s)\n",
		e.Requests, e.Errors, FormatPercent(e.ErrorRate))
	fmt.Fprintf(&builder, "Latensi p95 %s, p99 %s\n",
		FormatDuration(e.P95), FormatDuration(e.P99))
	return builder.String()
}
