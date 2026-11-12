package main

import (
	"encoding/csv"
	"github.com/maxmind/mmdbwriter"
	geoip2 "github.com/oschwald/geoip2-golang"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildRoundTrip(t *testing.T) {
	dir := t.TempDir()
	old := *dataDir
	*dataDir = dir
	t.Cleanup(func() { *dataDir = old })
	if err := os.WriteFile(filepath.Join(dir, "ranges.txt"), []byte("1.0.0.0/24\n2001:4860::/32\n"), 0600); err != nil {
		t.Fatal(err)
	}
	tree, err := mmdbwriter.New(mmdbwriter.Options{DatabaseType: "GeoIP2-City", Description: map[string]string{"en": "Test database"}, RecordSize: 28})
	if err != nil {
		t.Fatal(err)
	}
	country := LocationInfo{CountryCode: "CN", CountryName: "中国"}
	if err := loadIPRanges(tree, "data/ranges.txt", country); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "province.txt"), []byte("1.0.0.0/25\n"), 0600); err != nil {
		t.Fatal(err)
	}
	province := LocationInfo{CountryCode: "CN", CountryName: "中国", ProvinceCode: "BJ", ProvinceName: "北京市", ProvinceEnName: "Beijing"}
	if err := loadIPRanges(tree, "data/province.txt", province); err != nil {
		t.Fatal(err)
	}
	city := province
	city.CityName = "北京市"
	if err := os.WriteFile(filepath.Join(dir, "city.txt"), []byte("1.0.0.0/26\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := loadIPRanges(tree, "data/city.txt", city); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "test.mmdb")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tree.WriteTo(f); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := validate(path); err != nil {
		t.Fatal(err)
	}
	db, err := geoip2.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for ip, want := range map[string]LocationInfo{"1.0.0.1": city, "1.0.0.100": province, "1.0.0.200": country, "2001:4860::1": country} {
		got, err := db.City(net.ParseIP(ip))
		if err != nil {
			t.Fatal(err)
		}
		if got.Country.IsoCode != want.CountryCode || got.Country.Names["zh-CN"] != want.CountryName {
			t.Fatalf("country mismatch: %+v", got)
		}
		if want.ProvinceCode == "" {
			if len(got.Subdivisions) != 0 {
				t.Fatal("unexpected subdivision")
			}
		} else if len(got.Subdivisions) != 1 || got.Subdivisions[0].IsoCode != want.ProvinceCode || got.Subdivisions[0].Names["zh-CN"] != want.ProvinceName || got.Subdivisions[0].Names["en"] != want.ProvinceEnName {
			t.Fatalf("subdivision mismatch: %+v", got)
		}
		if got.City.Names["zh-CN"] != want.CityName {
			t.Fatalf("city mismatch: %+v", got.City)
		}
		countryResult, err := db.Country(net.ParseIP(ip))
		if err != nil || countryResult.Country.IsoCode != want.CountryCode {
			t.Fatalf("Country SDK failed: %v", err)
		}

	}
	if err := loadIPRanges(tree, "missing.txt", country); err == nil {
		t.Fatal("accepted missing input")
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.txt"), []byte("not-a-cidr\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := loadIPRanges(tree, "bad.txt", country); err == nil {
		t.Fatal("accepted malformed CIDR")
	}
	if err := validate(filepath.Join(dir, "bad.txt")); err == nil {
		t.Fatal("accepted corrupt database")
	}
}

func TestProvinceISOMapping(t *testing.T) {
	file, err := os.Open("cncity_name_code.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"11": "BJ", "12": "TJ", "13": "HE", "14": "SX", "15": "NM", "21": "LN", "22": "JL", "23": "HL", "31": "SH", "32": "JS", "33": "ZJ", "34": "AH", "35": "FJ", "36": "JX", "37": "SD", "41": "HA", "42": "HB", "43": "HN", "44": "GD", "45": "GX", "46": "HI", "50": "CQ", "51": "SC", "52": "GZ", "53": "YN", "54": "XZ", "61": "SN", "62": "GS", "63": "QH", "64": "NX", "65": "XJ", "71": "TW", "81": "HK", "82": "MO"}
	if len(rows) != len(want)+1 {
		t.Fatal("incomplete province mapping")
	}
	for _, row := range rows[1:] {
		if len(row) != 5 || want[row[2]] == "" || want[row[2]] != row[4] {
			t.Fatalf("invalid mapping: %v", row)
		}
		if row[2] == "53" && row[3] != "Yunnan" {
			t.Fatal("incorrect Yunnan name")
		}
		delete(want, row[2])
	}
	if len(want) != 0 {
		t.Fatalf("missing mappings: %v", want)
	}
}
