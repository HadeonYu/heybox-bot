# heybot-bot 小黑盒机器人

特性：
1. 支持配置一个纯文本模型和额外一个图片理解模型
2. 支持白名单和频率限制两种模式
3. 支持配置和系统提示词热更新
4. 支持字节火山引擎、kimi、anthropic、deepseek、openai和其他兼容openai的大模型厂商
5. 支持空闲时自动退避，降低无效请求频率

## 0. 声明
> 本项目仅供学习交流使用, 用户如何使用与作者无关

> golang只是爱好，非专业，若发现问题欢迎提issue或直接pr

## 1. 前言
起因是在小黑盒上刷到了这篇帖子 [在小黑盒上做了个grok](https://www.xiaoheihe.cn/app/bbs/link/180965857)，于是我突发奇想决定做一个真的会回复评论的AI。没想到等主体框架写完了，回头发现原作者已经开发出来了：[SomeOvO/xhhRobot](https://github.com/SomeOvO/xhhRobot)。但是既然已经写好框架了，我还是决定继续写完，然后就有了这个项目。

## 2. 使用方法
### 2.1 下载Release
在右边的release界面选择自己的系统架构的发行版，下载并解压。

目前的发行版有：x86-64架构的Windows和Linux系统、arm64架构的MacOS系统（Apple Silicon）。如有需要，可以参考下文 [参与开发](#3-参与开发)自行编译适合自己系统的可执行文件。
### 2.2 填入配置
配置存放在 `config.yaml` 文件中，请参考 [配置说明](./src/config/README.md) 填写自己的配置

填写完成后可以执行以下命令验证AI配置（和其他配置）是否正确：
- Linux/MacOS：
```bash
./heybox-bot --llm_test
```
- Windows:
1. 在文件资源管理器打开解压后的文件夹，点击文件夹路径后面的空白部分：
![windows进入cmd-step1](./assets/windows进入cmd-step1.png)
2. 输出cmd并按回车：
![windows进入cmd-step1](./assets/windows进入cmd-step2.png)
3. 在弹出的命令提示符窗口中执行：
```cmd
.\heybox-bot.exe --llm_test
```
如果输出结果不正确，请根据输出修改配置

### 2.3 启动
第一次启动需要扫码登录。Linux/MacOS系统可以扫终端输出的二维码，也可以扫生成的QRcode.png
Windows用户终端输出的二维码可能用不了，只能扫生成的QRcode.png

程序启动之前的@消息不会被处理，第一次启动后的@消息程序都会处理。

Linux/MacOS用户可以参考Makefile中的 `run_detach` 和 `exit` ，用screen工具后台运行，Windows暂不支持后台运行（因为我还不会）

### 2.4 退出
前台运行可以直接按 `Ctrl + c` 或 `Ctrl + \` 退出，若是后台运行，可以用一些手段向进程发送这些信号退出：
`SIGINT`, `SIGTERM`, `SIGABRT`, `SIGHUP`, `SIGQUIT`，或者极端一点的 `SIGKILL`


## 3. 参与开发
如果您愿意参与开发，希望您能遵守以下约定：
1. 在dev分支开发
2. git提交信息有意义，参考如下格式：
```
feat: 新特性
fix: 修复bug
doc: 文档相关
......
```
### 3.1 项目结构
```
.
├── Makefile    编译、打包工具
├── bin    可执行文件
├── config-example.yaml    配置文件
├── metadata    运行时信息
├── package     release文件
├── src
│   ├── go.mod
│   ├── go.sum
│   ├── config    配置项
│   ├── db        实现数据库接口
│   ├── heybox    机器人源码
│   ├── llm       大模型厂商适配
│   ├── logger    日志
│   └── main.go   入库函数
├── system_prompt.md
```
### 3.2 开发环境
- golang 1.23.0及以上版本
- GNU Make

### 3.3 增加功能
#### 3.3.1 bot新功能
bot用定时器实现功能，定时器源码见 `src/heybox/job.go` 。可以编写定时器回调函数，在bot启动前加入定时器，在回调函数中实现新功能

#### 3.3.2 大模型适配
1. 在 `src/llm/` 文件夹下添加新的源文件，实现 `complete` 和 `response` 接口
2. 在 `src/llm/chat.go` 中根据配置中的 `vendor` 分发

### 3.4 编译，运行，打包
先把 `config-example.yaml` 复制一份，重命名为 `config.yaml`，随后参考一下指令：
```bash
# 编译：
make

# 编译并运行
make run

# 后台运行（仅Linux/MacOS）
make run_detach

# 退出后台运行（仅Linux/MacOS）
make exit

# 打包发行（需要配置tar和zip）
make release
```
