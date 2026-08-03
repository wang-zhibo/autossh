package app

import (
	"autossh/src/utils"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/pkg/errors"
)

type ResType int

const (
	ResTypeSrc ResType = iota
	ResTypeDst
)

type TransferObject struct {
	raw     string  // 原始数据，如 vagrant:/root/example.txt
	resType ResType // 类型，ResTypeSrc-源，ResTypeDst-目的
	server  *Server // 服务器，当raw为本地址地时，该字段为空
	path    string  // 从raw解析得到的文件路径，如 /root/example.txt
}

type Cp struct {
	isDir bool
	cfg   *Config

	sources []*TransferObject
	target  *TransferObject
}

var transferSequence atomic.Uint64

// 复制
func showCp(configFile string, args []string) {
	var err error
	cfg, err := loadConfig(configFile)
	if err != nil {
		utils.Errorln(err)
		return
	}

	cp := Cp{cfg: cfg}
	if err := cp.parse(args); err != nil {
		utils.Errorln(err)
		return
	}

	var dstIoClient IOClient
	if cp.target.server == nil {
		dstIoClient = new(LocalIOClient)
	} else {
		sftpConnection, err := cp.target.server.GetSftpClient()
		if err != nil {
			utils.Errorln(err)
			return
		}

		defer func() {
			_ = sftpConnection.Close()
		}()

		c := SftpIOClient{SftpClient: sftpConnection.Client}
		dstIoClient = &c
	}

	if err := cp.prepareTarget(dstIoClient); err != nil {
		utils.Errorln(err)
		return
	}

	for _, source := range cp.sources {
		var srcIoClient IOClient
		var sftpConnection *SftpConnection

		if source.server == nil {
			srcIoClient = new(LocalIOClient)
		} else {
			sftpConnection, err = source.server.GetSftpClient()
			if err != nil {
				cp.printFileError(source.path, err)
				continue
			}

			srcIoClient = &SftpIOClient{SftpClient: sftpConnection.Client}
		}

		func() {
			defer func() {
				if sftpConnection != nil {
					_ = sftpConnection.Close()
				}
			}()

			if file, err := cp.transferNew(srcIoClient, dstIoClient, source.path, cp.target.path, ""); err != nil {
				cp.printFileError(file, err)
			}
		}()
	}
}

func (cp *Cp) prepareTarget(dstIO IOClient) error {
	if len(cp.sources) < 2 {
		return nil
	}

	info, err := dstIO.Stat(cp.target.path)
	if err == nil {
		if !info.IsDir() {
			return errors.New("复制多个源时，目标必须是目录")
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("读取目标路径失败: %w", err)
	}
	if !cp.isDir {
		return errors.New("复制多个源时，目标目录必须已存在；使用 -r 可自动创建目录")
	}
	if err := dstIO.MkdirAll(cp.target.path); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}
	return nil
}

// 解析参数
func (cp *Cp) parse(args []string) error {
	fs := flag.NewFlagSet("cp", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&cp.isDir, "r", false, "文件夹")
	if err := fs.Parse(args); err != nil {
		return err
	}

	restArgs := fs.Args()
	length := len(restArgs)
	var err error

	if len(restArgs) < 2 {
		return errors.New("请输入完整参数")
	}

	cp.target, err = newTransferObject(cp.cfg, restArgs[length-1])
	if err != nil {
		return err
	}

	cp.sources = make([]*TransferObject, 0)
	for _, arg := range restArgs[:length-1] {
		s, err := newTransferObject(cp.cfg, arg)
		if err != nil {
			return err
		}

		if s.resType == ResTypeSrc && s.resType == cp.target.resType {
			return errors.New("源和目标不能同时为本地地址")
		}

		cp.sources = append(cp.sources, s)
	}

	return nil
}

// ioCopy 将打开的源文件写入确定的目标文件名。内容先写入同目录临时文件，
// 传输完整后再重命名，失败时删除临时文件，避免破坏已有目标文件。
func (cp *Cp) ioCopy(dstIO IOClient, srcFile FileLike, dst string) (string, error) {
	temporaryDst := fmt.Sprintf("%s.autossh-part-%d-%d", dst, os.Getpid(), transferSequence.Add(1))
	dstFile, err := dstIO.Create(temporaryDst)
	if err != nil {
		return dst, err
	}

	completed := false
	defer func() {
		_ = dstFile.Close()
		if !completed {
			_ = dstIO.Remove(temporaryDst)
		}
	}()

	var bytesCount int64
	filename := path.Base(srcFile.Name())
	startTime := time.Now()
	lastPrint := time.Now()

	srcFileInfo, err := srcFile.Stat()
	if err != nil {
		return srcFile.Name(), err
	}

	bytes := make([]byte, 64*1024)
	for {
		n, err := srcFile.Read(bytes[:])
		eof := err == io.EOF
		if err != nil && err != io.EOF {
			return srcFile.Name(), err
		}

		if n > 0 {
			wn, writeErr := dstFile.Write(bytes[:n])
			if writeErr != nil {
				return srcFile.Name(), writeErr
			}
			if wn != n {
				return srcFile.Name(), io.ErrShortWrite
			}
			bytesCount += int64(wn)

			process := 100.0
			if size := srcFileInfo.Size(); size > 0 {
				process = float64(bytesCount) / float64(size) * 100
			}
			speed := float64(bytesCount) / time.Since(startTime).Seconds()
			if time.Since(lastPrint) >= time.Second && !eof {
				cp.printProcess(filename, process, startTime, speed)
				lastPrint = time.Now()
			}
		}

		if eof {
			speed := float64(bytesCount) / time.Since(startTime).Seconds()
			cp.printProcess(filename, 100.0, startTime, speed)
			break
		}
	}

	if err := dstFile.Close(); err != nil {
		return srcFile.Name(), err
	}
	if err := dstIO.Rename(temporaryDst, dst); err != nil {
		return srcFile.Name(), fmt.Errorf("提交传输文件失败: %w", err)
	}
	completed = true
	fmt.Println("")
	return "", nil
}

