// 原生 WebSocket 测试客户端（无 socket.io，直连 gorilla/websocket）
// 使用: node simple-test.js [cland-cid]
const WebSocket = require('ws');

const clandCid = process.argv[2] || '1';
const url = `ws://localhost:8081/ws?cland-cid=${clandCid}`;

console.log(`🔌 连接: ${url}`);
const ws = new WebSocket(url);

ws.on('open', () => {
  console.log('✅ 连接成功');

  const msg = {
    msgType: 1,
    sessionId: 'test-session',
    msgId: 'msg-001',
    src: 'U:user_001',
    dst: 'room:test-room',
    content: 'Hello from native ws test',
    contentType: 1,
    ts: Date.now().toString(),
    status: 1
  };
  ws.send(JSON.stringify(msg));
  console.log('📤 已发送:', JSON.stringify(msg, null, 2));
});

ws.on('message', (data) => {
  console.log('📥 收到消息:', data.toString());
});

ws.on('close', (code, reason) => {
  console.log('❌ 连接关闭. code:', code, 'reason:', reason.toString());
});

ws.on('error', (err) => {
  console.error('⚠️ 错误:', err.message);
});

// ws 库自动应答服务端 Ping 帧，无需手动处理心跳
console.log('（心跳由 ws 库自动处理）');
