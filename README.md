# iplist_mmdb

将 [metowolf/iplist](https://github.com/metowolf/iplist) 发布的 CIDR 数据转换为 MaxMind DB。

## 下载

[最新 Release](https://github.com/cyberphysalis/iplist_mmdb/releases/latest) 提供 `iplist.mmdb`、`build-info.json` 和 `SHA256SUMS`。固定下载地址：

https://github.com/cyberphysalis/iplist_mmdb/releases/latest/download/iplist.mmdb

## 本地构建

需要 Go（版本见 go.mod），在项目根目录执行：

```sh
git clone --depth 1 https://github.com/metowolf/iplist.git upstream
go test ./...
go run . --data-dir upstream/data --output dist/iplist.mmdb
(cd dist && sha256sum iplist.mmdb)
```

缺失输入文件、无效 CIDR、写入失败或数据库校验失败会中止构建。部分上游地区可能没有地址，空文件会提示警告。完成后验证数据库结构，遍历全部网络检查字段，并查询每段起始地址核对记录；通过后才替换输出文件。该校验验证结构和一致性，不证明地理位置的现实准确性。

## GeoIP2 兼容结构

接口使用 GeoIP2 City 嵌套结构。数据库类型为 `GeoIP2-City`，用于 SDK 类型检查；数据仍来自 iplist/OpenIPDB，并非 MaxMind 官方数据库。

```json
{
  "country": {"iso_code": "CN", "names": {"zh-CN": "中国"}},
  "subdivisions": [
    {"iso_code": "GD", "names": {"zh-CN": "广东省", "en": "Guangdong"}}
  ]
}
```

字段定义参考 [MaxMind City binary schema](https://dev.maxmind.com/geoip/docs/databases/city-and-country/city-binary/)，地区编码参考 [UNECE 中国地区表](https://service.unece.org/trade/locode/cn.htm)。

### SDK 验证

Go 使用 `github.com/oschwald/geoip2-golang` 的 City/Country API；构建时用 City API 对照全部网络记录。测试覆盖 IPv4、IPv6、较小省级网段覆盖、纯国家记录及无效输入。

发布前另外使用 MaxMind 官方 Python `geoip2` 的 City API 校验产物里的每一种不同记录，包括多语言名称、缺失字段和省级代码。

```sh
python -m pip install -r requirements-test.txt
python scripts/check_sdk.py dist/iplist.mmdb
```

Python 官方 SDK 对 City 数据库使用 `reader.city(ip).country` 读取国家；`reader.country(ip)` 只接受 Country 类型数据库。Go SDK 的 City/Country API 均已测试。

Python 查询示例：

```python
import geoip2.database
with geoip2.database.Reader("dist/iplist.mmdb") as reader:
    result = reader.city("1.0.1.1")
    print(result.country.iso_code, result.country.names.get("zh-CN"))
    if result.subdivisions:
        print(result.subdivisions[0].iso_code, result.subdivisions[0].names)
```

## 自动发布

GitHub Actions 每周一 UTC 00:23（新加坡时间 08:23）运行。

数据来源和使用条款请参见上游项目及其 OpenIPDB/IPinfo/bgp.tools 数据源；本项目不额外授予上游数据的使用权。
