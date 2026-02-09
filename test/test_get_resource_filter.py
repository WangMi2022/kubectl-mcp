#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
测试 get_resource_filter 和 get_pods 工具
验证 Pod 查询功能是否正常工作
"""

import requests
import json
import sys

# 默认配置
DEFAULT_URL = "http://localhost:8000"
TEST_POD_NAME = "easydata-d8bfc4f87-qc4ct"


def call_tool(base_url: str, tool_name: str, arguments: dict) -> dict:
    """调用 MCP 工具"""
    url = f"{base_url}/tool"
    payload = {
        "name": tool_name,
        "arguments": arguments
    }
    
    print(f"\n{'='*60}")
    print(f"调用工具: {tool_name}")
    print(f"参数: {json.dumps(arguments, ensure_ascii=False, indent=2)}")
    print(f"{'='*60}")
    
    try:
        response = requests.post(url, json=payload, timeout=30)
        result = response.json()
        
        if response.status_code == 200 and result.get("success"):
            print(f"✅ 成功")
            if "data" in result:
                print(f"返回数据: {json.dumps(result['data'], ensure_ascii=False, indent=2)}")
        else:
            print(f"❌ 失败")
            print(f"错误: {result.get('error', '未知错误')}")
        
        return result
    except Exception as e:
        print(f"❌ 请求异常: {e}")
        return {"success": False, "error": str(e)}


def test_health(base_url: str) -> bool:
    """检查服务健康状态"""
    try:
        response = requests.get(f"{base_url}/health", timeout=5)
        if response.status_code == 200:
            data = response.json()
            print(f"✅ 服务状态: {data.get('status', 'unknown')}")
            print(f"   版本: {data.get('version', 'unknown')}")
            return True
    except Exception as e:
        print(f"❌ 服务不可用: {e}")
    return False


def main():
    # 解析参数
    base_url = sys.argv[1] if len(sys.argv) > 1 else DEFAULT_URL
    pod_name = sys.argv[2] if len(sys.argv) > 2 else TEST_POD_NAME
    
    print(f"MCP 服务地址: {base_url}")
    print(f"测试 Pod 名称: {pod_name}")
    print()
    
    # 1. 检查服务健康状态
    print("=" * 60)
    print("1. 检查服务健康状态")
    print("=" * 60)
    if not test_health(base_url):
        print("服务不可用，退出测试")
        sys.exit(1)
    
    # 2. 测试 get_pods (使用 name 参数)
    print("\n" + "=" * 60)
    print("2. 测试 get_pods (使用 name 参数精确匹配)")
    print("=" * 60)
    result = call_tool(base_url, "get_pods", {"name": pod_name})
    # get_pods 返回的 data 直接是数组
    pods_by_name = result.get("data", []) if result.get("success") and isinstance(result.get("data"), list) else []
    print(f"   找到 Pod 数量: {len(pods_by_name)}")
    
    # 3. 测试 get_resource_filter (精确匹配)
    print("\n" + "=" * 60)
    print("3. 测试 get_resource_filter (精确匹配)")
    print("=" * 60)
    result = call_tool(base_url, "get_resource_filter", {
        "kind": "Pod",
        "name": pod_name,
        "matchMode": "exact"
    })
    
    # 4. 测试 get_resource_filter (模糊匹配 - 使用前缀)
    prefix = pod_name.split("-")[0] if "-" in pod_name else pod_name[:8]
    print("\n" + "=" * 60)
    print(f"4. 测试 get_resource_filter (模糊匹配, 前缀: {prefix})")
    print("=" * 60)
    result = call_tool(base_url, "get_resource_filter", {
        "kind": "Pod",
        "name": prefix,
        "matchMode": "fuzzy"
    })
    
    # 5. 测试 get_pods (不带 name，搜索所有)
    print("\n" + "=" * 60)
    print("5. 测试 get_pods (搜索所有 Pod，限制返回)")
    print("=" * 60)
    result = call_tool(base_url, "get_pods", {})
    total_pods = len(result.get("data", [])) if result.get("success") and isinstance(result.get("data"), list) else 0
    print(f"   集群中 Pod 总数: {total_pods}")
    
    # 6. 对比测试：使用 labelSelector
    print("\n" + "=" * 60)
    print("6. 测试 get_pods (使用 labelSelector)")
    print("=" * 60)
    # 从 pod_name 提取可能的 app 标签
    app_name = pod_name.split("-")[0] if "-" in pod_name else pod_name
    result = call_tool(base_url, "get_pods", {"labelSelector": f"app={app_name}"})
    
    print("\n" + "=" * 60)
    print("测试完成")
    print("=" * 60)


if __name__ == "__main__":
    main()
