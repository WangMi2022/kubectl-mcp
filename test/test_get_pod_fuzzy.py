#!/usr/bin/env python3
"""
测试 get_pod_filter 工具 - 模糊匹配模式
通过 HTTP API 调用 kubectl-mcp 服务
"""

import requests
import json
import sys


def test_get_pod_fuzzy(keyword, base_url="http://localhost:8000"):
    """
    测试 get_pod_filter 工具 - 模糊匹配
    
    Args:
        keyword: Pod 名称关键词（部分名称）
        base_url: kubectl-mcp 服务地址
    """
    print(f"=" * 60)
    print(f"测试 get_pod_filter 工具 - 模糊匹配")
    print(f"搜索关键词: {keyword}")
    print(f"=" * 60)
    
    # 构建请求
    url = f"{base_url}/tool"
    payload = {
        "name": "get_pod_filter",
        "arguments": {
            "name": keyword,
            "matchMode": "fuzzy"
        }
    }
    
    headers = {
        "Content-Type": "application/json"
    }
    
    try:
        # 发送请求
        print(f"\n发送请求到: {url}")
        print(f"请求体: {json.dumps(payload, indent=2, ensure_ascii=False)}")
        
        response = requests.post(url, json=payload, headers=headers, timeout=10)
        
        print(f"\n响应状态码: {response.status_code}")
        
        if response.status_code == 200:
            result = response.json()
            print(f"\n响应内容:")
            print(json.dumps(result, indent=2, ensure_ascii=False))
            
            # 解析结果
            if "content" in result and len(result["content"]) > 0:
                content = result["content"][0]
                if content.get("type") == "text":
                    text_content = content.get("text", "")
                    try:
                        data = json.loads(text_content)
                        print(f"\n" + "=" * 60)
                        print("解析后的结果:")
                        print("=" * 60)
                        
                        if data.get("found"):
                            count = data.get("count", 0)
                            match_mode = data.get("matchMode", "fuzzy")
                            print(f"✅ 找到 {count} 个匹配的 Pod")
                            print(f"匹配模式: {match_mode}")
                            print(f"消息: {data.get('message', '')}")
                            
                            pods = data.get("pods", [])
                            namespaces = data.get("namespaces", [])
                            
                            # 按命名空间分组显示
                            pods_by_ns = {}
                            for pod in pods:
                                ns = pod.get('namespace', 'unknown')
                                if ns not in pods_by_ns:
                                    pods_by_ns[ns] = []
                                pods_by_ns[ns].append(pod)
                            
                            print(f"\n📋 找到的 Pod 列表（按命名空间分组）:")
                            for ns, ns_pods in sorted(pods_by_ns.items()):
                                print(f"\n  命名空间: {ns} ({len(ns_pods)} 个)")
                                for i, pod in enumerate(ns_pods, 1):
                                    print(f"    {i}. {pod.get('name')}")
                                    print(f"       状态: {pod.get('status')} | 阶段: {pod.get('phase')}")
                                    print(f"       IP: {pod.get('ip')} | 节点: {pod.get('node')}")
                                    print(f"       重启: {pod.get('restarts')} 次 | 创建: {pod.get('createdAt')}")
                                    
                                    containers = pod.get('containers', [])
                                    if containers:
                                        container_names = [c.get('name') for c in containers]
                                        print(f"       容器: {', '.join(container_names)}")
                            
                            # 统计信息
                            print(f"\n📊 统计信息:")
                            print(f"   总计: {count} 个 Pod")
                            print(f"   分布: {len(pods_by_ns)} 个命名空间")
                            
                            # 状态统计
                            status_count = {}
                            for pod in pods:
                                status = pod.get('status', 'Unknown')
                                status_count[status] = status_count.get(status, 0) + 1
                            print(f"   状态分布: {', '.join([f'{k}={v}' for k, v in status_count.items()])}")
                            
                        else:
                            print(f"❌ 未找到匹配的 Pod")
                    except json.JSONDecodeError:
                        print(f"\n原始文本内容:\n{text_content}")
            
            return True
        else:
            print(f"\n❌ 请求失败")
            print(f"错误信息: {response.text}")
            return False
            
    except requests.exceptions.ConnectionError:
        print(f"\n❌ 连接失败: 无法连接到 {base_url}")
        print(f"请确保 kubectl-mcp 服务正在运行")
        return False
    except requests.exceptions.Timeout:
        print(f"\n❌ 请求超时")
        return False
    except Exception as e:
        print(f"\n❌ 发生错误: {str(e)}")
        import traceback
        traceback.print_exc()
        return False


def test_health_check(base_url="http://localhost:8000"):
    """测试服务健康状态"""
    print(f"\n检查服务健康状态...")
    try:
        response = requests.get(f"{base_url}/health", timeout=5)
        if response.status_code == 200:
            data = response.json()
            print(f"✅ 服务状态: {data.get('status')}")
            print(f"   版本: {data.get('version', 'unknown')}")
            return True
        else:
            print(f"❌ 服务异常: HTTP {response.status_code}")
            return False
    except Exception as e:
        print(f"❌ 无法连接到服务: {str(e)}")
        return False


def main():
    """主函数"""
    # 默认搜索关键词
    default_keyword = "nginx"
    
    # 从命令行参数获取关键词
    if len(sys.argv) > 1:
        keyword = sys.argv[1]
    else:
        keyword = default_keyword
        print(f"使用默认搜索关键词: {keyword}")
        print(f"提示: 可以通过命令行参数指定搜索关键词")
        print(f"用法: python test_get_pod_fuzzy.py <keyword> [base_url]\n")
    
    # 服务地址
    base_url = "http://localhost:8000"
    if len(sys.argv) > 2:
        base_url = sys.argv[2]
    
    # 先检查服务健康状态
    if not test_health_check(base_url):
        print("\n请先启动 kubectl-mcp 服务:")
        print("  cd kubectl-mcp")
        print("  go run cmd/server/main.go")
        sys.exit(1)
    
    # 测试模糊搜索 Pod
    print("\n")
    success = test_get_pod_fuzzy(keyword, base_url)
    
    print("\n" + "=" * 60)
    if success:
        print("✅ 测试完成")
    else:
        print("❌ 测试失败")
    print("=" * 60)


if __name__ == "__main__":
    main()
