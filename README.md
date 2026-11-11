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

CSV 保存国家和省级行政区名称映射；生成器先加载国家数据，再加载省级数据。字段为 `country_code`、`country_name`，省级记录额外包含 `province_code`、`province_name`、`province_en_name`。不是 GeoIP2 标准嵌套结构，消费者应按这些字段读取。当前沿用原生成器对港澳台省级数据写入 CN 的行为。

缺失输入文件、无效 CIDR、写入失败或数据库校验失败会中止构建。部分上游地区可能没有地址，空文件会提示警告。完成后验证数据库结构，遍历全部网络检查字段，并查询每段起始地址核对记录；通过后才替换输出文件。该校验验证结构和一致性，不证明地理位置的现实准确性。

## 自动发布

GitHub Actions 每周一 UTC 00:23（新加坡时间 08:23）运行，也支持手动触发和 main 分支推送。PR 只构建校验，不发布。定时任务由 GitHub 调度，可能延迟。

每次检出上游 master 的同一提交，用该快照构建。Release 标签包含上游和生成器提交的缩写，同一组合不会重复发布（仍运行构建校验）。先创建草稿并上传全部文件，再公开 Release；失败的草稿可在重跑时恢复。`build-info.json` 保存完整提交 SHA 和构建时间，`SHA256SUMS` 校验下载内容。二进制和上游数据不进入 Git 历史。

使用仓库内置 `GITHUB_TOKEN` 的 contents:write 权限，无需额外 Secret。需在仓库设置中允许 GitHub Actions 运行。

数据来源和使用条款请参见上游项目及其 OpenIPDB/IPinfo/bgp.tools 数据源；本项目不额外授予上游数据的使用权。
