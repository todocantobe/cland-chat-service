const { io } = require('socket.io-client');
const readline = require('readline');

// 命令行参数解析（默认值，新增 clandCid 配置）
const args = process.argv.slice(2).reduce((acc, arg) => {
    const [key, value] = arg.split('=');
    acc[key.slice(2)] = value;
    return acc;
}, {
    serverUrl: 'http://localhost:8081',
    namespace: '/',
    eventName: 'message',
    eventData: '{"msgType":1,"sessionId":"test-session","msgId":"msg-001","src":"U:user_001","dst":"room:test-room","content":"Hello World","contentType":1,"ts":"1745690716604","status":1}',
    roomName: 'test-room',
    clandCid: '1' // 新增：cland-cid 默认值（通过命令行 --cland-cid=1 覆盖）
});

// 命令行交互
const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
    prompt: 'socket-io-test> '
});

// 连接 Socket.IO 服务器
const socket = io(args.serverUrl + args.namespace, {
    reconnection: false,
    query: {
        'cland-cid': args.clandCid
    }
});

// 日志输出
function log(type, message) {
    const prefixes = {
        info: '\x1b[36m[INFO]\x1b[0m', // 青色
        send: '\x1b[35m[SEND]\x1b[0m',  // 紫色
        receive: '\x1b[34m[RECV]\x1b[0m',// 蓝色
        error: '\x1b[31m[ERROR]\x1b[0m'  // 红色
    };
    console.log(`${prefixes[type]} [${new Date().toLocaleTimeString()}] ${message}`);
}

// 【核心工具函数】给事件数据追加 clandCid（仅 JSON 数据追加，不破坏原有格式）
function addClandCidToData(rawData) {
    // 如果没有传入 cland-cid，直接返回原始数据
    if (!args.clandCid) return rawData;

    try {
        // 尝试解析为 JSON，成功则追加 clandCid
        const jsonData = typeof rawData === 'string' ? JSON.parse(rawData) : rawData;
        return { ...jsonData, clandCid: args.clandCid }; // 追加参数
    } catch (e) {
        // 非 JSON 数据（如普通字符串），不追加（避免破坏数据）
        return rawData;
    }
}

// 连接成功（新增：显示 cland-cid 配置）
socket.on('connect', () => {
    log('info', `连接成功！Socket ID: ${socket.id}`);
    log('info', `当前配置：服务器=${args.serverUrl}，命名空间=${args.namespace}，cland-cid=${args.clandCid || '未设置'}`);
    log('info', '支持命令：');
    log('info', '  send [事件名] [数据] → 发送事件（自动附加 cland-cid）');
    log('info', '  join [房间名]       → 加入房间');
    log('info', '  leave [房间名]      → 离开房间');
    log('info', '  broadcast [房间名] [事件名] [数据] → 广播事件（自动附加 cland-cid）');
    log('info', '  set cland-cid [值]  → 动态修改 cland-cid（如：set cland-cid=2）');
    log('info', '  help                → 显示消息格式帮助');
    log('info', '  exit                → 退出');
    rl.prompt();
});

// 连接失败
socket.on('connect_error', (err) => {
    log('error', `连接失败：${err.message}`);
    process.exit(1);
});

// 断开连接
socket.on('disconnect', (reason) => {
    log('info', `断开连接：${reason}`);
    process.exit(0);
});

// 监听所有服务器事件
socket.onAny((eventName, ...args) => {
    log('receive', `事件 [${eventName}]：${JSON.stringify(args, null, 2)}`);
    rl.prompt();
});

// 命令行输入处理（新增 set 命令，优化 send/broadcast 自动附加 cland-cid）
rl.on('line', (input) => {
    const cmd = input.trim().split(/\s+/);
    if (!cmd[0]) {
        rl.prompt();
        return;
    }

    switch (cmd[0]) {
        // 发送事件：自动附加 cland-cid（核心修改）
        case 'send': {
            const eventName = cmd[1] || args.eventName;
            let eventData = cmd[2] || args.eventData;
            // 追加 cland-cid 到事件数据
            eventData = addClandCidToData(eventData);
            
            socket.emit(eventName, eventData, (response) => {
                if (response !== undefined) {
                    log('info', `服务器响应：${JSON.stringify(response, null, 2)}`);
                }
            });
            log('send', `事件 [${eventName}]：${JSON.stringify(eventData, null, 2)}`);
            break;
        }

        // 加入房间
        case 'join': {
            const roomName = cmd[1] || args.roomName;
            socket.emit('join', roomName, (err) => {
                if (err) log('error', `加入房间失败：${err}`);
                else log('info', `成功加入房间：${roomName}`);
                rl.prompt();
            });
            break;
        }

        // 离开房间
        case 'leave': {
            const roomName = cmd[1] || args.roomName;
            socket.emit('leave', roomName, (err) => {
                if (err) log('error', `离开房间失败：${err}`);
                else log('info', `成功离开房间：${roomName}`);
                rl.prompt();
            });
            break;
        }

        // 广播事件：自动附加 cland-cid（核心修改）
        case 'broadcast': {
            const roomName = cmd[1] || args.roomName;
            const eventName = cmd[2] || args.eventName;
            let eventData = cmd[3] || args.eventData;
            // 追加 cland-cid 到事件数据
            eventData = addClandCidToData(eventData);
            
            socket.to(roomName).emit(eventName, eventData);
            log('send', `向房间 [${roomName}] 广播事件 [${eventName}]：${JSON.stringify(eventData, null, 2)}`);
            break;
        }

        // 新增命令：动态修改 cland-cid（无需重启脚本）
        case 'set': {
            if (cmd[1] === 'cland-cid' && cmd[2]) {
                args.clandCid = cmd[2];
                log('info', `已设置 cland-cid 为：${cmd[2]}`);
            } else {
                log('error', 'set 命令用法：set cland-cid [值]（如：set cland-cid=3）');
            }
            break;
        }

        // 显示帮助
        case 'help': {
            log('info', '消息格式示例：');
            log('info', '  {"msgType":1,"sessionId":"test-session","msgId":"msg-001","src":"U:user_001","dst":"room:test-room","content":"Hello World","contentType":1,"ts":"1745690716604","status":1}');
            log('info', '字段说明：');
            log('info', '  msgType: 1=普通消息, 2=通知, 3=确认');
            log('info', '  sessionId: 会话ID');
            log('info', '  msgId: 消息ID');
            log('info', '  src: 发送者 (U:用户, A:客服, S:系统)');
            log('info', '  dst: 接收者 (用户ID 或 room:房间名)');
            log('info', '  content: 消息内容');
            log('info', '  contentType: 1=文本, 2=图片, 3=文件, 520=转人工');
            log('info', '  ts: 时间戳 (Unix毫秒)');
            log('info', '  status: 1=新建, 2=历史, 3=离线, 4=撤回, 5=已发送, 6=已送达, 7=已读');
            break;
        }

        // 退出
        case 'exit': {
            socket.disconnect();
            break;
        }

        default:
            log('error', '未知命令！支持：send/join/leave/broadcast/set/help/exit');
    }

    rl.prompt();
});

// 错误处理
process.on('uncaughtException', (err) => {
    log('error', `未捕获错误：${err.message}`);
    rl.prompt();
});
