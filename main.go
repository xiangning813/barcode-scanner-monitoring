package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/gorilla/websocket"
)

// Windows API 常量
const (
	WH_KEYBOARD_LL = 13
	WM_KEYDOWN     = 0x0100
	WM_KEYUP       = 0x0101
	WM_SYSKEYDOWN  = 0x0104
	WM_SYSKEYUP    = 0x0105
	HC_ACTION      = 0
)

// Windows API 结构体
type KBDLLHOOKSTRUCT struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

type POINT struct {
	X, Y int32
}

type MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

// Windows API 函数
var (
	user32              = syscall.NewLazyDLL("user32.dll")
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	setWindowsHookEx    = user32.NewProc("SetWindowsHookExW")
	unhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	callNextHookEx      = user32.NewProc("CallNextHookEx")
	getMessage          = user32.NewProc("GetMessageW")
	translateMessage    = user32.NewProc("TranslateMessage")
	dispatchMessage     = user32.NewProc("DispatchMessageW")
	getModuleHandle     = kernel32.NewProc("GetModuleHandleW")
	getCurrentThreadId  = kernel32.NewProc("GetCurrentThreadId")
)

// 全局变量
var (
	hook          uintptr
	barcodeBuffer strings.Builder
	lastKeyTime   time.Time
	isRunning     = true
	clients       = make(map[*websocket.Conn]bool)
	upgrader      = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // 允许所有来源
		},
	}
)

// 配置常量
const (
	BARCODE_TIMEOUT_MS = 100     // 扫码枪输入超时时间（毫秒）
	MIN_BARCODE_LENGTH = 3       // 最小条码长度
	WEBSOCKET_PORT     = ":8080" // WebSocket服务器端口
)

// 键盘钩子回调函数
func keyboardHookProc(nCode int, wParam uintptr, lParam uintptr) uintptr {
	if nCode >= HC_ACTION && wParam == WM_KEYDOWN {
		// 获取键盘结构体
		kbStruct := (*KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))
		vkCode := kbStruct.VkCode

		currentTime := time.Now()
		timeDiff := currentTime.Sub(lastKeyTime).Milliseconds()

		// 如果按键间隔太长，清空缓冲区
		if timeDiff > BARCODE_TIMEOUT_MS {
			barcodeBuffer.Reset()
		}

		lastKeyTime = currentTime

		// 处理字符键
		if isCharacterKey(vkCode) {
			if ch := getCharFromVirtualKey(vkCode); ch != 0 {
				barcodeBuffer.WriteByte(ch)
				fmt.Printf("%c", ch) // 实时显示输入
			}
		} else if vkCode == 0x0D { // 回车键
			barcode := barcodeBuffer.String()
			if len(barcode) >= MIN_BARCODE_LENGTH {
				fmt.Printf("\n检测到条码: %s\n", barcode)
				processBarcode(barcode)
			}
			barcodeBuffer.Reset()
		}
	}

	// 调用下一个钩子
	ret, _, _ := callNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
	return ret
}

// 判断是否为字符键
func isCharacterKey(vkCode uint32) bool {
	return (vkCode >= 0x30 && vkCode <= 0x39) || // 数字 0-9
		(vkCode >= 0x41 && vkCode <= 0x5A) || // 字母 A-Z
		(vkCode >= 0x60 && vkCode <= 0x69) || // 小键盘数字 0-9
		vkCode == 0xBD || // 减号
		vkCode == 0xBB || // 等号/加号
		vkCode == 0xDB || // 左方括号
		vkCode == 0xDD || // 右方括号
		vkCode == 0xDC || // 反斜杠
		vkCode == 0xBA || // 分号
		vkCode == 0xDE || // 引号
		vkCode == 0xBC || // 逗号
		vkCode == 0xBE || // 句号
		vkCode == 0xBF // 斜杠
}

// 从虚拟键码获取字符
func getCharFromVirtualKey(vkCode uint32) byte {
	// 数字键 0-9
	if vkCode >= 0x30 && vkCode <= 0x39 {
		return byte(vkCode)
	}

	// 字母键 A-Z
	if vkCode >= 0x41 && vkCode <= 0x5A {
		return byte(vkCode)
	}

	// 小键盘数字 0-9
	if vkCode >= 0x60 && vkCode <= 0x69 {
		return byte(vkCode - 0x60 + '0')
	}

	// 特殊字符
	switch vkCode {
	case 0xBD:
		return '-' // 减号
	case 0xBB:
		return '=' // 等号
	case 0xDB:
		return '[' // 左方括号
	case 0xDD:
		return ']' // 右方括号
	case 0xDC:
		return '\\' // 反斜杠
	case 0xBA:
		return ';' // 分号
	case 0xDE:
		return '\'' // 引号
	case 0xBC:
		return ',' // 逗号
	case 0xBE:
		return '.' // 句号
	case 0xBF:
		return '/' // 斜杠
	default:
		return 0
	}
}

