// Package pdf merender dokumen PDF untuk kebutuhan internal aplikasi.
//
// Font Tinos (SIL Open Font License) di-embed ke dalam binary agar
// hasil render identik di seluruh lingkungan (Windows, Linux, Docker).
// Lihat assets/OFL.txt untuk lisensi font.
package pdf

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"

	"github.com/signintech/gopdf"
	"github.com/skip2/go-qrcode"
)

//go:embed assets/Tinos-Regular.ttf
var tinosRegular []byte

//go:embed assets/Tinos-Bold.ttf
var tinosBold []byte

const (
	fontRegular = "Tinos"
	fontBold    = "Tinos-Bold"
	pageWidth   = 420.0
	pageHeight  = 595.0
	marginX     = 34.0
	contentW    = pageWidth - 2*marginX
)

var (
	colorSea      = [3]uint8{0x17, 0x3a, 0x40}
	colorLagoon   = [3]uint8{0x2f, 0x6a, 0x4a}
	colorSand     = [3]uint8{0xe8, 0xec, 0xe3}
	colorGold     = [3]uint8{0xc9, 0xa2, 0x27}
	colorSeaSoft  = [3]uint8{0x41, 0x61, 0x66}
	colorWhite    = [3]uint8{0xff, 0xff, 0xff}
	colorHairline = [3]uint8{0xd5, 0xd9, 0xd1}
)

// SlipRow adalah satu baris penerimaan atau potongan pada slip.
type SlipRow struct {
	Jenis   string
	Nominal string
}

// SlipData memuat seluruh isi slip gaji yang dirender ke PDF.
type SlipData struct {
	OrgName        string
	NomorSlip      string
	Nama           string
	PeriodeLabel   string
	TanggalGaji    string
	StatusLabel    string
	GajiPokok      string
	Pendapatan     []SlipRow
	Potongan       []SlipRow
	TotalGaji      string
	TotalPotongan  string
	GajiBersih     string
	KodeVerifikasi string
	VerifyURL      string
}

// RenderSlip merender slip gaji menjadi dokumen PDF satu halaman ukuran A5.
func RenderSlip(data SlipData) ([]byte, error) {
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA5})
	pdf.AddPage()
	if err := pdf.AddTTFFontData(fontRegular, tinosRegular); err != nil {
		return nil, fmt.Errorf("register regular font: %w", err)
	}
	if err := pdf.AddTTFFontData(fontBold, tinosBold); err != nil {
		return nil, fmt.Errorf("register bold font: %w", err)
	}
	if err := pdf.SetFont(fontRegular, "", 10); err != nil {
		return nil, fmt.Errorf("set font: %w", err)
	}

	r := slipRenderer{pdf: &pdf}
	r.masthead(data)
	r.identity(data)
	r.columns(data)
	r.takeHome(data)
	r.footer(data)

	var buf bytes.Buffer
	if _, err := pdf.WriteTo(&buf); err != nil {
		return nil, fmt.Errorf("write pdf: %w", err)
	}
	return buf.Bytes(), nil
}

type slipRenderer struct {
	pdf *gopdf.GoPdf
	y   float64
}

func (r *slipRenderer) setText(rgb [3]uint8)   { r.pdf.SetTextColor(rgb[0], rgb[1], rgb[2]) }
func (r *slipRenderer) setFill(rgb [3]uint8)   { r.pdf.SetFillColor(rgb[0], rgb[1], rgb[2]) }
func (r *slipRenderer) setStroke(rgb [3]uint8) { r.pdf.SetStrokeColor(rgb[0], rgb[1], rgb[2]) }

func (r *slipRenderer) rect(x0, y0, x1, y1 float64, fill bool) error {
	style := "S"
	if fill {
		style = "F"
	}
	return r.pdf.Rectangle(x0, y0, x1, y1, style, 0, 0)
}

func (r *slipRenderer) line(x1, y1, x2, y2 float64) {
	r.pdf.SetLineWidth(0.75)
	r.pdf.Line(x1, y1, x2, y2)
}

func (r *slipRenderer) text(x, y float64, family string, size float64, rgb [3]uint8, text string) error {
	if err := r.pdf.SetFont(family, "", size); err != nil {
		return err
	}
	r.setText(rgb)
	r.pdf.SetXY(x, y)
	return r.pdf.Text(clip(text, 96))
}

func (r *slipRenderer) right(y float64, rightX float64, family string, size float64, rgb [3]uint8, text string) error {
	if err := r.pdf.SetFont(family, "", size); err != nil {
		return err
	}
	r.setText(rgb)
	width, err := r.pdf.MeasureTextWidth(text)
	if err != nil {
		width = 0
	}
	r.pdf.SetXY(rightX-width, y)
	return r.pdf.Text(text)
}

func (r *slipRenderer) masthead(data SlipData) {
	org := strings.TrimSpace(data.OrgName)
	if org == "" {
		org = "SIMPEG"
	}
	r.setFill(colorSea)
	_ = r.rect(0, 0, pageWidth, 92, true)
	_ = r.text(marginX, 22, fontRegular, 8, colorGold, "DOKUMEN RESMI — BERSIFAT RAHASIA")
	_ = r.text(marginX, 38, fontBold, 11, colorWhite, strings.ToUpper(org))
	_ = r.text(marginX, 56, fontBold, 15, colorWhite, "S L I P   G A J I")

	_ = r.right(26, pageWidth-marginX, fontRegular, 7.5, colorWhite, "Periode")
	_ = r.right(40, pageWidth-marginX, fontBold, 10.5, colorWhite, data.PeriodeLabel)
	_ = r.right(58, pageWidth-marginX, fontRegular, 7.5, colorWhite, "Tanggal gaji")
	_ = r.right(72, pageWidth-marginX, fontBold, 8.5, colorWhite, data.TanggalGaji)

	r.setFill(colorGold)
	_ = r.rect(0, 92, pageWidth, 96, true)
	r.y = 108
}

