package scanner

import (
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"
	
	"github.com/sirupsen/logrus"
	
	"userclient/internal/config"
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

// BarcodeHandler 条码处理器接口
type BarcodeHandler interface {
	HandleBarcode(barcode string) error
}

// Hook 键盘钩子管理器
type Hook struct {
	hook          uintptr
	barcodeBuffer strings.Builder
	lastKeyTime   time.Time
	isRunning     bool
	config        *config.ScannerConfig
	handler       BarcodeHandler
	logger        *logrus.Logger
}

// NewHook 创建新的键盘钩子管理器
func NewHook(cfg *config.ScannerConfig, handler BarcodeHandler, logger *logrus.Logger) *Hook {
	return &Hook{
		config:    cfg,
		handler:   handler,
		logger:    logger,
		isRunning: false,
	}
}

// Install 安装键盘钩子
func (h *Hook) Install() error {
	if !h.config.EnableHook {
		h.logger.Info("键盘钩子已禁用")
		return nil
	}
	
	// 获取模块句柄
	moduleHandle, _, _ := getModuleHandle.Call(0)
	if moduleHandle == 0 {
		return fmt.Errorf("获取模块句柄失败")
	}
	
	// 安装钩子
	hookProc := syscall.NewCallback(h.keyboardHookProc)
	hookHandle, _, _ := setWindowsHookEx.Call(
		uintptr(WH_KEYBOARD_LL),
		hookProc,
		moduleHandle,
		0,
	)
	
	if hookHandle == 0 {
		return fmt.Errorf("安装键盘钩子失败")
	}
	
	h.hook = hookHandle
	h.isRunning = true
	h.logger.Info("键盘钩子已启动，等待扫码枪输入...")
	return nil
}

// Uninstall 卸载键盘钩子
func (h *Hook) Uninstall() {
	if h.hook != 0 {
		unhookWindowsHookEx.Call(h.hook)
		h.hook = 0
		h.isRunning = false
		h.logger.Info("键盘钩子已停止")
	}
}

// IsRunning 检查钩子是否运行中
func (h *Hook) IsRunning() bool {
	return h.isRunning
}

// MessageLoop 消息循环
func (h *Hook) MessageLoop() {
	var msg MSG
	for h.isRunning {
		ret, _, _ := getMessage.Call(
			uintptr(unsafe.Pointer(&msg)),
			0,
			0,
			0,
		)
		
		if ret == 0 { // WM_QUIT
			break
		} else if ret == ^uintptr(0) { // -1, error
			h.logger.Error("获取消息时出错")
			break
		}
		
		translateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		dispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

// Stop 停止钩子
func (h *Hook) Stop() {
	h.isRunning = false
	h.Uninstall()
}

// keyboardHookProc 键盘钩子回调函数
func (h *Hook) keyboardHookProc(nCode int, wParam uintptr, lParam uintptr) uintptr {
	if nCode >= HC_ACTION && wParam == WM_KEYDOWN {
		// 获取键盘结构体
		kbStruct := (*KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))
		vkCode := kbStruct.VkCode
		
		currentTime := time.Now()
		timeDiff := currentTime.Sub(h.lastKeyTime).Milliseconds()
		
		// 如果按键间隔太长，清空缓冲区
		if timeDiff > int64(h.config.TimeoutMS) {
			h.barcodeBuffer.Reset()
		}
		
		h.lastKeyTime = currentTime
		
		// 处理字符键
		if h.isCharacterKey(vkCode) {
			if ch := h.getCharFromVirtualKey(vkCode); ch != 0 {
				h.barcodeBuffer.WriteByte(ch)
				fmt.Printf("%c", ch) // 实时显示输入
			}
		} else if vkCode == 0x0D { // 回车键
			barcode := h.barcodeBuffer.String()
			if len(barcode) >= h.config.MinLength && len(barcode) <= h.config.MaxLength {
				fmt.Printf("\n检测到条码: %s\n", barcode)
				if h.handler != nil {
					if err := h.handler.HandleBarcode(barcode); err != nil {
						h.logger.WithError(err).Error("处理条码失败")
					}
				}
			}
			h.barcodeBuffer.Reset()
		}
	}
	
	// 调用下一个钩子
	ret, _, _ := callNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
	return ret
}

// isCharacterKey 判断是否为字符键
func (h *Hook) isCharacterKey(vkCode uint32) bool {
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

// getCharFromVirtualKey 从虚拟键码获取字符
func (h *Hook) getCharFromVirtualKey(vkCode uint32) byte {
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