// 条码数据结构
type BarcodeData struct {
	Content   string    `json:"content"`
	Length    int       `json:"length"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
}

// 处理扫描到的条码
func processBarcode(barcode string) {
	timestamp := time.Now()
	fmt.Printf("[%s] 扫码成功!\n", timestamp.Format("15:04:05"))
	fmt.Printf("条码内容: %s\n", barcode)
	fmt.Printf("条码长度: %d\n", len(barcode))
	fmt.Printf("条码类型: %s\n", getBarcodeType(barcode))
	fmt.Println(strings.Repeat("-", 50))

	// 创建条码数据
	barcodeData := BarcodeData{
		Content:   barcode,
		Length:    len(barcode),
		Type:      getBarcodeType(barcode),
		Timestamp: timestamp,
		Status:    "success",
	}

	// 业务逻辑处理
	if strings.HasPrefix(barcode, "PRD") {
		barcodeData.Message = "识别为产品条码，正在查询产品信息..."
		fmt.Println(barcodeData.Message)
	} else if len(barcode) == 13 && isAllDigits(barcode) {
		barcodeData.Message = "识别为EAN-13条码，正在验证..."
		fmt.Println(barcodeData.Message)
	} else if len(barcode) == 12 && isAllDigits(barcode) {
		barcodeData.Message = "识别为UPC-A条码，正在处理..."
		fmt.Println(barcodeData.Message)
	} else {
		barcodeData.Message = "通用条码，正在记录..."
		fmt.Println(barcodeData.Message)
	}

	// 推送到前端
	broadcastToClients(barcodeData)

	// 模拟处理延时
	time.Sleep(100 * time.Millisecond)
	fmt.Println("处理完成!\n")
}

// 获取条码类型
func getBarcodeType(barcode string) string {
	if barcode == "" {
		return "未知"
	}

	switch {
	case len(barcode) == 8 && isAllDigits(barcode):
		return "EAN-8"
	case len(barcode) == 12 && isAllDigits(barcode):
		return "UPC-A"
	case len(barcode) == 13 && isAllDigits(barcode):
		return "EAN-13"
	case len(barcode) == 14 && isAllDigits(barcode):
		return "ITF-14"
	case isAlphaNumeric(barcode):
		return "Code 128"
	default:
		return "其他类型"
	}
}

// 检查是否全为数字
func isAllDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// 检查是否为字母数字字符
func isAlphaNumeric(s string) bool {
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '-' || r == '.') {
			return false
		}
	}
	return true
}

// 安装键盘钩子
func installHook() error {
	// 获取模块句柄
	moduleHandle, _, _ := getModuleHandle.Call(0)
	if moduleHandle == 0 {
		return fmt.Errorf("获取模块句柄失败")
	}

	// 安装钩子
	hookProc := syscall.NewCallback(keyboardHookProc)
	hookHandle, _, _ := setWindowsHookEx.Call(
		uintptr(WH_KEYBOARD_LL),
		hookProc,
		moduleHandle,
		0,
	)

	if hookHandle == 0 {
		return fmt.Errorf("安装键盘钩子失败")
	}

	hook = hookHandle
	fmt.Println("键盘钩子已启动，等待扫码枪输入...")
	return nil
}

// 卸载键盘钩子
func uninstallHook() {
	if hook != 0 {
		unhookWindowsHookEx.Call(hook)
		hook = 0
		fmt.Println("键盘钩子已停止")
	}
}

// 消息循环
// WebSocket处理函数
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket升级失败: %v", err)
		return
	}
	defer conn.Close()

	// 添加客户端连接
	clients[conn] = true
	fmt.Printf("新的WebSocket客户端连接，当前连接数: %d\n", len(clients))

	// 发送欢迎消息
	welcomeMsg := map[string]interface{}{
		"type":    "welcome",
		"message": "WebSocket连接成功，等待扫码数据...",
		"time":    time.Now(),
	}
	conn.WriteJSON(welcomeMsg)

	// 监听客户端消息
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Printf("读取WebSocket消息失败: %v", err)
			delete(clients, conn)
			fmt.Printf("客户端断开连接，当前连接数: %d\n", len(clients))
			break
		}
	}
}

// 广播消息到所有客户端
func broadcastToClients(data BarcodeData) {
	if len(clients) == 0 {
		return
	}

	message := map[string]interface{}{
		"type": "barcode",
		"data": data,
	}

	for client := range clients {
		err := client.WriteJSON(message)
		if err != nil {
			log.Printf("发送消息到客户端失败: %v", err)
			client.Close()
			delete(clients, client)
		}
	}

	fmt.Printf("条码数据已推送到 %d 个客户端\n", len(clients))
}

// 启动WebSocket服务器
func startWebSocketServer() {
	http.HandleFunc("/ws", handleWebSocket)

	// 提供静态文件服务（可选）
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, getTestHTML())
		} else {
			http.NotFound(w, r)
		}
	})

	fmt.Printf("WebSocket服务器启动在端口 %s\n", WEBSOCKET_PORT)
	fmt.Printf("测试页面: http://localhost%s\n", WEBSOCKET_PORT)
	fmt.Printf("WebSocket地址: ws://localhost%s/ws\n", WEBSOCKET_PORT)

	go func() {
		if err := http.ListenAndServe(WEBSOCKET_PORT, nil); err != nil {
			log.Printf("WebSocket服务器启动失败: %v", err)
		}
	}()
}

// 获取测试HTML页面
func getTestHTML() string {
	return `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>扫码枪WebSocket测试</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .container { max-width: 800px; margin: 0 auto; }
        .status { padding: 10px; margin: 10px 0; border-radius: 5px; }
        .connected { background-color: #d4edda; color: #155724; }
        .disconnected { background-color: #f8d7da; color: #721c24; }
        .barcode-item { 
            border: 1px solid #ddd; 
            padding: 15px; 
            margin: 10px 0; 
            border-radius: 5px;
            background-color: #f9f9f9;
        }
        .timestamp { color: #666; font-size: 0.9em; }
        .barcode-content { font-weight: bold; font-size: 1.2em; color: #007bff; }
        .barcode-info { margin: 5px 0; }
        #messages { max-height: 400px; overflow-y: auto; }
    </style>
</head>
<body>
    <div class="container">
        <h1>扫码枪WebSocket实时监控</h1>
        <div id="status" class="status disconnected">未连接</div>
        <div id="messages"></div>
    </div>

    <script>
        const status = document.getElementById('status');
        const messages = document.getElementById('messages');
        
        const ws = new WebSocket('ws://localhost:8080/ws');
        
        ws.onopen = function() {
            status.textContent = 'WebSocket已连接，等待扫码数据...';
            status.className = 'status connected';
        };
        
        ws.onmessage = function(event) {
            const data = JSON.parse(event.data);
            
            if (data.type === 'welcome') {
                console.log('收到欢迎消息:', data.message);
            } else if (data.type === 'barcode') {
                displayBarcode(data.data);
            }
        };
        
        ws.onclose = function() {
            status.textContent = 'WebSocket连接已断开';
            status.className = 'status disconnected';
        };
        
        ws.onerror = function(error) {
            status.textContent = 'WebSocket连接错误';
            status.className = 'status disconnected';
            console.error('WebSocket错误:', error);
        };
        
        function displayBarcode(barcode) {
            const item = document.createElement('div');
            item.className = 'barcode-item';
            
            const timestamp = new Date(barcode.timestamp).toLocaleString();
            
            item.innerHTML = ` + "`" + `
                <div class="timestamp">${timestamp}</div>
                <div class="barcode-content">条码: ${barcode.content}</div>
                <div class="barcode-info">长度: ${barcode.length} | 类型: ${barcode.type}</div>
                <div class="barcode-info">状态: ${barcode.status} | ${barcode.message}</div>
            ` + "`" + `;
            
            messages.insertBefore(item, messages.firstChild);
            
            // 限制显示的条码数量
            while (messages.children.length > 20) {
                messages.removeChild(messages.lastChild);
            }
        }
    </script>
</body>
</html>`
}

func messageLoop() {
	var msg MSG
	for isRunning {
		ret, _, _ := getMessage.Call(
			uintptr(unsafe.Pointer(&msg)),
			0,
			0,
			0,
		)

		if ret == 0 { // WM_QUIT
			break
		} else if ret == ^uintptr(0) { // -1, error
			fmt.Println("获取消息时出错")
			break
		}

		translateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		dispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func main() {
	fmt.Println("=== Go 扫码枪对接程序 (WebSocket版) ===")
	fmt.Println("基于Win32 API键盘钩子实现")
	fmt.Println("支持WebSocket实时推送到前端")
	fmt.Println("按 Ctrl+C 退出程序\n")

	// 启动WebSocket服务器
	startWebSocketServer()

	// 等待服务器启动
	time.Sleep(1 * time.Second)

	// 设置信号处理
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("\n正在退出程序...")
		isRunning = false
		uninstallHook()
		// 关闭所有WebSocket连接
		for client := range clients {
			client.Close()
		}
		os.Exit(0)
	}()

	// 安装键盘钩子
	if err := installHook(); err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}

	defer uninstallHook()

	// 启动消息循环
	messageLoop()

	fmt.Println("程序已退出")
}