// 传输
// 上传时，src = 本地，dst = 远程
// 下载时，src = 远程，dst = 本地
func (cp *Cp) transferNew(srcIO IOClient, dstIO IOClient, src string, dst string, _ string) (string, error) {
	srcInfo, err := srcIO.Lstat(src)
	if err != nil {
		return src, err
	}
	if srcInfo.Mode()&os.ModeSymlink != 0 {
		return src, errors.New("不支持复制符号链接")
	}

	if srcInfo.IsDir() {
		if !cp.isDir {
			return src, errors.New("是一个目录")
		}
		dstDir, err := cp.resolveDestinationDirectory(dstIO, dst)
		if err != nil {
			return dst, err
		}
		return cp.copyDirectoryContents(srcIO, dstIO, src, dstDir)
	}

	dstFile, err := cp.resolveDestinationFile(dstIO, srcInfo.Name(), dst)
	if err != nil {
		return dst, err
	}
	return cp.copyFile(srcIO, dstIO, src, dstFile)
}

func (cp *Cp) copyDirectoryContents(srcIO IOClient, dstIO IOClient, srcDir string, dstDir string) (string, error) {
	childFiles, err := srcIO.ReadDir(srcDir)
	if err != nil {
		return srcDir, err
	}
	for _, childFile := range childFiles {
		if file, err := cp.copyPath(srcIO, dstIO, path.Join(srcDir, childFile.Name()), path.Join(dstDir, childFile.Name())); err != nil {
			return file, err
		}
	}
	return "", nil
}

func (cp *Cp) copyPath(srcIO IOClient, dstIO IOClient, src string, dst string) (string, error) {
	srcInfo, err := srcIO.Lstat(src)
	if err != nil {
		return src, err
	}
	if srcInfo.Mode()&os.ModeSymlink != 0 {
		return src, errors.New("不支持复制符号链接")
	}
	if srcInfo.IsDir() {
		if err := dstIO.MkdirAll(dst); err != nil {
			return dst, fmt.Errorf("创建目录失败: %w", err)
		}
		return cp.copyDirectoryContents(srcIO, dstIO, src, dst)
	}
	return cp.copyFile(srcIO, dstIO, src, dst)
}

func (cp *Cp) copyFile(srcIO IOClient, dstIO IOClient, src string, dst string) (string, error) {
	srcFile, err := srcIO.Open(src)
	if err != nil {
		return src, err
	}
	defer func() { _ = srcFile.Close() }()

	if _, err := srcFile.Stat(); err != nil {
		return srcFile.Name(), err
	}
	return cp.ioCopy(dstIO, srcFile, dst)
}

func (cp *Cp) resolveDestinationDirectory(dstIO IOClient, dst string) (string, error) {
	info, err := dstIO.Stat(dst)
	if err == nil {
		if !info.IsDir() {
			return dst, errors.New("目标路径不是目录")
		}
		return dst, nil
	}
	if !os.IsNotExist(err) {
		return dst, err
	}
	if err := dstIO.MkdirAll(dst); err != nil {
		return dst, err
	}
	return dst, nil
}

func (cp *Cp) resolveDestinationFile(dstIO IOClient, srcName string, dst string) (string, error) {
	info, err := dstIO.Stat(dst)
	if err == nil {
		if info.IsDir() {
			return path.Join(dst, srcName), nil
		}
		return dst, nil
	}
	if !os.IsNotExist(err) {
		return dst, err
	}

	parent := path.Dir(dst)
	if parent != "." && parent != "/" {
		if err := dstIO.MkdirAll(parent); err != nil {
			return dst, err
		}
	}
	return dst, nil
}

func (cp *Cp) printProcess(name string, process float64, startTime time.Time, speed float64) {
	execTime := time.Now().Sub(startTime)

	type winSize struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}
	ws := &winSize{}
	retCode, _, _ := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdin),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)))

	padding := 0
	if int(retCode) != -1 {
		padding = int(ws.Col) - utils.ZhLen(name) - 40
		if padding < 0 {
			padding = 0
		}
	}

	extInfo := fmt.Sprintf("%.2f%%  %10s/s  %02.0f:%02.0f:%02.0f",
		process,
		utils.SizeFormat(speed),
		execTime.Hours(),
		execTime.Minutes(),
		execTime.Seconds())

	format := "\r%s%-" + strconv.Itoa(padding) + "s%40s"
	fmt.Printf(format, name, "", extInfo)
}

func (cp *Cp) printFileError(name string, err error) {
	fmt.Println(name, ": ", err)
}

// 创建传输对象
func newTransferObject(cfg *Config, raw string) (*TransferObject, error) {
	obj := TransferObject{
		raw: raw,
	}

	args := strings.SplitN(raw, ":", 2)
	switch len(args) {
	case 1:
		if strings.TrimSpace(args[0]) == "" {
			return nil, errors.New("本地路径不能为空")
		}
		obj.resType = ResTypeSrc
		obj.path = args[0]
	case 2:
		obj.path = strings.TrimSpace(args[1])
		if obj.path == "" {
			return nil, errors.New("远程路径不能为空")
		}
		serverName := normalizeLookupKey(args[0])
		serverIndex, exists := cfg.serverIndex[serverName]
		if !exists {
			return nil, errors.New("服务器" + args[0] + "不存在")
		}
		obj.resType = ResTypeDst
		obj.server = serverIndex.server

	default:
		return nil, errors.New(raw + " 格式错误")
	}

	return &obj, nil
}
