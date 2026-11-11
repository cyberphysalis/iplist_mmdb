package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	geoip2 "github.com/oschwald/geoip2-golang"
	"github.com/oschwald/maxminddb-golang"
)

// LocationInfo 存储位置信息
type LocationInfo struct {
	CountryCode    string
	CountryName    string
	ProvinceCode   string
	ProvinceName   string
	ProvinceEnName string
}

var dataDir = flag.String("data-dir", "upstream/data", "upstream data directory")
var output = flag.String("output", "dist/iplist.mmdb", "output database")

func main() {
	flag.Parse()
	// 创建 MMDB writer
	writer, err := mmdbwriter.New(
		mmdbwriter.Options{
			DatabaseType: "GeoIP2-City",
			Description: map[string]string{
				"en": "IP Location Database based on OpenIPDB",
				"zh": "基于 OpenIPDB 的 IP 地理位置数据库",
			},
			RecordSize: 28,
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	// 读取国家数据
	if err := loadCountryData(writer); err != nil {
		log.Fatalf("Failed to load country data: %v", err)
	}

	// 读取中国省级数据
	if err := loadChinaCityData(writer); err != nil {
		log.Fatalf("Failed to load China city data: %v", err)
	}

	// 写入到文件
	if err := os.MkdirAll(filepath.Dir(*output), 0755); err != nil {
		log.Fatal(err)
	}
	fh, err := os.CreateTemp(filepath.Dir(*output), ".mmdb-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(fh.Name())
	defer fh.Close()

	_, err = writer.WriteTo(fh)
	if err != nil {
		log.Fatal(err)
	}

	if err := fh.Close(); err != nil {
		log.Fatal(err)
	}
	if err := validate(fh.Name()); err != nil {
		log.Fatal(err)
	}
	if err := os.Rename(fh.Name(), *output); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Validated and created", *output)
}

// loadCountryData 加载国家级 IP 数据
func loadCountryData(writer *mmdbwriter.Tree) error {
	file, err := os.Open("country_name_code.csv")
	if err != nil {
		return err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}

	// 跳过表头
	for i, record := range records {
		if i == 0 {
			continue
		}
		if len(record) < 3 {
			continue
		}

		code := strings.TrimSpace(record[0])
		name := strings.TrimSpace(record[1])
		path := strings.TrimSpace(record[2])

		if code == "" || path == "" {
			continue
		}

		// 读取该国家的 IP 列表
		if err := loadIPRanges(writer, path, LocationInfo{
			CountryCode: code,
			CountryName: name,
		}); err != nil {
			return fmt.Errorf("load %s: %w", path, err)
		} else {
			fmt.Printf("Loaded country: %s (%s)\n", name, code)
		}
	}

	return nil
}

// loadChinaCityData 加载中国省级 IP 数据
func loadChinaCityData(writer *mmdbwriter.Tree) error {
	file, err := os.Open("cncity_name_code.csv")
	if err != nil {
		return err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}

	// 跳过表头
	for i, record := range records {
		if i == 0 {
			continue
		}
		if len(record) != 5 {
			return fmt.Errorf("invalid province mapping row %d", i+1)
		}

		path := strings.TrimSpace(record[0])
		name := strings.TrimSpace(record[1])
		provinceCode := strings.TrimSpace(record[4])
		enName := strings.TrimSpace(record[3])

		if path == "" || len(provinceCode) != 2 || provinceCode != strings.ToUpper(provinceCode) {
			return fmt.Errorf("invalid province ISO mapping row %d", i+1)
		}

		// 读取该省份的 IP 列表
		if err := loadIPRanges(writer, path, LocationInfo{
			CountryCode:    "CN",
			CountryName:    "中国",
			ProvinceCode:   provinceCode,
			ProvinceName:   name,
			ProvinceEnName: enName,
		}); err != nil {
			return fmt.Errorf("load %s: %w", path, err)
		} else {
			fmt.Printf("Loaded province: %s (%s)\n", name, provinceCode)
		}
	}

	return nil
}

// loadIPRanges 从文件中读取 CIDR 格式的 IP 段并写入数据库
func loadIPRanges(writer *mmdbwriter.Tree, path string, info LocationInfo) error {
	file, err := os.Open(filepath.Join(*dataDir, strings.TrimPrefix(path, "data/")))
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// 解析 CIDR
		_, network, err := net.ParseCIDR(line)
		if err != nil {
			return fmt.Errorf("invalid CIDR %s in %s: %w", line, path, err)
		}

		// 将数据写入 MMDB
		record := mmdbtype.Map{
			"country": mmdbtype.Map{"iso_code": mmdbtype.String(info.CountryCode), "names": mmdbtype.Map{"zh-CN": mmdbtype.String(info.CountryName)}},
		}
		if info.ProvinceCode != "" {
			record["subdivisions"] = mmdbtype.Slice{mmdbtype.Map{
				"iso_code": mmdbtype.String(info.ProvinceCode),
				"names":    mmdbtype.Map{"zh-CN": mmdbtype.String(info.ProvinceName), "en": mmdbtype.String(info.ProvinceEnName)},
			}}
		}

		if err := writer.Insert(network, record); err != nil {
			return fmt.Errorf("insert %s: %w", line, err)
		}

		count++
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if count == 0 {
		log.Printf("Warning: no IP ranges loaded from %s", path)
	}

	return nil
}

// Validate every stored network, not a fixed set of potentially changing IP locations.
func validate(path string) error {
	db, err := maxminddb.Open(path)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.Verify(); err != nil {
		return err
	}
	if db.Metadata.DatabaseType != "GeoIP2-City" {
		return fmt.Errorf("unexpected database type")
	}
	sdk, err := geoip2.Open(path)
	if err != nil {
		return err
	}
	defer sdk.Close()
	networks := db.Networks()
	count := 0
	for networks.Next() {
		var record geoip2.City
		network, err := networks.Network(&record)
		if err != nil {
			return err
		}
		if len(record.Country.IsoCode) != 2 || record.Country.Names["zh-CN"] == "" {
			return fmt.Errorf("invalid country record at %s", network)
		}
		for _, subdivision := range record.Subdivisions {
			if len(subdivision.IsoCode) != 2 || subdivision.Names["zh-CN"] == "" || subdivision.Names["en"] == "" {
				return fmt.Errorf("incomplete province at %s", network)
			}
		}
		found, err := sdk.City(network.IP)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(*found, record) {
			return fmt.Errorf("SDK lookup mismatch at %s", network)
		}

		count++
	}
	if err := networks.Err(); err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("empty database")
	}
	fmt.Printf("Validated %d networks\n", count)
	return nil
}