func (r *slipRenderer) identity(data SlipData) {
	idW := contentW
	r.setFill(colorSand)
	_ = r.rect(marginX, r.y, marginX+idW, r.y+52, true)

	row := func(label, value string, x, y float64) {
		_ = r.text(x+8, y, fontRegular, 6.5, colorSeaSoft, strings.ToUpper(label))
		_ = r.text(x+8, y+11, fontBold, 9, colorSea, value)
	}
	row("Nomor slip", data.NomorSlip, marginX, r.y+6)
	row("Status", data.StatusLabel, marginX+idW/2, r.y+6)
	row("Nama pegawai", data.Nama, marginX, r.y+28)
	r.y += 64
}

func (r *slipRenderer) columns(data SlipData) {
	colW := (contentW - 16) / 2
	leftX := marginX
	rightX := marginX + colW + 16

	header := func(x float64, title string) {
		_ = r.text(x, r.y, fontBold, 8.5, colorGold, strings.ToUpper(title))
		r.setStroke(colorGold)
		r.line(x, r.y+10, x+colW, r.y+10)
	}

	headerY := r.y
	header(leftX, "Penerimaan")
	header(rightX, "Potongan")

	pendapatan := append([]SlipRow{{Jenis: "Gaji pokok", Nominal: data.GajiPokok}}, data.Pendapatan...)
	potongan := data.Potongan

	rowY := headerY + 22
	maxRows := len(pendapatan)
	if len(potongan) > maxRows {
		maxRows = len(potongan)
	}
	for i := 0; i < maxRows; i++ {
		if i < len(pendapatan) {
			r.tableRow(leftX, colW, rowY, pendapatan[i].Jenis, pendapatan[i].Nominal, false)
		}
		if i < len(potongan) {
			r.tableRow(rightX, colW, rowY, potongan[i].Jenis, potongan[i].Nominal, false)
		}
		rowY += 15
	}

	r.line(leftX, rowY-2, leftX+colW, rowY-2)
	r.line(rightX, rowY-2, rightX+colW, rowY-2)
	r.tableRow(leftX, colW, rowY+2, "Total penerimaan", data.TotalGaji, true)
	r.tableRow(rightX, colW, rowY+2, "Total potongan", data.TotalPotongan, true)

	r.y = rowY + 34
}

func (r *slipRenderer) tableRow(x, colW, y float64, jenis, nominal string, bold bool) {
	family := fontRegular
	if bold {
		family = fontBold
	}
	_ = r.text(x, y, family, 8, colorSea, jenis)
	_ = r.right(y, x+colW, family, 8, colorSea, nominal)
}

func (r *slipRenderer) takeHome(data SlipData) {
	bandY := r.y
	r.setFill(colorLagoon)
	_ = r.rect(marginX, bandY, marginX+contentW, bandY+30, true)
	_ = r.text(marginX+14, bandY+10, fontBold, 9, colorWhite, "GAJI BERSIH")
	_ = r.right(bandY+7, marginX+contentW-14, fontBold, 14, colorWhite, data.GajiBersih)
	r.y = bandY + 44
}

func (r *slipRenderer) footer(data SlipData) {
	footerY := pageHeight - 84
	r.setStroke(colorHairline)
	r.line(marginX, footerY-6, marginX+contentW, footerY-6)

	qrText := strings.TrimSpace(data.VerifyURL)
	if qrText != "" {
		r.qr(qrText, marginX, footerY, 66)
		textX := marginX + 78
		_ = r.text(textX, footerY+4, fontBold, 8, colorSea, "Verifikasi slip ini")
		_ = r.text(textX, footerY+16, fontRegular, 6.5, colorSeaSoft, "Pindai QR di samping atau buka:")
		r.wrappedText(textX, footerY+28, contentW-78, 8.5, fontRegular, 6.5, colorSeaSoft, qrText)
	} else if kode := strings.TrimSpace(data.KodeVerifikasi); kode != "" {
		_ = r.text(marginX, footerY+8, fontRegular, 8, colorSeaSoft, "Kode verifikasi: "+kode)
	}

	_ = r.text(marginX, footerY+66, fontRegular, 6, colorSeaSoft,
		"Simpan dengan aman; dilarang menyebarluaskan isi slip gaji. SIMPEG — dicetak otomatis.")
}

func (r *slipRenderer) qr(text string, x, y float64, size float64) {
	code, err := qrcode.New(text, qrcode.Medium)
	if err != nil {
		return
	}
	_ = r.pdf.ImageFrom(code.Image(int(size)), x, y, &gopdf.Rect{W: size, H: size})
}

func (r *slipRenderer) wrappedText(x, y, maxWidth float64, step float64, family string, size float64, rgb [3]uint8, text string) {
	words := strings.Fields(text)
	line := ""
	cy := y
	for _, w := range words {
		candidate := strings.TrimSpace(line + " " + w)
		if family != "" && line != "" {
			if err := r.pdf.SetFont(family, "", size); err != nil {
				return
			}
			width, err := r.pdf.MeasureTextWidth(candidate)
			if err == nil && width > maxWidth {
				_ = r.text(x, cy, family, size, rgb, line)
				line = w
				cy += step
				continue
			}
		}
		line = candidate
	}
	if line != "" {
		_ = r.text(x, cy, family, size, rgb, line)
	}
}

func clip(text string, max int) string {
	text = strings.TrimSpace(text)
	text = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' {
			return ' '
		}
		return r
	}, text)
	runes := []rune(text)
	if len(runes) > max {
		return string(runes[:max]) + "..."
	}
	return text
}
