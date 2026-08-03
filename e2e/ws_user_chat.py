#!/usr/bin/env python3
"""用户通信 WebSocket 端到端测试（原生 ws，无 socket.io）

前置: 服务已启动 (HTTP :8080 / WS :8081)，数据库已初始化
运行: python3 e2e/ws_user_chat.py [--host 127.0.0.1] [--port 8081]
依赖: pip3 install websocket-client
"""
import argparse
import json
import sys
import time

import websocket

PASS = "✅"
FAIL = "❌"


def parse_args():
    p = argparse.ArgumentParser(description="用户通信 WS e2e")
    p.add_argument("--host", default="127.0.0.1")
    p.add_argument("--port", default="8081")
    return p.parse_args()


def connect(url, cid):
    """建立连接（带重试）"""
    last_err = None
    for _ in range(5):
        try:
            ws = websocket.create_connection(f"{url}?cland-cid={cid}", timeout=10)
            return ws
        except Exception as e:
            last_err = e
            time.sleep(1)
    raise last_err


def recv_json(ws, timeout=5):
    """读取并解析 JSON 消息"""
    ws.settimeout(timeout)
    return json.loads(ws.recv())


def send_msg(ws, msg):
    ws.send(json.dumps(msg, ensure_ascii=False))


def expect_no_message(ws, wait=1.0):
    """断言一段时间内无消息到达"""
    ws.settimeout(wait)
    try:
        data = ws.recv()
        return False, f"意外收到消息: {data}"
    except websocket.WebSocketTimeoutException:
        return True, ""
    except Exception as e:
        return False, f"读取异常: {e}"


def main():
    args = parse_args()
    base = f"ws://{args.host}:{args.port}/ws"
    results = []

    def check(name, ok, detail=""):
        results.append((name, ok))
        print(f"{PASS if ok else FAIL} {name}" + (f"  -> {detail}" if detail else ""))

    print(f"== WS 用户通信 e2e ({base}) ==\n")

    # ---------- 1. 连接 ----------
    alice = connect(base, "user_001")
    check("A(user_001) 连接成功", True)
    bob = connect(base, "user_002")
    check("B(user_002) 连接成功", True)

    try:
        # ---------- 2. A → B 直发消息 ----------
        msg_id = f"msg-a2b-{int(time.time()*1000)}"
        content = "你好 Bob，我是 Alice"
        send_msg(alice, {
            "msgType": 1, "sessionId": "e2e-session", "msgId": msg_id,
            "src": "U:user_001", "dst": "U:user_002",
            "content": content, "contentType": 1,
            "ts": int(time.time()*1000), "status": 1,
        })
        check("A 发送消息给 B", True, msg_id)

        # B 应收到 {code:200, msg:success, data:{...}}
        try:
            got = recv_json(bob)
            ok = (got.get("code") == 200 and
                  got.get("data", {}).get("content") == content and
                  got.get("data", {}).get("dst") == "U:user_002")
            check("B 收到 A 的消息", ok, json.dumps(got, ensure_ascii=False))
        except Exception as e:
            check("B 收到 A 的消息", False, str(e))

        # ---------- 3. B → A 回发消息（双向） ----------
        msg_id2 = f"msg-b2a-{int(time.time()*1000)}"
        send_msg(bob, {
            "msgType": 1, "sessionId": "e2e-session", "msgId": msg_id2,
            "src": "U:user_002", "dst": "U:user_001",
            "content": "收到收到", "contentType": 1,
            "ts": int(time.time()*1000), "status": 1,
        })
        check("B 发送消息给 A", True, msg_id2)
        try:
            got = recv_json(alice)
            ok = (got.get("code") == 200 and
                  got.get("data", {}).get("content") == "收到收到")
            check("A 收到 B 的消息", ok, json.dumps(got, ensure_ascii=False))
        except Exception as e:
            check("A 收到 B 的消息", False, str(e))

        # ---------- 4. B 发送已读回执 (msgType=3 ACK) ----------
        send_msg(bob, {
            "msgType": 3, "sessionId": "e2e-session", "msgId": msg_id,
            "src": "U:user_002", "dst": "U:user_001",
            "content": "", "contentType": 1,
            "ts": int(time.time()*1000), "status": 1,
        })
        ok, detail = expect_no_message(bob)
        check("B 已读回执处理无异常", ok, detail)

        # ---------- 5. A → 离线用户 C，不应报错 ----------
        send_msg(alice, {
            "msgType": 1, "sessionId": "e2e-session",
            "msgId": f"msg-a2c-{int(time.time()*1000)}",
            "src": "U:user_001", "dst": "U:user_003",
            "content": "离线消息", "contentType": 1,
            "ts": int(time.time()*1000), "status": 1,
        })
        ok, detail = expect_no_message(alice)
        check("A 发送给离线用户 C 无异常", ok, detail)

        # ---------- 6. 非法 JSON → 统一错误响应 ----------
        alice.send("not-json{")
        try:
            got = recv_json(alice)
            ok = got.get("code") == 50010010000
            check("非法 JSON 返回统一错误码", ok, json.dumps(got, ensure_ascii=False))
        except Exception as e:
            check("非法 JSON 返回统一错误码", False, str(e))

        # ---------- 7. 缺少 cland-cid 拒绝连接 ----------
        try:
            websocket.create_connection(f"{base}", timeout=5)
            check("缺少 cland-cid 拒绝连接", False, "意外连接成功")
        except Exception:
            check("缺少 cland-cid 拒绝连接", True)

    finally:
        alice.close()
        bob.close()

    # ---------- 汇总 ----------
    failed = [r for r in results if not r[1]]
    print(f"\n结果: {len(results)-len(failed)}/{len(results)} 通过")
    sys.exit(1 if failed else 0)


if __name__ == "__main__":
    main()
