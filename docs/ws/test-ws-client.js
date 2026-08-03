// 原生 WebSocket 交互式测试客户端（无 socket.io）
// 使用: node test-ws-client.js [--cland-cid=1]
const WebSocket = require('ws');
const readline = require('readline');

// 命令行参数（默认值）
const args = process.argv.slice(2).reduce((acc, arg) => {
    const [key, value] = arg.split('=');
    acc[key.slice(2)] = value;
    return acc;
}, {
    serverUrl: 'ws://localhost:8081/ws',
    clandCid: '1',
    eventData: '{"msgType":1,"sessionId":"test-session","msgId":"msg-001","src":"U:user_001","dst":"room:test-room","content":"Hello World","contentType":1,"ts":"1745690716604","status":1}'
});

const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
    prompt: 'ws-test> '
});

const url = `${args.serverUrl}?cland-cid=${args.clandCid}`;
const ws = new WebSocket(url);

// 日志输出
function log(type, message) {
    const prefixes = {
        info: '\x1b[36m[INFO]\x1b[0m',
        send: '\x1b[35m[SEND]\x1b[0m',
        receive: '\x1b[34m[RECV]\x1b[0m',
        error: '\x1b[31m[ERROR]\x1b[0m'
    };
    console.log(`${prefixes[type]} [${new Date().toLocaleTimeString()}] ${message}`);
}

// 连接成功
ws.on('open', () => {
    log('info', `连接成功！${url}`);
    log('info', '支持命令：');
    log('info', '  send [json]  → 发送消息（缺省用默认模板）');
    log('info', '  help         → 显示消息格式帮助');
    log('info', '  exit         → 退出');
    rl.prompt();
});

// 收到服务端消息
ws.on('message', (data) => {
    log('receive', data.toString());
    rl.prompt();
});

// 断开连接
ws.on('close', (code, reason) => {
    log('info', `连接关闭. code=${code} reason=${reason.toString()}`);
    process.exit(0);
});

// 连接错误
ws.on('error', (err) => {
    log('error', `连接错误：${err.message}`);
    process.exit(1);
});

// 命令行输入处理
rl.on('line', (input) => {
    const cmd = input.trim().split(/\s+/);
    if (!cmd[0]) {
        rl.prompt();
        return;
    }

    switch (cmd[0]) {
        // 发送消息（默认使用模板 JSON）
        case 'send': {
            let data = cmd.slice(1).join(' ') || args.eventData;
            try {
                JSON.parse(data); // 校验 JSON
                ws.send(data);
                log('send', data);
            } catch (e) {
                log('error', `JSON 格式错误：${e.message}`);
            }
            break;
        }

        // 帮助
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

        case 'exit': {
            ws.close();
            break;
        }

        default:
            log('error', '未知命令！支持：send/help/exit');
    }

    rl.prompt();
});

process.on('uncaughtException', (err) => {
    log('error', `未捕获错误：${err.message}`);
    rl.prompt();
});
