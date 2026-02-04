#!/usr/bin/env python3
"""
测试 get_pod_filter 工具 - 精确匹配模式
通过 HTTP API 调用 kubectl-mcp 服务
"""

import requests
import json
import sys


def test_get_pod_exact(pod_name, base_url="http://localhost:8000"):
    """
    测试 get_pod_filter 工具 - 精确匹配
    
    Args:
        pod_name: Pod 完整名称
        base_url: kubectl-mcp 服务地址
    """
    print(f"=" * 60)
    print(f"测试 get_pod_filter 工具 - 精确匹配")
    print(f"查询 Pod: {pod_name}")
    print(f"=" * 60)
    
    # 构建请求
    url = f"{base_url}/tool"
    payload = {
        "name": "get_pod_filter",
        "arguments": {
            "name": pod_name,
            "matchMode": "exact"
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
                            match_mode = data.get("matchMode", "exact")
                            print(f"✅ 找到 {count} 个匹配的 Pod")
                            print(f"匹配模式: {match_mode}")
                            print(f"消息: {data.get('message', '')}")
                            
                            if count == 1:
                                # 单个 Pod
                                pod = data.get("pod", {})
                                namespace = data.get("namespace", "")
                                print(f"\n🎯 目标 Pod 定位成功！")
                                print(f"\nPod 详细信息:")
                                print(f"  名称: {pod.get('name')}")
                                print(f"  命名空间: {pod.get('namespace')}")
                                print(f"  状态: {pod.get('status')}")
                                print(f"  阶段: {pod.get('phase')}")
                                print(f"  IP: {pod.get('ip')}")
                                print(f"  节点: {pod.get('node')}")
                                print(f"  重启次数: {pod.get('restarts')}")
                                print(f"  创建时间: {pod.get('createdAt')}")
                                
                                containers = pod.get('containers', [])
                                if containers:
                                    print(f"\n  容器信息 ({len(containers)} 个):")
                                    for i, container in enumerate(containers, 1):
                                        print(f"    {i}. {container.get('name')}")
                                        print(f"       镜像: {container.get('image')}")
                                        print(f"       就绪: {container.get('ready')}")
                                        print(f"       重启: {container.get('restartCount')}")
                                        print(f"       状态: {container.get('state')}")
                                
                                # 提供后续操作建议
                                print(f"\n💡 后续操作:")
                                print(f"   删除此 Pod:")
                                print(f"   kubectl delete pod {pod.get('name')} -n {namespace}")
                                print(f"   或使用 delete_pod 工具:")
                                print(f"   {{'name': 'delete_pod', 'arguments': {{'name': '{pod.get('name')}', 'namespace': '{namespace}'}}}}")
                            else:
                                # 多个同名 Pod（不同命名空间）
                                pods = data.get("pods", [])
                                namespaces = data.get("namespaces", [])
                                print(f"\n⚠️  发现多个同名 Pod，分布在不同命名空间:")
                                for i, pod in enumerate(pods, 1):
                                    print(f"  {i}. {pod.get('name')} (命名空间: {pod.get('namespace')})")
                                    print(f"     状态: {pod.get('status')} | IP: {pod.get('ip')} | 节点: {pod.get('node')}")
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
    # 默认要查询的 Pod 名称
    default_pod_name = "easydata-d8bfc4f87-qc4ct"
    
    # 从命令行参数获取 Pod 名称
    if len(sys.argv) > 1:
        pod_name = sys.argv[1]
    else:
        pod_name = default_pod_name
        print(f"使用默认 Pod 名称: {pod_name}")
        print(f"提示: 可以通过命令行参数指定 Pod 名称")
        print(f"用法: python test_get_pod_exact.py <pod_name> [base_url]\n")
    
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
    
    # 测试精确查询 Pod
    print("\n")
    success = test_get_pod_exact(pod_name, base_url)
    
    print("\n" + "=" * 60)
    if success:
        print("✅ 测试完成")
    else:
        print("❌ 测试失败")
    print("=" * 60)


if __name__ == "__main__":
    main()
