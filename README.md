# iplist_mmdb

将 [metowolf/iplist](https://github.com/metowolf/iplist) 发布的 CIDR 数据转换为 MaxMind DB。独立维护生成器，不需要同步上游 Git 历史，也不需要上游 S3 凭据。

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

## GeoIP2 兼容结构（破坏性变更）

新版本移除旧的扁平字段，改用 GeoIP2 City 嵌套结构。数据库类型为 `GeoIP2-City`，用于 SDK 类型检查；数据仍来自 iplist/OpenIPDB，并非 MaxMind 官方数据库。

```json
{
  "country": {"iso_code": "CN", "names": {"zh-CN": "中国"}},
  "subdivisions": [
    {"iso_code": "GD", "names": {"zh-CN": "广东省", "en": "Guangdong"}}
  ]
}
```

迁移：`country_code` → `country.iso_code`，`country_name` → `country.names["zh-CN"]`，省级名称 → `subdivisions[0].names`。`province_code` 原为国内代码（如 44），现在为 ISO 3166-2 后缀（如 GD），不能按旧编码解释。只有国家信息时省级数组字段省略。缺失的城市、经纬度、时区、GeoNames ID、注册国家不伪造填充。

`cncity_name_code.csv` 保留国内代码并新增 `iso_code` 列，覆盖 34 个省级映射。沿用原数据处理规则：省级数据覆盖国家数据，港澳台省级记录为 `country.iso_code=CN`、省级 `TW/HK/MO`。这是本项目的数据归类策略，不代表与 MaxMind 官方数据库的归属判断一致。

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

缺失输入文件、无效 CIDR、写入失败或数据库校验失败会中止构建。部分上游地区可能没有地址，空文件会提示警告。完成后验证数据库结构，遍历全部网络检查字段，并查询每段起始地址核对记录；通过后才替换输出文件。该校验验证结构和一致性，不证明地理位置的现实准确性。

## 自动发布

GitHub Actions 每周一 UTC 00:23（新加坡时间 08:23）运行，也支持手动触发和 main 分支推送。PR 只构建校验，不发布。定时任务由 GitHub 调度，可能延迟。

每次检出上游 master 的同一提交，用该快照构建。Release 标签包含上游和生成器提交的缩写，同一组合不会重复发布（仍运行构建校验）。先创建草稿并上传全部文件，再公开 Release；失败的草稿可在重跑时恢复。`build-info.json` 保存完整提交 SHA 和构建时间，`SHA256SUMS` 校验下载内容。二进制和上游数据不进入 Git 历史。

使用仓库内置 `GITHUB_TOKEN` 的 contents:write 权限，无需额外 Secret。需在仓库设置中允许 GitHub Actions 运行。

数据来源和使用条款请参见上游项目及其 OpenIPDB/IPinfo/bgp.tools 数据源；本项目不额外授予上游数据的使用权。
