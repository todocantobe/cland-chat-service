const { io } = require('socket.io-client');

// 测试客户端：补全 Ping/Pong 交互和协议兼容配置
const socket = io('http://localhost:8081', {
  query: { 'cland-cid': '1' },
  reconnection: false,
  transports: ['websocket'],
  pingInterval: 20000, // 改为 20 秒（和服务端主动 Ping 间隔一致）
  pingTimeout: 5000,
  autoUnref: false, // 禁用自动取消引用，确保心跳正常
  forceNew: true // 强制创建新连接，避免复用旧连接
});

// 1. 连接成功回调
socket.on('connect', () => {
  console.log('✅ Connected successfully! Socket ID:', socket.id);
  
  // 发送测试消息（保持你的原有逻辑）
  const testMessage = {
    msgType: 1,
    sessionId: "test-session",
    msgId: "msg-001",
    src: "U:user_001",
    dst: "room:test-room",
    content: "Hello from simple test",
    contentType: 1,
    ts: Date.now().toString(),
    status: 1
  };
  socket.emit('message', testMessage); // 注意：需用 socket.emit 发送事件（Socket.IO 标准）
  console.log('📤 Sent test message:', testMessage);
});

// 2. 关键：响应服务端的 Ping 消息（Engine.IO 协议要求）
socket.io.on('ping', () => {
  console.log('🔄 Received ping from server, sending pong...');
  socket.io.emit('pong'); // 必须回复 Pong，否则服务端会判定超时
});

// 3. 监听服务端发送的消息（可选，用于接收响应）
socket.on('message', (data) => {
  console.log('📥 Received message from server:', data);
});

// 4. 监听连接关闭事件（查看关闭原因）
socket.on('disconnect', (reason, details) => {
  console.log('❌ Disconnected. Reason:', reason);
  console.log('Close code:', details.code); // 查看关闭码（1005 或其他）
  console.log('Close description:', details.description);
});

// 5. 监听错误事件（排查连接问题）
socket.on('error', (err) => {
  console.error('⚠️ Socket error:', err);
});