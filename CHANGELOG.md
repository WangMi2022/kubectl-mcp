# Changelog

## [1.3.0] - 2026-01-30

### Added
- 新增 `get_pod_filter` 工具，支持在所有命名空间中快速查找Pod
  - 支持精确匹配模式（exact）：用于删除等精确操作
  - 支持模糊匹配模式（fuzzy）：用于搜索相关Pod
  - 自动返回Pod所在的命名空间，方便后续操作
- 新增Python测试脚本：
  - `test/test_get_pod_exact.py` - 精确匹配测试
  - `test/test_get_pod_fuzzy.py` - 模糊匹配测试

### Changed
- 优化查询工具的namespace参数描述，明确默认搜索所有命名空间
- 所有删除工具的namespace参数改为必填，强制先查询再删除，提高安全性
- 更新 `.gitignore`，添加更多编译产物和临时文件规则

### Fixed
- 修复AI助手在K8S查询时只搜索default命名空间的问题
- 修复删除操作可能误删default命名空间资源的安全隐患

### Removed
- 清理编译产物和敏感配置文件
- 移除旧的测试脚本 `test_get_pod_filter.py`（已拆分）

---

## [1.2.4] - 2026-01-27

### Initial Release
- 基础K8S MCP服务器功能
- 支持查询、创建、更新、删除K8S资源
- HTTP API接口
- 审计日志功能
