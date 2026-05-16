package heybox

import (
	"fmt"
	"heybox-bot/config"
	"heybox-bot/heybox/api"
	"heybox-bot/logger"
	"os"
	"sync"
	"time"

	"github.com/mdp/qrterminal/v3"
	qrcode "github.com/skip2/go-qrcode"
)

var (
	saved_sess *session
	runMu      sync.Mutex
	stopCh     chan struct{}
	doneCh     chan struct{}
)

// outputQRCode 在终端和本地图片文件中输出登录二维码。
func outputQRCode(qrURL string) error {
	qrterminal.GenerateHalfBlock(qrURL, qrterminal.L, os.Stdout)

	err := qrcode.WriteFile(qrURL, qrcode.Medium, 256, "QRcode.png")
	if err != nil {
		return fmt.Errorf("保存 QRCode.png 失败: %w", err)
	}

	return nil
}

// getLoginState 轮询二维码状态直到登录成功或超时。
func getLoginState(qrID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("二维码已超时")
		}
		state, err := api.GetQRCodeState(qrID)

		if err != nil {
			err = fmt.Errorf("获取二维码状态失败: %w", err)
			logger.Info("%v", err)
			return err
		}

		switch state.Error {
		case "wait":
			logger.Info("等待扫码...")
		case "ready":
			logger.Info("扫码成功，等待在 APP 中确认登录...")
		case "ok":
			saved_sess = newSession(state)
			if err := saved_sess.save(); err != nil {
				return fmt.Errorf("保存会话失败: %w", err)
			}
			return nil
		default:
			if state.ErrorMsg != "" {
				return fmt.Errorf("二维码状态异常: %s, %s", state.Error, state.ErrorMsg)
			}
			return fmt.Errorf("二维码状态异常: %s", state.Error)
		}

		time.Sleep(1500 * time.Millisecond)
	}
}

// loginSkipQRCode 尝试使用已保存会话跳过二维码登录。
func loginSkipQRCode() error {
	var err error
	saved_sess, err = loadSession()
	if err != nil {
		logger.Error("加载会话失败，将使用二维码登录")
		return err
	}
	api.SetCookies(saved_sess.Cookies)
	result, _, err := api.GetUserPermission(saved_sess.HeyboxID)
	if err != nil {
		logger.Error("请求用户权限失败: %v", err)
		return err
	}
	if result {
		logger.Info("权限校验通过，无需二维码登录")
		return nil
	}
	return fmt.Errorf("会话权限无效")
}

// login 完成会话复用或二维码登录流程。
func login() error {
	if err := loginSkipQRCode(); err == nil {
		return nil
	}
	qrInfo, err := api.GetQRCode()
	if err != nil {
		err = fmt.Errorf("获取二维码失败: %w", err)
		logger.Info("%v", err)
		return err
	}
	logger.Info("二维码地址: %v，%v 秒后过期", qrInfo.QRURL, qrInfo.Expire)

	if err = outputQRCode(qrInfo.QRURL); err != nil {
		err = fmt.Errorf("输出二维码失败: %w", err)
		logger.Error("%v", err)
		return err
	}

	if err = getLoginState(qrInfo.QRID, qrInfo.Expire); err != nil {
		err = fmt.Errorf("获取登录状态失败: %w", err)
		logger.Error("%v", err)
		return err
	}

	return nil
}

// Run 初始化元数据、登录账号并启动机器人后台循环。
func Run() error {
	if err := initMetadata(); err != nil {
		err = fmt.Errorf("初始化元数据失败: %v", err)
		logger.Error("%v", err)
		return err
	}

	if err := login(); err != nil {
		err = fmt.Errorf("登录失败: %v", err)
		logger.Error("%v", err)
		return err
	} else {
		logger.Info("登录成功")
	}

	if err := AddImmediateTimer(atMessageTimerName, config.GetBotInitWaitTime(), getAtMessageCb); err != nil {
		logger.Error("添加定时器 %q 失败: %v", atMessageTimerName, err)
	}
	if err := initRefuseTimer(); err != nil {
		logger.Error("初始化拒绝服务定时器失败: %v", err)
	}
	runMu.Lock()
	if stopCh != nil {
		runMu.Unlock()
		return nil
	}
	stopCh = make(chan struct{})
	doneCh = make(chan struct{})
	stop := stopCh
	done := doneCh
	resetScheduledJobs()
	runMu.Unlock()

	go runLoop(stop, done)

	return nil
}

// Stop 停止机器人后台循环并等待退出完成。
func Stop() {
	runMu.Lock()
	stop := stopCh
	done := doneCh
	if stop == nil {
		runMu.Unlock()
		return
	}
	stopCh = nil
	doneCh = nil
	close(stop)
	runMu.Unlock()

	<-done
}
