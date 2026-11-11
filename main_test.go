package main

import (
	"github.com/maxmind/mmdbwriter"
	"github.com/oschwald/maxminddb-golang"
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
	tree, err := mmdbwriter.New(mmdbwriter.Options{DatabaseType: "iplist-location", Description: map[string]string{"en": "Test database"}, RecordSize: 28})
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
	province := LocationInfo{CountryCode: "CN", CountryName: "中国", ProvinceCode: "11", ProvinceName: "北京市", ProvinceEnName: "Beijing"}
	if err := loadIPRanges(tree, "data/province.txt", province); err != nil {
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
	db, err := maxminddb.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for ip, want := range map[string]LocationInfo{"1.0.0.1": province, "1.0.0.200": country, "2001:4860::1": country} {
		var got LocationInfo
		if err := db.Lookup(net.ParseIP(ip), &got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("%s: got %+v want %+v", ip, got, want)
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
