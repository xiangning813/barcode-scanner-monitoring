# Go 扫码枪对接程序 (WebSocket版)

这是一个使用Go语言实现的扫码枪对接程序，通过Win32 API键盘钩子来监听扫码枪输入，并通过WebSocket实时推送数据到前端。

## 功能特性

- **全局键盘钩子监听**: 使用Win32 API的低级键盘钩子监听所有键盘输入
- **智能条码识别**: 基于按键时间间隔和回车键自动识别条码输入
- **多种条码类型支持**: 支持EAN-8、EAN-13、UPC-A、ITF-14、Code 128等常见条码格式
- **实时输入显示**: 实时显示扫码枪输入的字符
- **WebSocket实时推送**: 通过WebSocket将扫码数据实时推送到前端
- **Web测试界面**: 内置Web测试页面，可直观查看扫码结果
- **多客户端支持**: 支持多个前端客户端同时连接
- **业务逻辑扩展**: 可根据条码内容执行不同的业务逻辑
- **优雅退出**: 支持Ctrl+C优雅退出程序

## 技术实现

- **编程语言**: Go 1.21+
- **系统API**: Windows Win32 API
- **WebSocket库**: gorilla/websocket
- **核心技术**: 
  - `SetWindowsHookEx` - 安装低级键盘钩子
  - `syscall` 包 - 调用Windows API
  - `unsafe` 包 - 处理C结构体
  - 消息循环 - 处理Windows消息
  - WebSocket服务器 - 实时数据推送
  - HTTP服务器 - 提供测试页面

## 编译和运行

### 前置要求

- Go 1.21 或更高版本
- Windows 操作系统
- 管理员权限（推荐，用于全局钩子）
- 网络浏览器（用于测试WebSocket功能）

### 编译程序

```bash
# 进入项目目录
cd go-demo

# 下载依赖
go mod tidy

# 编译程序
go build -o barcode-scanner.exe main.go
```

### 运行程序

```bash
# 直接运行
go run main.go

# 或运行编译后的可执行文件
.\barcode-scanner.exe
```

## 使用说明

1. **启动程序**: 运行程序后会显示欢迎信息并启动键盘钩子和WebSocket服务器
2. **访问测试页面**: 打开浏览器访问 `http://localhost:8080` 查看实时扫码数据
3. **WebSocket连接**: 前端可连接到 `ws://localhost:8080/ws` 接收实时数据
4. **扫码测试**: 使用扫码枪扫描条码，程序会自动识别并推送到前端
5. **手动测试**: 也可以手动输入字符+回车来模拟扫码枪输入
6. **退出程序**: 按 `Ctrl+C` 优雅退出程序

## 条码识别原理

程序通过以下机制识别条码输入：

1. **时间间隔检测**: 扫码枪输入速度很快，字符间隔通常小于100毫秒
2. **结束符检测**: 大多数扫码枪会在条码末尾发送回车键
3. **长度验证**: 过滤掉过短的输入（少于3个字符）
4. **字符类型**: 主要处理数字、字母和常见符号

## 支持的条码类型

- **EAN-8**: 8位数字条码
- **UPC-A**: 12位数字条码
- **EAN-13**: 13位数字条码
- **ITF-14**: 14位数字条码
- **Code 128**: 字母数字混合条码
- **其他类型**: 通用条码处理

## 配置选项

可以通过修改以下常量来调整程序行为：

```go
const (
    BARCODE_TIMEOUT_MS  = 100    // 扫码枪输入超时时间（毫秒）
    MIN_BARCODE_LENGTH  = 3      // 最小条码长度
    WEBSOCKET_PORT      = ":8080" // WebSocket服务器端口
)
```

## WebSocket数据格式

### 发送到客户端的消息格式

**欢迎消息**:
```json
{
  "type": "welcome",
  "message": "WebSocket连接成功，等待扫码数据...",
  "time": "2024-01-01T12:00:00Z"
}
```

**条码数据**:
```json
{
  "type": "barcode",
  "data": {
    "content": "1234567890123",
    "length": 13,
    "type": "EAN-13",
    "timestamp": "2024-01-01T12:00:00Z",
    "status": "success",
    "message": "识别为EAN-13条码，正在验证..."
  }
}
```

## 业务逻辑扩展

在 `processBarcode` 函数中可以添加自定义的业务逻辑：

```go
func processBarcode(barcode string) {
    // 创建条码数据结构
    barcodeData := BarcodeData{
        Content:   barcode,
        Length:    len(barcode),
        Type:      getBarcodeType(barcode),
        Timestamp: time.Now(),
        Status:    "success",
    }
    
    // 根据条码内容执行不同操作
    if strings.HasPrefix(barcode, "PRD") {
        barcodeData.Message = "产品条码处理逻辑"
        // 添加产品查询逻辑
    } else if len(barcode) == 13 && isAllDigits(barcode) {
        barcodeData.Message = "EAN-13条码验证逻辑"
        // 添加EAN-13验证逻辑
    }
    
    // 推送到前端
    broadcastToClients(barcodeData)
}
```

## 前端集成示例

### JavaScript WebSocket客户端

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = function() {
    console.log('WebSocket连接已建立');
};

ws.onmessage = function(event) {
    const data = JSON.parse(event.data);
    
    if (data.type === 'barcode') {
        const barcode = data.data;
        console.log('收到条码:', barcode.content);
        // 处理条码数据
        handleBarcodeData(barcode);
    }
};

function handleBarcodeData(barcode) {
    // 在页面上显示条码信息
    document.getElementById('barcode-display').innerHTML = `
        <p>条码: ${barcode.content}</p>
        <p>类型: ${barcode.type}</p>
        <p>时间: ${new Date(barcode.timestamp).toLocaleString()}</p>
    `;
}
```

## 注意事项

1. **权限要求**: 全局键盘钩子可能需要管理员权限
2. **性能影响**: 键盘钩子会轻微影响系统性能
3. **安全考虑**: 程序会监听所有键盘输入，请确保在受信任的环境中使用
4. **兼容性**: 仅支持Windows系统

## 与C#版本对比

| 特性 | Go版本 | C#版本 |
|------|--------|--------|
| 编译后大小 | 较小 | 较大 |
| 启动速度 | 快 | 中等 |
| 内存占用 | 低 | 中等 |
| 开发复杂度 | 中等 | 简单 |
| 跨平台性 | 有限 | 有限 |
| 生态系统 | 丰富 | 非常丰富 |

## 故障排除

### 常见问题

1. **钩子安装失败**: 尝试以管理员身份运行程序
2. **无法检测扫码枪**: 检查扫码枪是否配置为键盘模式
3. **字符识别错误**: 调整 `getCharFromVirtualKey` 函数的映射关系
4. **程序无响应**: 检查消息循环是否正常运行

### 调试建议

- 添加更多日志输出来跟踪程序执行
- 使用调试器检查钩子回调函数的执行
- 测试不同类型的扫码枪兼容性

## 许可证

本项目仅供学习和测试使用。