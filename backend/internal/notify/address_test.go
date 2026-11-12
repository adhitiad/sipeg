package notify

import "testing"

func TestSplitAddressReturnsPlainAddress(t *testing.T) {
	address, ok := SplitAddress("pegawai@example.test")
	if !ok {
		t.Fatal("alamat polos harus dapat diurai")
	}
	if address != "pegawai@example.test" {
		t.Fatalf("alamat = %s, ingin pegawai@example.test", address)
	}
}

func TestSplitAddressExtractsAddressFromDisplayNameForm(t *testing.T) {
	address, ok := SplitAddress(`"Adhi T" <adhitiadwima@unsub.ac.id>`)
	if !ok {
		t.Fatal("bentuk nama tampilan harus dapat diurai")
	}
	if address != "adhitiadwima@unsub.ac.id" {
		t.Fatalf("alamat = %s, ingin adhitiadwima@unsub.ac.id", address)
	}
}

func TestSplitAddressRejectsUnterminatedDisplayName(t *testing.T) {
	if _, ok := SplitAddress("Adhi <adhitiadwima@unsub.ac.id"); ok {
		t.Fatal("bentuk tanpa penutup kurung sudut harus ditolak")
	}
}

func TestEnvelopeSenderUsesPlainAddress(t *testing.T) {
	got := EnvelopeSender(`"Adhi T" <adhitiadwima@unsub.ac.id>`)
	if got != "adhitiadwima@unsub.ac.id" {
		t.Fatalf("pengirim amplop = %s, ingin adhitiadwima@unsub.ac.id", got)
	}
}

func TestEnvelopeSenderKeepsPlainAddress(t *testing.T) {
	got := EnvelopeSender("pegawai@example.test")
	if got != "pegawai@example.test" {
		t.Fatalf("pengirim amplop = %s, ingin pegawai@example.test", got)
	}
}

func TestValidateAddressAcceptsBothForms(t *testing.T) {
	valid := []string{
		"pegawai@example.test",
		`"Adhi T" <adhitiadwima@unsub.ac.id>`,
		"Adhi T <adhitiadwima@unsub.ac.id>",
	}
	for _, address := range valid {
		if err := ValidateAddress(address); err != nil {
			t.Errorf("alamat %q seharusnya valid: %v", address, err)
		}
	}
}

func TestValidateAddressRejectsMalformedValues(t *testing.T) {
	invalid := []string{
		"",
		"tanpa-domain",
		"@example.test",
		"pegawai@",
		"pegawai@localhost",
		"pegawai @example.test",
		`<adhitiadwima@unsub.ac.id`,
	}
	for _, address := range invalid {
		if err := ValidateAddress(address); err == nil {
			t.Errorf("alamat %q seharusnya ditolak", address)
		}
	}
}
