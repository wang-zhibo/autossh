# autossh

一个SSH远程客户端，可一键登录远程服务器，主要用来弥补Mac/Linux Terminal SSH无法保存密码的不足。

## 当前版本

v1.1.1

## v1.1.1 更新

- 强化配置、日志和安装目录权限，默认启用 SSH HostKey 校验。
- 改进 SSH/SFTP 的连接超时、IPv6、心跳与资源关闭处理。
- 优化 `cp -r`，支持空目录，拒绝符号链接循环，并在失败时清理临时文件。
- 密码编辑时不再回显；路径解析不再依赖 Shell。
- 新增测试、竞态检测和 GitHub Actions CI；构建产物附带 SHA-256 校验文件。



## 功能说明
- SSH 快速登录
- 支持 cp 命令文件/文件夹复制功能 `autossh cp source:/file target:/file`
- 支持自动更新检测功能 `autossh upgrade`
- 新增快捷登录功能 `autossh [序号/别名]`

## 安装
- Mac/Linux 用户直接解压安装包并运行 `./install`。安装完成后重新打开终端，再运行 `autossh`。
- 配置文件保存在 `~/.autossh/config.json`，安装程序会保留已有配置并限制其仅当前用户可读写。
